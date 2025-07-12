# Blogging Platform Microservices

A modern, scalable blogging platform built using microservices architecture.

## Services

- **API Gateway**: Entry point for all client requests
- **Auth Service**: Handles user authentication and authorization
- **User Profile Service**: Manages user profiles and preferences
- **Post Service**: Handles blog post creation, updates, and deletion
- **Comment Service**: Manages comments on blog posts
- **Notification Service**: Handles notifications and alerts
- **Service Registry**: Maintains registry of active services

## Technology Stack

- Go (Golang)
- Redis
- Docker & Docker Compose
- Gin Web Framework

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.21 or later

### Running the Project

To start all services:
```bash
make up
```

To stop all services:
```bash
make down
```

## Service Registry

The Service Registry is a critical component that maintains a list of all active services in the system. It provides:

- Service registration
- Health check/heartbeat mechanism
- Service discovery

### Endpoints

- `POST /register`: Register a new service
- `POST /heartbeat`: Update service TTL
- `GET /services`: List all active services

## Project Structure

```
blogging-platform/
├── client/
├── server/
│   ├── api-gateway/
│   ├── auth/
│   ├── user-profile/
│   ├── post/
│   ├── comment/
│   ├── notification/
│   ├── service-registry/
│   └── shared/
├── docker-compose.yml
├── Makefile
└── README.md
```