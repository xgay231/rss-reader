<script setup>
import { defineProps, computed, ref, onMounted } from 'vue';
import { marked } from 'marked';
import { loadModel, summarizeText } from '../services/summarizer.js';

// This component receives the selected article as a prop
const props = defineProps({
  article: {
    type: Object,
    default: null,
  },
});

const summary = ref('');
const isLoadingSummary = ref(false);
const summaryError = ref('');

// Load the model when the component is mounted
onMounted(async () => {
  try {
    await loadModel();
  } catch (error) {
    summaryError.value = 'Failed to load the summarization model.';
    console.error(error);
  }
});

// A computed property to parse Markdown content into HTML
const renderedContent = computed(() => {
  if (props.article && props.article.content) {
    return marked.parse(props.article.content);
  }
  return '';
});

// Function to generate the summary
const generateSummary = async () => {
  if (!props.article || !props.article.content) return;

  isLoadingSummary.value = true;
  summaryError.value = '';
  summary.value = '';

  try {
    // We need to get the plain text from the HTML content for an accurate summary
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = props.article.content;
    const textContent = tempDiv.textContent || tempDiv.innerText || '';
    
    summary.value = await summarizeText(textContent);
  } catch (error) {
    summaryError.value = 'Failed to generate summary.';
    console.error(error);
  } finally {
    isLoadingSummary.value = false;
  }
};
</script>

<template>
  <div class="article-view-container">
    <div v-if="article">
      <h1>
        <a :href="article.url" target="_blank" rel="noopener noreferrer">
          {{ article.title }}
        </a>
      </h1>

      <div class="summary-section">
        <button @click="generateSummary" :disabled="isLoadingSummary">
          {{ isLoadingSummary ? 'Generating...' : 'Generate Summary' }}
        </button>
        <div v-if="summary" class="summary-content">
          <h3>Summary</h3>
          <p>{{ summary }}</p>
        </div>
        <div v-if="summaryError" class="summary-error">
          <p>{{ summaryError }}</p>
        </div>
      </div>

      <!-- Use v-html to render the parsed HTML content -->
      <div class="article-content" v-html="renderedContent"></div>
    </div>
    <div v-else class="no-article-selected">
      <p>Select an article from the list to read it here.</p>
    </div>
  </div>
</template>

<style scoped>
.article-view-container {
  padding: 2rem;
  height: 100vh;
  overflow-y: auto;
  background-color: var(--color-background);
}

.no-article-selected {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 80%;
  color: var(--color-text-secondary);
  font-size: 1.2rem;
}

h1 {
  margin-top: 0;
  font-size: 1.8rem;
  margin-bottom: 1rem;
}

h1 a {
  color: var(--color-text-primary);
  text-decoration: none;
  transition: color 0.2s;
}

h1 a:hover {
  color: var(--color-accent);
}

.article-content {
  line-height: 1.6;
  font-size: 1rem;
  color: var(--color-text-primary);
}

/* Basic styling for content that comes from v-html */
.article-content ::v-deep(p) {
  margin-bottom: 1rem;
}

.article-content ::v-deep(a) {
  color: var(--color-accent);
}

.article-content ::v-deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
}

.summary-section {
  margin-bottom: 2rem;
  padding: 1rem;
  background-color: var(--color-bg-pane);
  border-radius: 6px;
  border: 1px solid var(--color-border);
}

.summary-section button {
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  border: none;
  margin-bottom: 1rem;
}

.summary-section button:hover {
  background-color: var(--color-accent-hover);
}

.summary-section button:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}

.summary-content h3 {
  margin-top: 0;
  font-size: 1.1rem;
  color: var(--color-text-primary);
}

.summary-content p {
  font-style: italic;
  color: var(--color-text-secondary);
}

.summary-error {
  color: var(--color-danger);
}
</style>