<script setup>
import { ref, onMounted } from 'vue';

const feedUrl = ref('');
const articles = ref([]);

const fetchArticles = async () => {
  try {
    const response = await fetch('/api/articles');
    articles.value = await response.json();
  } catch (error) {
    console.error('Error fetching articles:', error);
  }
};

const addFeed = async () => {
  if (!feedUrl.value) return;
  try {
    await fetch('/api/feeds', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ url: feedUrl.value }),
    });
    feedUrl.value = '';
    fetchArticles(); // Refresh the list
  } catch (error) {
    console.error('Error adding feed:', error);
  }
};

onMounted(fetchArticles);

</script>

<template>
  <div id="app">
    <header>
      <h1>RSS Reader</h1>
    </header>
    <main>
      <div class="feed-form">
        <input type="text" v-model="feedUrl" @keyup.enter="addFeed" placeholder="Enter RSS feed URL">
        <button @click="addFeed">Add Feed</button>
      </div>
      <div class="articles-list">
        <div v-if="articles.length === 0">No articles found. Add a feed to get started!</div>
        <div v-for="article in articles" :key="article.guid" class="article-item">
          <h2><a :href="article.url" target="_blank">{{ article.title }}</a></h2>
          <p v-html="article.description"></p>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}

header {
  margin-bottom: 40px;
}

.feed-form {
  margin-bottom: 40px;
}

.feed-form input {
  padding: 10px;
  width: 300px;
  margin-right: 10px;
}

.feed-form button {
  padding: 10px 20px;
}

.articles-list {
  max-width: 800px;
  margin: 0 auto;
  text-align: left;
}

.article-item {
  border-bottom: 1px solid #eee;
  padding: 20px 0;
}

.article-item:last-child {
  border-bottom: none;
}

.article-item h2 {
  margin: 0 0 10px;
}

.article-item h2 a {
  color: #2c3e50;
  text-decoration: none;
}

.article-item h2 a:hover {
  text-decoration: underline;
}
</style>
