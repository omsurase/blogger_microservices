# Service Registry

A lightweight service registry for microservices, built with Go, Gin, and Redis. It allows services to register themselves, send heartbeats, and enables service discovery.

## Endpoints

### 1. Register a Service
- **URL:** `/register`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "name": "post-service",
    "address": "http://post-service:8083"
  }
  ```
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "message": "Service registered successfully"
    }
    ```

### 2. Heartbeat
- **URL:** `/heartbeat`
- **Method:** `POST`
- **Request Body:**
  ```json
  {
    "name": "post-service"
  }
  ```
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    {
      "message": "Heartbeat received"
    }
    ```

### 3. List Registered Services
- **URL:** `/services`
- **Method:** `GET`
- **Success Response:**
  - **Code:** 200
  - **Body:**
    ```json
    [
      {
        "name": "post-service",
        "address": "http://post-service:8083"
      }
      // ... other services
    ]
    ```

## Usage

- Register your service after startup using `/register`.
- Send periodic heartbeats (every 30-50 seconds) to `/heartbeat` to keep your service alive in the registry.
- Use `/services` to discover currently available services.

## Environment Variables
- `REDIS_ADDR`: Redis server address (default: `localhost:6379`)
- `PORT`: Port for the service registry (default: `8080`)

---