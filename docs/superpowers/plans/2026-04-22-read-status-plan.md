# 已读状态管理功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 RSS 阅读器添加已读/未读状态管理和筛选功能，支持多设备同步

**Architecture:** 在 Article 模型中添加 readStatus 字段，扩展 API 支持筛选和标记操作，前端添加筛选栏和视觉区分

**Tech Stack:** Go (backend), Vue.js (frontend), MongoDB

---

## 文件结构

```
backend/
├── main.go              # 修改: Article结构体添加readStatus字段, 添加API路由和处理器

frontend/src/
├── components/
│   └── ArticleList.vue  # 修改: 添加筛选栏组件, 添加已读视觉样式
└── App.vue              # 修改: 传递currentSourceId给ArticleList, 支持markAllRead回调
```

---

## 实现任务

### Task 1: 后端 - 修改 Article 模型

**Files:**
- Modify: `backend/main.go:85-99`

- [ ] **Step 1: 在 Article 结构体中添加 ReadStatus 字段**

找到 `backend/main.go` 第 85-99 行的 Article 结构体，在 `SummaryGeneratedAt` 字段后添加：

```go
ReadStatus       string             `json:"readStatus" bson:"readStatus"` // "unread" | "read"
```

---

### Task 2: 后端 - 创建文章时默认 readStatus 为 "unread"

**Files:**
- Modify: `backend/main.go:153` (article creation in fetchRssAndStore)
- Modify: `backend/main.go:312` (article creation in updateSource)

- [ ] **Step 2: 在 fetchRssAndStore 函数中创建文章时设置 ReadStatus: "unread"**

在第 153 行附近的 article 创建代码中，添加 `ReadStatus: "unread",` 字段。

- [ ] **Step 3: 在 updateSource 函数中创建文章时设置 ReadStatus: "unread"**

在第 312 行附近的 article 创建代码中，添加 `ReadStatus: "unread",` 字段。

---

### Task 3: 后端 - 添加数据库索引

**Files:**
- Modify: `backend/main.go` (位置待定，在数据库初始化处)

- [ ] **Step 4: 为 readStatus 字段添加索引**

在数据库初始化代码附近（查找 `CreateIndex` 或 `Indexes()` 的调用），为 `readStatus` 字段创建索引：

```go
db.ArticleCollection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
    Keys: bson.D{{Key: "readStatus", Value: 1}},
})
```

---

### Task 4: 后端 - 添加文章筛选 API

**Files:**
- Modify: `backend/main.go:586-612` (articles GET /starred 附近)

- [ ] **Step 5: 修改 GET /api/articles 实现筛选逻辑**

在 `articles.GET("/starred"` 处理函数附近，修改或添加 `articles.GET("")` 处理函数，支持 `showRead` 和 `showUnread` 查询参数：

```go
articles.GET("", func(c *gin.Context) {
    userID := getUserID(c)
    if userID == primitive.NilObjectID {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    showRead := c.Query("showRead") == "true"
    showUnread := c.Query("showUnread") == "true"

    // 构建筛选条件
    filter := bson.M{"userId": userID}

    // 如果两个都false，返回空数组（用户明确选择了空白状态）
    if !showRead && !showUnread {
        c.JSON(200, []Article{})
        return
    }

    // 添加 readStatus 筛选
    if showRead && !showUnread {
        filter["readStatus"] = "read"
    } else if !showRead && showUnread {
        filter["readStatus"] = "unread"
    }
    // 如果两者都为true，不添加 readStatus 筛选

    cursor, err := db.ArticleCollection.Find(context.Background(), filter)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to fetch articles"})
        return
    }
    defer cursor.Close(context.Background())

    var articles []Article
    if err = cursor.All(context.Background(), &articles); err != nil {
        c.JSON(500, gin.H{"error": "Failed to decode articles"})
        return
    }

    c.JSON(200, articles)
})
```

**注意:** 需要检查是否已有 `GET /api/articles` 路由，如果有则修改现有路由。

---

### Task 5: 后端 - 添加标记单篇文章已读 API

**Files:**
- Modify: `backend/main.go:709-730` (star endpoint 附近)

- [ ] **Step 6: 添加 PUT /api/articles/:id/read 路由**

在 unstar 路由后添加新的路由：

