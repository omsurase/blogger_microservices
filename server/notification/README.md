# Notification Service

A background worker service that handles notifications for the blogging platform.

## Overview

The Notification Service is responsible for processing events from RabbitMQ and sending notifications to users. Currently, it handles new comment notifications by simulating email sending to post authors.

## Features

- Consumes NewCommentCreated events from RabbitMQ
- Simulates sending email notifications to post authors
- Implements robust error handling with message requeuing
- Graceful shutdown handling

## Technology Stack

- Go 1.21
- RabbitMQ (for event consumption)
- Docker

## Configuration

The service can be configured using environment variables:

```env
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
```

## Event Format

The service expects comment events in the following format:

```json
{
  "comment_id": "uuid",
  "post_id": "uuid",
  "author_id": "uuid",
  "user_id": "uuid",
  "content": "string",
  "created_at": "timestamp"
}
```

## Running the Service

### Using Docker

```bash
docker build -t notification-service .
docker run -e RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/ notification-service
```

### Using Docker Compose

The service is included in the project's docker-compose.yml file and can be started with:

```bash
docker-compose up notification-service
``` 