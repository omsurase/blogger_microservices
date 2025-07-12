.PHONY: up down logs clean

# Start all services
up:
	docker compose up --build -d

# Stop all services
down:
	docker compose down

# View logs
logs:
	docker compose logs -f

# Clean up volumes
clean:
	docker compose down -v 