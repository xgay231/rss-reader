# 功能说明文档

## 1. 功能概述

Web RSS 阅读器提供以下核心功能：

1. **用户认证** - 注册、登录、登出、JWT Token 自动刷新
2. **订阅源管理** - 添加、删除、查看 RSS 订阅源
3. **分组管理** - 创建、编辑、删除分组，拖拽排序
4. **文章获取** - 自动从订阅源抓取并存储文章
5. **文章阅读** - 查看文章详细内容，自动检测内容类型
6. **已读/未读状态** - 标记文章已读/未读，筛选功能
7. **收藏功能** - 收藏/取消收藏文章，收藏夹视图
8. **本地摘要** - 使用 TextRank + MMR 算法生成本地摘要
9. **AI 摘要** - 调用 OpenAI API 生成智能摘要
10. **每日总结邮件** - 自动发送每日文章摘要到邮箱
11. **用户设置** - 自定义 Feed 更新间隔、自动摘要开关

## 2. 功能详情

### 2.1 用户认证

#### 注册账户

**前端实现** - [`AuthForm.vue:62-83`](frontend/src/components/AuthForm.vue:62)

```javascript
const handleSubmit = async () => {
  if (isLogin.value) {
    await auth.login(form.email, form.password)
  } else {
    await auth.register(form.email, form.username, form.password)
    success.value = '注册成功！请登录'
    isLogin.value = true
  }
}
```

**后端实现** - [`handlers/auth.go:19-82`](backend/handlers/auth.go:19)

1. 验证 email、username、password 必填
2. 检查 email 是否已存在
3. bcrypt 哈希密码
4. 创建用户，生成 JWT token（15分钟 Access + 7天 Refresh）
5. 返回用户信息和 token

```go
// handlers/auth.go:19-82 - Register 函数
func Register(c *gin.Context) {
    var json struct {
        Email    string `json:"email" binding:"required,email"`
        Username string `json:"username" binding:"required,min=2"`
        Password string `json:"password" binding:"required,min=6"`
    }

    if err := c.ShouldBindJSON(&json); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 检查 email 是否已存在
    err := db.UserCollection.FindOne(context.Background(), bson.M{"email": json.Email}).Decode(&existingUser)
    if err == nil {
        c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
        return
    }

    // bcrypt 哈希密码
    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(json.Password), bcrypt.DefaultCost)

    // 创建用户（默认 FeedUpdateInterval=15, AutoSummary=true）
    user := struct { ... }{
        Email:              json.Email,
        Username:           json.Username,
        PasswordHash:      string(hashedPassword),
        FeedUpdateInterval: 15,
        AutoSummary:       true,
    }
    db.UserCollection.InsertOne(context.Background(), user)
}
```

#### 登录

**前端实现** - [`composables/useAuth.js:19-35`](frontend/src/composables/useAuth.js:19)

```javascript
const login = async (email, password) => {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
  })

  const data = await response.json()
  user.value = data.user
  setTokens(data.accessToken, data.refreshToken)
  return data
}
```

**后端实现** - [`handlers/auth.go:84-138`](backend/handlers/auth.go:84)

1. 验证 email、password
2. 查询用户验证 email 存在
3. bcrypt 比较密码
4. 生成 Access Token（15分钟）和 Refresh Token（7天）
5. 返回用户信息和 token

#### Token 自动刷新

**前端实现** - [`utils/api.js:16-40`](frontend/src/utils/api.js:16)

```javascript
// 401 时自动尝试刷新 token
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

#### 登出

**前端实现** - [`composables/useAuth.js:52-73`](frontend/src/composables/useAuth.js:52)

```javascript
const logout = async () => {
  const token = accessToken.value
  user.value = null
  setTokens(null, null)
  // 调用后端登出接口，然后强制刷新页面
  if (token) {
    fetch('/api/auth/logout', { method: 'POST', ... })
  }
  window.location.reload()
}
```

### 2.2 订阅源管理

#### 添加订阅源

**前端实现** - [`SourceList.vue:196-226`](frontend/src/components/SourceList.vue:196)

```javascript
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

