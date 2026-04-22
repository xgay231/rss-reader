package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var DB *mongo.Database
var UserCollection *mongo.Collection
var ArticleCollection *mongo.Collection
var SourceCollection *mongo.Collection
var GroupCollection *mongo.Collection

func ConnectDatabase() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// Ping the primary
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

	// Create unique index on email
	_, err = UserCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]interface{}{"email": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Warning: failed to create email index: %v", err)
	} else {
		log.Println("User email index created successfully")
	}

	// Create index on readStatus for ArticleCollection
	_, err = ArticleCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "readStatus", Value: 1}},
	})
	if err != nil {
		log.Printf("Warning: failed to create readStatus index: %v", err)
	} else {
		log.Println("Article readStatus index created successfully")
	}
}