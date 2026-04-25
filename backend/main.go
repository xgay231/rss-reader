package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"rss-reader/backend/db"
	"rss-reader/backend/handlers"
	"rss-reader/backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-mail/mail/v2"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron/v3"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var aiClient *openai.Client
var aiModelName string
var feedUpdateInterval = 15 * time.Minute // default 15 minutes
var feedUpdateIntervalMins = 15           // interval in minutes for dynamic updates

// SMTP config
var (
	smtpHost = os.Getenv("SMTP_HOST")
	smtpPort = os.Getenv("SMTP_PORT")
	smtpUser = os.Getenv("SMTP_USER")
)

// ctx and cancel are used for graceful shutdown of background tasks
var ctx, stop = context.WithCancel(context.Background())

// updateTicker controls the feed update ticker
var updateTicker *time.Ticker
var tickerStopChan chan struct{}

// removeThinkTags removes <think>...</think> tags from AI-generated content
func removeThinkTags(text string) string {
	re := regexp.MustCompile(`<think>[\s\S]*?<\/think>`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

// generateSummary generates an AI summary for an article and saves it to the database
func generateSummary(articleID primitive.ObjectID, userID primitive.ObjectID) {
	if aiClient == nil {
		log.Printf("AI client not available, skipping summary generation for article %s", articleID.Hex())
		return
	}

	ctx := context.Background()
	var article Article
	err := db.ArticleCollection.FindOne(ctx, bson.M{"_id": articleID, "userId": userID}).Decode(&article)
	if err != nil {
		log.Printf("Failed to find article %s for summary generation: %v", articleID.Hex(), err)
		return
	}

	// Idempotency check: skip if summary already exists
	if article.Summary != "" {
		log.Printf("Summary already exists for article %s, skipping", articleID.Hex())
		return
	}

	// Create a chat completion request
	resp, err := aiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: aiModelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "你是一个帮助用户总结文章内容的助手。请用中文输出纯文本摘要，不要使用 markdown 格式，不要使用列表、标题、粗体等任何格式标记，只输出纯段落文本。",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Please summarize the following article content:\n\n" + article.Content,
				},
			},
		},
	)

	if err != nil {
		log.Printf("ChatCompletion error for article %s: %v", articleID.Hex(), err)
		return
	}

	if len(resp.Choices) == 0 {
		log.Printf("No summary content returned from AI for article %s", articleID.Hex())
		return
	}

	summary := removeThinkTags(resp.Choices[0].Message.Content)

	// Save summary to database
	now := time.Now()
	update := bson.M{"$set": bson.M{"summary": summary, "summaryGeneratedAt": now}}
	_, err = db.ArticleCollection.UpdateByID(ctx, articleID, update)
	if err != nil {
		log.Printf("Failed to save AI summary for article %s: %v", articleID.Hex(), err)
		return
	}

	log.Printf("Successfully generated summary for article %s", articleID.Hex())
}

func init() {
	// Load .env file
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
		log.Printf("Using custom OpenAI Base URL: %s", baseURL)
	}

	aiClient = openai.NewClientWithConfig(config)
	log.Println("OpenAI client initialized successfully.")

	aiModelName = os.Getenv("OPENAI_MODEL_NAME")
	if aiModelName == "" {
		aiModelName = openai.GPT3Dot5Turbo // Default model
		log.Printf("OPENAI_MODEL_NAME not set, using default: %s", aiModelName)
	} else {
		log.Printf("Using AI Model: %s", aiModelName)
	}
}

