package main

import (
	"context"
	"log"
	"github.com/gin-gonic/gin"
	"github.com/mmcdole/gofeed"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"rss-reader/backend/db"
)

// Article represents a single RSS feed item
type Article struct {
	GUID        string `json:"guid" bson:"_id,omitempty"`
	Title       string `json:"title" bson:"title"`
	URL         string `json:"url" bson:"url"`
	Description string `json:"description" bson:"description"`
}

func main() {
	db.ConnectDatabase() // Connect to the database

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
		api.POST("/feeds", func(c *gin.Context) {
			var json struct {
				URL string `json:"url" binding:"required"`
			}

			if err := c.ShouldBindJSON(&json); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			fp := gofeed.NewParser()
			feed, err := fp.ParseURL(json.URL)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to parse feed"})
				return
			}

			var articles []interface{}
			for _, item := range feed.Items {
				article := Article{
					GUID:        item.GUID,
					Title:       item.Title,
					URL:         item.Link,
					Description: item.Description,
				}
				articles = append(articles, article)
			}

			if len(articles) == 0 {
				c.JSON(200, gin.H{"status": "ok", "articles_added": 0, "message": "No new articles found"})
				return
			}

			opts := options.InsertMany().SetOrdered(false)
			_, err = db.ArticleCollection.InsertMany(context.Background(), articles, opts)
			if err != nil {
				log.Printf("Failed to insert articles: %v", err) // Log the actual error
				c.JSON(500, gin.H{"error": "Failed to save articles"})
				return
			}

			c.JSON(200, gin.H{"status": "ok", "articles_added": len(articles)})
		})

		api.GET("/articles", func(c *gin.Context) {
			var articles []Article
			cursor, err := db.ArticleCollection.Find(context.Background(), bson.M{})
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

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}