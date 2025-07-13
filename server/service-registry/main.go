package main

import (
	"os"

	"github.com/blogging-platform/service-registry/internal/handlers"
	"github.com/blogging-platform/service-registry/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisStore, err := store.NewRedisStore(redisAddr)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to Redis")
	}
	defer redisStore.Close()

	handler := handlers.NewHandler(redisStore, logger)

	router := gin.Default()

	router.POST("/register", handler.RegisterService)
	router.POST("/heartbeat", handler.Heartbeat)
	router.GET("/services", handler.GetServices)
	router.GET("/health", handler.Health)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.WithField("port", port).Info("Starting service registry")
	if err := router.Run(":" + port); err != nil {
		logger.WithError(err).Fatal("Failed to start server")
	}
} 