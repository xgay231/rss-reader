<script setup>
import { defineProps } from 'vue';

// This component will receive the list of articles as a prop
const props = defineProps({
  articles: {
    type: Array,
    required: true,
  },
});

// This component will emit an event when an article is selected
const emit = defineEmits(['article-selected']);

const selectArticle = (article) => {
  emit('article-selected', article);
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
</script>

<template>
  <div class="article-list-container">
    <h2>Articles</h2>
    <ul v-if="articles.length > 0">
      <li
        v-for="article in articles"
        :key="article.id"
        @click="selectArticle(article)"
      >
        <h3>{{ article.title }}</h3>
        <div class="article-meta">
          <span class="published-time">{{ formatTime(article.publishedAt) }}</span>
        </div>
        <p>{{ article.description }}</p>
      </li>
    </ul>
    <div v-else class="no-articles">
      <p>Select a feed to see its articles.</p>
    </div>
  </div>
</template>

<style scoped>
.article-list-container {
  padding: 1rem;
  border-right: 1px solid var(--color-border);
  height: 100vh;
  overflow-y: auto;
  background-color: var(--color-bg-pane);
}

h2 {
  margin-top: 0;
  font-size: 1.2rem;
  color: var(--color-text-primary);
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

ul {
  list-style-type: none;
  padding: 0;
  margin: 0;
}

li {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: background-color 0.2s;
}

li:hover {
  background-color: var(--color-bg-item-hover);
}

h3 {
  margin: 0 0 0.25rem 0;
  font-size: 1rem;
  color: var(--color-accent-hover);
}

.article-meta {
  margin-bottom: 0.5rem;
}

.published-time {
  font-size: 0.8rem;
  color: var(--color-text-secondary);
  opacity: 0.7;
}

p {
  margin: 0;
  font-size: 0.9rem;
  color: var(--color-text-secondary);
  /* Truncate long descriptions */
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.no-articles {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 80%;
  color: var(--color-text-secondary);
}
</style>