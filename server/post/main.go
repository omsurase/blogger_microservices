package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omsurase/blogger_microservices/server/post/internal/handlers"
	"github.com/omsurase/blogger_microservices/server/post/internal/store"
)

const (
	serviceName    = "post-service"
	retryAttempts  = 5
	retryDelay     = 5 * time.Second
	heartbeatDelay = 30 * time.Second
)

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "X-User-ID header missing"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func registerService() error {
	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		return fmt.Errorf("REGISTRY_URL environment variable is required")
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %v", err)
	}

	serviceAddress := fmt.Sprintf("http://%s:8080", hostname)
	registerURL := fmt.Sprintf("%s/register", registryURL)

	for i := 0; i < retryAttempts; i++ {
		payload := map[string]string{
			"name":    serviceName,
			"address": serviceAddress,
		}

		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal registration payload: %v", err)
		}

		resp, err := http.Post(registerURL, "application/json", bytes.NewBuffer(jsonPayload))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
				log.Printf("Successfully registered service with registry")
				return nil
			}
		}

		log.Printf("Failed to register service, attempt %d/%d: %v", i+1, retryAttempts, err)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("failed to register service after %d attempts", retryAttempts)
}

func startHeartbeat() {
	registryURL := os.Getenv("REGISTRY_URL")
	heartbeatURL := fmt.Sprintf("%s/heartbeat", registryURL)

	ticker := time.NewTicker(heartbeatDelay)
	go func() {
		for range ticker.C {
			payload := map[string]string{
				"name": serviceName,
			}

			jsonPayload, err := json.Marshal(payload)
			if err != nil {
				log.Printf("Failed to marshal heartbeat payload: %v", err)
				continue
			}

			resp, err := http.Post(heartbeatURL, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				log.Printf("Failed to send heartbeat: %v", err)
				continue
			}
			resp.Body.Close()
		}
	}()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pgStore, err := store.NewPostgresStore(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	redisStore, err := store.NewRedisStore(os.Getenv("REDIS_ADDR"), 1*time.Hour)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	handler := handlers.NewHandler(pgStore, redisStore)
	router := gin.Default()

	router.POST("/post/create", authMiddleware(), handler.CreatePost)
	router.PUT("/post/:id", authMiddleware(), handler.UpdatePost)
	router.DELETE("/post/:id", authMiddleware(), handler.DeletePost)
	router.GET("/post/:id", handler.GetPost)
	router.GET("/post/user/:id", handler.GetPostsByUser)
	router.GET("/post/tag/:tag", handler.GetPostsByTag)
	router.GET("/post", handler.GetAllPosts)

	// Health endpoint under /post prefix
	router.GET("/post/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"timestamp": time.Now(),
		})
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		if err := registerService(); err != nil {
			log.Printf("Failed to register service: %v", err)
		}
		startHeartbeat()
	}()

	go func() {
		log.Printf("Post service starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
} 