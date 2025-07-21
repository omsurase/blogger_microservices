package service

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"gopkg.in/mail.v2"
	"github.com/omsurase/blogger_microservices/server/notification/internal/models"
)

type NotificationService struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	fromEmail    string
	authServiceURL string
}

func NewNotificationService() *NotificationService {
	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if smtpPort == 0 {
		smtpPort = 587 // default SMTP port
	}

	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "http://auth-service:8080" // default URL for auth service in Docker
	}

	return &NotificationService{
		smtpHost:     os.Getenv("SMTP_HOST"),
		smtpPort:     smtpPort,
		smtpUsername: os.Getenv("SMTP_USERNAME"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
		fromEmail:    os.Getenv("FROM_EMAIL"),
		authServiceURL: authServiceURL,
	}
}

type UserResponse struct {
	Status int `json:"status"`
	Data   struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"data"`
}

func (s *NotificationService) getUserEmail(userID string) (string, error) {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "http://auth-service:8080"
	}

	url := fmt.Sprintf("%s/auth/users/%s", authServiceURL, userID)
	log.Printf("Fetching user details from: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch user: %v", err)
		return "", fmt.Errorf("failed to fetch user: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Auth service returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("failed to fetch user: %s", string(body))
	}

	log.Printf("Auth service response: %s", string(body))

	var userResp UserResponse
	if err := json.Unmarshal(body, &userResp); err != nil {
		log.Printf("Failed to decode user response: %v", err)
		return "", fmt.Errorf("failed to decode user response: %v", err)
	}

	log.Printf("Successfully fetched user details: %+v", userResp.Data)
	return userResp.Data.Email, nil
}

func (s *NotificationService) CreateNotification(event *models.CommentEvent) (*models.EmailNotification, error) {
	log.Printf("Creating notification for comment event: %+v", event)

	// Get the author's email from the auth service
	authorEmail, err := s.getUserEmail(event.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get author's email: %v", err)
	}

	notification := &models.EmailNotification{
		To:      authorEmail,
		Subject: "New Comment on Your Post",
		Body:    fmt.Sprintf("A new comment has been added to your post by user %s: %s", event.CommenterID, event.Content),
	}

	log.Printf("Sending notification: %+v", notification)
	return notification, nil
}

func (s *NotificationService) SendNotification(notification *models.EmailNotification) error {
	log.Printf("Attempting to send email via SMTP: %s:%d", s.smtpHost, s.smtpPort)

	m := mail.NewMessage()
	m.SetHeader("From", s.fromEmail)
	m.SetHeader("To", notification.To)
	m.SetHeader("Subject", notification.Subject)
	m.SetBody("text/plain", notification.Body)

	d := mail.NewDialer(s.smtpHost, s.smtpPort, s.smtpUsername, s.smtpPassword)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
} 