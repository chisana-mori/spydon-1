# AMD64 Offline Installation Package

This package contains the following AMD64 compatible software for **offline environments**:

## 1. Node.js
- **File**: `nodejs-amd64.tar.xz`
- **Version**: v24.2.0
- **Architecture**: Linux x64
- **Installation**: Extract to `/usr/local/` and add to PATH

## 2. PostgreSQL Docker Image
- **File**: `postgres-15-alpine.tar`
- **Version**: 15 Alpine
- **Architecture**: linux/amd64
- **Load command**: `docker load -i postgres-15-alpine.tar`

## 3. MinIO Docker Image
- **File**: `minio-latest.tar`
- **Version**: Latest
- **Architecture**: linux/amd64
- **Load command**: `docker load -i minio-latest.tar`

## 4. Redis Docker Image
- **File**: `redis-7-alpine.tar`
- **Version**: 7 Alpine
- **Architecture**: linux/amd64
- **Load command**: `docker load -i redis-7-alpine.tar`

## 5. Bun JavaScript Runtime
- **File**: `install-bun.sh`, `bun-linux-x64`
- **Version**: v1.1.38
- **Architecture**: Linux x64
- **Installation 1**: `chmod +x install-bun.sh && ./install-bun.sh`
- **Installation 2**: `chmod +x bun-linux-x64 && ./bun-linux-x64` (offline mode)

## 6. Docker Compose
- **File**: `install-docker-compose.sh`, `docker-compose-amd64`
- **Version**: v2.29.2 (included binary)
- **Architecture**: Linux x64
- **Installation**: `chmod +x install-docker-compose.sh && ./install-docker-compose.sh`
- **Note**: Offline installation using included binary

## 7. Traefik Reverse Proxy
- **File**: `traefik-v2.11.tar`, `docker-compose.yml`, `traefik.yml`
- **Version**: v2.11
- **Architecture**: linux/amd64
- **Load command**: `docker load -i traefik-v2.11.tar`
- **Features**: Pre-configured routing rules for mixed architecture (see README-MIXED-ARCHITECTURE.md)

## Architecture Overview

This package supports a **mixed architecture** setup:

- **Containerized Services**: PostgreSQL, Redis, MinIO, Traefik (managed by Docker)
- **External Services**: Navy, Wayne API, Wayne, Spydon, Spydon Frontend (run as local processes on localhost)

For detailed architecture and routing information, see [README-MIXED-ARCHITECTURE.md](README-MIXED-ARCHITECTURE.md).

## Installation Instructions

### Node.js
```bash
# Extract Node.js
sudo tar -xf nodejs-amd64.tar.xz -C /usr/local/
sudo ln -s /usr/local/node-v24.2.0-linux-x64/bin/node /usr/bin/node
sudo ln -s /usr/local/node-v24.2.0-linux-x64/bin/npm /usr/bin/npm
sudo ln -s /usr/local/node-v24.2.0-linux-x64/bin/npx /usr/bin/npx
```

### Bun
```bash
# Method 1: Use original installation script
chmod +x install-bun.sh
./install-bun.sh

# Method 2: Use offline installation script
chmod +x bun-linux-x64
./bun-linux-x64

# Method 3: Manual installation from zip (if available)
unzip bun-amd64.zip
sudo mkdir -p /usr/local/bun
sudo cp -r bun-linux-x64/* /usr/local/bun/
sudo ln -sf /usr/local/bun/bin/bun /usr/local/bin/bun
sudo ln -sf /usr/local/bun/bin/bunx /usr/local/bin/bunx
```

**Note**: The `bun-linux-x64` script downloads the latest Bun binary during installation. For completely offline environments, ensure you have the zip file available.

### Docker Compose
```bash
# Make script executable and run (Offline installation)
chmod +x install-docker-compose.sh
./install-docker-compose.sh

# Alternative: Manual installation
sudo chmod +x docker-compose-amd64
sudo mv docker-compose-amd64 /usr/local/bin/docker-compose
sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
```

**Note**: Docker Compose installation works completely offline using the included binary.

### Docker Images
```bash
# Load all Docker images
docker load -i postgres-15-alpine.tar
docker load -i minio-latest.tar
docker load -i redis-7-alpine.tar
docker load -i traefik-v2.11.tar

# Verify images are loaded
docker images
```

### Traefik Reverse Proxy Setup
```bash
# Load Traefik image first
docker load -i traefik-v2.11.tar

# Start all services with reverse proxy
docker-compose up -d

# Access services:
# - Traefik Dashboard: http://localhost:8080
# - Navy Service: http://localhost/navy (→ port 8081)
# - Wayne API: http://localhost/wayneapi (→ port 8082)
# - Wayne Service: http://localhost/wayne (→ port 8083)

# View routing rules
curl http://localhost:8080/api/http/routers
```

**Note**: See `README-TRAEFIK.md` for detailed Traefik configuration and routing rules.

### Quick Start with Docker Compose
Create a `docker-compose.yml` file:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: myapp
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin123
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

Start services:
```bash
docker-compose up -d
```

## Runtime Versions Check

After installation, you can verify versions:

```bash
# Node.js
node --version    # v24.2.0
npm --version     # Latest npm
npx --version

# Bun
bun --version     # v1.1.38
bunx --version

# Docker Compose
docker-compose --version  # Latest version

# Docker images
docker images | grep postgres
docker images | grep redis
docker images | grep minio
```

## Prerequisites

- **Docker**: Required for Docker Compose and Docker images
  ```bash
  # For online environments:
  curl -fsSL https://get.docker.com | sh
  sudo usermod -aG docker $USER

  # For offline environments: Docker must be pre-installed
  # This package does not include Docker Engine installation files
  ```
- **sudo access**: Required for system-wide installations

## Offline Installation Notes

This package is designed for **offline environments**:
- ✅ All binaries and images are included locally
- ✅ No internet connection required for installation
- ✅ Docker Compose uses the included binary (`docker-compose-amd64`)
- ✅ Docker images are pre-packaged as tar files
- ❌ Docker Engine must be pre-installed (not included in this package)
- ❌ Node.js and Bun require manual extraction (no network dependency)