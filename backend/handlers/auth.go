package handlers

import (
	"context"
	"net/http"
	"time"

	"rss-reader/backend/db"
	"rss-reader/backend/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// Register creates a new user account
func Register(c *gin.Context) {
	var json struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required,min=2"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	var existingUser struct{}
	err := db.UserCollection.FindOne(context.Background(), bson.M{"email": json.Email}).Decode(&existingUser)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(json.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user := struct {
		ID           primitive.ObjectID `bson:"_id,omitempty"`
		Email        string             `bson:"email"`
		Username     string             `bson:"username"`
		PasswordHash string             `bson:"passwordHash"`
		CreatedAt    time.Time          `bson:"createdAt"`
		UpdatedAt    time.Time          `bson:"updatedAt"`
	}{
		Email:        json.Email,
		Username:     json.Username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	result, err := db.UserCollection.InsertOne(context.Background(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	})
}

// Login authenticates a user and returns tokens
func Login(c *gin.Context) {
	var json struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user struct {
		ID           primitive.ObjectID `bson:"_id"`
		Email        string             `bson:"email"`
		Username     string             `bson:"username"`
		PasswordHash string             `bson:"passwordHash"`
	}
	err := db.UserCollection.FindOne(context.Background(), bson.M{"email": json.Email}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(json.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate tokens
	accessToken, err := generateAccessToken(user.ID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := generateRefreshToken(user.ID.Hex())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"expiresIn":    900, // 15 minutes in seconds
	})
}

// Refresh refreshes access token
func Refresh(c *gin.Context) {
	var json struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse refresh token
	claims := &middleware.Claims{}
	token, err := jwt.ParseWithClaims(json.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return middleware.JWTSecret, nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Verify user still exists
	userObjID, _ := primitive.ObjectIDFromHex(claims.UserID)
	count, _ := db.UserCollection.CountDocuments(context.Background(), bson.M{"_id": userObjID})
	if count == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User no longer exists"})
		return
	}

	// Generate new tokens
	accessToken, err := generateAccessToken(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	newRefreshToken, err := generateRefreshToken(claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  accessToken,
		"refreshToken": newRefreshToken,
		"expiresIn":    900,
	})
}

// Logout handles user logout
func Logout(c *gin.Context) {
	// In a production app, you might want to blacklist the token
	// For now, just return success - client will delete tokens
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetMe returns the current user's info
func GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	objID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user struct {
		ID        primitive.ObjectID `bson:"_id"`
		Email     string             `bson:"email"`
		Username  string             `bson:"username"`
		CreatedAt time.Time          `bson:"createdAt"`
	}
	err = db.UserCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        user.ID,
		"email":     user.Email,
		"username":  user.Username,
		"createdAt": user.CreatedAt,
	})
}

// Helper functions

func generateAccessToken(userID string) (string, error) {
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.JWTSecret)
}

func generateRefreshToken(userID string) (string, error) {
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(middleware.JWTSecret)
}
