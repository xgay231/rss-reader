# Article List Sort Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sort article list by `publishedAt` descending (newest first) in the frontend.

**Architecture:** Modify the `filteredArticles` computed property in `ArticleList.vue` to sort articles before filtering.

**Tech Stack:** Vue 3 Composition API

---

### Task 1: Add sort to filteredArticles

**File:** `frontend/src/components/ArticleList.vue:41-47`

- [ ] **Step 1: Modify filteredArticles to sort by publishedAt descending**

Replace:
```js
const filteredArticles = computed(() => {
  return props.articles.filter((article) => {
    if (article.readStatus === 'read' && !showRead.value) return false;
    if (article.readStatus === 'unread' && !showUnread.value) return false;
    return true;
  });
});
```

With:
```js
const filteredArticles = computed(() => {
  return props.articles
    .slice()
    .sort((a, b) => new Date(b.publishedAt) - new Date(a.publishedAt))
    .filter((article) => {
      if (article.readStatus === 'read' && !showRead.value) return false;
      if (article.readStatus === 'unread' && !showUnread.value) return false;
      return true;
    });
});
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ArticleList.vue
git commit -m "feat(frontend): sort article list by publishedAt descending"
```

---

**Verification:** Start frontend with `npm run dev` and verify articles appear in reverse chronological order (newest first).