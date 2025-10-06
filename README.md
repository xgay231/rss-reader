# RSS 阅读器

基于 Vue 3 + Vite 的前端和 Go Gin + MongoDB 的后端实现的 Web RSS 阅读器。

## 技术栈

- 前端：Vue 3 + Vite + Pinia
- 后端：Go + Gin 框架
- 数据库：MongoDB
- 部署：Docker 容器化

## 项目结构

```
rss/
├── frontend/          # Vue 3 前端应用
├── backend/           # Go 后端服务
├── docker-compose.yml # Docker 编排文件
└── README.md
```

## 功能特性

- 添加和管理 RSS 源
- 自动抓取和解析 RSS 内容
- 显示文章列表
- 阅读文章详情

## 数据库设计

### RSS 源集合 (feeds)

```javascript
{
  _id: ObjectId,
  url: string,           // RSS 源地址
  title: string,         // 源标题
  description: string,   // 源描述
  createdAt: Date,       // 创建时间
  updatedAt: Date        // 更新时间
}
```

### 文章集合 (articles)

```javascript
{
  _id: ObjectId,
  feedId: ObjectId,      // 关联的 RSS 源 ID
  title: string,         // 文章标题
  link: string,          // 原文链接
  description: string,   // 文章摘要
  content: string,       // 文章内容
  publishedAt: Date,     // 发布时间
  createdAt: Date,       // 创建时间
  updatedAt: Date        // 更新时间
}
```
