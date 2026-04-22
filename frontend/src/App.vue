<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { provideAuth, useAuth } from "./composables/useAuth";
import { fetchWithAuth } from "./utils/api";
import SourceList from "./components/SourceList.vue";
import ArticleList from "./components/ArticleList.vue";
import ArticleView from "./components/ArticleView.vue";
import AuthForm from "./components/AuthForm.vue";

const auth = provideAuth()

const articles = ref([]);
const selectedArticle = ref(null);
const sourceListRef = ref(null);
const currentSourceId = ref(null); // Track current source ID to detect starred view

// Mobile navigation state - stack based navigation
const currentView = ref("sources"); // 'sources' | 'articles' | 'content'
const windowWidth = ref(window.innerWidth);

// Column width state (in pixels)
const leftWidth = ref(280);
const centerWidth = ref(350);

// Check if mobile view
const isMobile = computed(() => windowWidth.value < 768);

// Update window width on resize
const updateWindowWidth = () => {
  windowWidth.value = window.innerWidth;
};

// Clear data when user changes (user switch)
watch(() => auth.user.value, (newUser, oldUser) => {
  if (oldUser && !newUser) {
    // Logout - clear all data
    articles.value = [];
    selectedArticle.value = null;
    currentView.value = "sources";
  } else if (newUser && !oldUser) {
    // Login - clear old user data
    articles.value = [];
    selectedArticle.value = null;
    currentView.value = "sources";
  } else if (newUser && oldUser && newUser.id !== oldUser.id) {
    // User switched - clear old user's data
    articles.value = [];
    selectedArticle.value = null;
    currentView.value = "sources";
  }
});

onMounted(() => {
  window.addEventListener("resize", updateWindowWidth);
  // Check login status on mount
  auth.fetchCurrentUser();
});

onUnmounted(() => {
  window.removeEventListener("resize", updateWindowWidth);
});

// For tracking drag state
const dragging = ref(null); // 'left' or 'center'
const startX = ref(0);
const startWidth = ref(0);

const startDrag = (e, which) => {
  dragging.value = which;
  startX.value = e.clientX;
  startWidth.value = which === "left" ? leftWidth.value : centerWidth.value;
  document.addEventListener("mousemove", onDrag);
  document.addEventListener("mouseup", stopDrag);
  document.body.style.cursor = "col-resize";
  document.body.style.userSelect = "none";
};

const onDrag = (e) => {
  if (!dragging.value) return;
  const delta = e.clientX - startX.value;
  if (dragging.value === "left") {
    leftWidth.value = Math.max(220, Math.min(400, startWidth.value + delta));
  } else {
    centerWidth.value = Math.max(280, Math.min(500, startWidth.value + delta));
  }
};

const stopDrag = () => {
  dragging.value = null;
  document.removeEventListener("mousemove", onDrag);
  document.removeEventListener("mouseup", stopDrag);
  document.body.style.cursor = "";
  document.body.style.userSelect = "";
};

// Navigate back in mobile view
const goBack = () => {
  if (currentView.value === "content") {
    currentView.value = "articles";
    selectedArticle.value = null;
  } else if (currentView.value === "articles") {
    currentView.value = "sources";
    articles.value = [];
  }
};

// This function is called when a source is selected in the SourceList component
const handleSourceSelected = async (source) => {
  if (!source) {
    articles.value = [];
    selectedArticle.value = null;
    currentSourceId.value = null;
    return;
  }

  // Handle starred articles special case
  if (source.id === "starred") {
    currentSourceId.value = "starred";
    try {
      const response = await fetchWithAuth("/api/articles/starred");
      if (!response.ok) {
        throw new Error("Failed to fetch starred articles");
      }
      articles.value = await response.json();
      selectedArticle.value = null;
      currentView.value = "articles";
    } catch (error) {
      console.error("Error fetching starred articles:", error);
      articles.value = [];
    }
    return;
  }

  currentSourceId.value = source.id;
  try {
    const response = await fetchWithAuth(`/api/sources/${source.id}/articles`);
    if (!response.ok) {
      throw new Error("Failed to fetch articles for the source");
    }
    articles.value = await response.json();
    selectedArticle.value = null; // Reset article view when a new source is selected
    currentView.value = "articles";
  } catch (error) {
    console.error("Error fetching articles:", error);
    articles.value = []; // Clear articles on error
  }
};

