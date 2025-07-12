package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/post/internal/handlers"
	"github.com/omsurase/blogger_microservices/server/post/internal/store"
)

type ServiceRegistration struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format. Use 'Bearer <token>'"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("email", claims["email"])
		c.Next()
	}
}

func registerService(serviceID string, port string) error {
	registration := ServiceRegistration{
		Name:    "post-service",
		Address: fmt.Sprintf("http://post:%s", port),
	}

	jsonData, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		registryURL = "http://service-registry:8080"
	}

	resp, err := http.Post(fmt.Sprintf("%s/register", registryURL), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register service: %d", resp.StatusCode)
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			heartbeat := ServiceRegistration{
				Name: "post-service",
			}
			jsonData, err := json.Marshal(heartbeat)
			if err != nil {
				log.Printf("Error marshaling heartbeat: %v", err)
				continue
			}
			resp, err := http.Post(fmt.Sprintf("%s/heartbeat", registryURL), "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("Error sending heartbeat: %v", err)
				continue
			}
			resp.Body.Close()
		}
	}()

	return nil
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

	serviceID := uuid.New().String()
	if err := registerService(serviceID, port); err != nil {
		log.Printf("Failed to register service: %v", err)
	}

	router := gin.Default()

	router.POST("/post/create", authMiddleware(), handler.CreatePost)
	router.PUT("/post/:id", authMiddleware(), handler.UpdatePost)
	router.DELETE("/post/:id", authMiddleware(), handler.DeletePost)
	router.GET("/post/:id", handler.GetPost)
	router.GET("/post/user/:id", handler.GetPostsByUser)
	router.GET("/post/tag/:tag", handler.GetPostsByTag)

	log.Printf("Post service starting on port %s", port)
	if err := router.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
} 