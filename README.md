# Solomon Microservices Platform

Uma plataforma de microserviços para geração de IPs e escaneamento de portas com arquitetura hexagonal, usando RabbitMQ para comunicação assíncrona e MongoDB para persistência de dados.

## 🏗️ Arquitetura

A plataforma consiste em dois microserviços principais:

### 1. IP Generator (`ip-generator`)
- Gera endereços IPv4 aleatórios (excluindo 127.*.*.*)
- Publica IPs em filas RabbitMQ
- API REST para geração sob demanda
- Logs estruturados com Zap

### 2. Port Scanner (`port-scanner`)
- Consome IPs das filas RabbitMQ
- Escaneamento rápido e concorrente de portas
- Banner grabbing com ZGrab2
- Detecção de versões de serviços
- Persistência automática em MongoDB
- API REST para consultas e estatísticas

## 🗄️ Banco de Dados

### MongoDB
- **Database**: `solomon`
- **Collections**:
  - `scan_results`: Resultados completos de escaneamento

### Índices Otimizados
- IP + timestamp para consultas por endereço
- Batch ID + timestamp para consultas por lote
- Status + timestamp para filtros
- Portas abertas para análise de serviços

## 🚀 Início Rápido

### Pré-requisitos
- Docker e Docker Compose
- Go 1.23+ (para desenvolvimento)

### 1. Clone e Configure
```bash
git clone <repository>
cd z
```

### 2. Configure as Variáveis de Ambiente
```bash
cp env.example .env
# Edite o arquivo .env conforme necessário
```

### 3. Inicie os Serviços
```bash
# Iniciar todos os serviços
make up

# Ou individualmente
make up-rabbitmq
make up-mongodb
make up-ip-generator
make up-port-scanner
```

### 4. Verifique o Status
```bash
make status
```

## 📊 APIs Disponíveis

### IP Generator (Porta 8080)
- `GET /api/v1/health` - Status do serviço
- `POST /api/v1/generate` - Gerar IPs
- `GET /api/v1/stats` - Estatísticas

### Port Scanner (Porta 8081)
- `GET /api/v1/health` - Status do serviço (inclui MongoDB)
- `POST /api/v1/scan` - Escanear IP individual
- `POST /api/v1/scan/batch` - Escanear múltiplos IPs
- `GET /api/v1/stats` - Estatísticas de escaneamento
- `GET /api/v1/status/:ip` - Status de escaneamento por IP
- `GET /api/v1/ports/:ip` - Portas abertas por IP

#### Endpoints MongoDB
- `GET /api/v1/db/stats` - Estatísticas do banco de dados
- `GET /api/v1/db/result/:ip` - Resultado de escaneamento por IP
- `GET /api/v1/db/batch/:batch_id` - Resultados por lote
- `GET /api/v1/db/search` - Busca avançada (em desenvolvimento)

## 🔧 Configuração

### Variáveis de Ambiente

#### IP Generator
```yaml
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
RABBITMQ_QUEUE=ip_queue
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
LOG_LEVEL=info
```

#### Port Scanner
```yaml
RABBITMQ_URL=amqp://admin:admin123@rabbitmq:5672/
RABBITMQ_IP_QUEUE=ip_queue
RABBITMQ_SCAN_RESULT_QUEUE=scan_result_queue
RABBITMQ_ENRICHMENT_QUEUE=enrichment_queue
RABBITMQ_SERVICE_ANALYSIS_QUEUE=service_analysis_queue
MONGODB_CONNECTION_STRING=mongodb://admin:admin123@mongodb:27017/solomon?authSource=admin
MONGODB_DATABASE_NAME=solomon
MONGODB_COLLECTION_NAME=scan_results
MONGODB_ENABLE_DATABASE=true
SERVER_HOST=0.0.0.0
SERVER_PORT=8081
LOG_LEVEL=info
```

### Configuração de Escaneamento
```yaml
scan:
  ping_timeout: "5s"
  connect_timeout: "3s"
  banner_timeout: "2s"
  max_retries: 3
  retry_delay: "1s"
  concurrency: 100
  zgrab_concurrency: 20
  enable_banner: true
  enable_ping: true
  priority_ports: [80, 443, 22, 21, 25, 3306, 5432]
```

