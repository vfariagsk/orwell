# Port Scanner Microservice

A high-performance port scanner microservice built with Go, featuring hexagonal architecture, concurrent scanning, and optimized banner grabbing with ZGrab2 integration.

## Features

### Core Scanning
- **Fast Concurrent Scanning**: Uses goroutines and semaphores for efficient parallel port scanning
- **TCP SYN Scanning**: Fast connection-based port detection
- **Ping Detection**: Safe ping service with input validation and timeouts
- **Retry Logic**: Configurable retry mechanism with exponential backoff
- **Timeout Management**: Comprehensive timeout handling for all operations

### Banner Grabbing
- **ZGrab2 Integration**: Advanced banner grabbing using ZGrab2 with protocol-specific modules
- **Worker Pool Architecture**: Optimized concurrency with dedicated worker pools for ZGrab2 processes
- **Priority-Based Processing**: High-priority ports get preferential treatment for banner grabbing
- **Dynamic Module Selection**: Automatically selects appropriate ZGrab2 modules based on port
- **Fallback Mechanisms**: Graceful fallback to basic banner grabbing when ZGrab2 fails
- **Version Detection**: Comprehensive version extraction from multiple protocols
- **Confidence Levels**: Indicates whether banner info comes from ZGrab2 or basic grabbing

### Performance Optimizations
- **Concurrency Limits**: Separate limits for general scanning and ZGrab2 processes
- **Resource Management**: Prevents system overload with configurable worker pools
- **Memory Efficiency**: Optimized data structures and garbage collection
- **Network Optimization**: Connection pooling and timeout management
- **CPU Utilization**: Efficient use of system resources with proper goroutine management

### Architecture
- **Hexagonal Architecture**: Clean separation of concerns with domain, application, and infrastructure layers
- **Dependency Injection**: Flexible service composition and testing
- **Interface-Based Design**: Loose coupling between components
- **Configuration Management**: Flexible configuration with Viper
- **Error Handling**: Comprehensive error handling and logging

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                     │
├─────────────────────────────────────────────────────────────┤
│  HTTP Server  │  RabbitMQ  │  ZGrab2  │  Ping Service      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
├─────────────────────────────────────────────────────────────┤
│  Scan Engine  │  Queue Manager  │  Banner Worker Pool      │
└─────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Domain Layer                           │
├─────────────────────────────────────────────────────────────┤
│  Scanner Service  │  Banner Grabber  │  Port/Scan Entities  │
└─────────────────────────────────────────────────────────────┘
```

## Performance Features

### Concurrency Model
- **General Scanning**: Uses semaphore-based concurrency control (default: 100 concurrent scans)
- **ZGrab2 Processes**: Dedicated worker pool with configurable limits (default: 20 workers)
- **Banner Grabbing**: Priority-based processing with intelligent fallback
- **Resource Isolation**: Separate pools prevent resource contention

### Optimization Strategies
1. **Port Priority System**: Critical ports (80, 443, 22) get priority for ZGrab2 processing
2. **Dynamic Module Selection**: Only runs relevant ZGrab2 modules per port
3. **Aggressive Timeouts**: Prevents hanging processes and resource exhaustion
4. **Connection Reuse**: Efficient connection management
5. **Memory Pooling**: Reduces garbage collection pressure

### Monitoring and Metrics
- **Real-time Statistics**: Track scanning and banner grabbing performance
- **Worker Pool Metrics**: Monitor ZGrab2 worker utilization
- **Error Rates**: Track success/failure rates for optimization
- **Response Times**: Monitor average processing times

## Configuration

### Scan Configuration
```yaml
scan:
  ping_timeout: "5s"
  connect_timeout: "3s"
  banner_timeout: "2s"
  max_retries: 3
  retry_delay: "1s"
  concurrency: 100              # General scanning concurrency
  zgrab_concurrency: 20         # ZGrab2 worker pool size
  enable_banner: true
  enable_ping: true
  priority_ports: [80, 443, 22, 21, 25, 3306, 5432]  # High-priority ports
