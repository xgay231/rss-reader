# 每日文章总结功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 RSS 阅读器添加每日总结功能，每天定时将收到的新文章聚合为一份摘要，发送至用户邮箱，支持手动触发。

**Architecture:** 后端 Go 添加每日总结 API 和定时调度，前端 Vue 在设置页面添加配置项。核心流程：查询当天文章 → 调用 AI 生成合并摘要 → 组装 HTML 邮件 → 通过 SMTP 发送。

**Tech Stack:** Go (gin, mongo-driver, robfig/cron, go-mail), Vue.js

---

## 文件变更概览

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `backend/models/user.go` | 修改 | 新增每日总结相关字段 |
| `backend/main.go` | 修改 | 新增 API handlers、cron 调度、邮件发送逻辑 |
| `frontend/src/components/SettingsModal.vue` | 修改 | 添加每日总结设置 UI |

---

## Task 1: 扩展 User 模型

**Files:**
- Modify: `backend/models/user.go`

- [ ] **Step 1: 添加每日总结字段到 User 结构体**

在 `backend/models/user.go` 的 `User` 结构体中添加新字段：

```go
DailySummaryEnabled bool   `json:"dailySummaryEnabled" bson:"dailySummaryEnabled"`
DailySummaryTime    string `json:"dailySummaryTime" bson:"dailySummaryTime"`         // 格式 "HH:MM"
DailySummaryEmail   string `json:"dailySummaryEmail" bson:"dailySummaryEmail"`       // 发送目标邮箱
SmtpPassword        string `json:"-" bson:"smtpPassword"`                           // SMTP 密码，不暴露给前端
```

- [ ] **Step 2: 提交**

```bash
git add backend/models/user.go
git commit -m "feat(backend): add daily summary fields to User model"
```

---

## Task 2: 安装依赖

**Files:**
- Modify: `backend/go.mod`, `backend/go.sum`

- [ ] **Step 1: 添加 cron 和 mail 依赖**

```bash
cd backend
go get github.com/robfig/cron/v3
go get github.com/go-mail/mail/v2
```

- [ ] **Step 2: 验证依赖已添加**

Run: `go mod tidy && go build ./...`
Expected: 无错误

- [ ] **Step 3: 提交**

```bash
git add backend/go.mod backend/go.sum
git commit -m "deps(backend): add robfig/cron and go-mail dependencies"
```

---

## Task 3: 创建每日总结 Handler

**Files:**
- Modify: `backend/main.go`

首先在 `main.go` 顶部 import 区域添加新的导入：

```go
"github.com/robfig/cron/v3"
"github.com/go-mail/mail"
```

然后在文件末尾添加以下代码：

- [ ] **Step 1: 添加每日总结设置结构体和常量**

```go
// DailySummarySettings 每日总结设置
type DailySummarySettings struct {
	Enabled bool   `json:"enabled"`
	Time    string `json:"time"`    // 格式 "HH:MM"
	Email   string `json:"email"`   // 目标邮箱
}

// DailySummarySendResult 发送结果
type DailySummarySendResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
```

- [ ] **Step 2: 添加获取每日总结设置的 Handler**

```go
// GetDailySummarySettings 获取每日总结设置
func GetDailySummarySettings(c *gin.Context) {
	userID := getUserID(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user struct {
		DailySummaryEnabled bool   `bson:"dailySummaryEnabled"`
		DailySummaryTime    string `bson:"dailySummaryTime"`
		DailySummaryEmail   string `bson:"dailySummaryEmail"`
	}

	err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	// 返回默认值
	enabled := user.DailySummaryEnabled
	time := user.DailySummaryTime
	email := user.DailySummaryEmail

	if time == "" {
		time = "09:00" // 默认早上 9 点
	}

	c.JSON(http.StatusOK, DailySummarySettings{
		Enabled: enabled,
		Time:    time,
		Email:   email,
	})
}
```

- [ ] **Step 3: 添加更新每日总结设置的 Handler**

