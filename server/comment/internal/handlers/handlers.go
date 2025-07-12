package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
		CommenterID: comment.UserID.String(),
		Content:     comment.Content,
		CreatedAt:   comment.CreatedAt,
	}

	if err := h.publisher.PublishNewComment(event); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to publish comment event", err)
		return
	}

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
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			sendError(c, http.StatusUnauthorized, "Authorization header is required", nil)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			sendError(c, http.StatusUnauthorized, "Invalid token format. Use 'Bearer <token>'", nil)
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			sendError(c, http.StatusUnauthorized, "Invalid or expired token", err)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			sendError(c, http.StatusInternalServerError, "Failed to parse token claims", nil)
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("email", claims["email"])
		c.Next()
	}
} 