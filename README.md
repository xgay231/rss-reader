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

## 开发计划

### 第一阶段：环境搭建
- [ ] 初始化 Vue 3 + Vite 项目
- [ ] 配置 Pinia 状态管理
- [ ] 初始化 Go Gin 项目
- [ ] 配置 MongoDB 连接
- [ ] 创建 Docker 配置文件

### 第二阶段：后端开发
- [ ] 实现 RSS 源管理 API
- [ ] 实现文章获取和存储逻辑
- [ ] 实现文章查询 API
- [ ] 添加 RSS 抓取定时任务

### 第三阶段：前端开发
- [ ] 实现 RSS 源添加页面
- [ ] 实现文章列表页面
- [ ] 实现文章详情页面
- [ ] 添加页面间导航

### 第四阶段：测试与部署
- [ ] 进行集成测试
- [ ] 优化性能
- [ ] 编写部署文档
- [ ] 完成 Docker 部署配置

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
