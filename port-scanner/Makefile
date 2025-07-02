# Port Scanner Microservice Makefile

.PHONY: help build run test clean docker-build docker-run docker-stop lint format

# Variables
BINARY_NAME=port-scanner
DOCKER_IMAGE=port-scanner
DOCKER_TAG=latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

run: ## Run the application locally
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -f coverage.html

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-stop: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-logs: ## Show Docker Compose logs
	docker-compose logs -f

docker-clean: ## Clean Docker resources
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

format: ## Format code
	@echo "Formatting code..."
	go fmt ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

dev: ## Run in development mode with hot reload
	@echo "Running in development mode..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Development shortcuts
dev-setup: install-tools deps ## Setup development environment
	@echo "Development environment setup complete!"

# Production shortcuts
prod-build: clean build ## Build for production
	@echo "Production build complete!"

prod-run: docker-build docker-run ## Deploy to production
	@echo "Production deployment complete!"

# Monitoring and debugging
status: ## Check service status
	@echo "Checking service status..."
	@curl -s http://localhost:8081/api/v1/health || echo "Service not responding"
	@echo ""
	@curl -s http://localhost:8081/api/v1/stats || echo "Stats not available"

logs: ## Show application logs
	@echo "Showing application logs..."
	docker-compose logs -f port-scanner

# Queue management
queue-status: ## Check RabbitMQ status
	@echo "Checking RabbitMQ status..."
	@curl -s -u guest:guest http://localhost:15672/api/overview || echo "RabbitMQ management not available"

# Testing utilities
test-scan: ## Test scan a single IP
	@echo "Testing scan functionality..."
	@curl -X POST http://localhost:8081/api/v1/scan \
		-H "Content-Type: application/json" \
		-d '{"ip": "8.8.8.8", "ports": [80, 443]}' || echo "Test failed"

test-batch: ## Test batch scan
	@echo "Testing batch scan functionality..."
	@curl -X POST http://localhost:8081/api/v1/scan/batch \
		-H "Content-Type: application/json" \
		-d '{"ips": ["8.8.8.8", "1.1.1.1"], "ports": [80, 443]}' || echo "Test failed" 