**后端实现** - [`main.go:868-939`](backend/main.go:868)

1. 验证 URL 必填
2. 检查订阅源是否已存在（同一用户不能重复添加相同 URL）
3. 使用 `gofeed.NewParser()` 解析 RSS
4. 插入订阅源到 `sources` 集合
5. 处理孤立收藏文章（重新订阅时恢复关联）
6. 批量插入历史文章到 `articles` 集合

**限制**：
- URL 必须为有效的 RSS/Atom 链接
- 不允许重复添加相同 URL 的订阅源 (返回 409)

#### 查看订阅源

**前端实现** - [`SourceList.vue:159-170`](frontend/src/components/SourceList.vue:159)

```javascript
const fetchSources = async () => {
  const response = await fetchWithAuth('/api/sources')
  sources.value = data || []
}
```

- 左侧面板显示所有已添加的订阅源
- 按分组显示，支持折叠/展开
- 显示收藏夹入口和收藏数量

#### 删除订阅源

**前端实现** - [`SourceList.vue:358-380`](frontend/src/components/SourceList.vue:358)

```javascript
const deleteSource = async (sourceId, event) => {
  event.stopPropagation()
  if (!confirm('Are you sure you want to delete this feed and all its articles?')) {
    return
  }
  const response = await fetchWithAuth(`/api/sources/${sourceId}`, { method: 'DELETE' })
  sources.value = sources.value.filter((s) => s.id !== sourceId)
}
```

**后端实现** - [`main.go:966-994`](backend/main.go:966)

```go
sources.DELETE("/:id", func(c *gin.Context) {
    userID := getUserID(c)
    id, _ := primitive.ObjectIDFromHex(c.Param("id"))

    // 删除订阅源
    db.SourceCollection.DeleteOne(context.Background(), bson.M{"_id": id, "userId": userID})

    // 删除关联文章（保留已收藏的文章）
    db.ArticleCollection.DeleteMany(context.Background(), bson.M{"sourceId": id, "isStarred": false, "userId": userID})

    c.JSON(200, gin.H{"status": "ok"})
})
```

**注意**：删除订阅源时，已收藏的文章会被保留。

### 2.3 分组管理

#### 创建分组

**前端实现** - [`SourceList.vue:228-253`](frontend/src/components/SourceList.vue:228)

```javascript
const addGroup = async () => {
  const response = await fetchWithAuth('/api/groups', {
    method: 'POST',
    body: JSON.stringify({ name: newGroupName.value }),
  })
  const newGroup = await response.json()
  groups.value.push(newGroup)
  newGroupName.value = ''
}
```

#### 分组内移动订阅源

**前端实现** - [`SourceList.vue:382-407`](frontend/src/components/SourceList.vue:382)

```javascript
const assignSourceToGroup = async (source, groupId, event) => {
  event.stopPropagation()
  openMenuSourceId.value = null
  const response = await fetchWithAuth(`/api/sources/${source.id}/group`, {
    method: 'PUT',
    body: JSON.stringify({ groupId: groupId }),
  })
  source.groupId = groupId || null
  sources.value = [...sources.value]
}
```

**后端实现** - [`main.go:1059-1099`](backend/main.go:1059)

```go
sources.PUT("/:id/group", func(c *gin.Context) {
    var json struct {
        GroupID string `json:"groupId"`
    }
    c.ShouldBindJSON(&json)

    var groupID primitive.ObjectID
    if json.GroupID != "" {
        groupID, _ = primitive.ObjectIDFromHex(json.GroupID)
    }

    update := bson.M{"$set": bson.M{"groupId": groupID}}
    db.SourceCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
    c.JSON(200, gin.H{"status": "ok"})
})
```

#### 拖拽排序

**前端实现** - [`SourceList.vue:50-145`](frontend/src/components/SourceList.vue:50)

- 分组可以拖拽排序
- 订阅源可以拖拽到不同分组或"未分组"
- 拖拽时显示视觉反馈（虚线边框）

