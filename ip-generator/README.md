# IP Generator Microservice

A hexagonal architecture microservice responsible for generating IPv4 addresses (excluding private and special purpose ranges) and sending them to a message queue for scanning operations. Built with **Gin** framework for high-performance HTTP handling.

## Architecture

This microservice follows the **Hexagonal Architecture** (also known as Ports and Adapters) pattern:

```
┌──────────────────────────────────────────────────────────────┐
│                    External World                            │
├──────────────────────────────────────────────────────────────┤
│                    Infrastructure Layer                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │
│  │ Gin Server  │  │ RabbitMQ    │  │ Configuration       │   │
│  │ (Port)      │  │ (Adapter)   │  │ (Adapter)           │   │
│  └─────────────┘  └─────────────┘  └─────────────────────┘   │
├──────────────────────────────────────────────────────────────┤
│                    Application Layer                         │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │           IP Generation Service                         │ │
│  │           (Use Cases)                                   │ │
│  └─────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────┤
│                    Domain Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │
│  │ IP Address  │  │ IP Generator│  │ Queue Interfaces    │   │
│  │ (Entity)    │  │ (Service)   │  │ (Ports)             │   │
│  └─────────────┘  └─────────────┘  └─────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

1. **Domain Layer** (`internal/domain/`)
   - Core business entities (`IPAddress`)
   - Business logic (`IPGeneratorService`)
   - Interface definitions (`QueuePublisher`, `QueueSubscriber`)

2. **Application Layer** (`internal/application/`)
   - Use cases and orchestration (`IPGenerationService`)
   - Coordinates between domain and infrastructure

3. **Infrastructure Layer** (`internal/infrastructure/`)
   - Gin HTTP handlers and server (`http/`)
   - RabbitMQ implementation (`queue/`)
   - Configuration management (`config/`)

## Features

- **IP Generation**: 
  - Random IPv4 generation (excluding private and special purpose ranges)
  - Sequential IPv4 generation from a starting point
  - Configurable batch sizes
  - **Pseudo-random permutation** for uniform distribution

- **Queue Integration**: 
  - RabbitMQ integration for message publishing
  - Structured message format with batch IDs
  - Batch processing support

- **REST API with Gin**:
  - High-performance HTTP handling
  - Built-in middleware (CORS, Request ID, Rate Limiting)
  - Request validation and error handling
  - Multiple endpoint formats (JSON and Query Parameters)

- **Configuration**:
  - YAML configuration file
  - Environment variable support
  - Default values and validation

- **Containerization**:
  - Multi-stage Dockerfile
  - Docker Compose setup with RabbitMQ
  - Production-ready container configuration

## Excluded IP Ranges

The service automatically excludes the following IP ranges to ensure only valid public IPs are generated:

- **0.0.0.0/8** - Current network (only valid as source address)
- **10.0.0.0/8** - Private network
- **127.0.0.0/8** - Loopback addresses
- **192.168.0.0/16** - Private network
- **224.0.0.0/4** - Multicast addresses
- **240.0.0.0/4** - Reserved for future use

## Pseudo-Random Permutation

The service uses a **bijective permutation function** to ensure:

- **Uniform Distribution**: All valid IP addresses have equal probability
- **No Sequential Generation**: Prevents predictable patterns
- **Complete Cycle**: Covers the entire valid IP space without repetition
- **Deterministic**: Same seed produces same sequence (useful for testing)

### Permutation Algorithm

```go
func (p *IPPermutation) permute32(x uint32) uint32 {
    const prime = 0x7fffffff // 2^31 - 1 (Mersenne prime)
    
    x = (x * prime) ^ p.seed
    x = x ^ (x >> 16)
    x = x * 0x85ebca6b
    x = x ^ (x >> 13)
    x = x * 0xc2b2ae35
    x = x ^ (x >> 16)
    
    return x
}
```

## API Endpoints

### Generate Random IPs (JSON)
```http
POST /api/v1/ips/generate
Content-Type: application/json

{
  "count": 1000,
  "batch_size": 100
}
```

### Generate Sequential IPs (JSON)
```http
POST /api/v1/ips/generate/sequential
Content-Type: application/json

{
  "start_ip": "8.8.8.8",
  "count": 1000,
  "batch_size": 100
}
```

### Generate IPs (Query Parameters)
```http
GET /api/v1/ips/generate/query?count=1000&batch_size=100
```

### Health Check
```http
GET /health
```

### Service Information
```http
GET /api/v1/info
```

## Configuration

The service can be configured via `config.yaml` or environment variables:

```yaml
server:
  port: "8080"
  host: "localhost"

rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
  queue: "ip-scan-queue"
  exchange: ""

app:
  default_batch_size: 100
  max_ips_per_batch: 1000
```

### Environment Variables

- `RABBITMQ_URL`: RabbitMQ connection URL
- `RABBITMQ_QUEUE`: Queue name for IP messages
- `SERVER_PORT`: HTTP server port
- `APP_DEFAULT_BATCH_SIZE`: Default batch size for IP generation
- `APP_MAX_IPS_PER_BATCH`: Maximum IPs per batch

## Queue Message Format

```json
{
  "ips": ["8.8.8.8", "1.1.1.1", "208.67.222.222"],
  "batch_id": "batch-1234567890-0",
  "count": 3
}
```

## Getting Started

### Prerequisites

- Go 1.21+
- RabbitMQ (or Docker for containerized setup)

### Local Development

1. **Clone and navigate to the project**:
   ```bash
   cd ip-generator
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Start RabbitMQ** (using Docker):
   ```bash
   docker run -d --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3-management
   ```

4. **Run the service**:
   ```bash
   go run cmd/server/main.go
   ```

### Using Docker Compose

1. **Start all services**:
   ```bash
   docker-compose up -d
   ```

2. **View logs**:
   ```bash
   docker-compose logs -f ip-generator
   ```

3. **Stop services**:
   ```bash
   docker-compose down
   ```

### Building and Running

1. **Build the application**:
   ```bash
   go build -o ip-generator cmd/server/main.go
   ```

2. **Run the binary**:
   ```bash
   ./ip-generator
   ```

## Testing

### Manual Testing

1. **Generate random IPs (JSON)**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/ips/generate \
     -H "Content-Type: application/json" \
     -d '{"count": 10, "batch_size": 5}'
   ```

2. **Generate sequential IPs (JSON)**:
   ```bash
   curl -X POST http://localhost:8080/api/v1/ips/generate/sequential \
     -H "Content-Type: application/json" \
     -d '{"start_ip": "8.8.8.8", "count": 10, "batch_size": 5}'
   ```

3. **Generate IPs (Query Parameters)**:
   ```bash
   curl "http://localhost:8080/api/v1/ips/generate/query?count=10&batch_size=5"
   ```

4. **Health check**:
   ```bash
   curl http://localhost:8080/health
   ```

5. **Service information**:
   ```bash
   curl http://localhost:8080/api/v1/info
   ```

### Unit Testing

```bash
# Run all tests
go test ./...

# Run specific test package
go test ./internal/domain/

# Run with coverage
go test -cover ./...
```

## Middleware

The service includes several middleware components:

- **CORS**: Cross-Origin Resource Sharing support
- **Request ID**: Unique request identification for tracing
- **Rate Limiting**: Basic rate limiting (extensible)
- **Recovery**: Panic recovery and error handling
- **Logging**: Request/response logging

## Monitoring

- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Health Endpoint**: http://localhost:8080/health
- **Service Info**: http://localhost:8080/api/v1/info
- **Application Logs**: Check console output for detailed logs

## Performance

With Gin framework and permutation-based randomization, the service provides:
- High-performance HTTP handling
- Low memory footprint
- Fast JSON serialization/deserialization
- Efficient routing
- Built-in middleware stack
- **Uniform IP distribution** across the entire valid IP space
- **No sequential patterns** in generated IPs

## Security Considerations

- **IP Range Validation**: Automatically excludes private and special purpose ranges
- **Input Validation**: All API inputs are validated and sanitized
- **Rate Limiting**: Built-in rate limiting to prevent abuse
- **Error Handling**: Secure error responses without information leakage

## Deployment

### Docker Deployment

1. **Build image**:
   ```bash
   docker build -t ip-generator .
   ```

2. **Run container**:
   ```bash
   docker run -p 8080:8080 \
     -e RABBITMQ_URL=amqp://guest:guest@rabbitmq-host:5672/ \
     ip-generator
   ```

### Kubernetes Deployment

See `k8s/` directory for Kubernetes manifests.

## Contributing

1. Follow the hexagonal architecture pattern
2. Add tests for new features
3. Update documentation
4. Ensure all dependencies are properly managed
5. Use Gin best practices for HTTP handling
6. Maintain IP range exclusion rules
7. Test permutation algorithm for uniform distribution

## License

This project is licensed under the MIT License. 