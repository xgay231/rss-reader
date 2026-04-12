# 功能说明文档

## 1. 功能概述

Web RSS 阅读器提供以下核心功能：

1. **订阅源管理** - 添加、删除、查看 RSS 订阅源
2. **文章获取** - 自动从订阅源抓取并存储文章
3. **文章阅读** - 查看文章详细内容
4. **本地摘要** - 使用 TextRank + MMR 算法生成本地摘要
5. **AI 摘要** - 调用 OpenAI API 生成智能摘要

## 2. 功能详情

### 2.1 订阅源管理

#### 添加订阅源

**前端实现** - [`SourceList.vue:24-53`](frontend/src/components/SourceList.vue:24)

```javascript
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

**后端实现** - [`main.go:168-233`](backend/main.go:168)

1. 验证 URL 必填 (`binding:"required"`)
2. 检查订阅源是否已存在 (`bson.M{"url": json.URL}`)
3. 使用 `gofeed.NewParser()` 解析 RSS
4. 插入订阅源到 `sources` 集合
5. 批量插入历史文章到 `articles` 集合

```go
// main.go:168-176 - API 入口
sources.POST("", func(c *gin.Context) {
    var json struct {
        URL string `json:"url" binding:"required"`
    }

    if err := c.ShouldBindJSON(&json); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 检查重复
    var existingSource FeedSource
    err := db.SourceCollection.FindOne(context.Background(), bson.M{"url": json.URL}).Decode(&existingSource)
    if err == nil {
        c.JSON(409, gin.H{"error": "Feed source already exists"})
        return
    }

    // 解析 RSS
    fp := gofeed.NewParser()
    feed, err := fp.ParseURL(json.URL)

    // 存储订阅源和文章
    res, _ := db.SourceCollection.InsertOne(context.Background(), newSource)
    db.ArticleCollection.InsertMany(context.Background(), articles)
})
```

**限制**：

- URL 必须为有效的 RSS/Atom 链接
- 不允许重复添加相同 URL 的订阅源 (返回 409)

#### 查看订阅源

**前端实现** - [`SourceList.vue:11-21`](frontend/src/components/SourceList.vue:11)

```javascript
onMounted(async () => {
  const response = await fetch("/api/sources");
  sources.value = await response.json();
});
```

- 左侧面板显示所有已添加的订阅源
- 订阅源按添加时间顺序排列
- 点击订阅源触发 `selectSource` 函数发出事件

**后端实现** - [`main.go:236-251`](backend/main.go:236)

```go
sources.GET("", func(c *gin.Context) {
    var sources []FeedSource
    cursor, err := db.SourceCollection.Find(context.Background(), bson.M{})
    cursor.All(context.Background(), &sources)
    c.JSON(200, sources)
})
```

#### 删除订阅源

**前端实现** - [`SourceList.vue:61-83`](frontend/src/components/SourceList.vue:61)

```javascript
const deleteSource = async (sourceId, event) => {
  event.stopPropagation(); // 防止触发选择事件

  if (
    !confirm("Are you sure you want to delete this feed and all its articles?")
  ) {
    return;
  }

  const response = await fetch(`/api/sources/${sourceId}`, {
    method: "DELETE",
  });

  sources.value = sources.value.filter((s) => s.id !== sourceId);

  if (selectedSourceId.value === sourceId) {
    emit("source-selected", null);
  }
};
```

**后端实现** - [`main.go:254-276`](backend/main.go:254)

```go
sources.DELETE("/:id", func(c *gin.Context) {
    id, _ := primitive.ObjectIDFromHex(c.Param("id"))

    // 删除订阅源
    db.SourceCollection.DeleteOne(context.Background(), bson.M{"_id": id})

    // 删除关联文章（保留已收藏的文章）
    db.ArticleCollection.DeleteMany(context.Background(), bson.M{"sourceId": id, "isStarred": false})

    c.JSON(200, gin.H{"status": "ok"})
})
```

**注意**：删除订阅源时，已收藏的文章会被保留，其 sourceId 保持不变。当重新订阅同一 URL 时，系统会自动将孤立已收藏文章的 sourceId 更新到新订阅源，避免重复。

### 2.2 文章获取

#### 初始获取

添加订阅源时，后端自动抓取该订阅源的所有历史文章并存储到数据库。

**实现** - [`main.go:205-220`](backend/main.go:205)

```go
for _, item := range feed.Items {
    content := item.Content
    if content == "" {
        content = item.Description
    }
    article := Article{
        SourceID:    sourceID,
        GUID:        item.GUID,
        Title:       item.Title,
        URL:         item.Link,
        Description: item.Description,
        Content:     content,
    }
    articles = append(articles, article)
}

