# 本地测试 RSS 服务器开发设计

## 1. 概述

### 1.1 功能目标

创建一个独立的本地测试 RSS 服务器，用于在开发阶段测试 RSS 抓取功能。该服务器可以生成可控的测试 RSS feeds，支持随机/固定数据生成，便于验证 RSS 阅读器的各项功能。

### 1.2 解决的问题

- 依赖外部 RSS 源进行测试（不稳定、无法控制数据）
- 测试边界情况（空数据、大量文章、特殊字符等）
- 开发阶段离线测试

## 2. 技术方案

### 2.1 技术选型

| 项目 | 选择 | 理由 |
|------|------|------|
| 实现语言 | Go (与后端一致) | 可复用现有代码结构，便于集成 |
| HTTP 框架 | Gin | 与主后端一致，减少学习成本 |
| 数据格式 | RSS 2.0 + Atom | 兼容主流 RSS 格式 |

### 2.2 架构设计

```
backend/
├── main.go                 # 主应用入口
├── test_server/
│   └── server.go          # 测试 RSS 服务器 (新)
└── ...
```

## 3. 功能设计

### 3.1 服务器配置

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| 端口 | 8095 | 避免与主后端 (8080) 冲突 |
| 基础路径 | /feeds | 所有 feed 的基础路径 |

### 3.2 API 端点

#### 3.2.1 预定义测试 Feeds

| 端点 | 方法 | 说明 |
|------|------|------|
| `/feeds/simple` | GET | 简单 RSS，包含 5 篇固定测试文章 |
| `/feeds/random` | GET | 随机数据 RSS，每次请求生成不同内容 |
| `/feeds/empty` | GET | 空列表 RSS |
| `/feeds/single` | GET | 单篇文章 RSS |

#### 3.2.2 动态测试 Feeds

| 端点 | 方法 | 说明 |
|------|------|------|
| `/feeds/custom` | GET | 自定义数量的测试文章 |
| `/feeds/custom?count=10` | GET | 生成 10 篇文章 |
| `/feeds/custom?count=100` | GET | 生成 100 篇文章 |

#### 3.2.3 Feeds 管理 API

| 端点 | 方法 | 说明 |
|------|------|------|
| `/feeds/articles` | POST | 动态添加测试文章（内存存储） |
| `/feeds/articles` | GET | 获取当前所有测试文章 |
| `/feeds/articles/:id` | DELETE | 删除指定测试文章 |
| `/feeds/reset` | POST | 重置所有动态文章 |

### 3.3 测试数据生成

#### 3.3.1 固定测试文章 (simple feed)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test RSS Feed - Simple</title>
    <link>http://localhost:8095/feeds/simple</link>
    <description>A simple test feed for development</description>
    <item>
      <title>测试文章 1 - 正常标题</title>
      <link>http://example.com/article/1</link>
      <description>这是第一篇测试文章的描述内容</description>
      <guid>test-article-1</guid>
      <pubDate>Sat, 11 Apr 2026 10:00:00 GMT</pubDate>
    </item>
    <!-- ... 更多文章 -->
  </channel>
</rss>
```

#### 3.3.2 随机测试文章 (random feed)

- 标题：从预设模板 + 随机关键词生成
- 内容：随机长度的 Lorem ipsum 类型文本
- 发布日期：最近 7 天内的随机日期
- GUID：每次请求重新生成

### 3.4 数据结构

#### 测试文章结构

```go
type TestArticle struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Link        string    `json:"link"`
    Description string    `json:"description"`
    Content     string    `json:"content"`
    GUID        string    `json:"guid"`
    PublishedAt time.Time `json:"publishedAt"`
}
```

#### 预定义文章池

```go
var predefinedArticles = []TestArticle{
    {
        Title:       "技术架构更新公告",
        Description: "本次更新包含多项性能优化和新功能",
        Content:     "详细的技术更新内容...",
    },
    // ... 更多预设文章
}
```

## 4. 实现步骤

### 4.1 创建服务器文件

创建 `backend/test_server/server.go`

```
Step 1: 初始化 Gin 路由器
Step 2: 实现预定义 feed 处理器 (simple, empty, single)
Step 3: 实现随机 feed 处理器 (random)
Step 4: 实现自定义 count feed 处理器 (custom)
Step 5: 实现动态文章管理 API
Step 6: 添加 RSS/Atom XML 响应生成
Step 7: 添加启动命令和配置
```

### 4.2 启动方式

两种启动方式（二选一）：

1. **独立进程启动**
   ```bash
   cd backend
   go run test_server/server.go
   ```

2. **集成到主应用**（可选）
   ```go
   // main.go 中添加
   testServer := testserver.New()
   go testServer.Run(":8081")
   ```

### 4.3 测试验证

#### 手动测试

```bash
# 测试简单 feed
curl http://localhost:8095/feeds/simple

# 测试随机 feed
curl http://localhost:8095/feeds/random

# 测试自定义数量
curl http://localhost:8095/feeds/custom?count=20

# 添加测试文章
curl -X POST http://localhost:8095/feeds/articles \
  -H "Content-Type: application/json" \
  -d '{"title":"Custom Article","description":"Test"}'

# 获取所有文章
curl http://localhost:8095/feeds/articles
```

#### RSS 阅读器配置

```
订阅 URL: http://localhost:8095/feeds/simple
```

## 5. 边界情况处理

| 场景 | 处理方式 |
|------|----------|
| count <= 0 | 返回空列表 |
| count > 1000 | 限制最大为 1000 |
| count 参数缺失 | 默认 5 篇 |
| 文章 ID 不存在 | 返回 404 |
| 无动态文章 | custom feed 返回空列表 |

## 6. 相关文件

| 文件路径 | 说明 |
|----------|------|
| `backend/test_server/server.go` | 新建 - 测试 RSS 服务器 |
| `backend/go.mod` | 可能需要添加 gin 依赖（如未安装） |

## 7. 使用示例

### 7.1 开发调试流程

```
1. 启动主后端: cd backend && go run main.go
2. 启动测试服务器: cd backend && go run test_server/server.go
3. 在 RSS 阅读器前端添加订阅源: http://localhost:8095/feeds/simple
4. 验证文章抓取、显示、AI 摘要等功能
```

### 7.2 自动化测试

```bash
# 循环测试随机 feed
for i in {1..10}; do
  curl -s http://localhost:8095/feeds/random | grep -c "<item>"
  sleep 1
done
```

## 8. 待办事项

- [ ] 创建 `backend/test_server/` 目录
- [ ] 实现 RSS XML 生成辅助函数
- [ ] 实现预定义 feed 处理器
- [ ] 实现随机 feed 处理器
- [ ] 实现动态文章管理 API
- [ ] 测试并验证各端点
