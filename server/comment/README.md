# Comment Service

The Comment Service is responsible for managing comments on blog posts in the blogging platform. It handles CRUD operations for comments and publishes events when new comments are created.

## Features

- Create comments on posts
- Retrieve comments for a specific post
- Delete comments (with authorization)
- Event publishing to RabbitMQ
- Service registration with Service Registry
- Heartbeat mechanism for service health monitoring

## API Endpoints

### POST /comment/create
Creates a new comment on a post.

**Request:**
```
POST /comment/create
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
    "post_id": "123e4567-e89b-12d3-a456-426614174000",
    "content": "This is a great post!"
}
```

**Response:**
```json
{
    "status": 201,
    "data": {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "post_id": "123e4567-e89b-12d3-a456-426614174000",
        "user_id": "123e4567-e89b-12d3-a456-426614174000",
        "content": "This is a great post!",
        "created_at": "2024-01-01T00:00:00Z"
    }
}
```

### GET /comment/post/:postId
Retrieves all comments for a specific post.

**Request:**
```
GET /comment/post/123e4567-e89b-12d3-a456-426614174000
```

**Response:**
```json
{
    "status": 200,
    "data": [
        {
            "id": "123e4567-e89b-12d3-a456-426614174000",
            "post_id": "123e4567-e89b-12d3-a456-426614174000",
            "user_id": "123e4567-e89b-12d3-a456-426614174000",
            "content": "This is a great post!",
            "created_at": "2024-01-01T00:00:00Z"
        }
    ]
}
```

### DELETE /comment/:id
Deletes a comment. Only the comment author can delete their comments.

**Request:**
```
DELETE /comment/123e4567-e89b-12d3-a456-426614174000
Authorization: Bearer <jwt_token>
```

**Response:**
```json
{
    "status": 200,
    "data": {
        "message": "Comment deleted successfully"
    }
}
```

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `RABBITMQ_URL`: RabbitMQ connection string
- `REGISTRY_URL`: Service Registry URL
- `JWT_SECRET_KEY`: Secret key for JWT validation

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL,
    user_id UUID NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## Event Publishing

The service publishes a `NewCommentCreated` event to RabbitMQ when a new comment is created. The event payload includes:

```json
{
    "comment_id": "123e4567-e89b-12d3-a456-426614174000",
    "post_id": "123e4567-e89b-12d3-a456-426614174000",
    "commenter_id": "123e4567-e89b-12d3-a456-426614174000",
    "content": "This is a great post!",
    "created_at": "2024-01-01T00:00:00Z"
}
```

## Running the Service

1. Set the required environment variables
2. Run with Go:
   ```bash
   go mod download
   go run main.go
   ```
3. Or using Docker:
   ```bash
   docker build -t comment-service .
   docker run -p 8080:8080 comment-service
   ``` 