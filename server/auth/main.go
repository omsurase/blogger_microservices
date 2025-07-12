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
)

const (
	serviceName    = "auth-service"
	retryAttempts  = 5
	retryDelay     = 5 * time.Second
	heartbeatDelay = 30 * time.Second
)

func main() {
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

	authHandler := handlers.NewAuthHandler(store)
	router := gin.Default()

	router.POST("/signup", authHandler.SignUp)
	router.POST("/login", authHandler.Login)
	router.GET("/validate-token", handlers.AuthMiddleware(), authHandler.ValidateToken)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := registerService(); err != nil {
			log.Printf("Failed to register service: %v", err)
		}
		startHeartbeat()
	}()

	go func() {
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

	serviceAddress := fmt.Sprintf("http://%s:8080", serviceName)
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
			if resp.StatusCode == http.StatusOK {
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