package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user account
type User struct {
	ID                  primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email               string             `json:"email" bson:"email"`
	Username            string             `json:"username" bson:"username"`
	PasswordHash        string             `json:"-" bson:"passwordHash"` // never expose to client
	FeedUpdateInterval  int                `json:"feedUpdateInterval" bson:"feedUpdateInterval"`
	AutoSummary         bool               `json:"autoSummary" bson:"autoSummary"`
	DailySummaryEnabled bool               `json:"dailySummaryEnabled" bson:"dailySummaryEnabled"`
	DailySummaryTime    string             `json:"dailySummaryTime" bson:"dailySummaryTime"`         // 格式 "HH:MM"
	DailySummaryEmail   string             `json:"dailySummaryEmail" bson:"dailySummaryEmail"`       // 发送目标邮箱
	SmtpPassword        string             `json:"-" bson:"smtpPassword"`                            // SMTP 密码，不暴露给前端
	CreatedAt           time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt           time.Time          `json:"updatedAt" bson:"updatedAt"`
}
