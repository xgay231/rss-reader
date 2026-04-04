# 系统架构文档

## 1. 项目概述

Web RSS 阅读器是一款现代化的在线 RSS 阅读器，采用前后端分离架构：

- **后端**：Go 语言 + Gin 框架，提供 RESTful API
- **前端**：Vue 3 + Vite，构建单页应用
- **数据库**：MongoDB，存储订阅源和文章数据
- **AI 服务**：OpenAI API，提供智能摘要功能

## 2. 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         前端 (Vue 3)                            │
│  ┌──────────┐    ┌──────────────┐    ┌────────────────────┐   │
│  │ SourceList│    │ ArticleList  │    │   ArticleView      │   │
│  └─────┬────┘    └──────┬───────┘    └─────────┬──────────┘   │
│        │                │                      │              │
│        └────────────────┼──────────────────────┘              │
│                         │                                       │
│                  ┌──────▼──────┐                                │
│                  │  App.vue   │                                │
│                  └──────┬──────┘                                │
└─────────────────────────┼──────────────────────────────────────┘
                          │ HTTP API
┌─────────────────────────▼──────────────────────────────────────┐
│                      后端 (Go + Gin)                            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    API 路由层                            │   │
│  │  /ping   /api/sources   /api/sources/:id   /api/articles│   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│  ┌───────────────────────────▼────────────────────────────┐   │
│  │                    业务逻辑层                           │   │
│  │  • FeedSource 管理 (添加、删除、获取)                    │   │
│  │  • Article 管理 (获取、摘要生成)                         │   │
│  │  • AI 摘要服务 (OpenAI 集成)                            │   │
│  │  • 定时更新任务 (后台抓取 RSS)                           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│  ┌───────────────────────────▼────────────────────────────┐   │
│  │                    数据访问层                           │   │
│  │                  MongoDB 数据库                          │   │
│  │  • sources (订阅源集合)  • articles (文章集合)          │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 技术栈

### 后端技术栈

| 技术      | 版本   | 用途              |
| --------- | ------ | ----------------- |
| Go        | ≥1.21  | 编程语言          |
| Gin       | latest | Web 框架          |
| MongoDB   | -      | 数据库            |
| gofeed    | latest | RSS 解析          |
| go-openai | latest | OpenAI API 客户端 |
| godotenv  | latest | 环境变量管理      |

### 前端技术栈

| 技术                                          | 版本    | 用途           |
| --------------------------------------------- | ------- | -------------- |
| Vue                                           | 3.x     | 框架           |
| Vite                                          | -       | 构建工具       |
| @tensorflow/tfjs                              | ^4.22.0 | 机器学习运行时 |
| @tensorflow-models/universal-sentence-encoder | ^1.3.3  | 句子嵌入模型   |
| marked                                        | ^16.3.0 | Markdown 解析  |

## 4. 核心模块

### 4.1 后端模块

#### 数据库连接模块 ([`backend/db/db.go`](backend/db/db.go))

**代码位置**: [`db/db.go:16-33`](backend/db/db.go:16)

- 初始化 MongoDB 连接：`mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:password@localhost:27017"))`
- 连接数据库 `rss_reader`
- 创建两个集合：
  - `ArticleCollection` - 存储文章数据
  - `SourceCollection` - 存储订阅源数据
- 10 秒超时控制，使用 `context.WithTimeout`

```go
// db/db.go:16-33
func ConnectDatabase() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:password@localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal(err)
    }
    log.Println("Successfully connected to MongoDB!")
    Client = client
    ArticleCollection = client.Database("rss_reader").Collection("articles")
    SourceCollection = client.Database("rss_reader").Collection("sources")
}
```

#### 主程序模块 ([`backend/main.go`](backend/main.go))

##### 初始化配置 ([`main.go:24-54`](backend/main.go:24))

- 加载 `.env` 文件 (使用 `godotenv.Load()`)
- 从环境变量读取 OpenAI 配置：
  - `OPENAI_API_KEY` - API 密钥
  - `OPENAI_BASE_URL` - 自定义 API 端点
  - `OPENAI_MODEL_NAME` - 模型名称 (默认 `gpt-3.5-turbo`)