### 2.4 文章获取

#### 初始获取

添加订阅源时，后端自动抓取该订阅源的所有历史文章并存储到数据库。

**实现** - [`main.go:709-790`](backend/main.go:709)

```go
func insertArticlesAndGenerateSummary(source FeedSource, items []*gofeed.Item, ctx context.Context) []interface{} {
    for _, item := range items {
        content := item.Content
        if content == "" {
            content = item.Description
        }
        article := Article{
            SourceID:    source.ID,
            UserID:      source.UserID,
            GUID:        item.GUID,
            Title:       item.Title,
            URL:         item.Link,
            Description: item.Description,
            Content:     content,
            PublishedAt: item.PublishedParsed,
            ReadStatus:  "unread",
        }
        newArticles = append(newArticles, article)
    }

    if len(newArticles) > 0 {
        opts := options.InsertMany().SetOrdered(false)
        db.ArticleCollection.InsertMany(ctx, newArticles, opts)
    }
}
```

#### 定时更新

**实现** - [`main.go:792-810`](backend/main.go:792)

```go
func startFeedWorker() {
    updateFeeds() // 启动时执行一次

    for {
        ticker := time.NewTicker(feedUpdateInterval)
        select {
        case <-ticker.C:
            updateFeeds()
        case <-tickerStopChan:
            // Interval changed, restart ticker with new interval
        }
    }
}
```

- 默认 15 分钟更新一次
- 用户可自定义间隔（1-1440 分钟）

### 2.5 文章阅读

#### 文章列表

**前端实现** - [`ArticleList.vue:41-55`](frontend/src/components/ArticleList.vue:41)

```javascript
const filteredArticles = computed(() => {
  return props.articles
    .slice()
    .sort((a, b) => {
      const dateA = new Date(a.publishedAt).getTime()
      const dateB = new Date(b.publishedAt).getTime()
      if (isNaN(dateA) || isNaN(dateB)) return 0
      return dateB - dateA  // 降序排列
    })
    .filter((article) => {
      if (article.readStatus === 'read' && !showRead.value) return false
      if (article.readStatus === 'unread' && !showUnread.value) return false
      return true
    })
})
```

- 按发布时间降序排序
- 支持已读/未读状态筛选
- 显示相对时间（刚刚、分钟前、小时前、天前）

#### 文章内容渲染

**前端实现** - [`ArticleView.vue:64-100`](frontend/src/components/ArticleView.vue:64)

```javascript
const contentType = computed(() => {
  return detectContentType(props.article.content)
})

const renderedContent = computed(() => {
  switch (contentType.value) {
    case 'html':
      return DOMPurify.sanitize(content, { ALLOWED_TAGS: [...], ALLOWED_ATTR: [...] })
    case 'markdown':
      return marked.parse(content.replace(/\\n/g, '\n'))
    case 'plain':
    default:
      const escaped = content.replace(/\\n/g, '\n')...
      return `<pre style="white-space: pre-wrap; word-wrap: break-word;">${escaped}</pre>`
  }
})
```

- 自动检测内容类型（HTML/Markdown/纯文本）
- HTML 使用 DOMPurify 消毒防 XSS
- Markdown 使用 marked 解析

### 2.6 已读/未读状态

#### 标记已读

**前端实现** - [`App.vue:167-182`](frontend/src/App.vue:167)

```javascript
if (article.readStatus !== 'read') {
  fetchWithAuth(`/api/articles/${article.id}/read`, { method: 'PUT' })
    .then(() => {
      const index = articles.value.findIndex(a => a.id === article.id)
      if (index !== -1) {
        articles.value[index] = { ...articles.value[index], readStatus: 'read' }
      }
    })
}
```

**后端实现** - [`main.go:1455-1484`](backend/main.go:1455)

```go
articles.PUT("/:id/read", func(c *gin.Context) {
    update := bson.M{"$set": bson.M{"readStatus": "read"}}
    db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
    c.JSON(200, article)
})
```

#### 标记未读

