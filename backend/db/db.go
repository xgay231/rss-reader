package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var ArticleCollection *mongo.Collection

func ConnectDatabase() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:password@localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// Ping the primary
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully connected to MongoDB!")
	Client = client
	ArticleCollection = client.Database("rss_reader").Collection("articles")
}