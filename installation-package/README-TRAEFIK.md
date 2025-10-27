# Traefik Reverse Proxy Setup

This package includes Traefik v2.11 as a reverse proxy with pre-configured routing rules.

## Quick Start

1. **Start all services:**
   ```bash
   docker-compose up -d
   ```

2. **Access services:**
   - Traefik Dashboard: http://localhost:8080
   - Navy Service: http://localhost/navy
   - Wayne API: http://localhost/wayneapi
   - Wayne Service: http://localhost/wayne

## Routing Configuration

### Traefik Routes (No TLS)

| Route | Target Port | Path Stripping | Status |
|-------|-------------|----------------|--------|
| `/navy` | 8081 | ✅ `/navy` removed | ✅ Active |
| `/wayneapi` | 8082 | ✅ `/wayneapi` removed | ✅ Active |
| `/wayne` | 8083 | ✅ `/wayne` removed | ✅ Active |

### Example Requests

```bash
# Navy Service (internal port 8081)
curl http://localhost/navy/api/v1/ships
# Internally becomes: http://navy-service:80/api/v1/ships

# Wayne API (internal port 8082)
curl http://localhost/wayneapi/api/v1/bat-signals
# Internally becomes: http://wayneapi-service:80/api/v1/bat-signals

# Wayne Service (internal port 8083)
curl http://localhost/wayne/dashboard
# Internally becomes: http://wayne-service:80/dashboard
```

## Configuration Files

### 1. docker-compose.yml
- Main configuration with host networking
- Traefik uses `network_mode: host`
- Backend services use explicit port mappings
- Auto-configures Traefik routing labels

### 2. docker-compose-production.yml
- Production-ready configuration
- Uses nginx containers for backend services
- Host networking for direct performance
- Custom HTML pages for testing

### 3. traefik.yml
- Traefik static configuration
- Docker provider configured for host network
- Dashboard enabled on port 8080
- HTTP to HTTPS redirection (without actual TLS)

## Service Details

### Traefik
- **Image**: traefik:v2.11
- **Network Mode**: host (uses host network stack)
- **Direct Port Access**: 80, 443, 8080 (dashboard)
- **No Network Isolation**: Direct access to host network interfaces

### Backend Services
- **navy-service**: Port 8081 (accessible via /navy)
- **wayneapi-service**: Port 8082 (accessible via /wayneapi)
- **wayne-service**: Port 8083 (accessible via /wayne)

### Database Services
- **PostgreSQL**: Port 5432
- **Redis**: Port 6379
- **MinIO**: Ports 9000, 9001

## Path Stripping Configuration

Each service uses Traefik middleware to strip the path prefix:

```yaml
labels:
  - "traefik.http.middlewares.navy-stripprefix.stripprefix.prefixes=/navy"
  - "traefik.http.routers.navy.middlewares=navy-stripprefix"
```

**How it works:**
1. Request comes to `http://localhost/navy/some/path`
2. Traefik matches router rule `PathPrefix(/navy)`
3. Middleware strips `/navy` prefix
4. Request forwarded to backend as `/some/path`

## Monitoring

### Traefik Dashboard
- URL: http://localhost:8080
- Shows active routers, services, and middleware
- Real-time request monitoring

### Access Logs
Traefik logs are configured for JSON format:
```yaml
accessLog:
  filePath: "/var/log/traefik/access.log"
  format: json
```

## Host Network Benefits

### Performance Advantages
- ✅ **No Network Overhead** - Direct host network access
- ✅ **Better Performance** - Eliminates Docker network bridge latency
- ✅ **Simplified Configuration** - No custom network management
- ✅ **Direct Port Access** - Services accessible on host ports directly

### Security Considerations
- ⚠️ **No Network Isolation** - Services share host network namespace
- ⚠️ **Port Conflicts** - Ensure ports are not already in use on host
- ⚠️ **Direct Host Access** - Services have direct access to host network interfaces

## Customization

### Adding New Services

1. Add service to docker-compose.yml
2. Configure explicit port mapping
3. Configure Traefik labels:

```yaml
myservice:
  image: your-image:latest
  container_name: myservice
  ports:
    - "8084:80"  # Host port 8084 → Container port 80
  labels:
    - "traefik.enable=true"
    - "traefik.http.routers.myservice.rule=Host(`localhost`) && PathPrefix(`/myservice`)"
    - "traefik.http.routers.myservice.entrypoints=websecure"
    - "traefik.http.middlewares.myservice-stripprefix.stripprefix.prefixes=/myservice"
    - "traefik.http.routers.myservice.middlewares=myservice-stripprefix"
    - "traefik.http.services.myservice.loadbalancer.server.port=80"
```

### Changing Routes

Modify the `PathPrefix` and `stripprefix.prefixes` values in the service labels.

## Testing

### Test Each Route
```bash
# Test Navy service
curl -I http://localhost/navy

# Test Wayne API
curl -I http://localhost/wayneapi

# Test Wayne service
curl -I http://localhost/wayne
```

### View Traefik Configuration
```bash
# Check active routers
curl http://localhost:8080/api/http/routers

# Check services
curl http://localhost:8080/api/http/services
```

## Security Notes

- ⚠️ TLS is disabled (as requested)
- ⚠️ Traefik dashboard is exposed without authentication
- 🔒 Production should include proper TLS certificates
- 🔒 Dashboard should be secured in production environments