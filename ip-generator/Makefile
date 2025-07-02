.PHONY: build run test clean docker-build docker-run docker-compose-up docker-compose-down help

# Variables
BINARY_NAME=ip-generator
MAIN_PATH=cmd/server/main.go
DOCKER_IMAGE=ip-generator

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	go run $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	go clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 $(DOCKER_IMAGE)

# Docker Compose up
docker-compose-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

# Docker Compose down
docker-compose-down:
	@echo "Stopping services with Docker Compose..."
	docker-compose down

# Docker Compose logs
docker-compose-logs:
	docker-compose logs -f

# Generate go.sum
go-sum:
	go mod tidy
	go mod download

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  run                - Run the application"
	@echo "  test               - Run tests"
	@echo "  clean              - Clean build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  docker-compose-up  - Start services with Docker Compose"
	@echo "  docker-compose-down- Stop services with Docker Compose"
	@echo "  docker-compose-logs- View Docker Compose logs"
	@echo "  go-sum             - Generate go.sum"
	@echo "  help               - Show this help message" 