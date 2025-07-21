package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/comment/internal/models"
	"github.com/omsurase/blogger_microservices/server/comment/internal/publisher"
	"github.com/omsurase/blogger_microservices/server/comment/internal/store"
)

type CommentHandler struct {
	store     *store.PostgresStore
	publisher *publisher.RabbitMQPublisher
}

func NewCommentHandler(store *store.PostgresStore, publisher *publisher.RabbitMQPublisher) *CommentHandler {
	return &CommentHandler{
		store:     store,
		publisher: publisher,
	}
}

func sendError(c *gin.Context, status int, message string, err error) {
	errResponse := models.ErrorResponse{
		Status:  status,
		Message: message,
	}
	if err != nil {
		errResponse.Error = err.Error()
	}
	c.Header("Content-Type", "application/json")
	c.JSON(status, errResponse)
}

func sendSuccess(c *gin.Context, status int, data interface{}) {
	c.Header("Content-Type", "application/json")
	c.JSON(status, models.SuccessResponse{
		Status: status,
		Data:   data,
	})
}

func (h *CommentHandler) getPost(postID string) (*models.Post, error) {
    postServiceURL := os.Getenv("POST_SERVICE_URL")
    if postServiceURL == "" {
        postServiceURL = "http://post-service:8080"
    }

    postURL := fmt.Sprintf("%s/post/%s", postServiceURL, postID)
    log.Printf("Fetching post details from: %s", postURL)
    resp, err := http.Get(postURL)
    if err != nil {
        log.Printf("Failed to fetch post: %v", err)
        return nil, fmt.Errorf("failed to fetch post: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        log.Printf("Post service returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
        return nil, fmt.Errorf("failed to fetch post: %s", string(body))
    }

    body, _ := io.ReadAll(resp.Body)
    log.Printf("Post service response: %s", string(body))

    var response struct {
        Status int         `json:"status"`
        Data   models.Post `json:"data"`
    }
    if err := json.Unmarshal(body, &response); err != nil {
        log.Printf("Failed to decode post response: %v", err)
        return nil, fmt.Errorf("failed to decode post response: %v", err)
    }

    log.Printf("Successfully fetched post details: %+v", response.Data)
    return &response.Data, nil
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "User ID not found in context", nil)
		return
	}

	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid post ID format", err)
		return
	}

	post, err := h.getPost(req.PostID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch post details", err)
		return
	}

	parsedUserID, err := uuid.Parse(userID.(string))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	comment := &models.Comment{
		ID:        uuid.New(),
		PostID:    postID,
		UserID:    parsedUserID,
		Content:   req.Content,
		CreatedAt: time.Now().UTC(),
	}

	if err := h.store.CreateComment(comment); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create comment", err)
		return
	}

	event := &models.CommentEvent{
		CommentID:   comment.ID.String(),
		PostID:      comment.PostID.String(),
		AuthorID:    post.UserID,
		CommenterID: comment.UserID.String(),
		Content:     comment.Content,
		CreatedAt:   comment.CreatedAt,
	}

	eventJSON, _ := json.Marshal(event)
	log.Printf("Publishing comment event: %s", string(eventJSON))

	if err := h.publisher.PublishNewComment(event); err != nil {
		log.Printf("Failed to publish comment event: %v", err)
		sendError(c, http.StatusInternalServerError, "Failed to publish comment event", err)
		return
	}

	log.Printf("Successfully published comment event")
	sendSuccess(c, http.StatusCreated, comment)
}

func (h *CommentHandler) GetCommentsByPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("postId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid post ID format", err)
		return
	}

	comments, err := h.store.GetCommentsByPostID(postID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch comments", err)
		return
	}

	sendSuccess(c, http.StatusOK, comments)
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "User ID not found in context", nil)
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid comment ID format", err)
		return
	}

	comment, err := h.store.GetCommentByID(commentID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch comment", err)
		return
	}

	if comment == nil {
		sendError(c, http.StatusNotFound, "Comment not found", nil)
		return
	}

	parsedUserID, err := uuid.Parse(userID.(string))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	if comment.UserID != parsedUserID {
		sendError(c, http.StatusForbidden, "Not authorized to delete this comment", nil)
		return
	}

	if err := h.store.DeleteComment(commentID); err != nil {
		if err == sql.ErrNoRows {
			sendError(c, http.StatusNotFound, "Comment not found", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to delete comment", err)
		return
	}

	sendSuccess(c, http.StatusOK, map[string]string{"message": "Comment deleted successfully"})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			sendError(c, http.StatusUnauthorized, "X-User-ID header missing", nil)
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
} 