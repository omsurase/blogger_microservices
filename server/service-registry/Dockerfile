FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o service-registry

FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/service-registry .

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./service-registry"] 