**后端实现** - [`main.go:1487-1516`](backend/main.go:1487)

```go
articles.DELETE("/:id/read", func(c *gin.Context) {
    update := bson.M{"$set": bson.M{"readStatus": "unread"}}
    db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
    c.JSON(200, article)
})
```

#### 标记全部已读

**前端实现** - [`App.vue:205-222`](frontend/src/App.vue:205)

```javascript
const handleMarkAllRead = async () => {
  const response = await fetchWithAuth(
    `/api/sources/${currentSourceId.value}/mark-all-read`,
    { method: "PUT" }
  )
  if (response.ok) {
    handleSourceSelected({ id: currentSourceId.value }) // 刷新列表
  }
}
```

**后端实现** - [`main.go:1027-1057`](backend/main.go:1027)

```go
sources.PUT("/:id/mark-all-read", func(c *gin.Context) {
    update := bson.M{"$set": bson.M{"readStatus": "read"}}
    filter := bson.M{"sourceId": id, "userId": userID, "readStatus": "unread"}
    result, _ := db.ArticleCollection.UpdateMany(context.Background(), filter, update)
    c.JSON(200, gin.H{"modifiedCount": result.ModifiedCount})
})
```

### 2.7 收藏功能

#### 收藏文章

**前端实现** - [`ArticleView.vue:142-154`](frontend/src/components/ArticleView.vue:142)

```javascript
const toggleStar = async () => {
  const method = props.article.isStarred ? 'DELETE' : 'POST'
  const response = await fetchWithAuth(`/api/articles/${props.article.id}/star`, { method })
  if (response.ok) {
    props.article.isStarred = !props.article.isStarred
    emit('starred-changed')
  }
}
```

**后端实现** - [`main.go:1397-1423`](backend/main.go:1397)

```go
articles.POST("/:id/star", func(c *gin.Context) {
    update := bson.M{"$set": bson.M{"isStarred": true, "starredAt": time.Now()}}
    db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
    c.JSON(200, gin.H{"status": "ok", "isStarred": true})
})
```

#### 查看收藏夹

**前端实现** - [`SourceList.vue:338-349`](frontend/src/components/SourceList.vue:338)

```javascript
const selectStarred = async () => {
  selectedSourceId.value = 'starred'
  const response = await fetchWithAuth('/api/articles/starred')
  // ...
  emit('source-selected', { id: 'starred', name: '收藏夹' })
}
```

**后端实现** - [`main.go:1235-1257`](backend/main.go:1235)

```go
articles.GET("/starred", func(c *gin.Context) {
    filter := bson.M{"isStarred": true, "userId": userID}
    cursor, _ := db.ArticleCollection.Find(context.Background(), filter)
    articles := []Article{}
    cursor.All(context.Background(), &articles)
    c.JSON(200, articles)
})
```

### 2.8 本地摘要生成

#### 技术原理

使用 **TextRank + MMR** 算法

**前端实现** - [`summarizer.js:57-159`](frontend/src/services/summarizer.js:57)

```javascript
export async function summarizeText(content, numSentences = 3, lambda = 0.7) {
  // 1. 句子切分 - 按 .!? 句末标点切分
  const sentences = content.match(/[^.!?]+[.!?]+/g) || []

  // 2. 句子嵌入 - 使用 Universal Sentence Encoder
  const embeddings = await model.embed(sentences)

  // 3. 相似度矩阵 - 计算余弦相似度
  // ...

  // 4. TextRank 迭代 - 阻尼系数 0.85
  // ...

  // 5. MMR 选择 - 平衡相关性和多样性
  // ...
}
```

### 2.9 AI 摘要生成

#### 后端实现 - [`main.go:1328-1394`](backend/main.go:1328)

