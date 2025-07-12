package main

import (
	"log"
	"os"

	"github.com/omsurase/blogger_microservices/server/notification/internal/consumer"
	"github.com/omsurase/blogger_microservices/server/notification/internal/service"
	"github.com/omsurase/blogger_microservices/server/notification/internal/models"
)

func main() {
	// Initialize notification service
	notificationService := service.NewNotificationService()

	// Initialize RabbitMQ consumer
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://guest:guest@localhost:5672/"
	}

	consumer, err := consumer.NewRabbitMQConsumer(rabbitMQURL, func(event *models.CommentEvent) error {
		notification, err := notificationService.CreateNotification(event)
		if err != nil {
			log.Printf("Error creating notification: %v", err)
			return err
		}

		if err := notificationService.SendNotification(notification); err != nil {
			log.Printf("Error sending notification: %v", err)
			return err
		}

		log.Printf("Successfully sent notification")
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ consumer: %v", err)
	}

	// Start consuming messages
	if err := consumer.Start(); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
} 