```go
// main.go:24-54
func init() {
    err := godotenv.Load()
    if err != nil {
        log.Println("No .env file found, will use environment variables from OS")
    }

    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        log.Println("Warning: OPENAI_API_KEY environment variable not set. AI summarization will be disabled.")
        return
    }

    baseURL := os.Getenv("OPENAI_BASE_URL")
    config := openai.DefaultConfig(apiKey)
    if baseURL != "" {
        config.BaseURL = baseURL
    }

    aiClient = openai.NewClientWithConfig(config)
    aiModelName = os.Getenv("OPENAI_MODEL_NAME")
    if aiModelName == "" {
        aiModelName = openai.GPT3Dot5Turbo
    }
}
```

##### 数据模型 ([`main.go:56-72`](backend/main.go:56))

```go
// FeedSource 订阅源 - main.go:56-61
type FeedSource struct {
    ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Name string             `json:"name" bson:"name"`
    URL  string             `json:"url" bson:"url"`
}

// Article 文章 - main.go:63-72
type Article struct {
    ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    SourceID    primitive.ObjectID `json:"sourceId" bson:"sourceId"`
    GUID        string             `json:"guid" bson:"guid"`
    Title       string             `json:"title" bson:"title"`
    URL         string             `json:"url" bson:"url"`
    Description string             `json:"description" bson:"description"`
    Content     string             `json:"content" bson:"content"`
}
```

##### RSS 更新后台任务 ([`main.go:74-135`](backend/main.go:74))

- 从数据库获取所有订阅源
- 使用 `gofeed.NewParser()` 解析每个订阅源
- 增量更新：检查 GUID 是否已存在，避免重复
- 使用 `InsertMany` 批量插入新文章

```go
// main.go:74-135 - 核心更新逻辑
func updateFeeds() {
    // 1. 获取所有订阅源
    cursor, err := db.SourceCollection.Find(ctx, bson.M{})

    // 2. 遍历每个订阅源
    fp := gofeed.NewParser()
    for _, source := range sources {
        feed, err := fp.ParseURL(source.URL)

        // 3. 检查每篇文章是否存在
        for _, item := range feed.Items {
            filter := bson.M{"guid": item.GUID, "sourceId": source.ID}
            err := db.ArticleCollection.FindOne(ctx, filter).Err()

            if err == mongo.ErrNoDocuments {
                // 4. 新文章，添加到列表
                newArticles = append(newArticles, article)
            }
        }

        // 5. 批量插入
        if len(newArticles) > 0 {
            opts := options.InsertMany().SetOrdered(false)
            db.ArticleCollection.InsertMany(ctx, newArticles, opts)
        }
    }
}
```

##### API 路由实现

| 路由                                | 代码位置                                 | 功能说明                        |
| ----------------------------------- | ---------------------------------------- | ------------------------------- |
| `GET /ping`                         | [`main.go:155-159`](backend/main.go:155) | 健康检查                        |
| `POST /api/sources`                 | [`main.go:168-233`](backend/main.go:168) | 添加订阅源，解析 RSS 并存储文章 |
| `GET /api/sources`                  | [`main.go:236-251`](backend/main.go:236) | 获取所有订阅源                  |
| `DELETE /api/sources/:id`           | [`main.go:254-276`](backend/main.go:254) | 删除订阅源及其关联文章          |
| `GET /api/sources/:id/articles`     | [`main.go:279-300`](backend/main.go:279) | 获取订阅源下的所有文章          |
| `GET /api/articles/:id`             | [`main.go:307-322`](backend/main.go:307) | 获取单篇文章详情                |
| `POST /api/articles/:id/ai-summary` | [`main.go:325-374`](backend/main.go:325) | 调用 OpenAI API 生成摘要        |

### 4.2 前端模块

#### 主应用组件 ([`frontend/src/App.vue`](frontend/src/App.vue))

**代码位置**: [`App.vue:36-72`](frontend/src/App.vue:36)

