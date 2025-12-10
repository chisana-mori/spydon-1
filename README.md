# Spydon - Intelligent Alert Management Platform

![Spydon Logo](https://img.shields.io/badge/Spydon-Alert%20Management-blue?style=for-the-badge&logo=kubernetes)

A modern, intelligent alert management and root cause analysis platform for Kubernetes environments. Spydon provides real-time alert monitoring, AI-powered analysis, and comprehensive incident management capabilities.

## ✨ Features

### 🚨 **Alert Management**
- Real-time alert aggregation and visualization
- Multi-cluster alert correlation
- Severity-based alert prioritization
- Historical alert tracking and trend analysis
- Custom alert enrichment and tagging

### 🤖 **AI-Powered Root Cause Analysis**
- **HolmesGPT Integration**: Intelligent root cause analysis powered by AI
- Multi-source data analysis (Kubernetes API, Prometheus, Elasticsearch)
- Automated incident investigation and diagnosis
- Interactive chat interface for alert analysis
- Structured RCA reporting with actionable insights

### 📊 **Advanced Analytics & Visualization**
- Interactive dashboards with real-time metrics
- 30-day alert trend analysis
- Cluster health monitoring
- Custom reporting and alerting patterns
- Export capabilities for compliance and audit

### 🔐 **Enterprise Security**
- **CAS Single Sign-On**: Centralized authentication
- Role-based access control (RBAC)
- API key management for integrations
- Secure audit logging and compliance tracking

### 🏗️ **Infrastructure Management**
- Multi-cluster Kubernetes support
- MinIO object storage integration
- MySQL database for persistent data
- Redis caching for performance optimization
- Docker and Kubernetes deployment ready

## 🛠️ Tech Stack

### Backend
- **Go 1.23+** with Gin framework
- **MySQL** for data persistence
- **Redis** for caching and session management
- **MinIO** for object storage
- **Docker** & **Kubernetes** deployment

### Frontend
- **Next.js 16** with React 19
- **TypeScript** for type safety
- **Tailwind CSS** for responsive design
- **Radix UI** component library
- **Recharts** for data visualization
- **TanStack Query** for state management

### AI & Analytics
- **HolmesGPT** for intelligent analysis
- **Prometheus** integration for metrics
- **Elasticsearch** connectivity for log analysis

## 🚀 Quick Start

### Prerequisites

- [Docker](https://www.docker.com/get-started) and Docker Compose
- [Go](https://golang.org/doc/install) (version 1.23+)
- [Node.js](https://nodejs.org/en/download/) (version 18+) and npm
- [kubectl](https://kubernetes.io/docs/tasks/tools/) (for cluster deployment)
- [Helm](https://helm.sh/docs/intro/install/) (for cluster deployment)

### One-Click Docker Setup

The fastest way to get Spydon running is with Docker Compose:


# Clone the repository
git clone <repository-url>
cd robusta-web

# Configure environment
cp apps/backend/.env.example apps/backend/.env

# Start all services
make docker-run

# Or using docker compose directly
docker compose -f infrastructure/docker/docker-compose.yml up -d


Access your Spydon instance:
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **MinIO Console**: http://localhost:9001

### Development Mode

For local development with hot reload:


# Start dependencies
docker compose -f infrastructure/docker/docker-compose.yml up -d mysql minio redis

# Start backend
cd apps/backend
cp .env.example .env
go mod tidy
make run

# Start frontend (new terminal)
cd apps/frontend
npm install
npm run dev


## 📁 Project Structure


robusta-web/
├── apps/
│   ├── backend/           # Go backend API server
│   │   ├── cmd/          # Application entry points
│   │   ├── internal/     # Core application logic
│   │   │   ├── api/      # HTTP handlers and routes
│   │   │   ├── config/   # Configuration management
│   │   │   ├── db/       # Database operations
│   │   │   ├── models/   # Data models
│   │   │   └── services/ # Business logic
│   │   └── migrations/   # Database migrations
│   └── frontend/         # Next.js frontend application
│       ├── src/
│       │   ├── app/      # Next.js app router pages
│       │   ├── components/ # React components
│       │   └── lib/      # Utilities and configurations
│       └── public/       # Static assets
├── infrastructure/
│   ├── cas/             # CAS SSO configuration
│   ├── docker/          # Docker compose files
│   ├── k8s/             # Kubernetes manifests
│   └── scripts/         # Deployment scripts
└── packages/            # Shared packages and extensions


## 🔧 Configuration

Spydon uses environment variables for configuration. Key configuration options:

### Backend Configuration (apps/backend/.env)


# Database
DB_HOST=localhost
DB_PORT=3306
DB_NAME=robusta
DB_USER=robusta
DB_PASSWORD=password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# HolmesGPT
HOLMES_GPT_URL=http://localhost:8081
HOLMES_GPT_API_KEY=your-api-key
HOLMES_GPT_ENABLED=true

# CAS SSO
CAS_ENABLED=true
CAS_SERVER_URL=https://cas.example.com


### Frontend Configuration

The frontend configuration is managed through runtime JSON files in `apps/frontend/config/`.

## 🔗 Integrations

### Robusta Kubernetes Platform

Spydon is designed to integrate seamlessly with the Robusta Kubernetes monitoring platform:


# Add Robusta Helm repository
helm repo add robusta https://robusta-charts.storage.googleapis.com
helm repo update

# Create namespace
kubectl create namespace robusta

# Install Robusta with HolmesGPT
helm install robusta robusta/robusta -n robusta \
  -f infrastructure/scripts/robusta-holmesgpt-values-clean.yaml


### Alert Flow Integration


graph TD
    A[Prometheus] --> B[AlertManager]
    B --> C[Robusta Agent]
    C --> D[Spydon Backend]
    D --> E[Spydon Frontend]
    E --> F[HolmesGPT Analysis]
    F --> E


## 📊 Usage Examples

### Alert Management

1. **View Active Alerts**: Navigate to the dashboard for real-time alert overview
2. **Analyze Specific Alert**: Click on any alert to view detailed information
3. **Root Cause Analysis**: Use the HolmesGPT chat interface for AI-powered analysis
4. **Historical Trends**: Access the reports section for trend analysis

### HolmesGPT Integration


# Forward HolmesGPT service for local development
kubectl port-forward svc/holmesgpt -n robusta 8081:80


### API Usage


# Get all alerts
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/alerts

# Get cluster summary
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/clusters/summary

# Trigger RCA analysis
curl -X POST -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"alert_id": "alert-123"}' \
  http://localhost:8080/api/v1/rca/analyze


## 🚀 Deployment

### Kubernetes Deployment


# Apply configurations
kubectl apply -f infrastructure/k8s/

# Deploy application
./infrastructure/scripts/deploy.sh


### Production Considerations

- **High Availability**: Deploy multiple replicas for frontend and backend
- **Database**: Use managed MySQL service for production
- **Storage**: Configure persistent MinIO storage
- **Monitoring**: Enable Prometheus metrics collection
- **Security**: Configure proper SSL/TLS certificates

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Guidelines

- Follow Go conventions for backend code
- Use TypeScript strictly for frontend development
- Write unit tests for new features
- Update documentation for API changes
- Ensure code passes linting and type checking

### Code Quality


# Backend
go mod tidy
go fmt ./...
go test ./...

# Frontend
npm run lint
npm run type-check
npm run test


## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: Check our comprehensive documentation
- **Issues**: Report bugs and request features via GitHub Issues
- **Discussions**: Join our community discussions
- **Email**: Contact our support team for enterprise inquiries

## 🎯 Roadmap

- [ ] **Mobile Application**: Native mobile apps for iOS and Android
- [ ] **Enhanced AI Models**: Advanced machine learning capabilities
- [ ] **Multi-Cloud Support**: Integration with AWS, GCP, Azure
- [ ] **Advanced Analytics**: Predictive alert analysis
- [ ] **Slack/Teams Integration**: Native chat platform integration
- [ ] **Custom Dashboards**: Drag-and-drop dashboard builder

---

**Built with ❤️ for the Kubernetes community**

*Spydon transforms alert management from reactive to proactive with intelligent analysis and automation.*
