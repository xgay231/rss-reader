# 已读文章管理功能设计方案

## 1. 概述

为 RSS 阅读器添加已读/未读状态管理和批量操作功能，支持多设备同步。

## 2. 数据模型

### Article 模型扩展

在 `Article` 结构体中添加 `readStatus` 字段：

```go
type Article struct {
    ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Title       string             `json:"title" bson:"title"`
    Link        string             `json:"link" bson:"link"`
    Description string             `json:"description" bson:"description"`
    PublishedAt time.Time          `json:"publishedAt" bson:"publishedAt"`
    SourceID    primitive.ObjectID `json:"sourceId" bson:"sourceId"`
    Summary     string             `json:"summary" bson:"summary"`        // AI生成的摘要
    ReadStatus  string             `json:"readStatus" bson:"readStatus"`  // "unread" | "read"
}
```

**默认值：** 新抓取的文章 `readStatus` 默认设为 `"unread"`

## 3. API 设计

### 3.1 获取文章列表（带筛选）

`GET /api/articles`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| showRead | bool | 是否显示已读文章 |
| showUnread | bool | 是否显示未读文章 |

**筛选逻辑：**
- `showRead=true, showUnread=true` → 返回全部文章
- `showRead=true, showUnread=false` → 仅返回已读文章
- `showRead=false, showUnread=true` → 仅返回未读文章
- `showRead=false, showUnread=false` → 返回空数组

**响应示例：**
```json
{
  "articles": [
    {
      "id": "...",
      "title": "文章标题",
      "readStatus": "unread"
    }
  ]
}
```

### 3.2 标记文章已读

`PUT /api/articles/:id/read`

**请求体：**
```json
{
  "readStatus": "read"
}
```

**响应：** 更新后的文章对象

### 3.3 标记订阅源下所有未读文章为已读

`PUT /api/sources/:sourceId/mark-all-read`

**响应：**
```json
{
  "modifiedCount": 42
}
```

含义：将指定订阅源下的所有未读文章标记为已读。

## 4. 前端设计

### 4.1 筛选区域布局

```
[显示已读 ○] [显示未读 ●]  |  [标记已读]
```

两个 Toggle 按钮，默认：
- 显示已读 = 开启
- 显示未读 = 开启

即默认显示全部文章。

### 4.2 组件结构

```vue
<template>
  <div class="filter-bar">
    <ToggleButton v-model="showRead">显示已读</ToggleButton>
    <ToggleButton v-model="showUnread">显示未读</ToggleButton>
    <button @click="markAllAsRead" :disabled="!canMarkAllRead">
      标记已读
    </button>
  </div>
</template>
```

### 4.3 逻辑

- `showRead && showUnread` → 显示全部
- `showRead && !showUnread` → 仅显示已读
- `!showRead && showUnread` → 仅显示未读
- `!showRead && !showUnread` → 显示空状态

- 标记已读按钮：仅在 `showUnread=true` 时启用（因为只有未读的文章才能被标记为已读）

### 4.4 文章列表视觉区分

已读文章在列表中显示时添加视觉区分（如降低透明度或添加灰色背景）：

```css
.article-item.read {
  opacity: 0.6;
  background-color: var(--color-bg-read);
}
```

## 5. 数据库索引

为 `readStatus` 字段添加索引以优化查询性能：

```javascript
db.articles.createIndex({ readStatus: 1 })
```

## 6. 实现步骤

1. **后端**
   - 修改 Article 模型，添加 `readStatus` 字段
   - 更新 `GET /api/articles` 接口，添加筛选参数
   - 添加 `PUT /api/articles/:id/read` 接口（标记单篇已读）
   - 添加 `PUT /api/sources/:sourceId/mark-all-read` 接口（标记订阅源下全部已读）
   - 添加数据库索引

2. **前端**
   - 修改 ArticleList.vue，添加筛选栏（两个Toggle + 标记已读按钮）
   - 修改文章列表的获取逻辑，传递筛选参数
   - 添加已读文章的视觉样式
   - 添加"标记已读"功能

## 7. 状态定义

| 状态 | 说明 |
|------|------|
| `unread` | 未读（默认） |
| `read` | 已读 |