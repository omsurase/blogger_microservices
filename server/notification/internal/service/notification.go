package service

import (
	"encoding/json"
	"fmt"
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
		authServiceURL = "http://auth:8080" // default URL for auth service in Docker
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
	url := fmt.Sprintf("%s/api/users/%s", s.authServiceURL, userID)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get user details: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user details: status code %d", resp.StatusCode)
	}

	var userResp UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return "", fmt.Errorf("failed to decode user response: %v", err)
	}

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