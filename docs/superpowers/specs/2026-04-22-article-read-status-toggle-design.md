# 文章已读/未读切换按钮设计

## 概述

在文章内容查看页面（ArticleView.vue）右上角添加一个切换按钮，允许用户直接在阅读文章时标记为已读或未读状态。

## 需求

- 位置：AI总结按钮和收藏按钮之间
- 功能：点击切换文章的已读/未读状态
- 交互：直接切换，无需确认

## 设计

### 按钮设计

- **位置**：标题行内，AI总结按钮右侧，收藏按钮左侧
- **文字**：根据当前状态显示
  - 未读状态：显示"标记已读"
  - 已读状态：显示"标记未读"
- **样式**：
  - 使用与 AI 总结按钮相近的样式
  - 背景色使用主题色 `var(--color-accent)`
  - 文字使用 `var(--color-accent-text)`

### 行为

1. 点击按钮
2. 判断当前状态：
   - 如果是 `unread`，调用 `PUT /api/articles/:id/read` 标记为已读
   - 如果是 `read`，调用 `DELETE /api/articles/:id/read` 标记为未读
3. 成功后更新本地 `article.readStatus` 状态
4. 可选：触发 `read-status-changed` 事件通知父组件

### API 端点

- 标记已读：`PUT /api/articles/:id/read`
- 标记未读：`DELETE /api/articles/:id/read`

## 实现要点

- 需要在 `ArticleView.vue` 中添加切换按钮
- 需要在 `api.js` 或相关工具文件中添加调用 API 的方法（如果尚未存在）
- 按钮文字根据 `article.readStatus` 动态显示
- 成功后更新本地状态并通知父组件

## 状态

- 设计完成，等待用户确认