```

### Performance Tuning
- **Concurrency**: Adjust based on system resources and network capacity
- **ZGrab Concurrency**: Balance between performance and system load
- **Timeouts**: Optimize for your network environment
- **Priority Ports**: Focus resources on most important services

## API Endpoints

### Core Endpoints
- `GET /api/v1/health` - Service health check
- `GET /api/v1/stats` - Scanning statistics
- `GET /api/v1/banner-stats` - Banner grabbing performance metrics
- `POST /api/v1/scan` - Scan single IP
- `POST /api/v1/scan/batch` - Batch scan multiple IPs
- `GET /api/v1/status/:ip` - Get scan status for IP
- `GET /api/v1/ports/:ip` - Get open ports for IP

### Banner Statistics Endpoint
```bash
curl http://localhost:8080/api/v1/banner-stats
```

Response includes:
- Total banner grabs performed
- ZGrab2 vs basic banner grab counts
- Average processing times
- Error rates and worker pool statistics
- Real-time performance metrics

## Installation and Setup

### Prerequisites
- Go 1.21+
- Docker and Docker Compose
- ZGrab2 (automatically installed in Docker)

### Quick Start
```bash
# Clone and build
git clone <repository>
cd port-scanner
make build

# Run with Docker Compose
docker-compose up -d

# Or run locally
./server
```

### Docker Setup
```bash
# Build and run
docker build -t port-scanner .
docker run -p 8080:8080 port-scanner
```

## Performance Benchmarks

### Typical Performance
- **Port Scanning**: 1000 ports/second per IP
- **Banner Grabbing**: 50-200 banners/second (depending on ZGrab2 usage)
- **Concurrent IPs**: 100+ simultaneous scans
- **Memory Usage**: ~50-100MB per worker pool
- **CPU Usage**: Efficient utilization with proper limits

### Optimization Tips
1. **Adjust Concurrency**: Monitor system resources and adjust limits
2. **Priority Ports**: Focus ZGrab2 on critical services
3. **Timeout Tuning**: Optimize for your network environment
4. **Resource Monitoring**: Use banner stats endpoint for optimization
5. **Load Balancing**: Distribute load across multiple instances

## Security Considerations

### Safe Execution
- **Input Validation**: All inputs are validated and sanitized
- **Command Sandboxing**: ZGrab2 executed with proper isolation
- **Timeout Enforcement**: Prevents hanging processes
- **Resource Limits**: Prevents system overload
- **Error Handling**: Graceful degradation on failures

### Network Security
- **Rate Limiting**: Built-in rate limiting for scanning
- **Timeout Management**: Prevents resource exhaustion
- **Connection Limits**: Prevents connection flooding
- **Error Logging**: Comprehensive security event logging

## Monitoring and Observability

### Metrics Available
- **Scan Performance**: Total scans, success rates, average times
- **Banner Grabbing**: ZGrab2 usage, fallback rates, processing times
- **Resource Utilization**: Worker pool status, queue depths
- **Error Tracking**: Failure rates, timeout statistics
- **Network Performance**: Response times, connection success rates

### Health Checks
- **Service Health**: Overall service status
- **Dependency Health**: RabbitMQ, ZGrab2 availability
- **Resource Health**: Memory, CPU, network utilization
- **Performance Health**: Response time monitoring

## Troubleshooting

### Common Issues
1. **High Memory Usage**: Reduce ZGrab2 concurrency
2. **Slow Performance**: Increase concurrency limits
3. **ZGrab2 Failures**: Check ZGrab2 installation and permissions
4. **Network Timeouts**: Adjust timeout values for your environment
5. **Resource Exhaustion**: Monitor and adjust worker pool sizes

### Performance Tuning
1. **Monitor Banner Stats**: Use `/api/v1/banner-stats` endpoint
2. **Adjust Concurrency**: Balance performance vs resource usage
3. **Optimize Timeouts**: Match your network characteristics
4. **Priority Configuration**: Focus resources on important ports
5. **Resource Monitoring**: Track CPU, memory, and network usage

## Contributing

1. Follow hexagonal architecture principles
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Follow Go best practices and conventions
5. Ensure proper error handling and logging

## License

This project is licensed under the MIT License. 