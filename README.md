# Task Manager API - DevOps CI/CD Project

A production-grade Go REST API with comprehensive CI/CD pipeline demonstrating DevSecOps best practices.

## Project Overview

This project implements a Task Manager API with:
- CRUD operations for tasks
- PostgreSQL database persistence
- Health check endpoint
- Comprehensive CI/CD pipelines using GitHub Actions

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CI/CD Pipeline                           │
├─────────────────────────────────────────────────────────────────┤
│  CI Pipeline                                                    │
│  ┌─────┐  ┌──────┐  ┌────┐  ┌─────┐  ┌──────┐  ┌─────────────┐ │
│  │Lint │→ │ SAST │→ │SCA │→ │Test │→ │Build │→ │Docker Build │ │
│  └─────┘  └──────┘  └────┘  └─────┘  └──────┘  └─────────────┘ │
│                                                       ↓         │
│  ┌────────────┐  ┌───────────────┐  ┌──────────────────────────┐│
│  │Image Scan  │→ │Container Test │→ │Registry Push (DockerHub) ││
│  └────────────┘  └───────────────┘  └──────────────────────────┘│
├─────────────────────────────────────────────────────────────────┤
│  CD Pipeline                                                    │
│  ┌────────────┐  ┌────────┐  ┌──────────────┐  ┌─────────────┐ │
│  │Setup K8s   │→ │Deploy  │→ │Health Check  │→ │DAST (ZAP)   │ │
│  │(kind)      │  │        │  │              │  │             │ │
│  └────────────┘  └────────┘  └──────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/tasks` | List all tasks |
| GET | `/api/v1/tasks/{id}` | Get task by ID |
| POST | `/api/v1/tasks` | Create new task |
| PUT | `/api/v1/tasks/{id}` | Update task |
| DELETE | `/api/v1/tasks/{id}` | Delete task |

## Prerequisites

- Go 1.24+
- Docker & Docker Compose
- kubectl (for K8s deployment)
- kind (for local K8s cluster)

## Development

### Run Tests

```bash
go test -v ./...
```

### Build

```bash
go build -o taskmanager ./cmd/server
```

## CI/CD Pipeline

### CI Pipeline Stages

| Stage | Tool | Why It Matters |
|-------|------|----------------|
| **Linting** | golangci-lint | Enforces coding standards, prevents technical debt |
| **SAST** | GitHub CodeQL | Detects OWASP Top 10 vulnerabilities in code |
| **SCA** | govulncheck | Identifies vulnerable dependencies (supply-chain risks) |
| **Unit Tests** | go test | Validates business logic, prevents regressions |
| **Build** | go build | Compiles application binary |
| **Docker Build** | docker/build-push-action | Creates container image |
| **Image Scan** | Trivy | Detects OS & library vulnerabilities in container |
| **Container Test** | docker run + curl | Validates container is runnable and healthy |
| **Registry Push** | docker/build-push-action | Publishes trusted image to DockerHub |

### CD Pipeline Stages

| Stage | Tool | Why It Matters |
|-------|------|----------------|
| **Setup K8s** | kind | Creates isolated K8s environment for testing |
| **Deploy** | kubectl apply | Infrastructure as Code, GitOps approach |
| **Health Check** | curl | Validates deployment works end-to-end |
| **DAST** | OWASP ZAP | Tests running application for security issues |

## GitHub Secrets Configuration

Configure these secrets in your GitHub repository (Settings → Secrets and variables → Actions):

| Secret | Description | Example |
|--------|-------------|---------|
| `DOCKERHUB_USERNAME` | Your DockerHub username | `myusername` |
| `DOCKERHUB_TOKEN` | DockerHub access token | |
| `DB_USER` | Database username | `postgres` |
| `DB_PASSWORD` | Database password | `<strong-password>` |
| `DB_NAME` | Database name | `taskmanager` |

### Creating DockerHub Token

1. Go to [DockerHub](https://hub.docker.com/)
2. Account Settings → Security → New Access Token
3. Copy the token and add it as `DOCKERHUB_TOKEN` secret

## Local Testing

### Option 1: Docker Compose (Quickest)

```bash
docker-compose up --build
```

Test the API:
```bash
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Test task"}'
curl http://localhost:8080/api/v1/tasks
```

### Option 2: Kubernetes with kind

```bash
# Install kind (if not installed)
brew install kind

