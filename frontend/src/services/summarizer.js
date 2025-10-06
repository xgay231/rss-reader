import * as tf from '@tensorflow/tfjs';
import * as use from '@tensorflow-models/universal-sentence-encoder';

let model = null;

/**
 * Loads the Universal Sentence Encoder model.
 * This function will only load the model once.
 */
export async function loadModel() {
  if (model) {
    return; // Model is already loaded
  }
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
 * Generates a summary for the given text content.
 * @param {string} content The full text content of the article.
 * @param {number} numSentences The desired number of sentences in the summary.
 * @returns {Promise<string>} A promise that resolves to the summarized text.
 */
export async function summarizeText(content, numSentences = 3) {
  if (!model) {
    throw new Error('Model is not loaded yet. Call loadModel() first.');
  }

  // 1. Split the content into sentences.
  const sentences = content.match(/[^.!?]+[.!?]+/g) || [];
  if (sentences.length <= numSentences) {
    return content; // Not enough sentences to summarize
  }

  // 2. Embed the sentences into vectors.
  const embeddings = await model.embed(sentences);
  const sentenceVectors = tf.unstack(embeddings);

  // 3. Calculate the centroid (average vector) of all sentences.
  const centroid = tf.mean(embeddings, 0);

  // 4. Calculate the cosine similarity of each sentence to the centroid.
  const scores = sentenceVectors.map(sentenceVector => {
    return tf.losses.cosineDistance(centroid, sentenceVector, 0).dataSync()[0];
  });

  // 5. Rank sentences by their similarity score.
  const rankedSentences = sentences
    .map((sentence, index) => ({
      sentence,
      score: scores[index],
      index,
    }))
    .sort((a, b) => a.score - b.score); // Lower cosine distance is better

  // 6. Select the top N sentences and sort them by their original order.
  const topSentences = rankedSentences
    .slice(0, numSentences)
    .sort((a, b) => a.index - b.index);

  // 7. Join the sentences to form the final summary.
  return topSentences.map(item => item.sentence).join(' ');
}