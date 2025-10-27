# Mixed Architecture Setup Guide

## Overview

This setup uses a mixed architecture combining:
- **Containerized Services**: PostgreSQL, Redis, MinIO, Traefik (managed by Docker)
- **External Services**: Navy, Wayne API, Wayne (running as local processes on localhost)

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Host System                              │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ Navy        │  │ Wayne API   │  │ Wayne       │          │
│  │ :8081       │  │ :8082       │  │ :8083       │          │
│  │ (Process)   │  │ (Process)   │  │ (Process)   │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
│  ┌─────────────┐  ┌─────────────┐                          │
│  │ Spydon      │  │ Spydon FE   │                          │
│  │ :8086       │  │ :3000       │                          │
│  │ (Process)   │  │ (Process)   │                          │
│  └─────────────┘  └─────────────┘                          │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                 Docker Engine                       │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │    │
│  │ │Traefik  │ │PostgreSQL│ │  Redis  │ │  MinIO  │   │    │
│  │ │ :80,443 │ │ :5432   │ │ :6379   │ │:9000,9001│   │    │
│  │ └─────────┘ └─────────┘ └─────────┘ └─────────┘   │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Service Details

### Containerized Services (Docker Managed)

| Service | Port | Description | Status |
|---------|------|-------------|---------|
| Traefik | 80 | Reverse proxy (HTTP only) | ✅ Container |
| PostgreSQL | 5432 | Primary database | ✅ Container |
| Redis | 6379 | Cache and session storage | ✅ Container |
| MinIO | 9000, 9001 | Object storage with console | ✅ Container |

### External Services (Local Processes)

| Service | Port | Description | Startup Method |
|---------|------|-------------|----------------|
| Navy Service | 8081 | Navy application service | Local process |
| Wayne API | 8082 | Wayne API service | Local process |
| Wayne Service | 8083 | Wayne application service | Local process |
| Spydon Service | 8086 | Spydon backend service | Local process |
| Spydon Frontend | 3000 | Spydon frontend application | Local process |

## Routing Configuration

### Traefik Routing Rules

| Path | Target Service | Path Stripping | Description |
|------|----------------|----------------|-------------|
| `/navy/*` | `localhost:8081` | ✅ Yes | Routes to Navy Service |
| `/wayneapi/*` | `localhost:8082` | ✅ Yes | Routes to Wayne API |
| `/wayne/*` | `localhost:8083` | ✅ Yes | Routes to Wayne Service |
| `/spydon/*` | `localhost:8086` | ✅ Yes | Routes to Spydon Service |
| `/spydon-fe/*` | `localhost:3000` | ✅ Yes | Routes to Spydon Frontend |

**Note**: All services are now accessible via HTTP on port 80 only (no HTTPS).

### URL Examples

- `http://localhost/navy/dashboard` → `http://localhost:8081/dashboard`
- `http://localhost/wayneapi/users` → `http://localhost:8082/users`
- `http://localhost/wayne/status` → `http://localhost:8083/status`
- `http://localhost/spydon/api/health` → `http://localhost:8086/api/health`
- `http://localhost/spydon-fe/home` → `http://localhost:3000/home`

## Quick Start

### 1. Start Containerized Services

```bash
# Navigate to installation package directory
cd /path/to/installation-package

# Load Docker images (if not already loaded)
docker load -i postgres-15-alpine.tar
docker load -i minio-latest.tar
docker load -i redis-7-alpine.tar
docker load -i traefik-v2.11.tar

# Start all containerized services
docker-compose up -d

# Verify services are running
docker-compose ps
```

### 2. Start External Services

External services must be started manually on the specified ports:

```bash
# Example commands (replace with actual startup commands)

# Start Navy Service on port 8081
cd /path/to/navy-service
./navy-service --port=8081 &

# Start Wayne API on port 8082
cd /path/to/wayne-api
./wayne-api --port=8082 &

# Start Wayne Service on port 8083
cd /path/to/wayne-service
./wayne-service --port=8083 &
```

### 3. Verify Setup

```bash
# Check containerized services
curl http://localhost/traefik/dashboard  # Traefik dashboard
curl http://localhost/minio/console        # MinIO console

# Check external services (after starting them)
curl http://localhost/navy/health         # Navy service
curl http://localhost/wayneapi/health     # Wayne API
curl http://localhost/wayne/health        # Wayne service
curl http://localhost/spydon/health       # Spydon service
curl http://localhost/spydon-fe/          # Spydon Frontend
```

## Configuration Files

### docker-compose.yml

