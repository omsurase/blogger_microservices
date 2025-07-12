package publisher

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/omsurase/blogger_microservices/server/comment/internal/models"
	"github.com/streadway/amqp"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(url string) (*RabbitMQPublisher, error) {
	log.Printf("Connecting to RabbitMQ at %s", url)
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	log.Printf("Successfully connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Printf("Failed to open channel: %v", err)
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}
	log.Printf("Successfully opened channel")

	log.Printf("Declaring events_exchange")
	err = ch.ExchangeDeclare(
		"events_exchange",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Printf("Failed to declare exchange: %v", err)
		return nil, fmt.Errorf("failed to declare exchange: %v", err)
	}
	log.Printf("Successfully declared events_exchange")

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
	}, nil
}

func (p *RabbitMQPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}

func (p *RabbitMQPublisher) PublishNewComment(event *models.CommentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal event: %v", err)
		return fmt.Errorf("failed to marshal event: %v", err)
	}

	log.Printf("Publishing message to events_exchange: %s", string(body))
	err = p.channel.Publish(
		"events_exchange",
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:       body,
			Timestamp:  time.Now(),
		},
	)
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return fmt.Errorf("failed to publish message: %v", err)
	}
	log.Printf("Successfully published message to events_exchange")
	return nil
}

func (p *RabbitMQPublisher) Reconnect(url string) error {
	log.Printf("Attempting to reconnect to RabbitMQ at %s", url)
	if p.conn != nil {
		p.conn.Close()
	}
	if p.channel != nil {
		p.channel.Close()
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("Failed to reconnect to RabbitMQ: %v", err)
		return fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}
	log.Printf("Successfully reconnected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Printf("Failed to open channel after reconnect: %v", err)
		return fmt.Errorf("failed to open channel: %v", err)
	}
	log.Printf("Successfully opened channel after reconnect")

	log.Printf("Declaring events_exchange after reconnect")
	err = ch.ExchangeDeclare(
		"events_exchange",
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Printf("Failed to declare exchange after reconnect: %v", err)
		return fmt.Errorf("failed to declare exchange: %v", err)
	}
	log.Printf("Successfully declared events_exchange after reconnect")

	p.conn = conn
	p.channel = ch
	return nil
} 