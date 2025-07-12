package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/post/internal/models"
	"github.com/omsurase/blogger_microservices/server/post/internal/store"
)

type Handler struct {
	pgStore    *store.PostgresStore
	redisStore *store.RedisStore
}

func NewHandler(pgStore *store.PostgresStore, redisStore *store.RedisStore) *Handler {
	return &Handler{
		pgStore:    pgStore,
		redisStore: redisStore,
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

func getUserIDFromToken(c *gin.Context) (uuid.UUID, error) {
	claims := c.MustGet("claims").(jwt.MapClaims)
	userID, err := uuid.Parse(claims["id"].(string))
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (h *Handler) CreatePost(c *gin.Context) {
	if c.ContentType() != "application/json" {
		sendError(c, http.StatusUnsupportedMediaType, "Content-Type must be application/json", nil)
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "user ID not found in context", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid user ID format", err)
		return
	}

	var req models.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	post := &models.Post{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     req.Title,
		Content:   req.Content,
		Tags:      req.Tags,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.pgStore.CreatePost(post); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create post", err)
		return
	}

	sendSuccess(c, http.StatusCreated, post)
}

func (h *Handler) GetPost(c *gin.Context) {
	postIDStr := c.Param("id")
	if postIDStr == "" {
		sendError(c, http.StatusBadRequest, "Post ID is required", nil)
		return
	}

	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid post ID format", err)
		return
	}

	post, err := h.pgStore.GetPost(postID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Post not found", err)
		return
	}

	sendSuccess(c, http.StatusOK, post)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "user ID not found in context", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid user ID format", err)
		return
	}

	postIDStr := c.Param("id")
	if postIDStr == "" {
		sendError(c, http.StatusBadRequest, "Post ID is required", nil)
		return
	}

	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid post ID format", err)
		return
	}

	post, err := h.pgStore.GetPost(postID)
	if err != nil {
		sendError(c, http.StatusNotFound, "Post not found", err)
		return
	}

	if post.UserID != userID {
		sendError(c, http.StatusForbidden, "Not authorized to update this post", nil)
		return
	}

	var req models.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	post.Title = req.Title
	post.Content = req.Content
	post.Tags = req.Tags
	post.UpdatedAt = time.Now()

	if err := h.pgStore.UpdatePost(post); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update post", err)
		return
	}

	sendSuccess(c, http.StatusOK, post)
}

func (h *Handler) DeletePost(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "user ID not found in context", nil)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid user ID format", err)
		return
	}

	postIDStr := c.Param("id")
	if postIDStr == "" {
		sendError(c, http.StatusBadRequest, "Post ID is required", nil)
		return
	}

	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid post ID format", err)
		return
	}

	if err := h.pgStore.DeletePost(postID, userID); err != nil {
		if err.Error() == "post not found or user not authorized" {
			sendError(c, http.StatusNotFound, "Post not found or not authorized to delete", nil)
			return
		}
		sendError(c, http.StatusInternalServerError, "Failed to delete post", err)
		return
	}

	sendSuccess(c, http.StatusOK, gin.H{"message": "Post deleted successfully"})
}

func (h *Handler) GetPostsByUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		sendError(c, http.StatusBadRequest, "Invalid page number", err)
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		sendError(c, http.StatusBadRequest, "Invalid page size (must be between 1 and 100)", err)
		return
	}

	posts, totalCount, err := h.pgStore.GetPostsByUser(userID, page, pageSize)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch posts", err)
		return
	}

	response := models.PaginatedPostsResponse{
		Posts:      make([]models.PostResponse, len(posts)),
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}

	for i, post := range posts {
		response.Posts[i] = models.PostResponse(post)
	}

	c.Header("Cache-Control", "public, max-age=60")
	sendSuccess(c, http.StatusOK, response)
}

func (h *Handler) GetPostsByTag(c *gin.Context) {
	tag := c.Param("tag")
	if tag == "" {
		sendError(c, http.StatusBadRequest, "Tag parameter is required", nil)
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		sendError(c, http.StatusBadRequest, "Invalid page number", err)
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		sendError(c, http.StatusBadRequest, "Invalid page size (must be between 1 and 100)", err)
		return
	}

	posts, totalCount, err := h.pgStore.GetPostsByTag(tag, page, pageSize)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch posts", err)
		return
	}

	response := models.PaginatedPostsResponse{
		Posts:      make([]models.PostResponse, len(posts)),
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}

	for i, post := range posts {
		response.Posts[i] = models.PostResponse(post)
	}

	c.Header("Cache-Control", "public, max-age=60")
	sendSuccess(c, http.StatusOK, response)
} 