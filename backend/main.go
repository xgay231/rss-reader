package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// A test route to make sure everything is working
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.Run(":8080") // listen and serve on 0.0.0.0:8080
}