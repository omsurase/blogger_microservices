package handlers

import (
	"net/http"
	"strings"

	"github.com/blogging-platform/service-registry/internal/models"
	"github.com/blogging-platform/service-registry/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

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
	c.Header("Content-Type", "application/json")

	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse register request")
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorResponse{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format: " + err.Error(),
			},
		})
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Address) == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorResponse{
				Code:    "MISSING_FIELDS",
				Message: "Service name and address are required",
			},
		})
		return
	}

	service := &models.Service{
		Name:    req.Name,
		Address: req.Address,
	}

	if err := h.store.RegisterService(c.Request.Context(), service); err != nil {
		h.logger.WithError(err).Error("Failed to register service")
		
		switch err {
		case store.ErrServiceExists:
			c.JSON(http.StatusConflict, Response{
				Success: false,
				Error: &ErrorResponse{
					Code:    "SERVICE_EXISTS",
					Message: "A service with this name already exists",
				},
			})
		case store.ErrInvalidService:
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorResponse{
					Code:    "INVALID_SERVICE",
					Message: "Invalid service data provided",
				},
			})
		default:
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorResponse{
					Code:    "REGISTRATION_FAILED",
					Message: "Failed to register service: internal server error",
				},
			})
		}
		return
	}

	h.logger.WithFields(logrus.Fields{
		"service": service.Name,
		"address": service.Address,
	}).Info("Service registered successfully")

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data: gin.H{
			"message": "Service registered successfully",
			"service": service,
		},
	})
}

func (h *Handler) Heartbeat(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	var req models.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse heartbeat request")
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorResponse{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request format: " + err.Error(),
			},
		})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorResponse{
				Code:    "MISSING_SERVICE_NAME",
				Message: "Service name is required",
			},
		})
		return
	}

	if err := h.store.UpdateTTL(c.Request.Context(), req.Name); err != nil {
		h.logger.WithError(err).Error("Failed to update service TTL")

		switch err {
		case store.ErrServiceNotFound:
			c.JSON(http.StatusNotFound, Response{
				Success: false,
				Error: &ErrorResponse{
					Code:    "SERVICE_NOT_FOUND",
					Message: "Service not found",
				},
			})
		case store.ErrInvalidService:
			c.JSON(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorResponse{
					Code:    "INVALID_SERVICE",
					Message: "Invalid service name provided",
				},
			})
		default:
			c.JSON(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorResponse{
					Code:    "TTL_UPDATE_FAILED",
					Message: "Failed to update service TTL: internal server error",
				},
			})
		}
		return
	}

	h.logger.WithField("service", req.Name).Info("Service heartbeat received")
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"message": "Heartbeat received",
			"service": req.Name,
		},
	})
}

func (h *Handler) GetServices(c *gin.Context) {
	c.Header("Content-Type", "application/json")

	services, err := h.store.GetServices(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get services")
		c.JSON(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorResponse{
				Code:    "FETCH_FAILED",
				Message: "Failed to fetch services: internal server error",
			},
		})
		return
	}

	if len(services) == 0 {
		c.JSON(http.StatusOK, Response{
			Success: true,
			Data: gin.H{
				"services": []interface{}{},
				"message": "No services found",
			},
		})
		return
	}

	h.logger.WithField("count", len(services)).Info("Retrieved services")
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data: gin.H{
			"services": services,
			"count":    len(services),
		},
	})
} 