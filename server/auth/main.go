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
	"github.com/omsurase/blogger_microservices/server/auth/internal/handlers"
	"github.com/omsurase/blogger_microservices/server/auth/internal/store"
	"github.com/redis/go-redis/v9"
)

const (
	serviceName    = "auth-service"
	retryAttempts  = 5
	retryDelay     = 5 * time.Second
	heartbeatDelay = 30 * time.Second
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	store, err := store.NewPostgresStore(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	if err := store.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Redis client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// simple ping to confirm connectivity (non-fatal if fails, but log)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Printf("Warning: failed to connect to Redis at %s: %v", redisAddr, err)
	}

	authHandler := handlers.NewAuthHandler(store, redisClient)
	router := gin.Default()

	// Add health check endpoint under auth prefix
	router.GET("/auth/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	router.POST("/auth/signup", authHandler.SignUp)
	router.POST("/auth/login", authHandler.Login)
	router.GET("/auth/validate-token", handlers.AuthMiddleware(), authHandler.ValidateToken)
	router.POST("/auth/logout", handlers.AuthMiddleware(), authHandler.Logout)
	router.GET("/auth/users/:id", authHandler.GetUserByID)

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
		log.Printf("Auth service starting on port %s", port)
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