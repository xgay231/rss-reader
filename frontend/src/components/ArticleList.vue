<script setup>
import { ref, computed, defineProps } from 'vue';

// This component will receive the list of articles as a prop
const props = defineProps({
  articles: {
    type: Array,
    required: true,
  },
  currentSourceId: {
    type: String,
    default: null,
  },
});

// This component will emit an event when an article is selected
const emit = defineEmits(['article-selected', 'mark-all-read']);

const selectArticle = (article) => {
  emit('article-selected', article);
};

// Filter state
const showRead = ref(true);
const showUnread = ref(true);

const toggleShowRead = () => {
  showRead.value = !showRead.value;
};

const toggleShowUnread = () => {
  showUnread.value = !showUnread.value;
};

const markAllAsRead = async () => {
  // Emit event to parent component to mark all articles as read
  emit('mark-all-read', props.currentSourceId);
};

// Filtered articles based on filter state
const filteredArticles = computed(() => {
  return props.articles.filter((article) => {
    if (article.readStatus === 'read' && !showRead.value) return false;
    if (article.readStatus === 'unread' && !showUnread.value) return false;
    return true;
  });
});

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
    <div class="filter-bar">
      <button
        class="filter-btn"
        :class="{ active: showRead }"
        @click="toggleShowRead"
      >
        显示已读
      </button>
      <button
        class="filter-btn"
        :class="{ active: showUnread }"
        @click="toggleShowUnread"
      >
        显示未读
      </button>
      <button
        class="mark-read-btn"
        @click="markAllAsRead"
        :disabled="!showUnread"
      >
        标记已读
      </button>
    </div>
    <ul v-if="filteredArticles.length > 0">
      <li
        v-for="article in filteredArticles"
        :key="article.id"
        @click="selectArticle(article)"
        :class="{ read: article.readStatus === 'read' }"
      >
        <h3>{{ article.title }}</h3>
        <div class="article-meta">
          <span class="published-time">{{ formatTime(article.publishedAt) }}</span>
        </div>
        <p>{{ article.description }}</p>
      </li>
    </ul>
    <div v-else class="no-articles">
      <p v-if="!showRead && !showUnread">无符合筛选条件的文章</p>
      <p v-else>Select a feed to see its articles.</p>
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

li.read {
  opacity: 0.6;
  background-color: var(--color-bg-read, #f5f5f5);
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

.filter-bar {
  display: flex;
  gap: 0.5rem;
  padding: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.filter-btn {
  padding: 0.25rem 0.75rem;
  border: 1px solid var(--color-border);
  background: white;
  border-radius: 4px;
  cursor: pointer;
}

.filter-btn.active {
  background: var(--color-accent);
  color: white;
  border-color: var(--color-accent);
}

.mark-read-btn {
  margin-left: auto;
  padding: 0.25rem 0.75rem;
  background: var(--color-accent);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.mark-read-btn:disabled {
  background: #ccc;
  cursor: not-allowed;
}
</style>