```go
// Mark article as read
articles.PUT("/:id/read", func(c *gin.Context) {
    userID := getUserID(c)
    if userID == primitive.NilObjectID {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    id, err := primitive.ObjectIDFromHex(c.Param("id"))
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid Article ID"})
        return
    }

    var article Article
    err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&article)
    if err != nil {
        c.JSON(404, gin.H{"error": "Article not found"})
        return
    }

    update := bson.M{"$set": bson.M{"readStatus": "read"}}
    _, err = db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to update article"})
        return
    }

    article.ReadStatus = "read"
    c.JSON(200, article)
})
```

---

### Task 6: 后端 - 添加标记订阅源全部已读 API

**Files:**
- Modify: `backend/main.go:388-416` (sources GET /:id/articles 附近)

- [ ] **Step 7: 在 sources 路由组中添加标记全部已读接口**

在 `sources.GET("/:id/articles"` 处理函数后添加：

```go
// Mark all articles in a source as read
sources.PUT("/:id/mark-all-read", func(c *gin.Context) {
    userID := getUserID(c)
    if userID == primitive.NilObjectID {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    id, err := primitive.ObjectIDFromHex(c.Param("id"))
    if err != nil {
        c.JSON(400, gin.H{"error": "Invalid Source ID"})
        return
    }

    // Check if source exists and belongs to user
    var source Source
    err = db.SourceCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&source)
    if err != nil {
        c.JSON(404, gin.H{"error": "Source not found"})
        return
    }

    update := bson.M{"$set": bson.M{"readStatus": "read"}}
    filter := bson.M{"sourceId": id, "userId": userID, "readStatus": "unread"}
    result, err := db.ArticleCollection.UpdateMany(context.Background(), filter, update)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to mark articles as read"})
        return
    }

    c.JSON(200, gin.H{"modifiedCount": result.ModifiedCount})
})
```

---

### Task 7: 前端 - 修改 ArticleList 添加筛选栏

**Files:**
- Modify: `frontend/src/components/ArticleList.vue`

- [ ] **Step 8: 添加筛选栏组件**

在 ArticleList.vue 的 `<template>` 部分，在 `<h2>Articles</h2>` 后添加筛选栏：

```vue
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
```

- [ ] **Step 9: 添加筛选状态和相关逻辑**

在 `<script setup>` 部分添加：

```javascript
const showRead = ref(true);
const showUnread = ref(true);

const toggleShowRead = () => {
  showRead.value = !showRead.value;
  // 确保至少有一个为true
  if (!showRead.value && !showUnread.value) {
    showUnread.value = true;
  }
  refreshArticles();
};

const toggleShowUnread = () => {
  showUnread.value = !showUnread.value;
  if (!showRead.value && !showUnread.value) {
    showRead.value = true;
  }
  refreshArticles();
};

const markAllAsRead = async () => {
  // 调用 API 标记当前订阅源所有未读文章为已读
  // 需要 currentSourceId，从 props 或 emit 获取
};
```

- [ ] **Step 10: 添加已读文章视觉样式**

在 `<style scoped>` 中添加：

```css
.article-item.read {
  opacity: 0.6;
  background-color: var(--color-bg-read, #f5f5f5);
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
```

- [ ] **Step 11: 在文章列表项中添加已读样式类**

根据 `article.readStatus` 添加动态 class：

```vue
<li
  v-for="article in filteredArticles"
  :key="article.id"
  @click="selectArticle(article)"
  :class="{ read: article.readStatus === 'read' }"
>
```

---

### Task 8: 前端 - 修改 App.vue 传递 currentSourceId

**Files:**
- Modify: `frontend/src/App.vue:233-236`

- [ ] **Step 12: 传递 currentSourceId 和回调给 ArticleList**

在 ArticleList 组件上添加 `currentSourceId` prop 和 `onMarkAllRead` 回调：

```vue
<ArticleList
  :articles="articles"
  :currentSourceId="currentSourceId"
  @article-selected="handleArticleSelected"
  @mark-all-read="handleMarkAllRead"
/>
```

添加 `handleMarkAllRead` 方法：

```javascript
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
```

---

## 验证步骤

1. 启动后端服务：`cd backend && go run main.go`
2. 启动前端服务：`cd frontend && npm run dev`
3. 测试流程：
   - 选择一个订阅源，查看文章列表
   - 点击文章，应自动标记为已读
   - 测试筛选按钮：仅显示已读、仅显示未读、全部
   - 测试标记已读按钮
   - 检查已读文章是否有过期样式（降低透明度）