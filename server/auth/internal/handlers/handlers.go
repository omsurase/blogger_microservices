package handlers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omsurase/blogger_microservices/server/auth/internal/models"
	"github.com/omsurase/blogger_microservices/server/auth/internal/store"
	"golang.org/x/crypto/bcrypt"
	"github.com/redis/go-redis/v9"
)

type AuthHandler struct {
	store *store.PostgresStore
	redis *redis.Client
}

func NewAuthHandler(store *store.PostgresStore, redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{store: store, redis: redisClient}
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

func (h *AuthHandler) SignUp(c *gin.Context) {
	if c.ContentType() != "application/json" {
		sendError(c, http.StatusUnsupportedMediaType, "Content-Type must be application/json", nil)
		return
	}

	var req models.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	existingUser, _ := h.store.GetUserByEmail(req.Email)
	if existingUser != nil {
		sendError(c, http.StatusConflict, "User with this email already exists", nil)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to process password", err)
		return
	}

	user, err := h.store.CreateUser(req.Email, string(hashedPassword))
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to create user", err)
		return
	}

	sendSuccess(c, http.StatusCreated, user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	if c.ContentType() != "application/json" {
		sendError(c, http.StatusUnsupportedMediaType, "Content-Type must be application/json", nil)
		return
	}

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	user, err := h.store.GetUserByEmail(req.Email)
	if err != nil {
		sendError(c, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		sendError(c, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	iat := time.Now().Unix()
	jti := uuid.NewString()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     iat,
		"jti":     jti,
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to generate authentication token", err)
		return
	}

	sendSuccess(c, http.StatusOK, models.TokenResponse{Token: tokenString})
}

func (h *AuthHandler) ValidateToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		sendError(c, http.StatusUnauthorized, "Authorization header is required", nil)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		sendError(c, http.StatusUnauthorized, "Invalid token format. Use 'Bearer <token>'", nil)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})

	if err != nil || !token.Valid {
		sendError(c, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		sendError(c, http.StatusInternalServerError, "Failed to parse token claims", nil)
		return
	}

	sendSuccess(c, http.StatusOK, models.ValidationResponse{
		UserID: claims["user_id"].(string),
		Email:  claims["email"].(string),
	})
}

func (h *AuthHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		sendError(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	user, err := h.store.GetUserByID(userID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to fetch user", err)
		return
	}

	if user == nil {
		sendError(c, http.StatusNotFound, "User not found", nil)
		return
	}

	sendSuccess(c, http.StatusOK, user)
}

// Logout invalidates the provided JWT by blacklisting its jti in Redis.
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		sendError(c, http.StatusUnauthorized, "Authorization header is required", nil)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		sendError(c, http.StatusUnauthorized, "Invalid token format. Use 'Bearer <token>'", nil)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})

	if err != nil || !token.Valid {
		sendError(c, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		sendError(c, http.StatusInternalServerError, "Failed to parse token claims", nil)
		return
	}

	jtiClaim, ok := claims["jti"].(string)
	if !ok || jtiClaim == "" {
		sendError(c, http.StatusBadRequest, "Token missing jti claim", nil)
		return
	}

	expClaim, ok := claims["exp"].(float64)
	if !ok {
		sendError(c, http.StatusBadRequest, "Token missing exp claim", nil)
		return
	}

	// Calculate TTL as remaining seconds until expiration.
	ttlSeconds := int64(expClaim) - time.Now().Unix()
	if ttlSeconds <= 0 {
		// Token already expired, nothing to blacklist.
		sendSuccess(c, http.StatusOK, gin.H{"message": "Token already expired"})
		return
	}

	ctx := context.Background()
	err = h.redis.Set(ctx, jtiClaim, "blacklisted", time.Duration(ttlSeconds)*time.Second).Err()
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to blacklist token", err)
		return
	}

	sendSuccess(c, http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			sendError(c, http.StatusUnauthorized, "Authorization header is required", nil)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			sendError(c, http.StatusUnauthorized, "Invalid token format. Use 'Bearer <token>'", nil)
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET_KEY")), nil
		})

		if err != nil || !token.Valid {
			sendError(c, http.StatusUnauthorized, "Invalid or expired token", err)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			sendError(c, http.StatusInternalServerError, "Failed to parse token claims", nil)
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("email", claims["email"])
		c.Next()
	}
} 