<script setup>
import { defineProps, computed, ref, watch } from "vue";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { detectContentType } from "../utils/contentDetector.js";
import { generateAISummary } from "../services/summarizer.js";

// This component receives the selected article as a prop
const props = defineProps({
  article: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['starred-changed']);

const aiSummary = ref("");
const isLoadingAISummary = ref(false);
const aiSummaryError = ref("");

// Initialize aiSummary from article prop when available
watch(() => props.article, (newArticle) => {
  if (newArticle && newArticle.summary) {
    aiSummary.value = newArticle.summary;
  } else {
    aiSummary.value = "";
  }
}, { immediate: true });

// Detect content type
const contentType = computed(() => {
  if (props.article && props.article.content) {
    return detectContentType(props.article.content);
  }
  return 'plain';
});

// Render content based on detected type
const renderedContent = computed(() => {
  if (!props.article || !props.article.content) {
    return "";
  }

  const content = props.article.content;

  switch (contentType.value) {
    case 'html':
      // Sanitize HTML to prevent XSS
      return DOMPurify.sanitize(content, {
        ALLOWED_TAGS: ['p', 'br', 'b', 'i', 'em', 'strong', 'a', 'img',
          'ul', 'ol', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
          'blockquote', 'pre', 'code', 'span', 'div', 'table',
          'thead', 'tbody', 'tr', 'th', 'td', 'hr'],
        ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'target',
          'rel', 'width', 'height'],
        FORCE_BODY: true,
      });

    case 'markdown':
      // Convert literal \n to actual newlines before parsing
      return marked.parse(content.replace(/\\n/g, '\n'));

    case 'plain':
    default:
      // Convert literal \n to actual newlines, escape HTML entities and preserve whitespace
      const escaped = content
        .replace(/\\n/g, '\n')  // Convert literal \n to actual newlines
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
      return `<pre style="white-space: pre-wrap; word-wrap: break-word;">${escaped}</pre>`;
  }
});

// Function to generate the AI summary
const generateAISummaryHandler = async () => {
  if (!props.article || !props.article.id) return;

  // If summary already exists, just display it
  if (props.article.summary) {
    aiSummary.value = props.article.summary;
    return;
  }

  isLoadingAISummary.value = true;
  aiSummaryError.value = "";
  aiSummary.value = "";

  try {
    const result = await generateAISummary(props.article.id);
    aiSummary.value = result.summary;
    props.article.summary = result.summary;
  } catch (error) {
    aiSummaryError.value = "Failed to generate AI summary.";
    console.error(error);
  } finally {
    isLoadingAISummary.value = false;
  }
};

const formatTime = (timeString) => {
  if (!timeString) return '';

  const date = new Date(timeString);
  const now = new Date();
  const diff = now - date; // milliseconds

  const minutes = Math.floor(diff / 60000);
  const hours = Math.floor(diff / 3600000);
  const days = Math.floor(diff / 86400000);

  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes} 分钟前`;
  if (hours < 24) return `${hours} 小时前`;
  if (days === 1) return '昨天';
  if (days < 7) return `${days} 天前`;

  return date.toLocaleDateString('zh-CN');
};

const toggleStar = async () => {
  if (!props.article || !props.article.id) return;

  const method = props.article.isStarred ? 'DELETE' : 'POST';
  const response = await fetch(`/api/articles/${props.article.id}/star`, {
    method,
  });

  if (response.ok) {
    props.article.isStarred = !props.article.isStarred;
    emit('starred-changed');
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
        <button
          v-if="!article.summary"
          class="ai-summary-btn"
          @click="generateAISummaryHandler"
          :disabled="isLoadingAISummary"
        >
          {{ isLoadingAISummary ? "生成中..." : "AI 总结" }}
        </button>
        <span v-else class="summary-badge">✓ 已生成总结</span>
        <button
          class="star-btn"
          :class="{ starred: article.isStarred }"
          @click="toggleStar"
        >
          {{ article.isStarred ? '★' : '☆' }}
        </button>
      </h1>
      <p class="article-time">{{ formatTime(article.publishedAt) }}</p>

      <div class="summary-section">
        <div class="summary-controls">
          <button
            @click="generateAISummaryHandler"
            :disabled="isLoadingAISummary"
          >
            {{ isLoadingAISummary ? "Generating..." : "Generate AI Summary" }}
          </button>
        </div>

        <div class="summaries-container">
          <!-- AI Summary -->
          <div class="summary-content-wrapper">
            <div v-if="aiSummary" class="summary-content">
              <h3>AI Summary</h3>
              <p>{{ aiSummary }}</p>
            </div>
            <div v-if="aiSummaryError" class="summary-error">
              <p>{{ aiSummaryError }}</p>
            </div>
          </div>
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
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

h1 a {
  color: var(--color-text-primary);
  text-decoration: none;
  transition: color 0.2s;
  flex: 1;
}

h1 a:hover {
  color: var(--color-accent);
}

.star-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
  padding: 0;
  transition: color 0.2s;
}

.star-btn.starred {
  color: #f5c518;
}

.star-btn:hover {
  color: #f5c518;
}

.ai-summary-btn {
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  border: none;
  padding: 0.3rem 0.6rem;
  border-radius: 4px;
  font-size: 0.85rem;
  cursor: pointer;
}

.ai-summary-btn:hover {
  background-color: var(--color-accent-hover);
}

.ai-summary-btn:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}

.summary-badge {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: var(--color-bg-pane);
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  border: 1px solid var(--color-border);
}

.article-time {
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  opacity: 0.7;
  margin-bottom: 1rem;
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

.summary-controls {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.summary-section button {
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  border: none;
  /* margin-bottom: 1rem; */ /* Removed to use gap in controls */
}

.summary-section button:hover {
  background-color: var(--color-accent-hover);
}

.summaries-container {
  display: flex;
  gap: 1rem;
}

.summary-content-wrapper {
  flex: 1;
  min-width: 0;
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
