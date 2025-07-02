#!/bin/bash

# Solomon Microservices Startup Script
# Este script inicializa todos os serviços da plataforma Solomon

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
    print_status "Verificando se Docker está instalado..."
    if ! command -v docker &> /dev/null; then
        print_error "Docker não está instalado. Por favor, instale o Docker primeiro."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose não está instalado. Por favor, instale o Docker Compose primeiro."
        exit 1
    fi
    
    print_success "Docker e Docker Compose estão instalados"
}

# Verificar se os diretórios dos serviços existem
check_services() {
    print_status "Verificando estrutura dos serviços..."
    
    if [ ! -d "ip-generator" ]; then
        print_error "Diretório ip-generator não encontrado"
        exit 1
    fi
    
    if [ ! -d "port-scanner" ]; then
        print_error "Diretório port-scanner não encontrado"
        exit 1
    fi
    
    print_success "Estrutura dos serviços verificada"
}

# Criar diretórios necessários
create_directories() {
    print_status "Criando diretórios necessários..."
    
    mkdir -p logs/ip-generator
    mkdir -p logs/port-scanner
    mkdir -p rabbitmq/logs
    mkdir -p nginx/ssl
    mkdir -p backups
    
    print_success "Diretórios criados"
}

# Parar serviços existentes
stop_existing() {
    print_status "Parando serviços existentes..."
    docker-compose down --remove-orphans 2>/dev/null || true
    print_success "Serviços existentes parados"
}

# Construir imagens
build_images() {
    print_status "Construindo imagens Docker..."
    docker-compose build --no-cache
    print_success "Imagens construídas"
}

# Iniciar serviços
start_services() {
    print_status "Iniciando serviços..."
    docker-compose up -d
    
    print_status "Aguardando serviços ficarem prontos..."
    sleep 10
    
    print_success "Serviços iniciados"
}

# Verificar saúde dos serviços
check_health() {
    print_status "Verificando saúde dos serviços..."
    
    # Aguardar RabbitMQ ficar pronto
    print_status "Aguardando RabbitMQ..."
    for i in {1..30}; do
        if curl -s http://localhost:15672 > /dev/null 2>&1; then
            print_success "RabbitMQ está pronto"
            break
        fi
        if [ $i -eq 30 ]; then
            print_warning "RabbitMQ demorou para ficar pronto"
        fi
        sleep 2
    done
    
    # Verificar IP Generator
    print_status "Verificando IP Generator..."
    if curl -s http://localhost:8080/api/v1/health > /dev/null 2>&1; then
        print_success "IP Generator está saudável"
    else
        print_warning "IP Generator pode não estar pronto ainda"
    fi
    
    # Verificar Port Scanner
    print_status "Verificando Port Scanner..."
    if curl -s http://localhost:8081/api/v1/health > /dev/null 2>&1; then
        print_success "Port Scanner está saudável"
    else
        print_warning "Port Scanner pode não estar pronto ainda"
    fi
}

# Mostrar informações finais
show_info() {
    echo ""
    echo "=========================================="
    echo " Orwell Microservices Platform Started!"
    echo "=========================================="
    echo ""
    echo "📋 Services Available:"
    echo "  • IP Generator:     http://localhost:8080"
    echo "  • Port Scanner:     http://localhost:8081"
    echo "  • RabbitMQ UI:      http://localhost:15672"
    echo "  • Nginx Proxy:      http://localhost"
    echo ""
    echo "🔑 RabbitMQ Credentials:"
    echo "  • Usuário: admin"
    echo "  • Senha:  admin123"
    echo ""
    echo "📊 Useful Commands:"
    echo "  • Ver status:       make status"
    echo "  • Ver logs:         make logs"
    echo "  • Stop services:    make down"
    echo "  • Full demo:        make demo"
    echo ""
    echo "🚀 Quick Test:"
    echo "  curl -X POST http://localhost:8080/api/v1/generate -H 'Content-Type: application/json' -d '{\"count\": 5}'"
    echo ""
}

# Função principal
main() {
    echo "Starting Orwell Microservices Platform..."
    echo ""
    
    check_docker
    check_services
    create_directories
    stop_existing
    build_images
    start_services
    check_health
    show_info
}

# Executar função principal
main "$@" 