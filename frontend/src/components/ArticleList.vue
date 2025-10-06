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
  border-right: 1px solid #ccc;
  height: 100vh;
  overflow-y: auto;
}

h2 {
  margin-top: 0;
  font-size: 1.2rem;
  color: #333;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid #eee;
}

ul {
  list-style-type: none;
  padding: 0;
  margin: 0;
}

li {
  padding: 1rem;
  border-bottom: 1px solid #eee;
  cursor: pointer;
  transition: background-color 0.2s;
}

li:hover {
  background-color: #f0f0f0;
}

h3 {
  margin: 0 0 0.5rem 0;
  font-size: 1rem;
  color: #0056b3;
}

p {
  margin: 0;
  font-size: 0.9rem;
  color: #666;
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
  color: #888;
}
</style>