```go
// UpdateDailySummarySettings 更新每日总结设置
func UpdateDailySummarySettings(c *gin.Context) {
	userID := getUserID(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var json struct {
		Enabled     *bool   `json:"enabled"`
		Time        *string `json:"time"`
		Email       *string `json:"email"`
		SmtpPassword *string `json:"smtpPassword"` // 不返回给前端
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := bson.M{}
	if json.Enabled != nil {
		update["dailySummaryEnabled"] = *json.Enabled
	}
	if json.Time != nil {
		update["dailySummaryTime"] = *json.Time
	}
	if json.Email != nil {
		update["dailySummaryEmail"] = *json.Email
	}
	if json.SmtpPassword != nil {
		update["smtpPassword"] = *json.SmtpPassword
	}

	if len(update) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No settings to update"})
		return
	}

	_, err := db.UserCollection.UpdateOne(
		context.Background(),
		bson.M{"_id": userID},
		bson.M{"$set": update},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully"})
}
```

- [ ] **Step 4: 添加查询当天文章的函数**

```go
// getTodayArticles 查询用户当天新增的文章
func getTodayArticles(userID primitive.ObjectID) ([]Article, error) {
	// 获取当天 0:00 到现在的文章
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	filter := bson.M{
		"userId": userID,
		"publishedAt": bson.M{
			"$gte": startOfDay,
			"$lte": now,
		},
	}

	cursor, err := db.ArticleCollection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var articles []Article
	if err := cursor.All(context.Background(), &articles); err != nil {
		return nil, err
	}

	return articles, nil
}
```

- [ ] **Step 5: 添加 AI 生成合并摘要的函数**

```go
// generateMergedSummary 调用 AI 将多篇文章聚合成一段摘要
func generateMergedSummary(articles []Article) (string, error) {
	if aiClient == nil {
		return "AI client not available", nil
	}

	if len(articles) == 0 {
		return "今日暂无新文章", nil
	}

	// 构建文章列表
	var articleList string
	for i, article := range articles {
		articleList += fmt.Sprintf("%d. \"%s\" - 来源: %s\n", i+1, article.Title, article.URL)
	}

	prompt := fmt.Sprintf(`请为用户生成一段今日文章摘要。

文章列表：
%s

请生成一段 100-200 字的合并摘要，概括今日文章的核心内容。用中文输出。`, articleList)

	resp, err := aiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: aiModelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个帮助用户总结文章内容的助手。请用中文输出纯文本摘要，不要使用 markdown 格式，不要使用列表、标题、粗体等任何格式标记，只输出纯段落文本。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}
```

- [ ] **Step 6: 添加发送每日总结邮件的函数**

