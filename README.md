# Web RSS 阅读器

一款现代化的在线 RSS 阅读器，后端采用 Go 语言构建，前端基于 Vue 3 实现。

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
```