- 三栏响应式布局：
  - 左侧：`flex: 0 0 280px` - 订阅源列表
  - 中间：`flex: 0 0 350px` - 文章列表
  - 右侧：`flex: 1` - 文章阅读区域
- 组件通信：通过 `source-selected` 和 `article-selected` 事件传递数据

```vue
<!-- App.vue:36-47 - 模板结构 -->
<template>
  <div id="app-container">
    <div class="left-pane">
      <SourceList @source-selected="handleSourceSelected" />
    </div>
    <div class="center-pane">
      <ArticleList
        :articles="articles"
        @article-selected="handleArticleSelected"
      />
    </div>
    <div class="right-pane">
      <ArticleView :article="selectedArticle" />
    </div>
  </div>
</template>
```

#### 订阅源列表组件 ([`frontend/src/components/SourceList.vue`](frontend/src/components/SourceList.vue))

**代码位置**: [`SourceList.vue:1-84`](frontend/src/components/SourceList.vue:1)

- **状态管理**：
  - `sources` - 订阅源列表
  - `newSourceUrl` - 输入框绑定
  - `selectedSourceId` - 当前选中的订阅源 ID
- **功能实现**：
  - `onMounted` 时获取所有订阅源
  - `addSource` - POST 到 `/api/sources`
  - `deleteSource` - DELETE 到 `/api/sources/:id`

```javascript
// SourceList.vue:11-21 - 加载订阅源
onMounted(async () => {
  try {
    const response = await fetch("/api/sources");
    sources.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch sources:", error);
  }
});

// SourceList.vue:24-53 - 添加订阅源
const addSource = async () => {
  const response = await fetch("/api/sources", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url: newSourceUrl.value }),
  });

  if (response.status === 409) {
    alert("Feed source already exists.");
    return;
  }

  const newSource = await response.json();
  sources.value.push(newSource);
};
```

#### 文章列表组件 ([`frontend/src/components/ArticleList.vue`](frontend/src/components/ArticleList.vue))

**代码位置**: [`ArticleList.vue:1-18`](frontend/src/components/ArticleList.vue:1)

- 接收 `articles` 作为 prop (从 App.vue 传入)
- 发出 `article-selected` 事件通知 App.vue

```javascript
// ArticleList.vue:5-10 - Props 定义
const props = defineProps({
  articles: {
    type: Array,
    required: true,
  },
});

// ArticleList.vue:12-17 - 事件发出
const emit = defineEmits(["article-selected"]);
const selectArticle = (article) => {
  emit("article-selected", article);
};
```

#### 文章阅读组件 ([`frontend/src/components/ArticleView.vue`](frontend/src/components/ArticleView.vue))

**代码位置**: [`ArticleView.vue:1-243`](frontend/src/components/ArticleView.vue:1)

- **状态**：
  - `summary` - 本地摘要
  - `aiSummary` - AI 摘要
  - `isLoadingSummary` / `isLoadingAISummary` - 加载状态
- **功能**：
  - 使用 `marked` 库解析 Markdown (行 33-38)
  - 加载 USE 模型 (行 23-30)
  - 调用本地摘要生成 (行 41-61)
  - 调用 AI 摘要 API (行 64-79)

```javascript
// ArticleView.vue:33-38 - Markdown 渲染
const renderedContent = computed(() => {
  if (props.article && props.article.content) {
    return marked.parse(props.article.content);
  }
  return "";
});

// ArticleView.vue:41-61 - 本地摘要生成
const generateSummary = async () => {
  const tempDiv = document.createElement("div");
  tempDiv.innerHTML = props.article.content;
  const textContent = tempDiv.textContent || tempDiv.innerText || "";

  summary.value = await summarizeText(textContent);
};
```

#### 摘要服务 ([`frontend/src/services/summarizer.js`](frontend/src/services/summarizer.js))

**代码位置**: [`summarizer.js:1-183`](frontend/src/services/summarizer.js:1)

##### 模型加载 (行 16-26)

```javascript
// summarizer.js:16-26
export async function loadModel() {
  if (model) return;
  model = await use.load();
}
```

