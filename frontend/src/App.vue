<script setup>
import { ref } from 'vue';
import SourceList from './components/SourceList.vue';
import ArticleList from './components/ArticleList.vue';
import ArticleView from './components/ArticleView.vue';

const articles = ref([]);
const selectedArticle = ref(null);
const sourceListRef = ref(null);

// Column width state (in pixels)
const leftWidth = ref(280);
const centerWidth = ref(350);

// For tracking drag state
const dragging = ref(null); // 'left' or 'center'
const startX = ref(0);
const startWidth = ref(0);

const startDrag = (e, which) => {
  dragging.value = which;
  startX.value = e.clientX;
  startWidth.value = which === 'left' ? leftWidth.value : centerWidth.value;
  document.addEventListener('mousemove', onDrag);
  document.addEventListener('mouseup', stopDrag);
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
};

const onDrag = (e) => {
  if (!dragging.value) return;
  const delta = e.clientX - startX.value;
  if (dragging.value === 'left') {
    leftWidth.value = Math.max(220, Math.min(400, startWidth.value + delta));
  } else {
    centerWidth.value = Math.max(280, Math.min(500, startWidth.value + delta));
  }
};

const stopDrag = () => {
  dragging.value = null;
  document.removeEventListener('mousemove', onDrag);
  document.removeEventListener('mouseup', stopDrag);
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
};

// This function is called when a source is selected in the SourceList component
const handleSourceSelected = async (source) => {
  if (!source) {
    articles.value = [];
    selectedArticle.value = null;
    return;
  }

  // Handle starred articles special case
  if (source.id === 'starred') {
    try {
      const response = await fetch('/api/articles/starred');
      if (!response.ok) {
        throw new Error('Failed to fetch starred articles');
      }
      articles.value = await response.json();
      selectedArticle.value = null;
    } catch (error) {
      console.error('Error fetching starred articles:', error);
      articles.value = [];
    }
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

// Refresh starred count when article is starred/unstarred
const refreshStarredCount = () => {
  if (sourceListRef.value) {
    sourceListRef.value.refreshStarredCount();
  }
};
</script>

<template>
  <div id="app-container">
    <div class="left-pane" :style="{ width: leftWidth + 'px' }">
      <SourceList ref="sourceListRef" @source-selected="handleSourceSelected" />
    </div>
    <div
      class="divider"
      :class="{ dragging: dragging === 'left' }"
      @mousedown="(e) => startDrag(e, 'left')"
    ></div>
    <div class="center-pane" :style="{ width: centerWidth + 'px' }">
      <ArticleList :articles="articles" @article-selected="handleArticleSelected" />
    </div>
    <div
      class="divider"
      :class="{ dragging: dragging === 'center' }"
      @mousedown="(e) => startDrag(e, 'center')"
    ></div>
    <div class="right-pane">
      <ArticleView :article="selectedArticle" @starred-changed="refreshStarredCount" />
    </div>
  </div>
</template>

<style>
#app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}

.left-pane {
  flex-shrink: 0;
  min-width: 220px;
  max-width: 400px;
}

.center-pane {
  flex-shrink: 0;
  min-width: 280px;
  max-width: 500px;
}

.right-pane {
  flex: 1;
  min-width: 400px;
}

.divider {
  width: 4px;
  background: #e0e0e0;
  cursor: col-resize;
  flex-shrink: 0;
  transition: background 0.2s, opacity 0.2s;
  position: relative;
  z-index: 10;
}

.divider.dragging {
  opacity: 0.5;
  pointer-events: none;
}

.divider:hover {
  background: #bdbdbd;
}
</style>
