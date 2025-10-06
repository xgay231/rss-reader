# Web RSS 阅读器

一款现代化的在线 RSS 阅读器，后端采用 Go 语言构建，前端基于 Vue 3 实现。项目特色是其简洁的三栏式布局和由 TensorFlow.js 驱动的客户端 AI 文章摘要功能，旨在提供高效、纯粹的阅读体验。

## ✨ 功能特性

- **订阅源管理**: 轻松添加、刷新和删除 RSS 订阅源。
- **文章浏览**: 自动抓取并展示订阅源的文章列表。
- **三栏式布局**: 经典且高效的界面布局，方便在订阅源、文章列表和文章内容之间切换。
- **AI 文章摘要**: 利用 TensorFlow.js 在浏览器端直接生成文章摘要，无需依赖服务端，既保护了用户隐私，又提供了快速获取信息的核心途径。
- **现代化界面**: 经过精心设计的浅色主题，确保了视觉上的一致性和舒适的阅读体验。

## 🛠️ 技术选型

- **前端**:
  - **[Vue 3](https://cn.vuejs.org/)**: 采用组合式 API (Composition API) 构建。
  - **[Vite](https://cn.vitejs.dev/)**: 提供极速的开发服务器和构建体验。
  - **[TensorFlow.js](https://www.tensorflow.org/js?hl=zh-cn)**: 用于在客户端实现机器学习功能。
- **后端**:
  - **[Go](https://go.dev/)**: 高性能的后端编程语言。
  - **[Gin](https://gin-gonic.com/)**: 轻量级且高效的 Go Web 框架。
  - **[MongoDB](https://www.mongodb.com/)**: 灵活的 NoSQL 数据库，用于存储订阅源和文章数据。

## 🚀 快速开始

请按照以下步骤在您的本地环境中部署和运行此项目。

### 环境要求

- **Go**: 版本 `1.21` 或更高
- **Node.js**: 版本 `18.0` 或更高 (推荐使用 `pnpm` 作为包管理器)
- **MongoDB**: 确保本地 `mongodb://localhost:27017` 地址可访问

### 1. 启动后端服务

```bash
# 进入后端项目目录
cd backend

# 安装 Go 模块依赖
go mod tidy

# 启动后端 Gin 服务器
go run main.go
```
> 后端服务将在 `http://localhost:8080` 启动。

### 2. 启动前端应用

```bash
# 进入前端项目目录
cd frontend

# 安装 Node.js 依赖
pnpm install

# 启动 Vite 开发服务器
pnpm dev
```
> 前端应用将在 `http://localhost:5173` (或终端提示的其他端口) 启动。

现在，您可以打开浏览器访问前端地址，开始使用这款 RSS 阅读器了。

## 📁 项目结构

```
rss/
├── backend/           # Go 后端服务
│   ├── main.go        # 程序入口
│   └── ...
├── frontend/          # Vue 3 前端应用
│   ├── src/
│   │   ├── components/ # Vue 组件
│   │   ├── services/   # 服务模块 (如摘要服务)
│   │   └── App.vue
│   └── ...
└── README.md