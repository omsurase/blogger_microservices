# Post Service

Handles blog post creation, retrieval, and management for the blogging platform.

## Endpoints

### 1. Create Post
- **URL:** `/post/create`
- **Method:** `POST`
- **Auth Required:** Yes
- **Request Body:**
  ```json
  {
    "title": "My First Blog Post",
    "content": "This is the content of my blog post",
    "tags": ["tech", "golang"]
  }
  ```
- **Success Response:**
  - **Code:** 201
  - **Body:**
    ```json
    {
      "id": "<post_id>",
      "user_id": "<user_id>",
      "title": "My First Blog Post",
      "content": "This is the content of my blog post",
      "tags": ["tech", "golang"],
      "created_at": "2024-02-20T10:00:00Z",
      "updated_at": "2024-02-20T10:00:00Z"
    }
    ```

### 2. Get Post
- **URL:** `/post/:id`
- **Method:** `GET`
- **Auth Required:** No
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "id": "<post_id>",
      "user_id": "<user_id>",
      "title": "My First Blog Post",
      "content": "This is the content of my blog post",
      "tags": ["tech", "golang"],
      "created_at": "2024-02-20T10:00:00Z",
      "updated_at": "2024-02-20T10:00:00Z"
    }
    ```

### 3. Update Post
- **URL:** `/post/:id`
- **Method:** `PUT`
- **Auth Required:** Yes (only post author)
- **Request Body:**
  ```json
  {
    "title": "Updated Title",
    "content": "Updated content",
    "tags": ["tech", "golang", "updated"]
  }
  ```
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "id": "<post_id>",
      "user_id": "<user_id>",
      "title": "Updated Title",
      "content": "Updated content",
      "tags": ["tech", "golang", "updated"],
      "created_at": "2024-02-20T10:00:00Z",
      "updated_at": "2024-02-20T10:30:00Z"
    }
    ```

### 4. Delete Post
- **URL:** `/post/:id`
- **Method:** `DELETE`
- **Auth Required:** Yes (only post author)
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "message": "post deleted successfully"
    }
    ```

### 5. Get Posts by User
- **URL:** `/post/user/:id`
- **Method:** `GET`
- **Auth Required:** No
- **Query Parameters:**
  - `page` (optional, default: 1)
  - `page_size` (optional, default: 10)
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "posts": [
        {
          "id": "<post_id>",
          "user_id": "<user_id>",
          "title": "Post Title",
          "content": "Post content",
          "tags": ["tech"],
          "created_at": "2024-02-20T10:00:00Z",
          "updated_at": "2024-02-20T10:00:00Z"
        }
      ],
      "total_count": 1,
      "page": 1,
      "page_size": 10
    }
    ```

### 6. Get Posts by Tag
- **URL:** `/post/tag/:tag`
- **Method:** `GET`
- **Auth Required:** No
- **Query Parameters:**
  - `page` (optional, default: 1)
  - `page_size` (optional, default: 10)
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "posts": [
        {
          "id": "<post_id>",
          "user_id": "<user_id>",
          "title": "Post Title",
          "content": "Post content",
          "tags": ["tech"],
          "created_at": "2024-02-20T10:00:00Z",
          "updated_at": "2024-02-20T10:00:00Z"
        }
      ],
      "total_count": 1,
      "page": 1,
      "page_size": 10
    }
    ```

## Environment Variables
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string
- `JWT_SECRET_KEY`: Secret key for validating JWTs
- `REGISTRY_URL`: URL of the service registry
- `PORT`: Server port (default: 8080)

## Technology Stack
- Go (Golang)
- Gin Web Framework
- PostgreSQL
- Redis
- JWT for authentication

## Database Schema
- **posts** table:
  - `id` (UUID, primary key)
  - `user_id` (UUID, foreign key)
  - `title` (text)
  - `content` (text)
  - `tags` (text array)
  - `created_at` (timestamp with timezone)
  - `updated_at` (timestamp with timezone)

## Caching Strategy
- Post content is cached in Redis with a 1-hour TTL
- Cache is invalidated on post updates and deletes
- Cache is populated on first read (cache-aside pattern)

## Service Registration
- The service registers itself with the Service Registry on startup
- Sends heartbeat every 30 seconds to maintain registration
- Provides metadata about available endpoints 