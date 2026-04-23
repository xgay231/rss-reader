# 文章已读/未读切换按钮实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 ArticleView.vue 中添加切换按钮，允许用户标记文章为已读或未读状态

**Architecture:** 在现有 ArticleView.vue 组件中添加一个新按钮，根据文章当前状态显示"标记已读"或"标记未读"，点击后调用后端 API 切换状态

**Tech Stack:** Vue 3, Composition API, fetch API

---

## 文件结构

- **Modify:** `frontend/src/components/ArticleView.vue` - 添加切换按钮和相关逻辑

---

## 实施任务

### Task 1: 添加切换按钮到 ArticleView.vue

**Files:**
- Modify: `frontend/src/components/ArticleView.vue` - 添加切换按钮和相关逻辑
- Modify: `backend/main.go` - 添加 DELETE 端点
- Modify: `frontend/src/App.vue` - 添加状态同步处理

- [ ] **Step 1: 在标题区域添加切换按钮**

在 AI 总结按钮之后、收藏按钮之前添加以下代码：

```vue
<button
  class="read-status-btn"
  @click="toggleReadStatus"
  :disabled="isTogglingReadStatus"
>
  {{ article.readStatus === 'unread' ? '标记已读' : '标记未读' }}
</button>
```

- [ ] **Step 2: 添加响应式变量和方法**

在 script setup 区域（大约在第17行附近）添加：

```javascript
const isTogglingReadStatus = ref(false);

const toggleReadStatus = async () => {
  if (!props.article || !props.article.id) return;

  isTogglingReadStatus.value = true;

  try {
    const method = props.article.readStatus === 'unread' ? 'PUT' : 'DELETE';
    const response = await fetchWithAuth(`/api/articles/${props.article.id}/read`, {
      method,
    });

    if (response.ok) {
      props.article.readStatus = props.article.readStatus === 'unread' ? 'read' : 'unread';
      emit('read-status-changed');
    }
  } catch (error) {
    console.error('Failed to toggle read status:', error);
  } finally {
    isTogglingReadStatus.value = false;
  }
};
```

- [ ] **Step 3: 添加样式**

在样式区域（大约第246行后）添加：

```css
.read-status-btn {
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  border: none;
  padding: 0.3rem 0.6rem;
  border-radius: 4px;
  font-size: 0.85rem;
  cursor: pointer;
}

.read-status-btn:hover {
  background-color: var(--color-accent-hover);
}

.read-status-btn:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}
```

- [ ] **Step 4: 添加后端 DELETE 端点**

在 `backend/main.go` 的 articles 路由中添加 `DELETE /:id/read` 端点：

```go
// Mark article as unread
articles.DELETE("/:id/read", func(c *gin.Context) {
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

    update := bson.M{"$set": bson.M{"readStatus": "unread"}}
    _, err = db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to update article"})
        return
    }

    article.ReadStatus = "unread"
    c.JSON(200, article)
})
```

- [ ] **Step 5: 添加 App.vue 状态同步**

在 `App.vue` 中添加 `handleReadStatusChanged` 处理函数，并在 ArticleView 组件上监听 `@read-status-changed` 事件：

```javascript
const handleReadStatusChanged = () => {
  if (currentSourceId.value && currentSourceId.value !== "starred") {
    handleSourceSelected({ id: currentSourceId.value });
  } else if (currentSourceId.value === "starred") {
    loadStarredArticles();
  }
};
```

- [ ] **Step 6: 提交代码**

```bash
git add frontend/src/components/ArticleView.vue backend/main.go frontend/src/App.vue
git commit -m "feat: add read status toggle button in ArticleView"
```

---

## 验证

1. 启动后端服务器 `cd backend && go run main.go`
2. 启动前端开发服务器 `cd frontend && npm run dev`
3. 选择一篇文章查看
4. 确认标题行显示三个按钮：AI总结 | 标记已读/未读 | 收藏
5. 点击"标记已读"按钮，确认：
   - 文章状态变为已读
   - 文章列表中该文章变为已读样式（半透明）
6. 点击"标记未读"按钮，确认：
   - 文章状态变回未读
   - 文章列表中该文章恢复正常样式

---

## 状态

- [x] Task 1 完成（包含后端 DELETE 端点和 App.vue 状态同步）