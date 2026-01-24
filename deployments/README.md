# Deployments Directory

This directory contains deployment configurations and related files for containerization and CI/CD.

## Structure

```
deployments/
├── docker/
│   ├── Dockerfile.backend      # Backend Docker image definition
│   ├── Dockerfile.frontend     # Frontend Docker image definition
│   ├── docker-compose.yml      # Service orchestration
│   └── .dockerignore          # Docker build context exclusions
└── searxng/
    ├── docker-compose.yml      # SearXNG service orchestration
    ├── searxng/
    │   └── settings.yml        # SearXNG configuration
    ├── .env                    # Environment variables (gitignored)
    ├── .gitignore              # Exclude sensitive files
    └── README.md               # SearXNG deployment guide (中文)
```

## Docker Configuration

### `docker/`
AIGuides application containerization.

### `docker/Dockerfile.backend`
Multi-stage Docker build for the Go backend application.

**Features:**
- Based on Alpine Linux for minimal image size
- Multi-stage build (builder + runtime)
- Configuration via volume mount (not baked in)
- Exposes port 8080

### `docker/Dockerfile.frontend`
Multi-stage Docker build for the Next.js frontend application.

**Features:**
- Next.js standalone output mode
- Multi-stage build (deps + builder + runner)
- Non-root user for security
- Exposes port 3000

### `docker/docker-compose.yml`
Orchestrates frontend and backend services.

**Features:**
- Network isolation with dedicated bridge network
- Volume mounts for config and data persistence
- Service dependencies (frontend depends on backend)
- Auto-restart policies

### `searxng/`
Local SearXNG search engine deployment for providing real-time web search capabilities to AIGuides.

**Features:**
- Aggregates 3 major search engines (Google, Bing, DuckDuckGo)
- Completely free, no API key required
- Local deployment, no rate limits
- Redis caching for performance
- Chinese language optimized

**Quick Start:**
```bash
cd deployments/searxng
docker compose up -d
```

**Detailed Documentation:** [searxng/README.md](searxng/README.md) (中文)

## Building Images

From the project root:

```bash
# Build both images
make build

# Build individually
make build-backend
make build-frontend
```

## Running Locally

```bash
# Start services
docker compose -f deployments/docker/docker-compose.yml up -d

# View logs
docker compose -f deployments/docker/docker-compose.yml logs -f

# Stop services
docker compose -f deployments/docker/docker-compose.yml down
```

## CI/CD Integration

The GitHub Actions workflow (`.github/workflows/deploy.yml`) automatically:
1. Builds Docker images using these Dockerfiles
2. Saves images as tar files
3. Transfers to remote server
4. Deploys using docker-compose

See `docs/DEPLOYMENT.md` for complete deployment documentation.

## Configuration Requirements

Before deploying, ensure:
- Configuration file exists at `config/aiguide.yaml` (on server)
- Data directory exists at `data/` (on server)
- Required GitHub Secrets are configured
- Server has Docker and Docker Compose installed

## Related Documentation

- [DEPLOYMENT.md](../docs/DEPLOYMENT.md) - Complete deployment guide
- [CI_CD_SUMMARY.md](../docs/CI_CD_SUMMARY.md) - CI/CD architecture and details
