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
│  ┌──────────┐    ┌──────────────┐    ┌────────────────────┐     │
│  │ SourceList│   │ ArticleList  │    │   ArticleView      │     │
│  │ (订阅源)  │    │ (文章列表)   │    │   (文章阅读)       │     │
│  └─────┬────┘    └──────┬───────┘    └─────────┬──────────┘     │
│        │                │                      │                │
│        └────────────────┼──────────────────────┘                │
│                         │                                      │
│                  ┌──────▼──────┐                               │
│                  │  App.vue    │                               │
│                  │ (三栏布局)  │                               │
│                  └──────┬──────┘                               │
└─────────────────────────┼──────────────────────────────────────┘
                          │ HTTP API (JWT Protected)
┌─────────────────────────▼──────────────────────────────────────┐
│                      后端 (Go + Gin)                            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    API 路由层                            │    │
│  │  /ping  /api/auth/*  /api/sources/*  /api/articles/*   │    │
│  │  /api/groups/*  /api/settings/*  /api/daily-summary/*  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌───────────────────────────▼────────────────────────────┐    │
│  │                    业务逻辑层                           │    │
│  │  • 用户认证 (JWT, 注册/登录/刷新/登出)                    │    │
│  │  • FeedSource 管理 (添加、删除、获取、分组、拖拽排序)     │    │
│  │  • Article 管理 (获取、摘要、收藏、已读/未读状态)         │    │
│  │  • AI 摘要服务 (OpenAI 集成，自动摘要生成)               │    │
│  │  • 每日总结服务 (Cron 调度、SMTP 邮件发送)               │    │
│  │  • 定时更新任务 (后台抓取 RSS，动态间隔 1-1440 分钟)      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌───────────────────────────▼────────────────────────────┐    │
│  │                    数据访问层                           │    │
│  │                  MongoDB 数据库                         │    │
│  │  • users (用户集合)                                    │    │
│  │  • sources (订阅源集合)  • articles (文章集合)          │    │
│  │  • groups (分组集合)                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## 3. 技术栈

### 后端技术栈

| 技术              | 版本    | 用途                    |
| ----------------- | ------- | ---------------------- |
| Go                | ≥1.21   | 编程语言                |
| Gin               | latest  | Web 框架                |
| MongoDB           | -       | 数据库                  |
| gofeed            | latest  | RSS 解析                |
| go-openai         | latest  | OpenAI API 客户端       |
| godotenv          | latest  | 环境变量管理            |
| jwt               | v5      | JWT 身份验证            |
| bcrypt            | latest  | 密码哈希                |
| robfig/cron       | v3      | 定时任务调度             |
| go-mail           | v2      | SMTP 邮件发送           |

### 前端技术栈

| 技术                                          | 版本      | 用途           |
| --------------------------------------------- | --------- | -------------- |
| Vue                                           | 3.x       | 框架           |
| Vite                                          | -         | 构建工具       |
| @tensorflow/tfjs                              | ^4.22.0   | 机器学习运行时 |
| @tensorflow-models/universal-sentence-encoder| ^1.3.3    | 句子嵌入模型   |
| marked                                        | ^16.3.0   | Markdown 解析  |
| dompurify                                     | latest    | HTML  sanitization |

## 4. 核心模块

### 4.1 后端模块

#### 数据库连接模块 ([`backend/db/db.go`](backend/db/db.go))

**代码位置**: [`db/db.go:20-40`](backend/db/db.go:20)

- 初始化 MongoDB 连接：`mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))`
- 连接数据库 `rss_reader`
- 创建四个集合：
  - `UserCollection` - 存储用户账户
  - `ArticleCollection` - 存储文章数据
  - `SourceCollection` - 存储订阅源数据
  - `GroupCollection` - 存储分组数据
- 10 秒超时控制，使用 `context.WithTimeout`
- 创建 email 唯一索引和 readStatus 索引

```go
// db/db.go:20-40
func ConnectDatabase() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal(err)
    }
    log.Println("Successfully connected to MongoDB!")
    Client = client
    DB = client.Database("rss_reader")
    UserCollection = DB.Collection("users")
    ArticleCollection = DB.Collection("articles")
    SourceCollection = DB.Collection("sources")
    GroupCollection = DB.Collection("groups")

    // 创建 email 唯一索引
    _, err = UserCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys:    map[string]interface{}{"email": 1},
        Options: options.Index().SetUnique(true),
    })

    // 创建 readStatus 索引
    _, err = ArticleCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "readStatus", Value: 1}},
    })
}
```

#### 用户认证模块 ([`backend/middleware/auth.go`](backend/middleware/auth.go))

**代码位置**: [`middleware/auth.go:20-54`](backend/middleware/auth.go:20)

- JWT 中间件验证请求 Authorization header
- 提取 Bearer token 并验证有效性
- 将 userID 注入到 Gin context 供后续处理使用

```go
// middleware/auth.go:20-54
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
            c.Abort()
            return
        }

        tokenString := parts[1]
        claims := &Claims{}
        token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
            return JWTSecret, nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
            c.Abort()
            return
        }

        c.Set("userID", claims.UserID)
        c.Next()
    }
}
```

#### 用户认证处理器 ([`backend/handlers/auth.go`](backend/handlers/auth.go))

**代码位置**: [`handlers/auth.go:19-82`](backend/handlers/auth.go:19)

- `Register` - 用户注册，bcrypt 密码哈希，15 分钟 Access Token + 7 天 Refresh Token
- `Login` - 用户登录，验证密码，生成 JWT
- `Refresh` - 刷新 Access Token
- `Logout` - 登出（客户端删除 token）
- `GetMe` - 获取当前用户信息

```go
// handlers/auth.go:84-138 - Login 函数
func Login(c *gin.Context) {
    var json struct {
        Email    string `json:"email" binding:"required,email"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&json); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user struct {
        ID           primitive.ObjectID `bson:"_id"`
        Email        string             `bson:"email"`
        Username     string             `bson:"username"`
        PasswordHash string             `bson:"passwordHash"`
    }
    err := db.UserCollection.FindOne(context.Background(), bson.M{"email": json.Email}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(json.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
        return
    }

    accessToken, _ := generateAccessToken(user.ID.Hex())
    refreshToken, _ := generateRefreshToken(user.ID.Hex())

    c.JSON(http.StatusOK, gin.H{
        "user": gin.H{"id": user.ID, "email": user.Email, "username": user.Username},
        "accessToken":  accessToken,
        "refreshToken": refreshToken,
        "expiresIn":    900,
    })
}
```

#### 数据模型 ([`backend/models/user.go`](backend/models/user.go))

```go
// models/user.go:10-23 - User 模型
type User struct {
    ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    Email               string             `json:"email" bson:"email"`
    Username            string             `json:"username" bson:"username"`
    PasswordHash        string             `json:"-" bson:"passwordHash"`
    FeedUpdateInterval  int                `json:"feedUpdateInterval" bson:"feedUpdateInterval"`  // 默认 15 分钟
    AutoSummary         bool               `json:"autoSummary" bson:"autoSummary"`              // 默认 true
    DailySummaryEnabled bool               `json:"dailySummaryEnabled" bson:"dailySummaryEnabled"`
    DailySummaryTime    string             `json:"dailySummaryTime" bson:"dailySummaryTime"`    // 格式 "HH:MM"
    DailySummaryEmail   string             `json:"dailySummaryEmail" bson:"dailySummaryEmail"`
    SmtpPassword         string             `json:"-" bson:"smtpPassword"`
    CreatedAt           time.Time          `json:"createdAt" bson:"createdAt"`
    UpdatedAt           time.Time          `json:"updatedAt" bson:"updatedAt"`
}
```

#### FeedSource 和 Article 模型 ([`backend/main.go`](backend/main.go))

```go
// main.go:152-168 - FeedSource 模型
type FeedSource struct {
    ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    UserID  primitive.ObjectID `json:"userId" bson:"userId"`
    Name    string             `json:"name" bson:"name"`
    URL     string             `json:"url" bson:"url"`
    GroupID primitive.ObjectID `json:"groupId" bson:"groupId"`
}

// main.go:170-186 - Article 模型
type Article struct {
    ID                 primitive.ObjectID `json:"id" bson:"_id,omitempty"`
    UserID             primitive.ObjectID `json:"userId" bson:"userId"`
    SourceID           primitive.ObjectID `json:"sourceId" bson:"sourceId"`
    GUID               string             `json:"guid" bson:"guid"`
    Title              string             `json:"title" bson:"title"`
    URL                string             `json:"url" bson:"url"`
    Description        string             `json:"description" bson:"description"`
    Content            string             `json:"content" bson:"content"`
    PublishedAt        *time.Time        `json:"publishedAt" bson:"publishedAt"`
    IsStarred          bool               `json:"isStarred" bson:"isStarred"`
    StarredAt          time.Time         `json:"starredAt" bson:"starredAt"`
    Summary            string             `json:"summary" bson:"summary"`
    SummaryGeneratedAt *time.Time        `json:"summaryGeneratedAt" bson:"summaryGeneratedAt"`
    ReadStatus         string             `json:"readStatus" bson:"readStatus"` // "unread" | "read"
}
```

#### RSS 更新后台任务 ([`backend/main.go:670-699`](backend/main.go:670))

- 从数据库获取当前用户的所有订阅源
- 使用 `gofeed.NewParser()` 解析每个订阅源
- 增量更新：检查 GUID 是否已存在，避免重复
- 使用 `InsertMany` 批量插入新文章
- 新文章自动生成摘要（如果用户启用 autoSummary）

```go
// main.go:670-699 - updateFeeds 函数
func updateFeeds() {
    var sources []FeedSource
    cursor, err := db.SourceCollection.Find(ctx, bson.M{})
    cursor.All(ctx, &sources)

    fp := gofeed.NewParser()
    for _, source := range sources {
        feed, err := fp.ParseURL(source.URL)
        if err != nil {
            log.Printf("Failed to parse feed %s: %v", source.URL, err)
            continue
        }
        insertArticlesAndGenerateSummary(source, feed.Items, ctx)
    }
}
```

#### insertArticlesAndGenerateSummary ([`backend/main.go:709-790`](backend/main.go:709))

- 检查文章是否已存在（通过 GUID）
- 批量插入新文章
- 异步为新文章生成 AI 摘要（信号量限制 5 个并发）

```go
// main.go:709-790
func insertArticlesAndGenerateSummary(source FeedSource, items []*gofeed.Item, ctx context.Context) []interface{} {
    var newArticles []interface{}
    for _, item := range items {
        filter := bson.M{"guid": item.GUID, "sourceId": source.ID}
        err := db.ArticleCollection.FindOne(ctx, filter).Err()
        if err == mongo.ErrNoDocuments {
            // 新文章，添加到列表
            newArticles = append(newArticles, article)
        }
    }

    if len(newArticles) == 0 {
        return nil
    }

    opts := options.InsertMany().SetOrdered(false)
    result, err := db.ArticleCollection.InsertMany(ctx, newArticles, opts)

    // 自动生成摘要（异步，信号量限制 5 个并发）
    go func(source FeedSource, insertedIDs []interface{}) {
        semaphore := make(chan struct{}, 5)
        var wg sync.WaitGroup
        for _, id := range insertedIDs {
            articleID := id.(primitive.ObjectID)
            wg.Add(1)
            semaphore <- struct{}{}
            go func(aid primitive.ObjectID) {
                defer wg.Done()
                defer func() { <-semaphore }()
                generateSummary(aid, source.UserID)
            }(articleID)
        }
        wg.Wait()
    }(source, result.InsertedIDs)

    return result.InsertedIDs
}
```

#### 每日总结定时调度 ([`backend/main.go:1526-1543`](backend/main.go:1526))

- 使用 robfig/cron 每分钟检查一次
- 根据用户设置的 `dailySummaryTime` 发送邮件

```go
// main.go:1526-1543
func initDailySummaryScheduler() {
    c := cron.New()
    c.AddFunc("* * * * *", func() {
        checkAndSendDailySummaries()
    })
    c.Start()
}
```

#### API 路由实现

| 路由                                  | 代码位置              | 功能说明                    |
| ------------------------------------- | -------------------- | -------------------------- |
| `POST /api/auth/register`             | handlers/auth.go:19  | 用户注册                    |
| `POST /api/auth/login`                | handlers/auth.go:84  | 用户登录                    |
| `POST /api/auth/refresh`              | handlers/auth.go:140 | 刷新 Access Token           |
| `POST /api/auth/logout`               | handlers/auth.go:190 | 用户登出                    |
| `GET /api/auth/me`                    | handlers/auth.go:197 | 获取当前用户信息            |
| `GET /api/settings`                   | main.go:220           | 获取用户设置                |
| `PUT /api/settings`                   | main.go:274           | 更新用户设置                |
| `GET /api/daily-summary/settings`     | main.go:565           | 获取每日总结设置            |
| `PUT /api/daily-summary/settings`     | main.go:601           | 更新每日总结设置            |
| `POST /api/daily-summary/send`        | main.go:514           | 手动发送每日总结            |
| `POST /api/sources`                   | main.go:868           | 添加订阅源                  |
| `GET /api/sources`                    | main.go:942           | 获取用户所有订阅源          |
| `DELETE /api/sources/:id`             | main.go:966           | 删除订阅源（保留收藏文章）   |
| `GET /api/sources/:id/articles`       | main.go:997           | 获取订阅源下的所有文章      |
| `PUT /api/sources/:id/mark-all-read`  | main.go:1027          | 标记所有文章为已读          |
| `PUT /api/sources/:id/group`          | main.go:1059          | 将订阅源分配到分组          |
| `POST /api/sources/refresh`           | main.go:1101          | 手动刷新所有订阅源          |
| `POST /api/groups`                    | main.go:1110          | 创建分组                    |
| `GET /api/groups`                     | main.go:1143          | 获取用户所有分组            |
| `PUT /api/groups/:id`                 | main.go:1167          | 更新分组                    |
| `DELETE /api/groups/:id`              | main.go:1200          | 删除分组                    |
| `GET /api/articles/starred`           | main.go:1235          | 获取所有收藏文章            |
| `GET /api/articles`                   | main.go:1260          | 获取文章（支持已读/未读筛选）|
| `GET /api/articles/:id`               | main.go:1304          | 获取单篇文章详情            |
| `POST /api/articles/:id/ai-summary`   | main.go:1328          | 调用 OpenAI API 生成摘要    |
| `POST /api/articles/:id/star`         | main.go:1397          | 收藏文章                    |
| `DELETE /api/articles/:id/star`       | main.go:1426          | 取消收藏文章                |
| `PUT /api/articles/:id/read`          | main.go:1455          | 标记文章为已读              |
| `DELETE /api/articles/:id/read`       | main.go:1487          | 标记文章为未读              |

### 4.2 前端模块

#### 主应用组件 ([`frontend/src/App.vue`](frontend/src/App.vue))

**代码位置**: [`App.vue:1-96`](frontend/src/App.vue:1)

- 三栏响应式布局（可拖拽调整宽度）
- 移动端适配：stack-based 导航
- JWT 认证状态管理
- 组件通信：通过 `source-selected` 和 `article-selected` 事件传递数据

```vue
<!-- App.vue:249-314 - 模板结构 -->
<template>
  <div id="app-container">
    <AuthForm v-if="!auth.isAuthenticated()" />
    <template v-else>
      <SettingsModal :show="showSettings" @close="showSettings = false" />
      <div class="left-pane">
        <SourceList ref="sourceListRef" @source-selected="handleSourceSelected" @open-settings="showSettings = true" />
      </div>
      <div class="divider" @mousedown="(e) => startDrag(e, 'left')"></div>
      <div class="center-pane">
        <ArticleList :articles="articles" @article-selected="handleArticleSelected" />
      </div>
      <div class="divider" @mousedown="(e) => startDrag(e, 'center')"></div>
      <div class="right-pane">
        <ArticleView :article="selectedArticle" />
      </div>
    </template>
  </div>
</template>
```

#### 用户认证 ([`frontend/src/composables/useAuth.js`](frontend/src/composables/useAuth.js))

**代码位置**: [`composables/useAuth.js:1-144`](frontend/src/composables/useAuth.js:1)

- 提供/注入模式管理认证状态
- 登录、注册、登出、刷新 token
- 自动检查登录状态

```javascript
// composables/useAuth.js:19-35 - login 函数
const login = async (email, password) => {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
  })

  if (!response.ok) {
    const error = await response.json()
    throw new Error(error.error || 'Login failed')
  }

  const data = await response.json()
  user.value = data.user
  setTokens(data.accessToken, data.refreshToken)
  return data
}
```

#### 订阅源列表组件 ([`frontend/src/components/SourceList.vue`](frontend/src/components/SourceList.vue))

**代码位置**: [`SourceList.vue:1-630`](frontend/src/components/SourceList.vue:1)

- 订阅源管理：添加、删除、选择
- 分组管理：创建、编辑、删除、折叠/展开
- 拖拽排序：支持分组和订阅源的拖拽重排
- 订阅源分配到分组
- 收藏夹功能（显示收藏文章数）
- 用户菜单：设置、登出

```javascript
// SourceList.vue:196-226 - 添加订阅源
const addSource = async () => {
  const response = await fetchWithAuth('/api/sources', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url: newSourceUrl.value }),
  })

  if (response.status === 409) {
    alert('Feed source already exists.')
    return
  }

  const newSource = await response.json()
  sources.value.push(newSource)
  newSourceUrl.value = ''
}
```

#### 文章列表组件 ([`frontend/src/components/ArticleList.vue`](frontend/src/components/ArticleList.vue))

**代码位置**: [`ArticleList.vue:1-123`](frontend/src/components/ArticleList.vue:1)

- 已读/未读状态筛选
- 按发布时间排序（降序）
- 标记全部已读功能
- 时间格式化（相对时间显示）

```javascript
// ArticleList.vue:41-55 - 筛选和排序
const filteredArticles = computed(() => {
  return props.articles
    .slice()
    .sort((a, b) => {
      const dateA = new Date(a.publishedAt).getTime()
      const dateB = new Date(b.publishedAt).getTime()
      if (isNaN(dateA) || isNaN(dateB)) return 0
      return dateB - dateA
    })
    .filter((article) => {
      if (article.readStatus === 'read' && !showRead.value) return false
      if (article.readStatus === 'unread' && !showUnread.value) return false
      return true
    })
})
```

#### 文章阅读组件 ([`frontend/src/components/ArticleView.vue`](frontend/src/components/ArticleView.vue))

**代码位置**: [`ArticleView.vue:1-406`](frontend/src/components/ArticleView.vue:1)

- 自动检测内容类型（HTML/Markdown/纯文本）
- HTML 内容 sanitization（使用 DOMPurify）
- Markdown 内容渲染（使用 marked）
- AI 摘要生成（调用后端 API）
- 收藏/取消收藏
- 已读/未读状态切换
- 相对时间显示

```javascript
// ArticleView.vue:64-100 - 内容渲染
const renderedContent = computed(() => {
  if (!props.article || !props.article.content) return ""

  const content = props.article.content
  switch (contentType.value) {
    case 'html':
      return DOMPurify.sanitize(content, {
        ALLOWED_TAGS: ['p', 'br', 'b', 'i', 'em', 'strong', 'a', 'img', ...],
        ALLOWED_ATTR: ['href', 'src', 'alt', 'title', ...],
      })
    case 'markdown':
      return marked.parse(content.replace(/\\n/g, '\n'))
    case 'plain':
    default:
      return `<pre style="white-space: pre-wrap; word-wrap: break-word;">${escaped}</pre>`
  }
})
```

#### 设置弹窗 ([`frontend/src/components/SettingsModal.vue`](frontend/src/components/SettingsModal.vue))

**代码位置**: [`SettingsModal.vue:1-286`](frontend/src/components/SettingsModal.vue:1)

- Feed 更新间隔设置（15分钟/60分钟/自定义1-1440分钟）
- 自动摘要开关
- 每日总结邮件开关
- 发送时间、邮箱地址、SMTP 密码配置
- 立即发送每日总结功能

```javascript
// SettingsModal.vue:79-133 - 保存设置
const handleSave = async () => {
  let feedUpdateInterval;
  if (intervalOption.value === "custom") {
    feedUpdateInterval = customInterval.value
  } else {
    feedUpdateInterval = parseInt(intervalOption.value)
  }

  // 保存通用设置
  await fetchWithAuth("/api/settings", {
    method: "PUT",
    body: JSON.stringify({ feedUpdateInterval, autoSummary: autoSummary.value }),
  })

  // 保存每日总结设置
  await fetchWithAuth("/api/daily-summary/settings", {
    method: "PUT",
    body: JSON.stringify({
      enabled: dailySummaryEnabled.value,
      time: dailySummaryTime.value,
      email: dailySummaryEmail.value,
      smtpPassword: smtpPassword.value || undefined,
    }),
  })
}
```

#### 内容类型检测 ([`frontend/src/utils/contentDetector.js`](frontend/src/utils/contentDetector.js))

检测文章内容类型（HTML/Markdown/纯文本）

#### API 工具函数 ([`frontend/src/utils/api.js`](frontend/src/utils/api.js))

- 自动附加 JWT token
- 401 时自动尝试刷新 token
- 刷新失败则重新加载页面（显示登录界面）

```javascript
// api.js:16-40 - Token 刷新逻辑
if (response.status === 401 && refreshToken) {
  const refreshResponse = await fetch('/api/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refreshToken })
  })

  if (refreshResponse.ok) {
    const data = await refreshResponse.json()
    localStorage.setItem('accessToken', data.accessToken)
    localStorage.setItem('refreshToken', data.refreshToken)
    headers['Authorization'] = `Bearer ${data.accessToken}`
    return fetch(url, { ...options, headers })
  }
}
```

## 5. 数据流

### 添加订阅源流程

```
用户输入 RSS URL
       ↓
SourceList.vue POST /api/sources (SourceList.vue:196)
       ↓
后端解析 RSS XML (main.go:893)
       ↓
存储订阅源到 MongoDB (main.go:905)
       ↓
批量插入历史文章 (main.go:936 insertArticlesAndGenerateSummary)
       ↓
返回新订阅源，前端更新列表
```

### 阅读文章流程

```
用户点击订阅源
       ↓
App.vue GET /api/sources/:id/articles (App.vue:137)
       ↓
后端查询 MongoDB (main.go:997)
       ↓
返回文章列表，前端显示在 ArticleList
       ↓
用户点击文章
       ↓
ArticleView 渲染内容 (ArticleView.vue:64-100)
       ↓
自动标记文章为已读 (App.vue:167-182)
```

### 生成 AI 摘要流程

```
用户点击"AI 总结"按钮
       ↓
ArticleView POST /api/articles/:id/ai-summary (ArticleView.vue:103)
       ↓
后端调用 OpenAI API (main.go:1354)
       ↓
去除 <think> 标签 (main.go:106 removeThinkTags)
       ↓
保存摘要到数据库 (main.go:1387)
       ↓
返回摘要，前端显示
```

### 每日总结邮件流程

```
Cron 每分钟触发 checkAndSendDailySummaries (main.go:1546)
       ↓
查询所有启用每日总结且时间匹配的用户 (main.go:1551)
       ↓
异步发送邮件 sendDailySummaryEmail (main.go:1570)
       ↓
获取当天新文章 getTodayArticles (main.go:433)
       ↓
调用 AI 生成合并摘要 generateMergedSummary (main.go:439)
       ↓
构建 HTML 邮件并通过 SMTP 发送 (main.go:456-505)
```

## 6. 环境配置

### 后端环境变量

| 变量名              | 必需   | 说明                          |
| ------------------ | ------ | ----------------------------- |
| `OPENAI_API_KEY`   | 否     | OpenAI API 密钥               |
| `OPENAI_BASE_URL`  | 否     | OpenAI API 基础 URL           |
| `OPENAI_MODEL_NAME`| 否     | 模型名称 (默认 gpt-3.5-turbo)  |
| `SMTP_HOST`        | 否     | SMTP 服务器地址               |
| `SMTP_PORT`        | 否     | SMTP 端口                     |
| `SMTP_USER`        | 否     | SMTP 用户名                   |

### 数据库连接 ([`db/db.go:20`](backend/db/db.go:20))

```go
// 连接地址
mongodb://localhost:27017

// 数据库名
rss_reader

// 集合
- users (用户集合)
- sources (订阅源集合)
- articles (文章集合)
- groups (分组集合)
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