package models

import (
	"time"
)

type CommentEvent struct {
	CommentID   string    `json:"comment_id"`
	PostID      string    `json:"post_id"`
	AuthorID    string    `json:"author_id"`    // Post author's email
	CommenterID string    `json:"commenter_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

type EmailNotification struct {
	To      string
	Subject string
	Body    string
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type SuccessResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
} 