##### TextRank + MMR 算法实现

- **句子嵌入**：使用 Universal Sentence Encoder (行 68-70)
- **相似度矩阵**：基于句子向量计算余弦相似度 (行 73-83)
- **TextRank 迭代**：阻尼系数 0.85，收敛阈值 1e-4 (行 86-107)
- **MMR 选择**：平衡相关性和多样性 (行 114-152)

```javascript
// summarizer.js:57-159 - 摘要生成核心逻辑
export async function summarizeText(content, numSentences = 3, lambda = 0.7) {
  // 1. 句子切分
  const sentences = content.match(/[^.!?]+[.!?]+/g) || [];

  // 2. 句子嵌入
  const embeddings = await model.embed(sentences);

  // 3. TextRank 排序
  for (let iter = 0; iter < 100; iter++) {
    // 迭代更新 scores
    newScores[i] = 1 - damping + damping * weightedSum;
  }

  // 4. MMR 选择多样句子
  while (summarySentences.length < numSentences) {
    const mmrScore =
      lambda * relevance - (1 - lambda) * maxSimilarityWithSelected;
  }

  return summarySentences.map((item) => item.sentence).join(" ");
}
```

##### AI 摘要 API 调用 (行 166-183)

```javascript
// summarizer.js:166-183
export async function generateAISummary(articleId) {
  const response = await fetch(`/api/articles/${articleId}/ai-summary`, {
    method: "POST",
  });

  const data = await response.json();
  return data.summary;
}
```

## 5. 数据流

### 添加订阅源流程

```
用户输入 RSS URL
       ↓
SourceList.vue POST /api/sources (SourceList.vue:29)
       ↓
后端解析 RSS XML (main.go:186-191)
       ↓
存储到 MongoDB (main.go:198-228)
       ↓
返回新订阅源，前端更新列表 (SourceList.vue:46-48)
```

### 阅读文章流程

```
用户点击订阅源
       ↓
App.vue GET /api/sources/:id/articles (App.vue:18)
       ↓
后端查询 MongoDB (main.go:287-297)
       ↓
返回文章列表，前端显示在 ArticleList
       ↓
用户点击文章
       ↓
ArticleView 渲染 Markdown (ArticleView.vue:33-38)
```

### 生成摘要流程

```
用户点击"生成摘要"按钮
       ↓
ArticleView 调用 summarizer.js
       ↓
本地摘要：
  - 加载 USE 模型 (summarizer.js:16-26)
  - 句子切分和嵌入 (summarizer.js:57-70)
  - TextRank 排序 (summarizer.js:86-111)
  - MMR 选择 (summarizer.js:114-158)
       ↓
AI 摘要：
  - POST /api/articles/:id/ai-summary (summarizer.js:166-173)
  - 后端调用 OpenAI API (main.go:345-373)
       ↓
返回摘要，前端显示
```

## 6. 环境配置

### 后端环境变量 ([`main.go:24-54`](backend/main.go:24))

| 变量名              | 必需 | 说明                          | 代码位置      |
| ------------------- | ---- | ----------------------------- | ------------- |
| `OPENAI_API_KEY`    | 否   | OpenAI API 密钥               | main.go:31    |
| `OPENAI_BASE_URL`   | 否   | OpenAI API 基础 URL           | main.go:37-42 |
| `OPENAI_MODEL_NAME` | 否   | 模型名称 (默认 gpt-3.5-turbo) | main.go:47-53 |

### 数据库连接 ([`db/db.go:16-33`](backend/db/db.go:16))

```go
// 连接地址
mongodb://root:password@localhost:27017

// 数据库名
rss_reader

// 集合
- sources (订阅源集合)
- articles (文章集合)
```

## 7. 部署说明

### 开发环境

```bash
# 后端
cd backend
go run main.go
# 服务端口: http://localhost:8080

# 前端
cd frontend
pnpm dev
# 服务端口: http://localhost:5173
```

### 生产环境

- 后端：`go build -o rss-server main.go`
- 前端：`pnpm build` (输出到 `dist` 目录)
