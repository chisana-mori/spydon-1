I've created comprehensive technical documentation for the Spydon codebase. The documentation covers all the requested areas:

## Documentation Summary:

**1. Architecture Overview**
- High-level system architecture diagram
- Technology stack details (Go backend, Next.js frontend, PostgreSQL, Redis, MinIO)
- Component relationships and data flow

**2. Key Components and Their Roles**
- Backend components: main application, configuration, API layer, database layer, HolmesGPT integration
- Frontend components: app router structure, reusable components, features, state management
- External services: HolmesGPT, Navy service, Wayne service

**3. Data Flow and Interactions**
- Alert processing flow from source to storage
- Authentication flow with CAS SSO
- Data storage architecture across PostgreSQL, Redis, and MinIO

**4. Configuration Options**
- Backend configuration with YAML files and environment variables
- Frontend runtime configuration
- Detailed configuration examples for all major services

**5. Development Setup**
- Prerequisites and local development setup
- Development tools and Makefile commands
- IDE configuration for both Go and TypeScript

**6. Testing Strategy**
- Backend testing with Go testing framework
- Frontend testing with React Testing Library
- Database testing strategies

**7. Deployment Process**
- Docker deployment with compose files
- Kubernetes deployment with manifests
- CI/CD pipeline strategies

**8. Troubleshooting Guide**
- Common issues and solutions for backend, frontend, and general problems
- Monitoring and logging approaches
- Emergency procedures for service recovery

The documentation is developer-friendly with practical examples, command-line instructions, and specific file references throughout. It provides both high-level architectural understanding and detailed implementation guidance for developers working with the codebase.