// Group represents a feed source group
type Group struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	Name      string             `json:"name" bson:"name"`
	SortOrder int                `json:"sortOrder" bson:"sortOrder"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

// FeedSource represents an RSS feed source
type FeedSource struct {
	ID      primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID  primitive.ObjectID `json:"userId" bson:"userId"`
	Name    string             `json:"name" bson:"name"`
	URL     string             `json:"url" bson:"url"`
	GroupID primitive.ObjectID `json:"groupId" bson:"groupId"`
}

// Article represents a single RSS feed item
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

// Settings represents user settings
type Settings struct {
	FeedUpdateInterval int  `json:"feedUpdateInterval"`
	AutoSummary       bool `json:"autoSummary"`
}

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

// getUserID extracts and validates userID from context
func getUserID(c *gin.Context) primitive.ObjectID {
	userID, exists := c.Get("userID")
	if !exists {
		return primitive.NilObjectID
	}
	id, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		return primitive.NilObjectID
	}
	return id
}

// GetSettings returns the user's settings
func GetSettings(c *gin.Context) {
	userID := getUserID(c)
	if userID == primitive.NilObjectID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user struct {
		FeedUpdateInterval int   `bson:"feedUpdateInterval"`
		AutoSummary        *bool `bson:"autoSummary"` // pointer to distinguish "not set" from "explicitly false"
	}

	err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	// Return defaults if not set and persist to DB
	feedInterval := user.FeedUpdateInterval
	autoSummary := user.AutoSummary
	needsUpdate := false

	if feedInterval == 0 {
		feedInterval = 15
		needsUpdate = true
	}

	if autoSummary == nil {
		// Field not set, use default true
		autoSummary = new(bool)
		*autoSummary = true
		needsUpdate = true
	}

	if needsUpdate {
		// Persist defaults to database so updateFeeds reads correct values
		db.UserCollection.UpdateOne(
			context.Background(),
			bson.M{"_id": userID},
			bson.M{"$set": bson.M{
				"feedUpdateInterval": feedInterval,
				"autoSummary":        *autoSummary,
			}},
		)
	}

	c.JSON(http.StatusOK, Settings{
		FeedUpdateInterval: feedInterval,
		AutoSummary:        *autoSummary,
	})
}

// UpdateSettings updates the user's settings
func UpdateSettings(c *gin.Context) {
	userID := getUserID(c)
	if userID == primitive.NilObjectID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var json struct {
		FeedUpdateInterval *int  `json:"feedUpdateInterval"`
		AutoSummary        *bool `json:"autoSummary"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate feedUpdateInterval
	if json.FeedUpdateInterval != nil {
		if *json.FeedUpdateInterval < 1 || *json.FeedUpdateInterval > 1440 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "feedUpdateInterval must be between 1 and 1440"})
			return
		}
	}

	update := bson.M{}
	if json.FeedUpdateInterval != nil {
		update["feedUpdateInterval"] = *json.FeedUpdateInterval
	}
	if json.AutoSummary != nil {
		update["autoSummary"] = *json.AutoSummary
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

	// If feedUpdateInterval was changed, signal ticker to restart with new interval
	if json.FeedUpdateInterval != nil && *json.FeedUpdateInterval != feedUpdateIntervalMins {
		feedUpdateIntervalMins = *json.FeedUpdateInterval
		feedUpdateInterval = time.Duration(feedUpdateIntervalMins) * time.Minute
		signalTickerRestart()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
}

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

// generateMergedSummary calls AI to merge multiple articles into a single summary
// Returns (summary, error). If there are no articles, returns ("", nil).
// If AI client is not available, returns ( "", error).
func generateMergedSummary(articles []Article) (string, error) {
	if aiClient == nil {
		return "", fmt.Errorf("AI client not available")
	}

	if len(articles) == 0 {
		return "", nil
	}

	// Build article list using strings.Builder for efficiency
	var builder strings.Builder
	for i, article := range articles {
		builder.WriteString(fmt.Sprintf("%d. \"%s\" - 来源: %s\n", i+1, article.Title, article.URL))
	}

	prompt := fmt.Sprintf(`请为用户生成一段今日文章摘要。

文章列表：
%s

请生成一段 100-200 字的合并摘要，概括今日文章的核心内容。用中文输出。`, builder.String())

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

	// Build article list HTML
	var articleListHTML strings.Builder
	for _, article := range articles {
		articleListHTML.WriteString(fmt.Sprintf(
			`<li><a href="%s" style="color: #0066cc;">%s</a></li>`,
			article.URL, html.EscapeString(article.Title)))
	}

	if articleListHTML.Len() == 0 {
		articleListHTML.WriteString("<li>今日暂无新文章</li>")
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
  <h2 style="color: #333;">每日文章总结</h2>
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
</html>`, today, len(articles), summary, articleListHTML.String())

	// 发送邮件
	host := smtpHost
	if host == "" {
		host = "smtp.qq.com" // 默认使用 QQ 邮箱 SMTP
	}
	port := 587 // 默认 TLS 端口
	if smtpPort != "" {
		if p, err := strconv.Atoi(smtpPort); err == nil {
			port = p
		}
	}

	// 创建邮件消息
	msg := mail.NewMessage()
	msg.SetHeader("From", user.DailySummaryEmail)
	msg.SetHeader("To", user.DailySummaryEmail)
	msg.SetHeader("Subject", "每日文章总结")
	msg.SetBody("text/html", htmlBody)

	// 创建 Dialer 并发送
	dialer := mail.NewDialer(host, port, user.DailySummaryEmail, user.SmtpPassword)
	dialer.StartTLSPolicy = mail.MandatoryStartTLS
	dialer.Timeout = 30 * time.Second
	err = dialer.DialAndSend(msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("[DailySummary] Email sent for user %s: to=%s", userID.Hex(), user.DailySummaryEmail)
	return nil
}

// SendDailySummary 手动触发发送每日总结
func SendDailySummary(c *gin.Context) {
	userID := getUserID(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 同步检查配置是否有效，再决定是否启动异步发送
	var user struct {
		DailySummaryEnabled bool   `bson:"dailySummaryEnabled"`
		DailySummaryEmail   string `bson:"dailySummaryEmail"`
		SmtpPassword        string `bson:"smtpPassword"`
	}
	err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check settings"})
		return
	}

	if !user.DailySummaryEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "每日总结未启用"})
		return
	}
	if user.DailySummaryEmail == "" || user.SmtpPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱或SMTP密码未配置"})
		return
	}

	// 配置有效，异步发送
	go func() {
		if err := sendDailySummaryEmail(userID); err != nil {
			log.Printf("[DailySummary] Failed to send daily summary for user %s: %v", userID.Hex(), err)
		} else {
			log.Printf("[DailySummary] Daily summary sent successfully for user %s", userID.Hex())
		}
	}()

	c.JSON(http.StatusOK, DailySummarySendResult{
		Success: true,
		Message: "每日总结发送中...",
	})
}

// signalTickerRestart signals the background ticker to restart with new interval
func signalTickerRestart() {
	if tickerStopChan != nil {
		close(tickerStopChan)
	}
}

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

// UpdateDailySummarySettings 更新每日总结设置
func UpdateDailySummarySettings(c *gin.Context) {
	userID := getUserID(c)
	if userID.IsZero() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var json struct {
		Enabled      *bool   `json:"enabled"`
		Time         *string `json:"time"`
		Email        *string `json:"email"`
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

func updateFeeds() {
	log.Println("Starting feed update process...")
	var sources []FeedSource
	ctx := context.Background()

	cursor, err := db.SourceCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to get sources from DB: %v", err)
		return
	}
	if err = cursor.All(ctx, &sources); err != nil {
		log.Printf("Failed to decode sources: %v", err)
		return
	}

	log.Printf("Found %d sources to process", len(sources))

	fp := gofeed.NewParser()
	for _, source := range sources {
		log.Printf("Processing source: %s (ID: %s, UserID: %s)", source.Name, source.ID.Hex(), source.UserID.Hex())
		feed, err := fp.ParseURL(source.URL)
		if err != nil {
			log.Printf("Failed to parse feed %s: %v", source.URL, err)
			continue // Move to the next source
		}

		insertArticlesAndGenerateSummary(source, feed.Items, ctx)
	}
	log.Println("Feed update process finished.")
}

// RefreshFeeds triggers a manual feed update asynchronously
func RefreshFeeds(c *gin.Context) {
	go updateFeeds()
	c.JSON(http.StatusOK, gin.H{"message": "Feed refresh triggered"})
}

// insertArticlesAndGenerateSummary inserts new articles from a feed and triggers auto-summary
// Returns the IDs of newly inserted articles
func insertArticlesAndGenerateSummary(source FeedSource, items []*gofeed.Item, ctx context.Context) []interface{} {
	var newArticles []interface{}

	for _, item := range items {
		// Check if article already exists
		filter := bson.M{"guid": item.GUID, "sourceId": source.ID}
		err := db.ArticleCollection.FindOne(ctx, filter).Err()
		if err == mongo.ErrNoDocuments {
			// Article is new, add it to the list
			content := item.Content
			if content == "" {
				content = item.Description
			}
			var publishedAt *time.Time
			if item.PublishedParsed != nil {
				publishedAt = item.PublishedParsed
			}
			article := Article{
				SourceID:    source.ID,
				UserID:      source.UserID,
				GUID:        item.GUID,
				Title:       item.Title,
				URL:         item.Link,
				Description: item.Description,
				Content:     content,
				PublishedAt: publishedAt,
				ReadStatus:  "unread",
			}
			newArticles = append(newArticles, article)
		}
	}

	if len(newArticles) == 0 {
		return nil
	}

	// Insert articles
	opts := options.InsertMany().SetOrdered(false)
	result, err := db.ArticleCollection.InsertMany(ctx, newArticles, opts)
	if err != nil {
		log.Printf("Failed to insert %d new articles for source %s: %v", len(newArticles), source.Name, err)
		return nil
	}

	log.Printf("Inserted %d new articles for source %s", len(newArticles), source.Name)

	// Auto-generate summaries for new articles if user has autoSummary enabled
	go func(source FeedSource, insertedIDs []interface{}) {
		// Check if user has autoSummary enabled
		var user struct {
			AutoSummary bool `bson:"autoSummary"`
		}
		err := db.UserCollection.FindOne(context.Background(), bson.M{"_id": source.UserID}).Decode(&user)
		if err != nil {
			log.Printf("[AutoSummary] Failed to check autoSummary setting for user %s: %v", source.UserID.Hex(), err)
			return
		}

		// If autoSummary is disabled, skip
		if !user.AutoSummary {
			return
		}

		// Semaphore to limit concurrent goroutines to 5
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

// startFeedWorker starts the background feed update worker with dynamic interval support
func startFeedWorker() {
	// Run once on startup
	updateFeeds()

	for {
		tickerStopChan = make(chan struct{})
		ticker := time.NewTicker(feedUpdateInterval)
		defer ticker.Stop()

		select {
		case <-ticker.C:
			updateFeeds()
		case <-tickerStopChan:
			// Interval changed, restart ticker with new interval
			log.Printf("Feed update interval changed to %d minutes", feedUpdateIntervalMins)
		}
	}
}

func main() {
	db.ConnectDatabase() // Connect to the database

	// Start the background worker with dynamic interval
	go startFeedWorker()

	router := gin.Default()

	// A test route to make sure everything is working
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/refresh", handlers.Refresh)
			auth.POST("/logout", handlers.Logout)
		}

		// Protected routes - require authentication
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/auth/me", handlers.GetMe)
		}

		// Settings routes (protected)
		settings := api.Group("/settings")
		settings.Use(middleware.AuthMiddleware())
		{
			settings.GET("", GetSettings)
			settings.PUT("", UpdateSettings)
		}

		// Daily summary routes (protected)
		dailySummary := api.Group("/daily-summary")
		dailySummary.Use(middleware.AuthMiddleware())
		{
			dailySummary.GET("/settings", GetDailySummarySettings)
			dailySummary.PUT("/settings", UpdateDailySummarySettings)
			dailySummary.POST("/send", SendDailySummary)
		}

		// Source routes (protected)
		sources := api.Group("/sources")
		sources.Use(middleware.AuthMiddleware())
		{
			// Add a new feed source
			sources.POST("", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				var json struct {
					URL string `json:"url" binding:"required"`
				}

				if err := c.ShouldBindJSON(&json); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				// Check if feed already exists for this user
				var existingSource FeedSource
				err := db.SourceCollection.FindOne(context.Background(), bson.M{"url": json.URL, "userId": userID}).Decode(&existingSource)
				if err == nil {
					c.JSON(409, gin.H{"error": "Feed source already exists"})
					return
				}

				fp := gofeed.NewParser()
				feed, err := fp.ParseURL(json.URL)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to parse feed"})
					return
				}

				newSource := FeedSource{
					UserID: userID,
					Name:   feed.Title,
					URL:    json.URL,
				}

				res, err := db.SourceCollection.InsertOne(context.Background(), newSource)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to save feed source"})
					return
				}
				sourceID := res.InsertedID.(primitive.ObjectID)

				// Fix duplicate starred articles issue - move orphaned starred articles to new source BEFORE checking for duplicates
				for _, item := range feed.Items {
					var existingStarred Article
					findFilter := bson.M{"guid": item.GUID, "isStarred": true, "userId": userID}
					err := db.ArticleCollection.FindOne(context.Background(), findFilter).Decode(&existingStarred)
					if err == nil {
						var source FeedSource
						sourceErr := db.SourceCollection.FindOne(context.Background(), bson.M{"_id": existingStarred.SourceID}).Decode(&source)
						if sourceErr == mongo.ErrNoDocuments {
							_, updateErr := db.ArticleCollection.UpdateOne(
								context.Background(),
								bson.M{"_id": existingStarred.ID},
								bson.M{"$set": bson.M{"sourceId": sourceID}},
							)
							if updateErr != nil {
								log.Printf("Failed to update orphaned starred article: %v", updateErr)
							}
						}
					}
				}

				// Insert articles and trigger auto-summary using shared function
				newSource.UserID = userID
				insertArticlesAndGenerateSummary(newSource, feed.Items, context.Background())

				c.JSON(201, newSource)
			})

			// Get all feed sources
			sources.GET("", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				var sources []FeedSource
				cursor, err := db.SourceCollection.Find(context.Background(), bson.M{"userId": userID})
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to fetch sources"})
					return
				}
				defer cursor.Close(context.Background())

				if err = cursor.All(context.Background(), &sources); err != nil {
					c.JSON(500, gin.H{"error": "Failed to decode sources"})
					return
				}

				c.JSON(200, sources)
			})

			// Delete a feed source
			sources.DELETE("/:id", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid ID"})
					return
				}

				// Delete the source (only if belongs to user)
				_, err = db.SourceCollection.DeleteOne(context.Background(), bson.M{"_id": id, "userId": userID})
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to delete source"})
					return
				}

				// Delete associated articles (keep starred articles)
				_, err = db.ArticleCollection.DeleteMany(context.Background(), bson.M{"sourceId": id, "isStarred": false, "userId": userID})
				if err != nil {
					// Log this error but don't fail the request, as the source is already deleted.
					log.Printf("Failed to delete articles for source %s: %v", id.Hex(), err)
				}

				c.JSON(200, gin.H{"status": "ok"})
			})

			// Get articles for a specific source
			sources.GET("/:id/articles", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Source ID"})
					return
				}

				var articles []Article
				cursor, err := db.ArticleCollection.Find(context.Background(), bson.M{"sourceId": id, "userId": userID})
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to fetch articles"})
					return
				}
				defer cursor.Close(context.Background())

				if err = cursor.All(context.Background(), &articles); err != nil {
					c.JSON(500, gin.H{"error": "Failed to decode articles"})
					return
				}

				c.JSON(200, articles)
			})

			// Mark all articles in a source as read
			sources.PUT("/:id/mark-all-read", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Source ID"})
					return
				}

				// Check if source exists and belongs to user
				var source FeedSource
				err = db.SourceCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&source)
				if err != nil {
					c.JSON(404, gin.H{"error": "Source not found"})
					return
				}

				update := bson.M{"$set": bson.M{"readStatus": "read"}}
				filter := bson.M{"sourceId": id, "userId": userID, "readStatus": "unread"}
				result, err := db.ArticleCollection.UpdateMany(context.Background(), filter, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to mark articles as read"})
					return
				}

				c.JSON(200, gin.H{"modifiedCount": result.ModifiedCount})
			})

			// Assign source to group
			sources.PUT("/:id/group", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Source ID"})
					return
				}

				var json struct {
					GroupID string `json:"groupId"`
				}

				if err := c.ShouldBindJSON(&json); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				var groupID primitive.ObjectID
				if json.GroupID != "" {
					groupID, err = primitive.ObjectIDFromHex(json.GroupID)
					if err != nil {
						c.JSON(400, gin.H{"error": "Invalid Group ID"})
						return
					}
				}

				update := bson.M{"$set": bson.M{"groupId": groupID}}
				_, err = db.SourceCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to update source"})
					return
				}

				c.JSON(200, gin.H{"status": "ok"})
			})

			// Trigger a manual feed refresh
			sources.POST("/refresh", RefreshFeeds)
		}

		// Group routes (protected)
		groups := api.Group("/groups")
		groups.Use(middleware.AuthMiddleware())
		{
			// Create a group
			groups.POST("", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				var json struct {
					Name string `json:"name" binding:"required"`
				}

				if err := c.ShouldBindJSON(&json); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				group := Group{
					UserID:    userID,
					Name:      json.Name,
					CreatedAt: time.Now(),
				}

				res, err := db.GroupCollection.InsertOne(context.Background(), group)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to create group"})
					return
				}

				group.ID = res.InsertedID.(primitive.ObjectID)
				c.JSON(201, group)
			})

			// Get all groups
			groups.GET("", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				var groups []Group
				cursor, err := db.GroupCollection.Find(context.Background(), bson.M{"userId": userID})
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to fetch groups"})
					return
				}
				defer cursor.Close(context.Background())

				if err = cursor.All(context.Background(), &groups); err != nil {
					c.JSON(500, gin.H{"error": "Failed to decode groups"})
					return
				}

				c.JSON(200, groups)
			})

			// Update a group
			groups.PUT("/:id", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Group ID"})
					return
				}

				var json struct {
					Name string `json:"name"`
				}

				if err := c.ShouldBindJSON(&json); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				update := bson.M{"$set": bson.M{"name": json.Name}}
				_, err = db.GroupCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to update group"})
					return
				}

				c.JSON(200, gin.H{"status": "ok"})
			})

			// Delete a group
			groups.DELETE("/:id", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Group ID"})
					return
				}

				// Delete the group (only if belongs to user)
				_, err = db.GroupCollection.DeleteOne(context.Background(), bson.M{"_id": id, "userId": userID})
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to delete group"})
					return
				}

				// Remove groupId from sources in this group
				_, err = db.SourceCollection.UpdateMany(context.Background(), bson.M{"groupId": id, "userId": userID}, bson.M{"$set": bson.M{"groupId": primitive.NilObjectID}})
				if err != nil {
					log.Printf("Failed to update sources after group deletion: %v", err)
				}

				c.JSON(200, gin.H{"status": "ok"})
			})
		}

		// Article routes (protected)
		articles := api.Group("/articles")
		articles.Use(middleware.AuthMiddleware())
		{
			// Get all starred articles
			articles.GET("/starred", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				filter := bson.M{"isStarred": true, "userId": userID}
				cursor, err := db.ArticleCollection.Find(context.Background(), filter)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to fetch starred articles"})
					return
				}
				defer cursor.Close(context.Background())

				articles := []Article{}
				if err = cursor.All(context.Background(), &articles); err != nil {
					c.JSON(500, gin.H{"error": "Failed to decode articles"})
					return
				}

				c.JSON(200, articles)
			})

			// Get articles with optional filtering
			articles.GET("", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				showRead := c.Query("showRead") == "true"
				showUnread := c.Query("showUnread") == "true"

				// Build filter condition
				filter := bson.M{"userId": userID}

				// If both are false, return empty array (user explicitly chose blank state)
				if !showRead && !showUnread {
					c.JSON(200, []Article{})
					return
				}

				// Add readStatus filter
				if showRead && !showUnread {
					filter["readStatus"] = "read"
				} else if !showRead && showUnread {
					filter["readStatus"] = "unread"
				}
				// If both are true, don't add readStatus filter

				cursor, err := db.ArticleCollection.Find(context.Background(), filter)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to fetch articles"})
					return
				}
				defer cursor.Close(context.Background())

				var articles []Article
				if err = cursor.All(context.Background(), &articles); err != nil {
					c.JSON(500, gin.H{"error": "Failed to decode articles"})
					return
				}

				c.JSON(200, articles)
			})

			// Get a single article by its ID
			articles.GET("/:id", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				var article Article
				err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&article)
				if err != nil {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				c.JSON(200, article)
			})

			// Generate AI summary for an article
			articles.POST("/:id/ai-summary", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				if aiClient == nil {
					c.JSON(503, gin.H{"error": "AI service is not available"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				var article Article
				err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&article)
				if err != nil {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				// Create a chat completion request
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
								Content: "Please summarize the following article content:\n\n" + article.Content,
							},
						},
					},
				)

				if err != nil {
					log.Printf("ChatCompletion error: %v\n", err)
					c.JSON(500, gin.H{"error": "Failed to generate AI summary"})
					return
				}

				if len(resp.Choices) == 0 {
					c.JSON(500, gin.H{"error": "No summary content returned from AI"})
					return
				}

				summary := removeThinkTags(resp.Choices[0].Message.Content)

				// Save summary to database
				now := time.Now()
				update := bson.M{"$set": bson.M{"summary": summary, "summaryGeneratedAt": now}}
				_, err = db.ArticleCollection.UpdateByID(context.Background(), id, update)
				if err != nil {
					log.Printf("Failed to save AI summary for article %s: %v", id.Hex(), err)
					// Continue anyway - don't fail the request
				}

				c.JSON(200, gin.H{"summary": summary})
			})

			// Star an article
			articles.POST("/:id/star", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				update := bson.M{"$set": bson.M{"isStarred": true, "starredAt": time.Now()}}
				result, err := db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to star article"})
					return
				}

				if result.MatchedCount == 0 {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				c.JSON(200, gin.H{"status": "ok", "isStarred": true})
			})

			// Unstar an article
			articles.DELETE("/:id/star", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				update := bson.M{"$set": bson.M{"isStarred": false}}
				result, err := db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to unstar article"})
					return
				}

				if result.MatchedCount == 0 {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				c.JSON(200, gin.H{"status": "ok", "isStarred": false})
			})

			// Mark article as read
			articles.PUT("/:id/read", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				var article Article
				err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&article)
				if err != nil {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				update := bson.M{"$set": bson.M{"readStatus": "read"}}
				_, err = db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to update article"})
					return
				}

				article.ReadStatus = "read"
				c.JSON(200, article)
			})

			// Mark article as unread
			articles.DELETE("/:id/read", func(c *gin.Context) {
				userID := getUserID(c)
				if userID == primitive.NilObjectID {
					c.JSON(401, gin.H{"error": "Unauthorized"})
					return
				}

				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				var article Article
				err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id, "userId": userID}).Decode(&article)
				if err != nil {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				update := bson.M{"$set": bson.M{"readStatus": "unread"}}
				_, err = db.ArticleCollection.UpdateOne(context.Background(), bson.M{"_id": id, "userId": userID}, update)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to update article"})
					return
				}

				article.ReadStatus = "unread"
				c.JSON(200, article)
			})
		}
	}

	// 初始化每日总结定时调度器
	initDailySummaryScheduler()

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}

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