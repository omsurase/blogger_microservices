package models

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateCommentRequest struct {
	PostID  string `json:"post_id" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type CommentEvent struct {
	CommentID   string    `json:"comment_id"`
	PostID      string    `json:"post_id"`
	AuthorID    string    `json:"author_id"`
	CommenterID string    `json:"commenter_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
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

type Post struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
} 