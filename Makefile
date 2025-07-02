# Solomon Microservices Platform - Makefile
# Gerencia todos os serviços: RabbitMQ, MongoDB, IP Generator, Port Scanner

.PHONY: help build up down logs status clean test
.PHONY: up-rabbitmq up-mongodb up-ip-generator up-port-scanner
.PHONY: down-rabbitmq down-mongodb down-ip-generator down-port-scanner
.PHONY: logs-rabbitmq logs-mongodb logs-ip-generator logs-port-scanner
.PHONY: mongodb-shell mongodb-backup mongodb-restore
.PHONY: build-ip-generator build-port-scanner

# Variáveis
COMPOSE_FILE = docker-compose.yml
PROJECT_NAME = solomon

# Comandos principais
help: ## Mostra esta ajuda
	@echo "Solomon Microservices Platform - Comandos disponíveis:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: build-ip-generator build-port-scanner ## Build de todos os serviços

build-ip-generator: ## Build do IP Generator
	@echo "🔨 Building IP Generator..."
	cd ip-generator && make build

build-port-scanner: ## Build do Port Scanner
	@echo "🔨 Building Port Scanner..."
	cd port-scanner && make build

up: ## Inicia todos os serviços
	@echo "🚀 Starting all services..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d
	@echo "✅ All services started. Use 'make status' to check health."

down: ## Para todos os serviços
	@echo "🛑 Stopping all services..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down
	@echo "✅ All services stopped."

restart: down up ## Reinicia todos os serviços

# Comandos individuais para cada serviço
up-rabbitmq: ## Inicia apenas RabbitMQ
	@echo "🐰 Starting RabbitMQ..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d rabbitmq

up-mongodb: ## Inicia apenas MongoDB
	@echo "🍃 Starting MongoDB..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d mongodb

up-ip-generator: ## Inicia apenas IP Generator
	@echo "🔢 Starting IP Generator..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d ip-generator

up-port-scanner: ## Inicia apenas Port Scanner
	@echo "🔍 Starting Port Scanner..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) up -d port-scanner

down-rabbitmq: ## Para apenas RabbitMQ
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) stop rabbitmq

down-mongodb: ## Para apenas MongoDB
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) stop mongodb

down-ip-generator: ## Para apenas IP Generator
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) stop ip-generator

down-port-scanner: ## Para apenas Port Scanner
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) stop port-scanner

# Logs
logs: ## Mostra logs de todos os serviços
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f

logs-rabbitmq: ## Logs do RabbitMQ
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f rabbitmq

logs-mongodb: ## Logs do MongoDB
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f mongodb

logs-ip-generator: ## Logs do IP Generator
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f ip-generator

logs-port-scanner: ## Logs do Port Scanner
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) logs -f port-scanner

# Status e Health Checks
status: ## Verifica status de todos os serviços
	@echo "📊 Service Status:"
	@echo "=================="
	@docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) ps
	@echo ""
	@echo "🏥 Health Checks:"
	@echo "=================="
	@echo "RabbitMQ: $(shell curl -s -f http://localhost:15672/api/overview >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Unhealthy")"
	@echo "MongoDB: $(shell docker exec solomon-mongodb mongosh --eval "db.adminCommand('ping')" >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Unhealthy")"
	@echo "IP Generator: $(shell curl -s -f http://localhost:8080/api/v1/health >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Unhealthy")"
	@echo "Port Scanner: $(shell curl -s -f http://localhost:8081/api/v1/health >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Unhealthy")"

# MongoDB específicos
mongodb-shell: ## Acessa shell do MongoDB
	@echo "🍃 Connecting to MongoDB shell..."
	docker exec -it solomon-mongodb mongosh -u admin -p admin123 --authenticationDatabase admin solomon

mongodb-backup: ## Faz backup do MongoDB
	@echo "💾 Creating MongoDB backup..."
	@mkdir -p backups
	docker exec solomon-mongodb mongodump --uri="mongodb://admin:admin123@localhost:27017/solomon?authSource=admin" --out=/tmp/backup
	docker cp solomon-mongodb:/tmp/backup ./backups/$(shell date +%Y%m%d_%H%M%S)
	@echo "✅ Backup created in ./backups/"

mongodb-restore: ## Restaura backup do MongoDB (use BACKUP_PATH=path/to/backup)
	@if [ -z "$(BACKUP_PATH)" ]; then \
		echo "❌ Please specify BACKUP_PATH=path/to/backup"; \
		exit 1; \
	fi
	@echo "🔄 Restoring MongoDB from $(BACKUP_PATH)..."
	docker cp $(BACKUP_PATH) solomon-mongodb:/tmp/restore
	docker exec solomon-mongodb mongorestore --uri="mongodb://admin:admin123@localhost:27017/solomon?authSource=admin" /tmp/restore
	@echo "✅ Restore completed"

# Testes
test: ## Executa testes de todos os serviços
	@echo "🧪 Running tests..."
	cd ip-generator && make test
	cd port-scanner && make test

# Limpeza
clean: ## Remove containers, volumes e imagens
	@echo "🧹 Cleaning up..."
	docker-compose -f $(COMPOSE_FILE) -p $(PROJECT_NAME) down -v --rmi all
	docker system prune -f
	@echo "✅ Cleanup completed"

# Desenvolvimento
dev-setup: ## Configuração inicial para desenvolvimento
	@echo "🔧 Setting up development environment..."
	@if [ ! -f .env ]; then \
		cp env.example .env; \
		echo "✅ Created .env file from template"; \
	fi
	@mkdir -p logs/ip-generator logs/port-scanner backups
	@echo "✅ Development setup completed"

# APIs e Exemplos
api-examples: ## Mostra exemplos de uso das APIs
	@echo "📡 API Examples:"
	@echo "================"
	@echo ""
	@echo "🔢 IP Generator (Port 8080):"
	@echo "  Health Check: curl http://localhost:8080/api/v1/health"
	@echo "  Generate IPs: curl -X POST http://localhost:8080/api/v1/generate -H 'Content-Type: application/json' -d '{\"count\": 10}'"
	@echo "  Get Stats:    curl http://localhost:8080/api/v1/stats"
	@echo ""
	@echo "🔍 Port Scanner (Port 8081):"
	@echo "  Health Check: curl http://localhost:8081/api/v1/health"
	@echo "  Scan IP:      curl -X POST http://localhost:8081/api/v1/scan -H 'Content-Type: application/json' -d '{\"ip\": \"8.8.8.8\"}'"
	@echo "  Get Stats:    curl http://localhost:8081/api/v1/stats"
	@echo "  DB Stats:     curl http://localhost:8081/api/v1/db/stats"
	@echo "  Get Result:   curl http://localhost:8081/api/v1/db/result/8.8.8.8"

# Monitoramento
monitor: ## Inicia monitoramento em tempo real
	@echo "📊 Starting real-time monitoring..."
	@echo "Press Ctrl+C to stop"
	@watch -n 2 'make status'

# Default target
.DEFAULT_GOAL := help 