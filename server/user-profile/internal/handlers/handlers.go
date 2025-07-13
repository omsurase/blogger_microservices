package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/user-profile/internal/models"
	"github.com/omsurase/blogger_microservices/server/user-profile/internal/store"
)

type ProfileHandler struct {
	store *store.PostgresStore
}

func NewProfileHandler(store *store.PostgresStore) *ProfileHandler {
	return &ProfileHandler{store: store}
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

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	profile, err := h.store.GetProfileByUserID(userID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch profile", err)
		return
	}

	if profile == nil {
		profile, err = h.store.CreateProfile(userID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create profile", err)
			return
		}
	}

	sendSuccess(c, http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		sendError(c, http.StatusUnauthorized, "User ID not found in context", nil)
		return
	}

	if c.ContentType() != "application/json" {
		sendError(c, http.StatusUnsupportedMediaType, "Content-Type must be application/json", nil)
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	parsedUserID, err := uuid.Parse(userID.(string))
	if err != nil {
		sendError(c, http.StatusBadRequest, "Invalid user ID format", err)
		return
	}

	profile, err := h.store.GetProfileByUserID(parsedUserID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch profile", err)
		return
	}

	if profile == nil {
		profile, err = h.store.CreateProfile(parsedUserID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to create profile", err)
			return
		}
	}

	updatedProfile, err := h.store.UpdateProfile(parsedUserID, &req)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to update profile", err)
		return
	}

	sendSuccess(c, http.StatusOK, updatedProfile)
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