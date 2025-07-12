# User Profile Service

The User Profile Service is responsible for managing user profile information in the blogging platform. It handles operations related to user profiles, such as retrieving and updating profile information.

## Features

- Get user profile by ID
- Update user profile (authenticated)
- Automatic profile creation for new users
- JWT-based authentication
- Service registration with Service Registry
- Heartbeat mechanism for service health monitoring

## API Endpoints

### GET /profile/:id
Fetches the profile for a given user ID.

**Request:**
```
GET /profile/123e4567-e89b-12d3-a456-426614174000
```

**Response:**
```json
{
    "status": 200,
    "data": {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "user_id": "123e4567-e89b-12d3-a456-426614174000",
        "bio": "Software Engineer",
        "avatar_url": "https://example.com/avatar.jpg",
        "twitter_url": "https://twitter.com/username",
        "linkedin_url": "https://linkedin.com/in/username",
        "github_url": "https://github.com/username",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }
}
```

### PUT /profile/update
Updates the profile of the authenticated user.

**Request:**
```
PUT /profile/update
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
    "bio": "Software Engineer",
    "avatar_url": "https://example.com/avatar.jpg",
    "twitter_url": "https://twitter.com/username",
    "linkedin_url": "https://linkedin.com/in/username",
    "github_url": "https://github.com/username"
}
```

**Response:**
```json
{
    "status": 200,
    "data": {
        "id": "123e4567-e89b-12d3-a456-426614174000",
        "user_id": "123e4567-e89b-12d3-a456-426614174000",
        "bio": "Software Engineer",
        "avatar_url": "https://example.com/avatar.jpg",
        "twitter_url": "https://twitter.com/username",
        "linkedin_url": "https://linkedin.com/in/username",
        "github_url": "https://github.com/username",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }
}
```

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `REGISTRY_URL`: Service Registry URL
- `JWT_SECRET_KEY`: Secret key for JWT validation

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS profiles (
    id UUID PRIMARY KEY,
    user_id UUID UNIQUE NOT NULL,
    bio TEXT,
    avatar_url TEXT,
    twitter_url TEXT,
    linkedin_url TEXT,
    github_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
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
   docker build -t user-profile-service .
   docker run -p 8080:8080 user-profile-service
   ``` 