```go
articles.POST("/:id/ai-summary", func(c *gin.Context) {
    if aiClient == nil {
        c.JSON(503, gin.H{"error": "AI service is not available"})
        return
    }

    resp, err := aiClient.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: aiModelName,
            Messages: []openai.ChatCompletionMessage{
                { Role: openai.ChatMessageRoleSystem, Content: "你是一个帮助用户总结文章内容的助手..." },
                { Role: openai.ChatMessageRoleUser, Content: "Please summarize..." + article.Content },
            },
        },
    )

    summary := removeThinkTags(resp.Choices[0].Message.Content)

    // 保存摘要到数据库
    update := bson.M{"$set": bson.M{"summary": summary, "summaryGeneratedAt": now}}
    db.ArticleCollection.UpdateByID(context.Background(), id, update)

    c.JSON(200, gin.H{"summary": summary})
})
```

**前端实现** - [`ArticleView.vue:103-120`](frontend/src/components/ArticleView.vue:103)

```javascript
const generateAISummaryHandler = async () => {
  isLoadingAISummary.value = true
  try {
    const result = await generateAISummary(props.article.id)
    aiSummary.value = result
    emit('summary-updated', { ...props.article, summary: result })
  } catch (error) {
    aiSummaryError.value = "Failed to generate AI summary."
  } finally {
    isLoadingAISummary.value = false
  }
}
```

### 2.10 每日总结邮件

#### 配置设置

**前端实现** - [`SettingsModal.vue:220-271`](frontend/src/components/SettingsModal.vue:220)

```javascript
const dailySummaryEnabled = ref(false)
const dailySummaryTime = ref("09:00")
const dailySummaryEmail = ref("")
const smtpPassword = ref("")

// 保存设置
await fetchWithAuth("/api/daily-summary/settings", {
  method: "PUT",
  body: JSON.stringify({
    enabled: dailySummaryEnabled.value,
    time: dailySummaryTime.value,
    email: dailySummaryEmail.value,
    smtpPassword: smtpPassword.value || undefined,
  }),
})
```

#### 定时调度

**后端实现** - [`main.go:1526-1575`](backend/main.go:1526)

```go
func initDailySummaryScheduler() {
    c := cron.New()
    c.AddFunc("* * * * *", func() {
        checkAndSendDailySummaries()
    })
    c.Start()
}

func checkAndSendDailySummaries() {
    now := time.Now()
    currentTime := now.Format("15:04") // 精确到分钟

    // 查询所有启用了每日总结且时间匹配的用户
    cursor, _ := db.UserCollection.Find(context.Background(), bson.M{
        "dailySummaryEnabled": true,
        "dailySummaryTime":    currentTime,
    })

    for cursor.Next(context.Background()) {
        var user struct{ ID primitive.ObjectID }
        cursor.Decode(&user)
        go func(uid primitive.ObjectID) {
            sendDailySummaryEmail(uid)
        }(user.ID)
    }
}
```

#### 发送邮件

**后端实现** - [`main.go:413-512`](backend/main.go:413)

```go
func sendDailySummaryEmail(userID primitive.ObjectID) error {
    // 1. 获取用户信息
    // 2. 查询当天文章
    articles, _ := getTodayArticles(userID)

    // 3. 生成合并摘要
    summary, _ := generateMergedSummary(articles)

    // 4. 构建 HTML 邮件
    htmlBody := fmt.Sprintf(`<!DOCTYPE html>...`, today, len(articles), summary, articleListHTML)

    // 5. 通过 SMTP 发送
    msg := mail.NewMessage()
    msg.SetHeader("From", user.DailySummaryEmail)
    msg.SetHeader("To", user.DailySummaryEmail)
    msg.SetHeader("Subject", "每日文章总结")
    msg.SetBody("text/html", htmlBody)

    dialer := mail.NewDialer(host, port, user.DailySummaryEmail, user.SmtpPassword)
    dialer.StartTLSPolicy = mail.MandatoryStartTLS
    dialer.Timeout = 30 * time.Second
    return dialer.DialAndSend(msg)
}
```

### 2.11 用户设置

#### 获取设置

**后端实现** - [`main.go:220-272`](backend/main.go:220)

