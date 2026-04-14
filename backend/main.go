package main

import (
	"context"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"rss-reader/backend/db"
	"rss-reader/backend/handlers"
	"rss-reader/backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"
	openai "github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var aiClient *openai.Client
var aiModelName string

// removeThinkTags removes <think>...</think> tags from AI-generated content
func removeThinkTags(text string) string {
	re := regexp.MustCompile(`<think>[\s\S]*?<\/think>`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
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

	fp := gofeed.NewParser()
	for _, source := range sources {
		feed, err := fp.ParseURL(source.URL)
		if err != nil {
			log.Printf("Failed to parse feed %s: %v", source.URL, err)
			continue // Move to the next source
		}

		var newArticles []interface{}
		for _, item := range feed.Items {
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
					UserID:      source.UserID, // Preserve the userId from source
					GUID:        item.GUID,
					Title:       item.Title,
					URL:         item.Link,
					Description: item.Description,
					Content:     content,
					PublishedAt: publishedAt,
				}
				newArticles = append(newArticles, article)
			} else if err != nil {
				// An actual error occurred during the check
				log.Printf("Error checking for article existence %s: %v", item.GUID, err)
			}
		}

		if len(newArticles) > 0 {
			opts := options.InsertMany().SetOrdered(false)
			_, err := db.ArticleCollection.InsertMany(ctx, newArticles, opts)
			if err != nil {
				log.Printf("Failed to insert %d new articles for source %s: %v", len(newArticles), source.Name, err)
			} else {
				log.Printf("Inserted %d new articles for source %s", len(newArticles), source.Name)
			}
		}
	}
	log.Println("Feed update process finished.")
}

func main() {
	db.ConnectDatabase() // Connect to the database

	// Start the background worker
	go func() {
		// Run once on startup
		updateFeeds()

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			updateFeeds()
		}
	}()

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

				// Insert articles with deduplication
				for _, item := range feed.Items {
					content := item.Content
					if content == "" {
						content = item.Description
					}
					var publishedAt *time.Time
					if item.PublishedParsed != nil {
						publishedAt = item.PublishedParsed
					}

					// Check if article already exists for this source
					filter := bson.M{"guid": item.GUID, "sourceId": sourceID}
					err := db.ArticleCollection.FindOne(context.Background(), filter).Err()
					if err == mongo.ErrNoDocuments {
						// Article doesn't exist, insert it
						article := Article{
							UserID:     userID,
							SourceID:   sourceID,
							GUID:       item.GUID,
							Title:      item.Title,
							URL:        item.Link,
							Description: item.Description,
							Content:    content,
							PublishedAt: publishedAt,
						}
						_, insertErr := db.ArticleCollection.InsertOne(context.Background(), article)
						if insertErr != nil {
							log.Printf("Failed to insert article %s: %v", item.GUID, insertErr)
						}
					}
				}

				newSource.ID = sourceID
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
		}
	}

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}