## 📈 Monitoramento

### Logs Estruturados
Todos os serviços usam logs estruturados com Zap:
- **Campos**: service, instance_id, request_id, event, level, timestamp, message
- **Níveis**: debug, info, warn, error, fatal
- **Filtros**: Por serviço, evento, IP, batch_id

### Health Checks
- RabbitMQ: `rabbitmq-diagnostics ping`
- MongoDB: `mongosh --eval "db.adminCommand('ping')"`
- IP Generator: `curl -f http://localhost:8080/api/v1/health`
- Port Scanner: `curl -f http://localhost:8081/api/v1/health`

### Métricas Disponíveis
- Total de IPs gerados
- Total de IPs escaneados
- Taxa de sucesso/falha
- Tempo médio de escaneamento
- Estatísticas de banner grabbing
- Métricas do MongoDB

## 🔍 Consultas MongoDB

### Exemplos de Consultas

#### Buscar resultado por IP
```javascript
db.scan_results.findOne({ "ip": "192.168.1.1" })
```

#### Buscar IPs com portas abertas
```javascript
db.scan_results.find({ "is_up": true, "open_ports": { $gt: 0 } })
```

#### Estatísticas por status
```javascript
db.scan_results.aggregate([
  { $group: { _id: "$status", count: { $sum: 1 } } }
])
```

#### Buscar por lote
```javascript
db.scan_results.find({ "batch_id": "batch-123" })
```

## 🛠️ Desenvolvimento

### Estrutura do Projeto
```
z/
├── ip-generator/          # Gerador de IPs
│   ├── cmd/server/       # Entry point
│   ├── internal/         # Lógica interna
│   │   ├── domain/       # Entidades e interfaces
│   │   ├── application/  # Casos de uso
│   │   └── infrastructure/ # Implementações
│   └── pkg/log/          # Logging compartilhado
├── port-scanner/         # Escaneador de portas
│   ├── cmd/server/       # Entry point
│   ├── internal/         # Lógica interna
│   │   ├── domain/       # Entidades e interfaces
│   │   ├── application/  # Casos de uso
│   │   └── infrastructure/ # Implementações
│   │       ├── database/ # MongoDB
│   │       ├── banner/   # Banner grabbing
│   │       └── queue/    # RabbitMQ
│   └── pkg/log/          # Logging compartilhado
├── mongodb/              # Configuração MongoDB
│   └── init/             # Scripts de inicialização
├── docker-compose.yml    # Orquestração
└── Makefile              # Comandos úteis
```

### Comandos de Desenvolvimento
```bash
# Build dos serviços
make build

# Testes
make test

# Logs em tempo real
make logs

# Acessar MongoDB
make mongodb-shell

# Backup do banco
make mongodb-backup

# Restaurar backup
make mongodb-restore
```

## 🔒 Segurança

### Configurações de Segurança
- **Port Scanner**: Execução com privilégios mínimos
- **MongoDB**: Autenticação habilitada
- **RabbitMQ**: Usuário e senha configurados
- **Networks**: Isolamento entre serviços

### Recomendações de Produção
1. Use secrets management para credenciais
2. Configure TLS para MongoDB e RabbitMQ
3. Implemente rate limiting nas APIs
4. Configure backup automático do MongoDB
5. Monitore logs para atividades suspeitas

## 📝 Logs e Debugging

### Filtros de Log Úteis
```bash
# Logs do port-scanner
docker logs solomon-port-scanner | grep "scan_completed"

# Logs de erro
docker logs solomon-port-scanner | grep "ERROR"

# Logs por IP específico
docker logs solomon-port-scanner | grep "192.168.1.1"

# Logs de MongoDB
docker logs solomon-port-scanner | grep "mongodb"
```

### Debugging
```bash
# Acessar container do port-scanner
docker exec -it solomon-port-scanner /bin/sh

# Verificar conectividade MongoDB
docker exec -it solomon-mongodb mongosh --eval "db.adminCommand('ping')"

# Verificar filas RabbitMQ
docker exec -it solomon-rabbitmq rabbitmqctl list_queues
```

## 🤝 Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature
3. Commit suas mudanças
4. Push para a branch
5. Abra um Pull Request

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo LICENSE para detalhes. 