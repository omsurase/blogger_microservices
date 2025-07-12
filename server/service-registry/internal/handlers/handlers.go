package handlers

import (
	"net/http"

	"github.com/blogging-platform/service-registry/internal/models"
	"github.com/blogging-platform/service-registry/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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

func (h *Handler) RegisterService(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse register request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	service := &models.Service{
		Name:    req.Name,
		Address: req.Address,
	}

	if err := h.store.RegisterService(c.Request.Context(), service); err != nil {
		h.logger.WithError(err).Error("Failed to register service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register service"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"service": service.Name,
		"address": service.Address,
	}).Info("Service registered successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Service registered successfully"})
}

func (h *Handler) Heartbeat(c *gin.Context) {
	var req models.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse heartbeat request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.store.UpdateTTL(c.Request.Context(), req.Name); err != nil {
		h.logger.WithError(err).Error("Failed to update service TTL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service TTL"})
		return
	}

	h.logger.WithField("service", req.Name).Info("Service heartbeat received")
	c.JSON(http.StatusOK, gin.H{"message": "Heartbeat received"})
}

func (h *Handler) GetServices(c *gin.Context) {
	services, err := h.store.GetServices(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get services")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get services"})
		return
	}

	h.logger.WithField("count", len(services)).Info("Retrieved services")
	c.JSON(http.StatusOK, services)
} 