// This function is called when an article is selected in the ArticleList component
const handleArticleSelected = async (article) => {
  // Fetch full article to ensure we have summary field
  try {
    const response = await fetchWithAuth(`/api/articles/${article.id}`);
    if (response.ok) {
      selectedArticle.value = await response.json();
    } else {
      selectedArticle.value = article;
    }
  } catch (error) {
    console.error("Error fetching full article:", error);
    selectedArticle.value = article;
  }
  currentView.value = "content";

  // Mark article as read when clicked and update local state
  if (article.readStatus !== 'read') {
    fetchWithAuth(`/api/articles/${article.id}/read`, {
      method: 'PUT'
    })
      .then(() => {
        // Update local articles state to reflect the change
        const index = articles.value.findIndex(a => a.id === article.id);
        if (index !== -1) {
          articles.value[index] = { ...articles.value[index], readStatus: 'read' };
        }
      })
      .catch(error => {
        console.error('Failed to mark article as read:', error);
      });
  }
};

// Update article when summary is generated
const handleSummaryUpdated = (updatedArticle) => {
  selectedArticle.value = updatedArticle;
};

// Mark all articles as read for the current source
const handleMarkAllRead = async () => {
  if (!currentSourceId.value || currentSourceId.value === "starred") return;

  try {
    const response = await fetchWithAuth(
      `/api/sources/${currentSourceId.value}/mark-all-read`,
      { method: "PUT" }
    );
    if (response.ok) {
      // 刷新文章列表
      const source = { id: currentSourceId.value };
      handleSourceSelected(source);
    }
  } catch (error) {
    console.error("Error marking all as read:", error);
  }
};

// Refresh starred count and list when article is starred/unstarred
const refreshStarredCount = async () => {
  if (sourceListRef.value) {
    sourceListRef.value.refreshStarredCount();
  }

  // If viewing starred articles list, refresh it
  if (currentSourceId.value === "starred") {
    try {
      const response = await fetchWithAuth("/api/articles/starred");
      if (response.ok) {
        articles.value = await response.json();
        // If currently viewing an article that was unstarred, close it
        if (selectedArticle.value && !articles.value.find(a => a.id === selectedArticle.value.id)) {
          selectedArticle.value = null;
          currentView.value = "articles";
        }
      }
    } catch (error) {
      console.error("Error refreshing starred articles:", error);
    }
  }
};
</script>

<template>
  <div id="app-container">
    <!-- Auth Form when not logged in -->
    <AuthForm v-if="!auth.isAuthenticated()" />

    <!-- Main App when logged in -->
    <template v-else>
    <header class="mobile-header" v-if="isMobile && currentView !== 'sources'">
      <button class="back-btn" @click="goBack">返回</button>
      <span class="header-title">
        {{ currentView === "articles" ? "文章列表" : "文章内容" }}
      </span>
    </header>

    <!-- Source List Panel -->
    <div
      class="left-pane"
      :style="{ width: isMobile ? '100%' : leftWidth + 'px' }"
      v-show="!isMobile || currentView === 'sources'"
    >
      <SourceList ref="sourceListRef" :key="auth.user.value?.id" @source-selected="handleSourceSelected" />
    </div>

    <!-- Left Divider -->
    <div
      v-if="!isMobile"
      class="divider"
      :class="{ dragging: dragging === 'left' }"
      @mousedown="(e) => startDrag(e, 'left')"
    ></div>

    <!-- Article List Panel -->
    <div
      class="center-pane"
      :style="{ width: isMobile ? '100%' : centerWidth + 'px' }"
      v-show="!isMobile || currentView === 'articles'"
    >
      <ArticleList
        :articles="articles"
        :currentSourceId="currentSourceId"
        @article-selected="handleArticleSelected"
        @mark-all-read="handleMarkAllRead"
      />
    </div>

    <!-- Center Divider -->
    <div
      v-if="!isMobile"
      class="divider"
      :class="{ dragging: dragging === 'center' }"
      @mousedown="(e) => startDrag(e, 'center')"
    ></div>

    <!-- Article Content Panel -->
    <div class="right-pane" v-show="!isMobile || currentView === 'content'">
      <ArticleView
        :article="selectedArticle"
        @starred-changed="refreshStarredCount"
        @summary-updated="handleSummaryUpdated"
      />
    </div>
    </template>
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

/* Mobile Header */
.mobile-header {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 50px;
  background: var(--color-bg-pane);
  border-bottom: 1px solid var(--color-border);
  z-index: 1000;
  align-items: center;
  padding: 0 1rem;
}

.back-btn {
  background: none;
  border: none;
  color: var(--color-accent);
  font-size: 1rem;
  cursor: pointer;
  padding: 0.5rem 0;
  min-width: auto;
}

.header-title {
  margin-left: 1rem;
  font-weight: 500;
  color: var(--color-text-primary);
}

/* Responsive styles */
@media (max-width: 768px) {
  .mobile-header {
    display: flex;
  }

  .divider {
    display: none;
  }

  #app-container {
    padding-top: 50px;
  }

  .left-pane,
  .center-pane,
  .right-pane {
    flex: none !important;
    width: 100% !important;
    min-width: 0 !important;
    max-width: 100% !important;
  }
}
</style>