```go
// sendDailySummaryEmail 发送每日总结邮件
func sendDailySummaryEmail(userID primitive.ObjectID) error {
	// 获取用户信息
	var user struct {
		Email             string `bson:"email"`
		DailySummaryEmail string `bson:"dailySummaryEmail"`
		SmtpPassword      string `bson:"smtpPassword"`
		DailySummaryTime  string `bson:"dailySummaryTime"`
	}

	err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	if user.DailySummaryEmail == "" || user.SmtpPassword == "" {
		return fmt.Errorf("email or smtp password not configured")
	}

	// 查询当天文章
	articles, err := getTodayArticles(userID)
	if err != nil {
		return fmt.Errorf("failed to get today articles: %v", err)
	}

	// 生成合并摘要
	summary, err := generateMergedSummary(articles)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %v", err)
	}

	// 构建文章列表 HTML
	var articleListHTML string
	for _, article := range articles {
		articleListHTML += fmt.Sprintf(
			`<li><a href="%s" style="color: #0066cc;">%s</a></li>`,
			article.URL, article.Title)
	}

	if articleListHTML == "" {
		articleListHTML = "<li>今日暂无新文章</li>"
	}

	// 构建 HTML 邮件
	today := time.Now().Format("2006年1月2日")
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
  <h2 style="color: #333;">📬 每日文章总结</h2>
  <p style="color: #666;">%s</p>
  <hr style="border: 1px solid #eee;">

  <h3 style="color: #444;">今日概览</h3>
  <p>你今天收到了 <strong>%d</strong> 篇文章。</p>

  <h3 style="color: #444;">智能摘要</h3>
  <p style="line-height: 1.6;">%s</p>

  <h3 style="color: #444;">文章列表</h3>
  <ul style="line-height: 1.8;">
    %s
  </ul>
</body>
</html>`, today, len(articles), summary, articleListHTML)

	// 发送邮件
	// SMTP 配置 - 使用 QQ 邮箱或其他支持 SMTP 的邮箱
	// 这里需要用户配置 SMTP 服务器地址和端口
	m := mail.NewMSG()
	m.SetHeader("From", user.DailySummaryEmail)
	m.SetHeader("To", user.DailySummaryEmail)
	m.SetHeader("Subject", fmt.Sprintf("📬 每日文章总结 - %s", today))
	m.SetBody("text/html", htmlBody)

	// 注意：实际使用时需要配置 SMTP 服务器
	// 这里暂时返回成功，实际部署时需要配置 .env 或用户设置中的 SMTP 信息
	log.Printf("[DailySummary] Email prepared for user %s: to=%s, articles=%d",
		userID.Hex(), user.DailySummaryEmail, len(articles))

	return nil
}
```

- [ ] **Step 7: 添加手动触发发送的 Handler**

```go
// SendDailySummary 手动触发发送每日总结
func SendDailySummary(c *gin.Context) {
	userID := getUserID(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 异步发送，不阻塞响应
	go func() {
		if err := sendDailySummaryEmail(userID); err != nil {
			log.Printf("[DailySummary] Failed to send daily summary for user %s: %v", userID.Hex(), err)
		} else {
			log.Printf("[DailySummary] Daily summary sent successfully for user %s", userID.Hex())
		}
	}()

	c.JSON(http.StatusOK, DailySummarySendResult{
		Success: true,
		Message: "每日总结发送中，请稍候...",
	})
}
```

- [ ] **Step 8: 提交**

```bash
git add backend/main.go
git commit -m "feat(backend): add daily summary API handlers"
```

---

## Task 4: 添加定时调度器

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: 在 main 函数中添加 cron 调度器初始化**

在 `main()` 函数中，数据库初始化和 AI 初始化之后，添加：

```go
// 初始化每日总结定时调度器
initDailySummaryScheduler()

// initDailySummaryScheduler 初始化每日总结定时调度器
func initDailySummaryScheduler() {
	c := cron.New()

	// 每分钟检查一次
	c.AddFunc("* * * * *", func() {
		checkAndSendDailySummaries()
	})

	c.Start()
	log.Println("Daily summary scheduler started")

	// 确保进程退出时停止 cron
	go func() {
		<-ctx.Done()
		c.Stop()
	}()
}

// checkAndSendDailySummaries 检查所有用户是否需要发送每日总结
func checkAndSendDailySummaries() {
	now := time.Now()
	currentTime := now.Format("15:04") // 精确到分钟

	// 查询所有启用了每日总结的用户
	cursor, err := db.UserCollection.Find(context.Background(), bson.M{
		"dailySummaryEnabled": true,
		"dailySummaryTime":    currentTime,
	})
	if err != nil {
		log.Printf("[DailySummary] Failed to find users: %v", err)
		return
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var user struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		log.Printf("[DailySummary] Triggering daily summary for user %s", user.ID.Hex())
		go func(uid primitive.ObjectID) {
			if err := sendDailySummaryEmail(uid); err != nil {
				log.Printf("[DailySummary] Failed to send for user %s: %v", uid.Hex(), err)
			}
		}(user.ID)
	}
}
```

- [ ] **Step 2: 注册路由**

在现有路由注册部分添加新的路由：

```go
// 每日总结相关
dailySummary := r.Group("/api/daily-summary")
dailySummary.Use(authMiddleware())
{
	dailySummary.GET("/settings", GetDailySummarySettings)
	dailySummary.PUT("/settings", UpdateDailySummarySettings)
	dailySummary.POST("/send", SendDailySummary)
}
```

- [ ] **Step 3: 提交**

```bash
git add backend/main.go
git commit -m "feat(backend): add daily summary cron scheduler"
```

---

## Task 5: 更新前端设置页面

**Files:**
- Modify: `frontend/src/components/SettingsModal.vue`

- [ ] **Step 1: 添加每日总结相关的 ref**

在 `<script setup>` 中添加：

```javascript
const dailySummaryEnabled = ref(false);
const dailySummaryTime = ref("09:00");
const dailySummaryEmail = ref("");
const smtpPassword = ref("");
const sendingSummary = ref(false);
const summaryMessage = ref("");
```

- [ ] **Step 2: 更新 fetchSettings 函数**

在获取设置的响应处理中添加：

```javascript
dailySummaryEnabled.value = data.dailySummaryEnabled ?? false;
dailySummaryTime.value = data.dailySummaryTime ?? "09:00";
dailySummaryEmail.value = data.dailySummaryEmail ?? "";
```

- [ ] **Step 3: 更新 handleSave 函数**

在保存设置时添加每日总结相关字段：

```javascript
body: JSON.stringify({
  feedUpdateInterval,
  autoSummary: autoSummary.value,
  // 新增
  dailySummaryEnabled: dailySummaryEnabled.value,
  dailySummaryTime: dailySummaryTime.value,
  dailySummaryEmail: dailySummaryEmail.value,
  smtpPassword: smtpPassword.value || undefined,
}),
```

- [ ] **Step 4: 添加手动发送函数**

```javascript
const handleSendSummary = async () => {
  sendingSummary.value = true;
  summaryMessage.value = "";

  try {
    const response = await fetchWithAuth("/api/daily-summary/send", {
      method: "POST",
    });

    const data = await response.json();
    if (response.ok) {
      summaryMessage.value = "每日总结发送中，请稍候...";
    } else {
      summaryMessage.value = data.error || "发送失败";
    }
  } catch (e) {
    summaryMessage.value = "发送失败";
    console.error(e);
  } finally {
    sendingSummary.value = false;
  }
};
```

- [ ] **Step 5: 在模板中添加每日总结设置 UI**

在自动摘要设置之后、divider 之前添加：

```html
<hr class="divider" />

<div class="setting-group">
  <label class="setting-label">每日总结</label>
  <div class="checkbox-option">
    <label>
      <input type="checkbox" v-model="dailySummaryEnabled" />
      <span>启用每日总结邮件</span>
    </label>
  </div>
</div>

<div v-if="dailySummaryEnabled" class="daily-summary-settings">
  <div class="setting-row">
    <label>发送时间：</label>
    <input
      type="time"
      v-model="dailySummaryTime"
      class="time-input"
    />
  </div>

  <div class="setting-row">
    <label>邮箱地址：</label>
    <input
      type="email"
      v-model="dailySummaryEmail"
      placeholder="接收总结的邮箱"
      class="text-input"
    />
  </div>

  <div class="setting-row">
    <label>SMTP 密码：</label>
    <input
      type="password"
      v-model="smtpPassword"
      placeholder="邮箱 SMTP 授权码"
      class="text-input"
    />
    <p class="hint">QQ邮箱可使用授权码，其他邮箱请查看 SMTP 设置</p>
  </div>

  <div class="setting-row">
    <button
      class="btn-send-summary"
      @click="handleSendSummary"
      :disabled="sendingSummary"
    >
      {{ sendingSummary ? "发送中..." : "立即发送" }}
    </button>
    <span v-if="summaryMessage" class="summary-message">{{ summaryMessage }}</span>
  </div>
</div>
```

- [ ] **Step 6: 添加样式**

在 `<style scoped>` 中添加：

```css
.daily-summary-settings {
  margin-top: 1rem;
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  flex-wrap: wrap;
}

.setting-row label {
  min-width: 70px;
  color: #333;
}

.time-input,
.text-input {
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.95rem;
}

.time-input {
  width: 100px;
}

.text-input {
  flex: 1;
  min-width: 180px;
}

.hint {
  width: 100%;
  margin: 0.25rem 0 0 70px;
  font-size: 0.8rem;
  color: #888;
}

.btn-send-summary {
  padding: 0.5rem 1rem;
  background: #28a745;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
}

.btn-send-summary:hover:not(:disabled) {
  background: #218838;
}

.btn-send-summary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.summary-message {
  margin-left: 0.5rem;
  font-size: 0.9rem;
  color: #666;
}
```

- [ ] **Step 7: 提交**

```bash
git add frontend/src/components/SettingsModal.vue
git commit -m "feat(frontend): add daily summary settings to SettingsModal"
```

---

## Task 6: 实现 SMTP 邮件发送（需要配置）

**Files:**
- Modify: `backend/main.go` 中的 `sendDailySummaryEmail` 函数

- [ ] **Step 1: 添加 .env 示例配置**

在 `backend/.env` 中添加：

```env
# SMTP 配置（用于每日总结邮件）
SMTP_HOST=smtp.qq.com
SMTP_PORT=587
SMTP_USER=your-email@qq.com
# SMTP_PASSWORD 使用用户的个人授权码，不是登录密码
```

- [ ] **Step 2: 更新 sendDailySummaryEmail 函数使用全局 SMTP 配置**

在 `main.go` 顶部添加 SMTP 配置变量：

```go
var (
	// ... existing vars
	smtpHost     = os.Getenv("SMTP_HOST")
	smtpPort     = os.Getenv("SMTP_PORT")
	smtpUser     = os.Getenv("SMTP_USER")
)
```

更新 `sendDailySummaryEmail` 函数使用 SMTP 发送：

```go
func sendDailySummaryEmail(userID primitive.ObjectID) error {
	// ... 获取用户信息和生成摘要 ...

	// 发送邮件
	m := mail.NewMSG()
	m.SetHeader("From", user.DailySummaryEmail)
	m.SetHeader("To", user.DailySummaryEmail)
	m.SetHeader("Subject", fmt.Sprintf("📬 每日文章总结 - %s", today))
	m.SetBody("text/html", htmlBody)

	// 连接到 SMTP 服务器并发送
	port, _ := strconv.Atoi(smtpPort)
	d := mail.NewDialer(smtpHost, port, user.DailySummaryEmail, user.SmtpPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("[DailySummary] Email sent for user %s: to=%s", userID.Hex(), user.DailySummaryEmail)
	return nil
}
```

- [ ] **Step 3: 提交**

```bash
git add backend/.env backend/main.go
git commit -m "feat(backend): implement SMTP email sending for daily summary"
```

---

## Task 7: 测试验证

- [ ] **Step 1: 手动触发测试**

1. 启动后端服务器
2. 打开前端设置页面
3. 填写每日总结邮箱和 SMTP 密码
4. 点击"立即发送"按钮
5. 检查邮箱是否收到测试邮件

- [ ] **Step 2: 定时任务测试**

1. 将发送时间设置为当前时间 + 1 分钟
2. 保存设置
3. 等待 1 分钟
4. 检查是否收到邮件

- [ ] **Step 3: 提交最终版本**

```bash
git status
git log --oneline -5
```

---

## 实施检查清单

在开始下一项任务前，确认以下内容：

| 任务 | 状态 |
|------|------|
| User 模型字段已添加 | ☐ |
| 依赖已安装 | ☐ |
| API Handlers 已实现 | ☐ |
| Cron 调度器已添加 | ☐ |
| 前端设置 UI 已添加 | ☐ |
| SMTP 发送已实现 | ☐ |
| 测试通过 | ☐ |

---

**Plan complete.** 两种执行方式：

**1. Subagent-Driven (recommended)** - 每次 dispatch 一个 subagent 执行一个 task，中间 review，快速迭代

**2. Inline Execution** - 在当前 session 执行任务，使用 executing-plans，批量执行带检查点

选择哪种方式？
