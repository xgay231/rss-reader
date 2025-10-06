<script setup>
import { ref } from 'vue';
import SourceList from './components/SourceList.vue';
import ArticleList from './components/ArticleList.vue';
import ArticleView from './components/ArticleView.vue';

const articles = ref([]);
const selectedArticle = ref(null);

// This function is called when a source is selected in the SourceList component
const handleSourceSelected = async (source) => {
  if (!source) {
    articles.value = [];
    selectedArticle.value = null;
    return;
  }
  try {
    const response = await fetch(`/api/sources/${source.id}/articles`);
    if (!response.ok) {
      throw new Error('Failed to fetch articles for the source');
    }
    articles.value = await response.json();
    selectedArticle.value = null; // Reset article view when a new source is selected
  } catch (error) {
    console.error('Error fetching articles:', error);
    articles.value = []; // Clear articles on error
  }
};

// This function is called when an article is selected in the ArticleList component
const handleArticleSelected = (article) => {
  selectedArticle.value = article;
};
</script>

<template>
  <div id="app-container">
    <div class="left-pane">
      <SourceList @source-selected="handleSourceSelected" />
    </div>
    <div class="center-pane">
      <ArticleList :articles="articles" @article-selected="handleArticleSelected" />
    </div>
    <div class="right-pane">
      <ArticleView :article="selectedArticle" />
    </div>
  </div>
</template>

<style>
/* Global styles */
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
}

#app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}

.left-pane {
  flex: 0 0 280px; /* Fixed width for the source list */
  min-width: 220px;
}

.center-pane {
  flex: 0 0 350px; /* Fixed width for the article list */
  min-width: 280px;
}

.right-pane {
  flex: 1; /* Takes up the remaining space */
  min-width: 400px;
}
</style>