```go
func GetSettings(c *gin.Context) {
    userID := getUserID(c)
    var user struct {
        FeedUpdateInterval int   `bson:"feedUpdateInterval"`
        AutoSummary        *bool `bson:"autoSummary"` // 指针区分"未设置"和"显式 false"
    }

    // 返回默认值（FeedUpdateInterval=15, AutoSummary=true）并持久化到数据库
}
```

#### 更新设置

**后端实现** - [`main.go:274-331`](backend/main.go:274)

```go
func UpdateSettings(c *gin.Context) {
    var json struct {
        FeedUpdateInterval *int  `json:"feedUpdateInterval"`
        AutoSummary        *bool `json:"autoSummary"`
    }

    // 验证 feedUpdateInterval 在 1-1440 之间
    if *json.FeedUpdateInterval < 1 || *json.FeedUpdateInterval > 1440 {
        c.JSON(400, gin.H{"error": "feedUpdateInterval must be between 1 and 1440"})
        return
    }

    // 更新数据库
    db.UserCollection.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$set": update})

    // 如果 Feed 更新间隔改变，重启 ticker
    if *json.FeedUpdateInterval != feedUpdateIntervalMins {
        feedUpdateIntervalMins = *json.FeedUpdateInterval
        feedUpdateInterval = time.Duration(feedUpdateIntervalMins) * time.Minute
        signalTickerRestart()
    }
}
```

## 3. API 接口

### 3.1 认证接口

| 方法   | 路径                   | 代码位置         | 说明               |
| ------ | ---------------------- | --------------- | ------------------ |
| POST   | `/api/auth/register`   | handlers/auth.go:19 | 用户注册           |
| POST   | `/api/auth/login`      | handlers/auth.go:84 | 用户登录           |
| POST   | `/api/auth/refresh`    | handlers/auth.go:140 | 刷新 Access Token  |
| POST   | `/api/auth/logout`     | handlers/auth.go:190 | 用户登出           |
| GET    | `/api/auth/me`         | handlers/auth.go:197 | 获取当前用户信息   |

### 3.2 设置接口

| 方法   | 路径                      | 代码位置      | 说明               |
| ------ | ------------------------- | ------------ | ------------------ |
| GET    | `/api/settings`           | main.go:220  | 获取用户设置       |
| PUT    | `/api/settings`           | main.go:274  | 更新用户设置       |
| GET    | `/api/daily-summary/settings` | main.go:565 | 获取每日总结设置   |
| PUT    | `/api/daily-summary/settings` | main.go:601 | 更新每日总结设置   |
| POST   | `/api/daily-summary/send` | main.go:514  | 手动发送每日总结   |

### 3.3 订阅源接口

| 方法   | 路径                           | 代码位置      | 说明               |
| ------ | ------------------------------ | ------------ | ------------------ |
| POST   | `/api/sources`                 | main.go:868  | 添加订阅源         |
| GET    | `/api/sources`                 | main.go:942  | 获取用户所有订阅源 |
| DELETE | `/api/sources/:id`             | main.go:966  | 删除订阅源         |
| GET    | `/api/sources/:id/articles`    | main.go:997  | 获取订阅源下的文章 |
| PUT    | `/api/sources/:id/mark-all-read` | main.go:1027 | 标记所有文章为已读 |
| PUT    | `/api/sources/:id/group`       | main.go:1059 | 将订阅源分配到分组 |
| POST   | `/api/sources/refresh`         | main.go:1101 | 手动刷新所有订阅源 |

### 3.4 分组接口

| 方法   | 路径                   | 代码位置      | 说明         |
| ------ | ---------------------- | ------------ | ------------ |
| POST   | `/api/groups`          | main.go:1110 | 创建分组     |
| GET    | `/api/groups`          | main.go:1143 | 获取用户所有分组 |
| PUT    | `/api/groups/:id`      | main.go:1167 | 更新分组     |
| DELETE | `/api/groups/:id`      | main.go:1200 | 删除分组     |

### 3.5 文章接口

