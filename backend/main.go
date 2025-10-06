package main

import (
	"github.com/gin-gonic/gin"
	"rss-reader/backend/db"
)

// Article represents a single RSS feed item
type Article struct {
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

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}