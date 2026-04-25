# 文章列表排序修复方案

## 背景

文章列表当前按 MongoDB 自然顺序（插入顺序）返回，用户反映列表顺序不符合预期。

## 目标

按文章发布时间（`publishedAt`）降序排列，最新文章排在最前面。

## 方案

前端排序 — 在 `ArticleList.vue` 的 `filteredArticles` 计算属性中，先排序再筛选。

## 修改文件

- `frontend/src/components/ArticleList.vue`

## 改动详情

修改 `filteredArticles` 计算属性：

**原代码（第 41-47 行）：**
```js
const filteredArticles = computed(() => {
  return props.articles.filter((article) => {
    if (article.readStatus === 'read' && !showRead.value) return false;
    if (article.readStatus === 'unread' && !showUnread.value) return false;
    return true;
  });
});
```

**新代码：**
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

## 排序逻辑

- 使用 `Array.slice()` 复制数组，避免修改原 `props.articles`
- 按 `publishedAt` 降序（`b - a`），最新的文章排在前面
- 排序在筛选之前执行

## 验证方式

1. 启动前端 `npm run dev`
2. 打开文章列表，确认最新文章显示在最上方
3. 测试已读/未读筛选功能正常