The main configuration file that defines:
- Traefik reverse proxy with host networking
- Database services (PostgreSQL, Redis, MinIO)
- Volume mounts for Traefik dynamic configuration
- External service documentation

### traefik.yml

Static Traefik configuration:
- Docker provider for container discovery
- File provider for external service routing
- Entry points (HTTP→HTTPS redirection)
- Security settings

### dynamic/external-services.yml

Dynamic routing configuration for external services:
- Router definitions for each external service
- Service definitions pointing to localhost ports
- Middleware for path prefix stripping

## Service Dependencies

### Containerized Services
- **Traefik**: No dependencies (core routing)
- **PostgreSQL**: Independent (data persistence)
- **Redis**: Independent (caching layer)
- **MinIO**: Independent (object storage)

### External Services
- **Navy Service**: May depend on PostgreSQL, Redis
- **Wayne API**: May depend on PostgreSQL, MinIO
- **Wayne Service**: May depend on Wayne API, PostgreSQL

## Troubleshooting

### Common Issues

1. **External services not accessible**
   ```bash
   # Check if services are running on correct ports
   netstat -tlnp | grep -E ':(8081|8082|8083)'

   # Test direct connection
   curl http://localhost:8081/health
   curl http://localhost:8082/health
   curl http://localhost:8083/health
   ```

2. **Traefik routing not working**
   ```bash
   # Check Traefik logs
   docker-compose logs traefik

   # Verify dynamic configuration is loaded
   curl http://localhost:8080/api/http/services
   ```

3. **Connection errors**
   - This setup uses HTTP only (no SSL/TLS)
   - Ensure port 80 is accessible and not blocked by firewall
   - For production, consider adding HTTPS/TLS support

4. **Port conflicts**
   ```bash
   # Check port availability
   netstat -tlnp | grep -E ':(80|443|5432|6379|9000|9001|8081|8082|8083)'

   # Stop conflicting services if needed
   sudo systemctl stop nginx  # Example: stop nginx if it conflicts
   ```

### Debug Commands

```bash
# Docker service status
docker-compose ps

# Docker service logs
docker-compose logs -f

# Network connectivity test
docker exec -it traefik ping localhost

# Traefik API diagnostics
curl http://localhost:8080/api/http/routers
curl http://localhost:8080/api/http/services
```

## Development Workflow

### Making Configuration Changes

1. **Static Traefik changes**: Edit `traefik.yml`
2. **Dynamic routing changes**: Edit `dynamic/external-services.yml`
3. **Service changes**: Edit `docker-compose.yml`
4. **Reload configuration**:
   ```bash
   docker-compose restart traefik
   ```

### Adding New External Services

1. Start the service on an available port
2. Add routing configuration to `dynamic/external-services.yml`
3. Restart Traefik:
   ```bash
   docker-compose restart traefik
   ```

## Security Considerations

### Containerized Services
- PostgreSQL, Redis, MinIO are only accessible from localhost
- Traefik handles all external traffic
- Database passwords should be changed in production

### External Services
- Services run with local process permissions
- Traefik provides HTTP routing (no SSL/TLS)
- Consider adding authentication middleware in Traefik
- **Security Warning**: HTTP only configuration - no encryption

### Network Security
- Host networking gives containers access to host network
- External services are accessible only through Traefik
- Configure firewall rules to restrict direct access to external service ports

## Backup and Recovery

### Containerized Services
```bash
# Backup PostgreSQL
docker exec postgres pg_dump -U postgres myapp > backup.sql

# Backup MinIO data
docker exec minio mc mirror /data /backup/minio

# Backup Redis data
docker exec redis redis-cli BGSAVE
docker cp redis:/data/dump.rdb ./redis-backup.rdb
```

### External Services
External services should implement their own backup mechanisms since they're not managed by Docker.

## Performance Monitoring

### Container Metrics
```bash
# Docker stats
docker-compose ps
docker stats

# Traefik metrics
curl http://localhost:8080/metrics
```

### External Service Monitoring
External services should expose metrics for monitoring systems like Prometheus.

## Production Deployment

### Scaling Considerations
- Containerized services can be scaled using Docker Compose or Kubernetes
- External services may need process management (systemd, supervisor)
- Consider load balancing for multiple instances

### High Availability
- PostgreSQL: Configure replication
- Redis: Configure cluster mode
- MinIO: Configure distributed mode
- External services: Run multiple instances behind load balancer

## Support

For issues with:
- **Containerized services**: Check Docker logs and configuration
- **External services**: Consult respective service documentation
- **Traefik routing**: Check Traefik dashboard and logs
- **Network issues**: Verify port availability and firewall settings