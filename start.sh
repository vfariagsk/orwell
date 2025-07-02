#!/bin/bash

# Orwell Microservices Platform - Startup Script
# Este script configura e inicia toda a plataforma

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Função para imprimir mensagens coloridas
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verificar se Docker está instalado
check_docker() {
    print_status "Checking Docker installation..."
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    print_success "Docker and Docker Compose are installed"
}

# Verificar se Docker está rodando
check_docker_running() {
    print_status "Checking if Docker is running..."
    if ! docker info &> /dev/null; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    print_success "Docker is running"
}

# Configurar ambiente
setup_environment() {
    print_status "Setting up environment..."
    
    # Criar arquivo .env se não existir
    if [ ! -f .env ]; then
        print_status "Creating .env file from template..."
        cp env.example .env
        print_success "Created .env file"
    else
        print_warning ".env file already exists"
    fi
    
    # Criar diretórios necessários
    mkdir -p logs/ip-generator logs/port-scanner backups mongodb/init
    print_success "Created necessary directories"
}

# Build dos serviços
build_services() {
    print_status "Building services..."
    
    # Build IP Generator
    print_status "Building IP Generator..."
    cd ip-generator
    if [ -f Makefile ]; then
        make build
    else
        go mod tidy
        go build -o bin/ip-generator cmd/server/main.go
    fi
    cd ..
    
    # Build Port Scanner
    print_status "Building Port Scanner..."
    cd port-scanner
    if [ -f Makefile ]; then
        make build
    else
        go mod tidy
        go build -o bin/port-scanner cmd/server/main.go
    fi
    cd ..
    
    print_success "All services built successfully"
}

# Iniciar serviços
start_services() {
    print_status "Starting services..."
    
    # Iniciar RabbitMQ primeiro
    print_status "Starting RabbitMQ..."
    docker-compose up -d rabbitmq
    
    # Aguardar RabbitMQ estar pronto
    print_status "Waiting for RabbitMQ to be ready..."
    sleep 10
    
    # Iniciar MongoDB
    print_status "Starting MongoDB..."
    docker-compose up -d mongodb
    
    # Aguardar MongoDB estar pronto
    print_status "Waiting for MongoDB to be ready..."
    sleep 15
    
    # Iniciar microserviços
    print_status "Starting microservices..."
    docker-compose up -d ip-generator port-scanner
    
    print_success "All services started"
}

# Verificar status dos serviços
check_services() {
    print_status "Checking service status..."
    
    # Aguardar um pouco para os serviços inicializarem
    sleep 10
    
    # Verificar RabbitMQ
    if curl -s -f http://localhost:15672/api/overview >/dev/null 2>&1; then
        print_success "RabbitMQ is healthy"
    else
        print_warning "RabbitMQ health check failed"
    fi
    
    # Verificar MongoDB
    if docker exec solomon-mongodb mongosh --eval "db.adminCommand('ping')" >/dev/null 2>&1; then
        print_success "MongoDB is healthy"
    else
        print_warning "MongoDB health check failed"
    fi
    
    # Verificar IP Generator
    if curl -s -f http://localhost:8080/api/v1/health >/dev/null 2>&1; then
        print_success "IP Generator is healthy"
    else
        print_warning "IP Generator health check failed"
    fi
    
    # Verificar Port Scanner
    if curl -s -f http://localhost:8081/api/v1/health >/dev/null 2>&1; then
        print_success "Port Scanner is healthy"
    else
        print_warning "Port Scanner health check failed"
    fi
}

# Mostrar informações úteis
show_info() {
    echo ""
    echo "🎉 Solomon Microservices Platform is ready!"
    echo "=========================================="
    echo ""
    echo "📊 Service URLs:"
    echo "  RabbitMQ Management: http://localhost:15672 (admin/admin123)"
    echo "  IP Generator API:    http://localhost:8080"
    echo "  Port Scanner API:    http://localhost:8081"
    echo "  MongoDB:             localhost:27017"
    echo ""
    echo "🔧 Useful Commands:"
    echo "  Check status:        make status"
    echo "  View logs:           make logs"
    echo "  Stop services:       make down"
    echo "  MongoDB shell:       make mongodb-shell"
    echo "  API examples:        make api-examples"
    echo ""
    echo "📝 Quick Test:"
    echo "  curl http://localhost:8080/api/v1/health"
    echo "  curl http://localhost:8081/api/v1/health"
    echo ""
}

# Função principal
main() {
    echo "🚀 Starting Solomon Microservices Platform Setup"
    echo "================================================"
    echo ""
    
    check_docker
    check_docker_running
    setup_environment
    build_services
    start_services
    check_services
    show_info
}

# Executar função principal
main "$@" 