if len(articles) > 0 {
    opts := options.InsertMany().SetOrdered(false)
    db.ArticleCollection.InsertMany(context.Background(), articles, opts)
}
```

#### 定时更新

**实现** - [`main.go:141-150`](backend/main.go:141)

```go
go func() {
    // 启动时执行一次
    updateFeeds()

    // 每分钟定时更新
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        updateFeeds()
    }
}()
```

**更新逻辑** - [`main.go:74-135`](backend/main.go:74)

1. 获取所有订阅源
2. 遍历解析每个 RSS URL
3. 检查每篇文章 GUID 是否已存在
4. 仅插入不重复的新文章

```go
func updateFeeds() {
    // 1. 获取所有订阅源
    cursor, _ := db.SourceCollection.Find(ctx, bson.M{})

    fp := gofeed.NewParser()
    for _, source := range sources {
        // 2. 解析 RSS
        feed, _ := fp.ParseURL(source.URL)

        // 3. 检查重复
        for _, item := range feed.Items {
            filter := bson.M{"guid": item.GUID, "sourceId": source.ID}
            err := db.ArticleCollection.FindOne(ctx, filter).Err()

            if err == mongo.ErrNoDocuments {
                // 4. 新文章
                newArticles = append(newArticles, article)
            }
        }

        // 5. 批量插入
        if len(newArticles) > 0 {
            db.ArticleCollection.InsertMany(ctx, newArticles, opts)
        }
    }
}
```

### 2.3 文章阅读

#### 文章列表

**前端实现** - [`ArticleList.vue:21-37`](frontend/src/components/ArticleList.vue:21)

```vue
<template>
  <div class="article-list-container">
    <ul v-if="articles.length > 0">
      <li
        v-for="article in articles"
        :key="article.id"
        @click="selectArticle(article)"
      >
        <h3>{{ article.title }}</h3>
        <p>{{ article.description }}</p>
      </li>
    </ul>
    <div v-else class="no-articles">
      <p>Select a feed to see its articles.</p>
    </div>
  </div>
</template>
```

- 中间面板显示当前选中订阅源的所有文章
- 每条显示标题和描述摘要 (截断显示)
- 点击文章触发 `selectArticle` 发出 `article-selected` 事件

**后端实现** - [`main.go:279-300`](backend/main.go:279)

```go
sources.GET("/:id/articles", func(c *gin.Context) {
    id, _ := primitive.ObjectIDFromHex(c.Param("id"))

    var articles []Article
    cursor, _ := db.ArticleCollection.Find(context.Background(), bson.M{"sourceId": id})
    cursor.All(context.Background(), &articles)

    c.JSON(200, articles)
})
```

#### 文章内容

**前端实现** - [`ArticleView.vue:33-127`](frontend/src/components/ArticleView.vue:33)

```javascript
// Markdown 渲染
const renderedContent = computed(() => {
  if (props.article && props.article.content) {
    return marked.parse(props.article.content);
  }
  return "";
});
```

```vue
<template>
  <div class="article-view-container">
    <div v-if="article">
      <h1>
        <a :href="article.url" target="_blank" rel="noopener noreferrer">
          {{ article.title }}
        </a>
      </h1>

      <div class="article-content" v-html="renderedContent"></div>
    </div>
  </div>
