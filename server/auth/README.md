# Auth Service

Handles user authentication, registration, and JWT token management for the blogging platform.

## Endpoints

- **POST /signup**: Register a new user
- **POST /login**: Authenticate user and return JWT
- **GET /validate-token**: Validate JWT and return user info (protected)

## Technology Stack

- Go (Golang)
- Gin Web Framework
- PostgreSQL
- golang-jwt/jwt
- bcrypt

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string
- `JWT_SECRET_KEY`: Secret key for signing JWTs
- `REGISTRY_URL`: URL of the service registry

## Usage

- Register a user via `/signup` with email and password
- Login via `/login` to receive a JWT
- Access protected endpoints by including the JWT in the `Authorization` header as `Bearer <token>`

## Database Schema

- **users** table:
  - `id` (UUID, primary key)
  - `email` (unique)
  - `password_hash` (string)
  - `created_at` (timestamp)
  - `updated_at` (timestamp)

## Example Requests

### Register
```json
POST /signup
{
  "email": "user@example.com",
  "password": "password123"
}
```

### Login
```json
POST /login
{
  "email": "user@example.com",
  "password": "password123"
}
```

### Validate Token
```
GET /validate-token
Authorization: Bearer <token>
```

---

- The service registers itself with the Service Registry and sends periodic heartbeats for service discovery.
- Passwords are securely hashed using bcrypt.
- JWTs include user ID and email as claims. 