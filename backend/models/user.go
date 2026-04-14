package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user account
type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email        string             `json:"email" bson:"email"`
	Username     string             `json:"username" bson:"username"`
	PasswordHash string             `json:"-" bson:"passwordHash"` // never expose to client
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
}
