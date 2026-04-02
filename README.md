# YouTube Tracker

A backend service that collects and tracks statistics from YouTube channels (subscriber counts over time). It exposes a REST API for managing channels, triggering statistics collection jobs, and querying historical data.

## Features

- Track YouTube channel subscriber counts over time
- Background statistics collection with a configurable worker pool (default: 10 workers)
- Real-time job status polling
- Paginated channel and statistics listing
- Multi-channel notifications (Telegram, Slack)
- Kubernetes-native deployment with two environments (dev and prod)

## Architecture

```
REST API (ogen-generated)
    └── HTTP Handler
            └── Channel Service (business logic, transactions)
                    └── Channel Repository (PostgreSQL via pgx)

Background Jobs:
    Statistics Collector (job orchestrator)
            └── Worker Pool (10 workers)
                    ├── Channel Repository
                    └── YouTube Data API v3
```

### Key Components

| Component | Path | Description |
|-----------|------|-------------|
| HTTP Handler | `internal/http_handler/` | REST endpoints, CORS, request logging |
| Channel Service | `internal/youtube/stats/` | Business logic, transaction management |
| Channel Repository | `internal/youtube/stats/` | PostgreSQL data access via pgx |
| Statistics Collector | `internal/youtube/collector.go` | Background job orchestration |
| Statistics Worker | `internal/youtube/worker.go` | Per-channel stats fetcher |
| YouTube Client | `internal/youtube/client.go` | YouTube Data API v3 wrapper |
| Notifier | `internal/notify/` | Telegram / Slack / Noop alerts |
| API types | `internal/api/` | Auto-generated from `openapi.yaml` via ogen |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/status` | Health check (uptime, docker tag) |
| `POST` | `/youtube/channel` | Add a new channel to track |
| `GET` | `/youtube/channel/{id}` | Get channel by ID |
| `GET` | `/youtube/channel/{id}/statistics` | Get channel statistics history |
| `GET` | `/youtube/channels` | List channels (paginated) |
| `POST` | `/schedule` | Trigger statistics collection job |
| `GET` | `/schedule/job/{id}` | Poll job status |

The full API spec is in [`openapi.yaml`](openapi.yaml). Run `make generate` to regenerate Go types after any spec changes.

## Prerequisites

- Go 1.24+
- PostgreSQL 14+
- Docker (for integration tests and container builds)
- Kubernetes cluster (for deployment)
- YouTube Data API v3 key

## Database Schema

Schema is managed via [Atlas Operator](https://atlasgo.io/) in Kubernetes — see `k8s/atlas-schema/`.

## Build Commands

The following are the most commonly used commands. For the full list, see the [`Makefile`](Makefile).

```bash
make dep           # Download dependencies
make generate      # Regenerate API code from openapi.yaml (via ogen)
make test          # Run unit tests
make lint          # Run linter (golangci-lint)
make build         # Build Linux binary (runs generate, test, vet, lint)
```

## Running Locally

```bash
export DB_URL="postgres://user:password@localhost:5432/yttracker"
export GOOGLE_API_KEY="your-youtube-api-key"
export ALLOWED_ORIGIN="http://localhost:3000"

go run ./cmd/app/
```

## Testing

The project has four test categories controlled by Go build tags:

| Command | Tag | Description |
|---------|-----|-------------|
| `make test` | (none) | Unit tests — uses mocks, no external dependencies |
| `make database_test` | `database` | Repository tests against real PostgreSQL (testcontainers) |
| `make integration_test` | `integration,database` | Full app + database end-to-end tests |
| `make thirdparty_test` | `thirdparty` | Hits real YouTube API (requires credentials) |

Test coverage report: `make test_coverage`

## Deployment

### Prod Release Flow

1. Push a `prod-v*` tag (e.g., `prod-v2.1.0`)
2. GitHub Actions builds and pushes the Docker image to GHCR (`docker-build.yml`)
3. `deploy-prod.yml` automatically authenticates to GKE (via Workload Identity Federation) and applies the deployment

### Required GitHub Repository Variables (Prod)

| Variable | Description |
|----------|-------------|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Workload Identity Federation provider resource name |
| `GCP_SERVICE_ACCOUNT` | GCP service account email for deployment |

### Secrets Management

Secrets are managed via [Bitnami SealedSecrets](https://github.com/bitnami-labs/sealed-secrets). Each environment has its own set of sealed secrets.

```bash
# 1. Copy and fill a secret template
cp k8s/secrets-plain/postgres-secrets.yaml.example \
   k8s/secrets-plain/default/postgres-secrets.yaml
# Edit with real base64-encoded values

# 2. Seal the secrets
./seal-secrets.sh          # dev
./seal-secrets.sh prod     # prod

# 3. Commit files in k8s/sealed-secrets/<env>/
# 4. Deploy
make k8s                   # dev
./apply-k8s.sh prod        # prod
```

### Manual Deployment

```bash
# Dev
make k8s

# Prod
./apply-k8s.sh prod
```

`apply-k8s.sh` installs nginx Ingress Controller, Atlas Operator, and SealedSecrets controller automatically if not already present.

## CI/CD Workflows

| Workflow | Trigger | Description |
|----------|---------|-------------|
| `build.yml` | PR, push to main | Build, test, lint |
| `docker-build.yml` | Tags, PRs | Build & push Docker image to GHCR |
| `deploy-prod.yml` | `prod-v*` tag, manual | Deploy to GKE production |
| `deploy-infra.yml` | Manual | One-time GKE infrastructure setup |
| `claude-review.yml` | PR | AI-powered code review |