</template>
```

- 右侧面板显示文章详细内容
- 使用 `marked` 库解析 Markdown 为 HTML
- 标题链接指向原文

### 2.4 本地摘要生成

#### 技术原理

使用 **TextRank + MMR** 算法

**前端实现** - [`summarizer.js:57-159`](frontend/src/services/summarizer.js:57)

```javascript
export async function summarizeText(content, numSentences = 3, lambda = 0.7) {
  // 1. 句子切分 - 按 .!? 句末标点切分
  const sentences = content.match(/[^.!?]+[.!?]+/g) || [];
  if (sentences.length <= numSentences) {
    return content;
  }

  // 2. 句子嵌入 - 使用 Universal Sentence Encoder
  const embeddings = await model.embed(sentences);
  const sentenceVectors = tf.unstack(embeddings);

  // 3. 相似度矩阵 - 计算余弦相似度
  const similarityMatrix = [];
  for (let i = 0; i < sentences.length; i++) {
    for (let j = 0; j < sentences.length; j++) {
      if (i === j) row.push(0);
      else row.push(cosineSimilarity(sentenceVectors[i], sentenceVectors[j]));
    }
    similarityMatrix.push(row);
  }

  // 4. TextRank 迭代 - 阻尼系数 0.85
  let scores = new Array(sentences.length).fill(1);
  const damping = 0.85;
  const epsilon = 1e-4;

  for (let iter = 0; iter < 100; iter++) {
    const newScores = new Array(sentences.length).fill(0);
    for (let i = 0; i < sentences.length; i++) {
      let weightedSum = 0;
      for (let j = 0; j < sentences.length; j++) {
        if (i !== j) {
          const sumOfOutgoingWeights = similarityMatrix[j].reduce(
            (a, b) => a + b,
            0
          );
          weightedSum +=
            (similarityMatrix[j][i] / sumOfOutgoingWeights) * scores[j];
        }
      }
      newScores[i] = 1 - damping + damping * weightedSum;
    }
    scores = newScores;
    if (maxDiff < epsilon) break;
  }

  // 5. MMR 选择 - 平衡相关性和多样性
  const summarySentences = [];
  const selectedIndices = new Set();

  // 选择得分最高的句子
  if (rankedSentences.length > 0) {
    summarySentences.push(rankedSentences[0]);
    selectedIndices.add(rankedSentences[0].index);
  }

  // MMR 迭代选择
  while (summarySentences.length < numSentences) {
    let bestCandidate = null;
    let maxMmrScore = -Infinity;

    for (const candidate of rankedSentences) {
      if (selectedIndices.has(candidate.index)) continue;

      const relevance = candidate.score;
      let maxSimilarityWithSelected = 0;

      // 计算与已选句子的最大相似度
      for (const selected of summarySentences) {
        const similarity = cosineSimilarity(candidate.vector, selected.vector);
        maxSimilarityWithSelected = Math.max(
          maxSimilarityWithSelected,
          similarity
        );
      }

      // MMR 公式
      const mmrScore =
        lambda * relevance - (1 - lambda) * maxSimilarityWithSelected;
      if (mmrScore > maxMmrScore) {
        maxMmrScore = mmrScore;
        bestCandidate = candidate;
      }
    }

    if (bestCandidate) {
      summarySentences.push(bestCandidate);
      selectedIndices.add(bestCandidate.index);
    } else {
      break;
    }
  }

  // 按原文顺序返回
  return summarySentences
    .sort((a, b) => a.index - b.index)
    .map((item) => item.sentence)
    .join(" ");
}
```

**前端调用** - [`ArticleView.vue:41-61`](frontend/src/components/ArticleView.vue:41)

```javascript
const generateSummary = async () => {
  if (!props.article || !props.article.content) return;

  isLoadingSummary.value = true;

  // 提取纯文本
  const tempDiv = document.createElement("div");
  tempDiv.innerHTML = props.article.content;
  const textContent = tempDiv.textContent || tempDiv.innerText || "";

  summary.value = await summarizeText(textContent);

  isLoadingSummary.value = false;
};
```

#### 使用方式

1. 打开文章阅读视图
2. 点击 "Generate Local Summary" 按钮
3. 等待模型加载和摘要生成
4. 摘要显示在下方

**注意**：

- 首次使用需要下载模型 (约 30MB)
- 模型缓存后，后续使用更快

### 2.5 AI 摘要生成

#### 技术原理

调用 OpenAI API，使用 GPT 模型对文章内容进行摘要。

**后端实现** - [`main.go:325-374`](backend/main.go:325)

```go
articles.POST("/:id/ai-summary", func(c *gin.Context) {
    // 1. 检查 AI 服务是否可用
    if aiClient == nil {
        c.JSON(503, gin.H{"error": "AI service is not available"})
        return
    }

    // 2. 获取文章内容
    id, _ := primitive.ObjectIDFromHex(c.Param("id"))
    var article Article
    err := db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&article)

    // 3. 调用 OpenAI API
    resp, err := aiClient.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: aiModelName,
            Messages: []openai.ChatCompletionMessage{
                {
                    Role:    openai.ChatMessageRoleSystem,
                    Content: "You are a helpful assistant that summarizes articles.",
                },
                {
                    Role:    openai.ChatMessageRoleUser,
                    Content: "Please summarize the following article content:\n\n" + article.Content,
                },
            },
        },
    )

    // 4. 返回摘要
    c.JSON(200, gin.H{"summary": resp.Choices[0].Message.Content})
})
```

**前端实现** - [`summarizer.js:166-183`](frontend/src/services/summarizer.js:166)

```javascript
export async function generateAISummary(articleId) {
  try {
    const response = await fetch(`/api/articles/${articleId}/ai-summary`, {
      method: "POST",
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || "Failed to fetch AI summary");
    }

    const data = await response.json();
    return data.summary;
  } catch (error) {
    console.error("Error generating AI summary:", error);
    throw error;
  }
}
```

**前端调用** - [`ArticleView.vue:64-79`](frontend/src/components/ArticleView.vue:64)

```javascript
const generateAISummaryHandler = async () => {
  if (!props.article || !props.article.id) return;

  isLoadingAISummary.value = true;

  try {
    aiSummary.value = await generateAISummary(props.article.id);
  } catch (error) {
    aiSummaryError.value = "Failed to generate AI summary.";
  } finally {
    isLoadingAISummary.value = false;
  }
};
```

#### 配置要求

需要设置以下环境变量 ([`main.go:24-54`](backend/main.go:24)):

```go
func init() {
    apiKey := os.Getenv("OPENAI_API_KEY")
    baseURL := os.Getenv("OPENAI_BASE_URL")
    aiModelName := os.Getenv("OPENAI_MODEL_NAME")

    if aiModelName == "" {
        aiModelName = openai.GPT3Dot5Turbo // 默认模型
    }
}
```

| 变量名              | 必需 | 说明                          |
| ------------------- | ---- | ----------------------------- |
| `OPENAI_API_KEY`    | 是   | OpenAI API 密钥               |
| `OPENAI_BASE_URL`   | 否   | API 端点地址 (用于代理)       |
| `OPENAI_MODEL_NAME` | 否   | 模型名称 (默认 gpt-3.5-turbo) |

**注意**：

- 如未配置 `OPENAI_API_KEY`，AI 摘要功能将不可用 (返回 503)
- API 调用可能产生费用

## 3. API 接口

### 3.1 订阅源接口

| 方法   | 路径                        | 代码位置    | 说明               |
| ------ | --------------------------- | ----------- | ------------------ |
| POST   | `/api/sources`              | main.go:168 | 添加订阅源         |
| GET    | `/api/sources`              | main.go:236 | 获取所有订阅源     |
| DELETE | `/api/sources/:id`          | main.go:254 | 删除订阅源（保留收藏）|
| GET    | `/api/sources/:id/articles` | main.go:279 | 获取订阅源下的文章 |
| PUT    | `/api/sources/:id/group`    | main.go:334 | 将订阅源分配到分组 |

### 3.2 分组接口

| 方法   | 路径                  | 代码位置    | 说明         |
| ------ | --------------------- | ----------- | ------------ |
| POST   | `/api/groups`         | main.go:375 | 创建分组     |
| GET    | `/api/groups`         | main.go:399 | 获取所有分组 |
| PUT    | `/api/groups/:id`     | main.go:420 | 更新分组     |
| DELETE | `/api/groups/:id`     | main.go:447 | 删除分组     |

### 3.3 文章接口

| 方法   | 路径                           | 代码位置    | 说明         |
| ------ | ------------------------------ | ----------- | ------------ |
| GET    | `/api/articles/starred`        | main.go:496 | 获取收藏文章 |
| GET    | `/api/articles/:id`            | main.go:515 | 获取单篇文章 |
| POST   | `/api/articles/:id/ai-summary` | main.go:533 | 生成 AI 摘要 |
| POST   | `/api/articles/:id/star`       | main.go:596 | 收藏文章     |
| DELETE | `/api/articles/:id/star`       | main.go:614 | 取消收藏     |

## 4. 用户界面

### 布局结构

```
┌──────────────────────────────────────────────────────────────────┐
│                         三栏布局                                 │
├────────────┬─────────────────┬─────────────────────────────────┤
│            │                 │                                 │
│  订阅源列表  │    文章列表     │       文章阅读与摘要            │
│  (280px)   │    (350px)      │        (自适应宽度)              │
│            │                 │                                 │
└────────────┴─────────────────┴─────────────────────────────────┘
```

### 交互流程

1. **添加订阅源**：左侧输入框 → 添加按钮 → 列表更新
2. **查看文章**：点击订阅源 → 中间显示文章 → 点击文章 → 右侧阅读
3. **生成摘要**：阅读文章 → 点击摘要按钮 → 等待生成 → 显示结果

## 5. 错误处理

| 场景                | 代码位置        | 处理方式                          |
| ------------------- | --------------- | --------------------------------- |
| RSS URL 无效        | main.go:187-191 | 返回 500，前端提示"解析失败"      |
| 订阅源重复添加      | main.go:180-184 | 返回 409，前端提示"已存在"        |
| 数据库连接失败      | db/db.go:21-27  | 后端日志记录，进程退出            |
| AI 服务不可用       | main.go:326-329 | 返回 503，前端提示"AI 服务不可用" |
| OpenAI API 调用失败 | main.go:362-365 | 返回 500，日志记录错误详情        |

## 6. 性能考虑

| 优化项   | 说明                             | 代码位置           |
| -------- | -------------------------------- | ------------------ |
| 模型缓存 | USE 模型加载后缓存在内存中       | summarizer.js:4,16 |
| 增量更新 | RSS 抓取仅处理新文章 (检查 GUID) | main.go:100-121    |
| 异步处理 | 后台任务不影响 API 响应          | main.go:141-150    |
| 批量插入 | 文章批量写入数据库 (InsertMany)  | main.go:124-131    |

## 7. 扩展性

系统设计支持以下扩展：

- 添加更多摘要算法
- 支持用户认证和个性化
- 添加文章收藏功能
- 支持更多 RSS 格式
- 添加标签和分类功能
