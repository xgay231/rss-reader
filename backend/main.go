package main

import (
	"context"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"rss-reader/backend/db"

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

// FeedSource represents an RSS feed source
type FeedSource struct {
	ID   primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name string             `json:"name" bson:"name"`
	URL  string             `json:"url" bson:"url"`
}

// Article represents a single RSS feed item
type Article struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	SourceID    primitive.ObjectID `json:"sourceId" bson:"sourceId"`
	GUID        string             `json:"guid" bson:"guid"`
	Title       string             `json:"title" bson:"title"`
	URL         string             `json:"url" bson:"url"`
	Description string             `json:"description" bson:"description"`
	Content     string             `json:"content" bson:"content"`
	PublishedAt *time.Time        `json:"publishedAt" bson:"publishedAt"`
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
		// Source routes
		sources := api.Group("/sources")
		{
			// Add a new feed source
			sources.POST("", func(c *gin.Context) {
				var json struct {
					URL string `json:"url" binding:"required"`
				}

				if err := c.ShouldBindJSON(&json); err != nil {
					c.JSON(400, gin.H{"error": err.Error()})
					return
				}

				// Check if feed already exists
				var existingSource FeedSource
				err := db.SourceCollection.FindOne(context.Background(), bson.M{"url": json.URL}).Decode(&existingSource)
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
					Name: feed.Title,
					URL:  json.URL,
				}

				res, err := db.SourceCollection.InsertOne(context.Background(), newSource)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to save feed source"})
					return
				}
				sourceID := res.InsertedID.(primitive.ObjectID)

				var articles []interface{}
				for _, item := range feed.Items {
					content := item.Content
					if content == "" {
						content = item.Description
					}
					var publishedAt *time.Time
					if item.PublishedParsed != nil {
						publishedAt = item.PublishedParsed
					}
					article := Article{
						SourceID:    sourceID,
						GUID:        item.GUID,
						Title:       item.Title,
						URL:         item.Link,
						Description: item.Description,
						Content:     content,
						PublishedAt: publishedAt,
					}
					articles = append(articles, article)
				}

				if len(articles) > 0 {
					opts := options.InsertMany().SetOrdered(false)
					_, err = db.ArticleCollection.InsertMany(context.Background(), articles, opts)
					if err != nil {
						// Log the error but don't fail the request for adding the source
						log.Printf("Failed to insert articles for source %s: %v", sourceID.Hex(), err)
					}
				}

				newSource.ID = sourceID
				c.JSON(201, newSource)
			})

			// Get all feed sources
			sources.GET("", func(c *gin.Context) {
				var sources []FeedSource
				cursor, err := db.SourceCollection.Find(context.Background(), bson.M{})
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
				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid ID"})
					return
				}

				// Delete the source
				_, err = db.SourceCollection.DeleteOne(context.Background(), bson.M{"_id": id})
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to delete source"})
					return
				}

				// Delete associated articles
				_, err = db.ArticleCollection.DeleteMany(context.Background(), bson.M{"sourceId": id})
				if err != nil {
					// Log this error but don't fail the request, as the source is already deleted.
					log.Printf("Failed to delete articles for source %s: %v", id.Hex(), err)
				}

				c.JSON(200, gin.H{"status": "ok"})
			})

			// Get articles for a specific source
			sources.GET("/:id/articles", func(c *gin.Context) {
				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Source ID"})
					return
				}

				var articles []Article
				cursor, err := db.ArticleCollection.Find(context.Background(), bson.M{"sourceId": id})
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
		}

		// Article routes
		articles := api.Group("/articles")
		{
			// Get a single article by its ID
			articles.GET("/:id", func(c *gin.Context) {
				id, err := primitive.ObjectIDFromHex(c.Param("id"))
				if err != nil {
					c.JSON(400, gin.H{"error": "Invalid Article ID"})
					return
				}

				var article Article
				err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&article)
				if err != nil {
					c.JSON(404, gin.H{"error": "Article not found"})
					return
				}

				c.JSON(200, article)
			})

			// Generate AI summary for an article
			articles.POST("/:id/ai-summary", func(c *gin.Context) {
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
				err = db.ArticleCollection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&article)
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

				c.JSON(200, gin.H{"summary": removeThinkTags(resp.Choices[0].Message.Content)})
			})
		}
	}

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}