# Create cluster
kind create cluster --name taskmanager

# Build and load image into kind
docker build -t taskmanager:local .
kind load docker-image taskmanager:local --name taskmanager

# Deploy namespace and config
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml

# Create secret (not stored in git)
kubectl create secret generic taskmanager-secret --namespace=taskmanager --from-literal=DATABASE_URL="postgres://postgres:postgres@postgres-service:5432/taskmanager?sslmode=disable" --from-literal=POSTGRES_USER="postgres" --from-literal=POSTGRES_PASSWORD="postgres" --from-literal=POSTGRES_DB="taskmanager"

# Deploy PostgreSQL
kubectl apply -f k8s/postgres.yaml

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgres -n taskmanager --timeout=120s

# Deploy application (with local image)
sed 's|image: taskmanager:latest|image: taskmanager:local|' k8s/deployment.yaml | kubectl apply -f -
kubectl apply -f k8s/service.yaml

# Patch for local images (kind doesn't pull from registry)
kubectl patch deployment taskmanager -n taskmanager -p '{"spec":{"template":{"spec":{"containers":[{"name":"taskmanager","imagePullPolicy":"Never"}]}}}}'

# Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=taskmanager -n taskmanager --timeout=120s

# Port forward to access
kubectl port-forward svc/taskmanager-service 8080:8080 -n taskmanager &

# Test
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/v1/tasks -H "Content-Type: application/json" -d '{"title":"Test from kind"}'
```

### Cleanup

```bash
# Docker Compose
docker-compose down -v

# kind
kind delete cluster --name taskmanager
```

## Project Structure

```
devops_ci_cd/
├── .github/
│   └── workflows/
│       ├── ci.yml              # CI Pipeline
│       └── cd.yml              # CD Pipeline
├── .zap/
│   └── rules.tsv               # ZAP scan rules
├── cmd/
│   └── server/
│       └── main.go             # Application entry point
├── internal/
│   ├── handler/
│   │   ├── task.go             # HTTP handlers
│   │   └── task_test.go        # Handler tests
│   ├── model/
│   │   └── task.go             # Data models
│   └── store/
│       ├── postgres.go         # PostgreSQL store
│       └── postgres_test.go    # Store tests
├── k8s/
│   ├── namespace.yaml          # K8s Namespace
│   ├── configmap.yaml          # App configuration
│   ├── secret.yaml.example     # Secret template (copy and fill in)
│   ├── postgres.yaml           # PostgreSQL StatefulSet
│   ├── deployment.yaml         # App Deployment
│   └── service.yaml            # K8s Service
├── Dockerfile                  # Multi-stage Docker build
├── docker-compose.yml          # Local development setup
├── .golangci.yml              # Linting configuration
├── go.mod
├── go.sum
└── README.md
```

## DevSecOps Principles

This project demonstrates key DevSecOps principles:

### Shift-Left Security
- Security checks run early in the pipeline
- CodeQL (SAST) detects code vulnerabilities
- govulncheck (SCA) identifies dependency risks
- Trivy scans container images for CVEs

### Fail-Fast
- Linting runs first to catch style issues
- Tests run before building artifacts
- Security scans gate the deployment

### Defense in Depth
- Multiple security scanning tools
- Container runs as non-root user
- Distroless base image minimizes attack surface
- Kubernetes security contexts applied

### Infrastructure as Code
- All configuration in version control
- Kubernetes manifests define infrastructure
- Reproducible deployments

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | 8080 |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | info |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Run linting: `golangci-lint run`
6. Submit a pull request

## License

MIT License
