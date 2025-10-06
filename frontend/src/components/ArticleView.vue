<script setup>
import { defineProps, computed } from 'vue';

// This component receives the selected article as a prop
const props = defineProps({
  article: {
    type: Object,
    default: null,
  },
});

// A computed property to safely access article properties
const articleContent = computed(() => {
  if (!props.article) {
    return {
      title: '',
      url: '',
      content: '<p>Select an article to read its content.</p>',
    };
  }
  return props.article;
});
</script>

<template>
  <div class="article-view-container">
    <div v-if="article">
      <h1>
        <a :href="articleContent.url" target="_blank" rel="noopener noreferrer">
          {{ articleContent.title }}
        </a>
      </h1>
      <!-- Use v-html to render the HTML content of the article -->
      <div class="article-content" v-html="articleContent.content"></div>
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
}

.no-article-selected {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 80%;
  color: #888;
  font-size: 1.2rem;
}

h1 {
  margin-top: 0;
  font-size: 1.8rem;
  margin-bottom: 1rem;
}

h1 a {
  color: #333;
  text-decoration: none;
  transition: color 0.2s;
}

h1 a:hover {
  color: #007bff;
}

.article-content {
  line-height: 1.6;
  font-size: 1rem;
  color: #444;
}

/* Basic styling for content that comes from v-html */
.article-content ::v-deep(p) {
  margin-bottom: 1rem;
}

.article-content ::v-deep(a) {
  color: #007bff;
}

.article-content ::v-deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
}
</style>