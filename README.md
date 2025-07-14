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
- Go 1.24 or later

### Running the Project

To start all services:
```bash
make up
```

To stop all services:
```bash
make down
```

## API Gateway Endpoints

| Endpoint                   | Description                                      |
|---------------------------|--------------------------------------------------|
| `/health`                 | Returns the health status of the API Gateway.    |
| `/health/detailed`        | Returns detailed health information about the gateway and registered services. |
| `/api/v1/auth/login`      | User login endpoint.                             |
| `/api/v1/auth/signup`     | User signup/registration endpoint.               |
| `/api/v1/auth/register`   | Alternate user registration endpoint.            |
| `/api/v1/auth/health`     | Health check for Auth Service.                   |
| `/api/v1/auth/validate-token` | Validate an existing JWT token through Auth Service. |
| `/api/v1/auth/logout`            | User sign-out endpoint (token invalidation).     |
| `/api/v1/posts`           | Handles blog post creation, listing, etc.        |
| `/api/v1/posts/*`         | Handles specific post operations (CRUD).         |
| `/api/v1/posts/health`    | Health check for Post Service.                   |
| `/api/v1/comments/*`      | Handles comment operations for posts.            |
| `/api/v1/comments/health` | Health check for Comment Service.                |
| `/api/v1/profile/*`       | Handles user profile operations.                 |
| `/api/v1/profile/health`  | Health check for User Profile Service.           |
| `/service-registry/health`| Health check for Service Registry.               |

## Service Documentation

For more information about each service, refer to the following table:

| Service Name         | Documentation Link                                      |
|---------------------|--------------------------------------------------------|
| API Gateway         | -                                                      |
| Auth Service        | [Auth Service README](server/auth/README.md)           |
| User Profile Service| [User Profile Service README](server/user-profile/README.md) |
| Post Service        | [Post Service README](server/post/README.md)           |
| Comment Service     | [Comment Service README](server/comment/README.md)     |
| Notification Service| [Notification Service README](server/notification/README.md) |
| Service Registry    | [Service Registry README](server/service-registry/README.md) |

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

**Note:** Each service exposes a `/health` endpoint for health checks, accessible via the API Gateway as shown above.
