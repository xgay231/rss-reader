import * as tf from '@tensorflow/tfjs';
import * as use from '@tensorflow-models/universal-sentence-encoder';
import { fetchWithAuth } from '../utils/api';

let model = null;

// Basic Chinese stop words list. In a real-world app, this could be more extensive.
const STOP_WORDS = new Set([
  '的', '了', '在', '是', '我', '有', '和', '也', '不', '就', '都', '而', '及', '与', '或', '个',
  '么', '之', '以', '将', '还', '为', '被', '从', '到', '得', '他', '她', '它', '们', '地', '等',
  '你', '我', '他', '她', '它', '咱', '这', '那', '一', '二', '三', '四', '五', '六', '七', '八', '九', '十'
]);

/**
 * Loads the Universal Sentence Encoder model.
 */
export async function loadModel() {
  if (model) return;
  try {
    console.log('Loading Universal Sentence Encoder model...');
    model = await use.load();
    console.log('Model loaded successfully.');
  } catch (error) {
    console.error('Failed to load the model:', error);
    throw new Error('Failed to load summarization model.');
  }
}

/**
 * Calculates the cosine similarity between two TensorFlow tensors.
 */
function cosineSimilarity(vecA, vecB) {
  const dotProduct = tf.dot(vecA, vecB).dataSync()[0];
  const normA = tf.norm(vecA).dataSync()[0];
  const normB = tf.norm(vecB).dataSync()[0];
  if (normA === 0 || normB === 0) {
    return 0;
  }
  return dotProduct / (normA * normB);
}

/**
 * Preprocesses text by removing stop words.
 * @param {string} text The input text.
 * @returns {string} The processed text.
 */
function preprocessText(text) {
  return text.split('').filter(char => !STOP_WORDS.has(char)).join('');
}

/**
 * Generates a summary for the given text content using TextRank and MMR.
 * @param {string} content The full text content of the article.
 * @param {number} numSentences The desired number of sentences in the summary.
 * @param {number} lambda The MMR lambda parameter (controls diversity).
 * @returns {Promise<string>} A promise that resolves to the summarized text.
 */
export async function summarizeText(content, numSentences = 3, lambda = 0.7) {
  if (!model) {
    throw new Error('Model is not loaded yet. Call loadModel() first.');
  }

  const sentences = content.match(/[^.!?]+[.!?]+/g) || [];
  if (sentences.length <= numSentences) {
    return content;
  }

  // Preprocess sentences before embedding
  const processedSentences = sentences.map(preprocessText);
  const embeddings = await model.embed(processedSentences);
  const sentenceVectors = tf.unstack(embeddings);

  // 1. Build the similarity matrix (graph) based on original sentence vectors
  const originalEmbeddings = await model.embed(sentences);
  const originalSentenceVectors = tf.unstack(originalEmbeddings);
  const similarityMatrix = [];
  for (let i = 0; i < sentences.length; i++) {
    const row = [];
    for (let j = 0; j < sentences.length; j++) {
      if (i === j) row.push(0);
      else row.push(cosineSimilarity(originalSentenceVectors[i], originalSentenceVectors[j]));
    }
    similarityMatrix.push(row);
  }

  // 2. Run the TextRank algorithm
  let scores = new Array(sentences.length).fill(1);
  const damping = 0.85;
  const epsilon = 1e-4;
  for (let iter = 0; iter < 100; iter++) {
    const newScores = new Array(sentences.length).fill(0);
    let maxDiff = 0;
    for (let i = 0; i < sentences.length; i++) {
      let weightedSum = 0;
      for (let j = 0; j < sentences.length; j++) {
        if (i !== j) {
          const sumOfOutgoingWeights = similarityMatrix[j].reduce((a, b) => a + b, 0);
          if (sumOfOutgoingWeights > 0) {
            weightedSum += (similarityMatrix[j][i] / sumOfOutgoingWeights) * scores[j];
          }
        }
      }
      newScores[i] = (1 - damping) + damping * weightedSum;
      maxDiff = Math.max(maxDiff, Math.abs(newScores[i] - scores[i]));
    }
    scores = newScores;
    if (maxDiff < epsilon) break;
  }

  const rankedSentences = sentences
    .map((sentence, index) => ({ sentence, score: scores[index], index, vector: originalSentenceVectors[index] }))
    .sort((a, b) => b.score - a.score);

  // 3. Use Maximal Marginal Relevance (MMR) to select diverse sentences
  const summarySentences = [];
  const selectedIndices = new Set();

  if (rankedSentences.length > 0) {
    const firstSentence = rankedSentences[0];
    summarySentences.push(firstSentence);
    selectedIndices.add(firstSentence.index);
  }

  while (summarySentences.length < numSentences && summarySentences.length < rankedSentences.length) {
    let bestCandidate = null;
    let maxMmrScore = -Infinity;

    for (const candidate of rankedSentences) {
      if (selectedIndices.has(candidate.index)) continue;

      const relevance = candidate.score;
      let maxSimilarityWithSelected = 0;
      for (const selected of summarySentences) {
        const similarity = cosineSimilarity(candidate.vector, selected.vector);
        if (similarity > maxSimilarityWithSelected) {
          maxSimilarityWithSelected = similarity;
        }
      }
      
      const mmrScore = lambda * relevance - (1 - lambda) * maxSimilarityWithSelected;
      if (mmrScore > maxMmrScore) {
        maxMmrScore = mmrScore;
        bestCandidate = candidate;
      }
    }

    if (bestCandidate) {
      summarySentences.push(bestCandidate);
      selectedIndices.add(bestCandidate.index);
    } else {
      break; // No more candidates to add
    }
  }

  // 4. Sort by original order and join
  return summarySentences
    .sort((a, b) => a.index - b.index)
    .map(item => item.sentence)
    .join(' ');
}

/**
 * Calls the backend API to generate a summary using an AI model.
 * @param {string} articleId The ID of the article to summarize.
 * @returns {Promise<string>} A promise that resolves to the AI-generated summary.
 */
export async function generateAISummary(articleId) {
  try {
    const response = await fetchWithAuth(`/api/articles/${articleId}/ai-summary`, {
      method: 'POST',
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to fetch AI summary');
    }

    const data = await response.json();
    return data.summary;
  } catch (error) {
    console.error('Error generating AI summary:', error);
    throw error;
  }
}