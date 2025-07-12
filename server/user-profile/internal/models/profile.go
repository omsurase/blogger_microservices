package models

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Bio         string    `json:"bio"`
	AvatarURL   string    `json:"avatar_url"`
	TwitterURL  string    `json:"twitter_url"`
	LinkedInURL string    `json:"linkedin_url"`
	GithubURL   string    `json:"github_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UpdateProfileRequest struct {
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	TwitterURL  string `json:"twitter_url"`
	LinkedInURL string `json:"linkedin_url"`
	GithubURL   string `json:"github_url"`
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