| 方法   | 路径                          | 代码位置      | 说明               |
| ------ | ----------------------------- | ------------ | ------------------ |
| GET    | `/api/articles/starred`      | main.go:1235 | 获取收藏文章       |
| GET    | `/api/articles`               | main.go:1260 | 获取文章（支持筛选）|
| GET    | `/api/articles/:id`          | main.go:1304 | 获取单篇文章       |
| POST   | `/api/articles/:id/ai-summary` | main.go:1328 | 生成 AI 摘要       |
| POST   | `/api/articles/:id/star`     | main.go:1397 | 收藏文章           |
| DELETE | `/api/articles/:id/star`     | main.go:1426 | 取消收藏           |
| PUT    | `/api/articles/:id/read`     | main.go:1455 | 标记已读           |
| DELETE | `/api/articles/:id/read`     | main.go:1487 | 标记未读           |

## 4. 用户界面

### 布局结构

```
┌────────────────────────────────────────────────────────────────────┐
│                         三栏布局                                   │
├────────────┬─────────────────────┬────────────────────────────────┤
│            │                     │                                 │
│  订阅源列表   │    文章列表         │       文章阅读与摘要            │
│  (280px)   │    (350px)          │        (自适应宽度)              │
│            │                     │                                 │
│  - 收藏夹   │  - 筛选按钮         │  - AI 总结按钮                  │
│  - 未分组   │  - 标记已读按钮     │  - 收藏/已读按钮                │
│  - 分组列表 │  - 文章卡片         │  - 内容渲染                     │
│            │                     │                                 │
└────────────┴─────────────────────┴────────────────────────────────┘
```

### 移动端布局

- 移动端（<768px）使用 stack-based 导航
- 顶部显示返回按钮和当前视图标题
- 三栏变为单栏，点击返回切换

### 交互流程

1. **注册/登录**：显示 AuthForm，输入凭证，登录成功后显示主界面
2. **添加订阅源**：左侧输入 URL → 添加按钮 → 列表更新
3. **查看文章**：点击订阅源 → 中间显示文章 → 点击文章 → 右侧阅读
4. **生成摘要**：阅读文章 → 点击 AI 总结按钮 → 等待生成 → 显示结果
5. **标记已读**：阅读文章时自动标记，或点击标记已读按钮
6. **设置每日总结**：右上角用户菜单 → 设置 → 配置每日总结参数

## 5. 错误处理

| 场景                  | 代码位置           | 处理方式                          |
| --------------------- | ----------------- | --------------------------------- |
| RSS URL 无效          | main.go:894-896   | 返回 500，前端提示"解析失败"       |
| 订阅源重复添加        | main.go:887-889   | 返回 409，前端提示"已存在"         |
| 数据库连接失败        | db/db.go:26-27    | 后端日志记录，进程退出            |
| AI 服务不可用         | main.go:1335-1338 | 返回 503，前端提示"AI 服务不可用"  |
| OpenAI API 调用失败   | main.go:1371-1374 | 返回 500，日志记录错误详情        |
| 认证失败              | middleware/auth.go:24-26 | 返回 401，前端刷新 token      |
| Token 过期            | api.js:17-40      | 自动刷新，成功则重试请求          |

## 6. 性能考虑

| 优化项       | 说明                             | 代码位置             |
| ------------ | -------------------------------- | ------------------ |
| 模型缓存     | USE 模型加载后缓存在内存中       | summarizer.js:4,16 |
| 增量更新     | RSS 抓取仅处理新文章 (检查 GUID) | main.go:713-715    |
| 异步处理     | 后台任务不影响 API 响应          | main.go:141-150    |
| 批量插入     | 文章批量写入数据库 (InsertMany)  | main.go:747         |
| 摘要并发限制  | AI 摘要生成信号量限制 5 个并发   | main.go:773-786     |
| 索引优化     | email 唯一索引、readStatus 索引  | db/db.go:41-60      |

## 7. 扩展性

系统设计支持以下扩展：

- 添加更多摘要算法
- 支持更多 RSS 格式
- 添加标签和分类功能
- 支持文章分享
- 多语言支持