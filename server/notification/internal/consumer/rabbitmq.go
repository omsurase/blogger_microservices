package consumer

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/omsurase/blogger_microservices/server/notification/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(*models.CommentEvent) error

type RabbitMQConsumer struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	exchangeName  string
	queueName     string
	handleMessage MessageHandler
}

func NewRabbitMQConsumer(url string, handler MessageHandler) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %v", err)
	}

	consumer := &RabbitMQConsumer{
		conn:          conn,
		channel:       ch,
		exchangeName:  "events_exchange",
		queueName:     "notification_queue",
		handleMessage: handler,
	}

	// Setup exchange and queue
	if err := consumer.setup(); err != nil {
		consumer.Close()
		return nil, err
	}

	return consumer, nil
}

func (c *RabbitMQConsumer) setup() error {
	// Declare exchange
	err := c.channel.ExchangeDeclare(
		c.exchangeName,
		"fanout",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}

	// Declare queue
	queue, err := c.channel.QueueDeclare(
		c.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %v", err)
	}

	// Bind queue to exchange
	err = c.channel.QueueBind(
		queue.Name,
		"",            // routing key
		c.exchangeName,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %v", err)
	}

	return nil
}

func (c *RabbitMQConsumer) Start() error {
	msgs, err := c.channel.Consume(
		c.queueName,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %v", err)
	}

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			var event models.CommentEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				continue
			}

			log.Printf("Received message: %s", string(msg.Body))
			log.Printf("Processing comment event: %+v", event)

			if err := c.handleMessage(&event); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}()

	log.Printf("Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}

func (c *RabbitMQConsumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
} 