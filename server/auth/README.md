# Auth Service

Handles user authentication, registration, and JWT token management for the blogging platform.

## Endpoints

### 1. Register a User
- **URL:** `/signup`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "message": "User registered successfully"
    }
    ```

### 2. Login
- **URL:** `/login`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "token": "<jwt_token>"
    }
    ```

### 3. Validate Token
- **URL:** `/validate-token`
- **Method:** `GET`
- **Headers:**
  - `Authorization: Bearer <token>`
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "id": "<user_id>",
      "email": "user@example.com"
    }
    ```

### 4. Logout (Sign-Out)
- **URL:** `/logout`
- **Method:** `POST`
- **Headers:**
  - `Authorization: Bearer <token>`
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "message": "Successfully logged out"
    }
    ```

## Usage

- Register a user after startup using `/signup`.
- Login via `/login` to receive a JWT.
- Access protected endpoints by including the JWT in the `Authorization` header as `Bearer <token>`.

## Environment Variables
- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET_KEY`: Secret key for signing JWTs
- `REGISTRY_URL`: URL of the service registry

## Technology Stack
- Go (Golang)
- Gin Web Framework
- PostgreSQL
- golang-jwt/jwt
- bcrypt

## Database Schema
- **users** table:
  - `id` (UUID, primary key)
  - `email` (unique)
  - `password_hash` (string)
  - `created_at` (timestamp)
  - `updated_at` (timestamp)

---
- The service registers itself with the Service Registry and sends periodic heartbeats for service discovery.
- Passwords are securely hashed using bcrypt.
- JWTs include user ID and email as claims. 