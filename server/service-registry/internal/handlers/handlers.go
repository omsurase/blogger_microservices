package handlers

import (
	"net/http"
	"strings"

	"github.com/blogging-platform/service-registry/internal/models"
	"github.com/blogging-platform/service-registry/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"errors"
)

type Handler struct {
	store  *store.RedisStore
	logger *logrus.Logger
}

func NewHandler(store *store.RedisStore, logger *logrus.Logger) *Handler {
	return &Handler{
		store:  store,
		logger: logger,
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

func (h *Handler) RegisterService(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse register request")
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Address) == "" {
		sendError(c, http.StatusBadRequest, "Service name and address are required", nil)
		return
	}

	service := &models.Service{
		Name:    req.Name,
		Address: req.Address,
	}

	if err := h.store.RegisterService(c.Request.Context(), service); err != nil {
		h.logger.WithError(err).Error("Failed to register service")

		// If the service already exists, refresh the TTL and treat it as a successful registration.
		if errors.Is(err, store.ErrServiceExists) {
			if ttlErr := h.store.UpdateTTL(c.Request.Context(), service.Name); ttlErr != nil {
				h.logger.WithError(ttlErr).Error("Failed to refresh TTL for existing service")
				sendError(c, http.StatusInternalServerError, "Failed to refresh TTL for existing service", ttlErr)
				return
			}
			sendSuccess(c, http.StatusOK, map[string]string{"name": service.Name, "status": "already registered"})
			return
		}

		sendError(c, http.StatusInternalServerError, "Failed to register service", err)
		return
	}

	sendSuccess(c, http.StatusCreated, service)
}

func (h *Handler) GetServices(c *gin.Context) {
	services, err := h.store.GetServices(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get services")
		sendError(c, http.StatusInternalServerError, "Failed to get services", err)
		return
	}

	sendSuccess(c, http.StatusOK, services)
}

func (h *Handler) Heartbeat(c *gin.Context) {
	var req models.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse heartbeat request")
		sendError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		sendError(c, http.StatusBadRequest, "Service name is required", nil)
		return
	}

	if err := h.store.UpdateTTL(c.Request.Context(), req.Name); err != nil {
		h.logger.WithError(err).Error("Failed to update service TTL")
		
		switch err {
		case store.ErrServiceNotFound:
			sendError(c, http.StatusNotFound, "Service not found", err)
		default:
			sendError(c, http.StatusInternalServerError, "Failed to update service TTL", err)
		}
		return
	}

	sendSuccess(c, http.StatusOK, map[string]string{"name": req.Name})
} 