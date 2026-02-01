# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Youtube Tracker is an backend that downloads statistics information from different user-generated content sources (initially from Youtube).

## Build Commands

```bash
# Generate API code from OpenAPI spec (via ogen)
make generate

# Run unit tests
make test

# Build Linux binary (runs generate, test, vet, lint first)
make build

# Full build for all platforms (linux, darwin, windows) + integration tests
make full_build

# Build Docker image
make docker-build

# Download dependencies
make dep

# Run linter
make lint

# Clean build artifacts
make clean

# Generate Atlas schema and deploy to k8s
make k8s
```

## Architecture

### Layer Structure

```
HTTP Handler (internal/http_handler)    - REST API endpoints, request/response mapping
    ↓
Service (internal/youtube/stats)        - Business logic, transaction management
    ↓
Repository (internal/youtube/stats)     - Database access, SQL queries
    ↓
PostgreSQL                              - Data persistence

Collector (internal/youtube)            - Background job orchestration
    ↓
Worker (internal/youtube)               - Individual channel statistics fetching
    ↓
YouTube Client (internal/youtube)       - YouTube Data API integration
```

### Key Components

- **HTTP Handler** (`internal/http_handler/handler.go`): Implements `ogen`-generated server interface. Handles REST endpoints for channels, statistics, and job scheduling.

- **Channel Service** (`internal/youtube/stats/channel_service.go`): Business logic for channel CRUD operations and statistics retrieval. Manages transactions via `helpers.RunInTx()`.

- **Channel Repository** (`internal/youtube/stats/channel_repository.go`): Database access layer using `pgx`. Handles channel and statistics persistence.

- **Statistics Collector** (`internal/youtube/collector.go`): Orchestrates background statistics collection jobs. Manages worker pool (10 workers by default) and job status tracking.

- **Statistics Worker** (`internal/youtube/worker.go`): Fetches statistics for a single channel via YouTube API and stores results.

- **YouTube Client** (`internal/youtube/client.go`): Wraps YouTube Data API v3 for channel lookups and statistics retrieval.

### API Generation

- OpenAPI spec generates server/client code via `ogen` to `internal/api/`
- Run `make generate` to regenerate API types and handlers

## Frontend

The frontend for this project is in a separate repository: https://github.com/sergey-podunov/statistics-tracker-frontend

## Testing

The project uses multiple test categories controlled by Go build tags:

### Test Commands

```bash
# Run unit tests (no external dependencies)
make test

# Run database tests (requires Docker for testcontainers)
make database_test

# Run integration tests (full app + database, requires Docker)
make integration_test

# Run third-party tests (hits real YouTube API, requires API credentials)
make thirdparty_test

# Generate test coverage report
make test_coverage
```

### Running a Single Test

```bash
# Unit test
go test -v -run TestName ./internal/...

# Database test
go test -v -tags=database -run TestName ./internal/...

# Integration test
go test -v -tags=integration,database -run TestName ./cmd/app/...
```

### Test Categories

- **Unit tests** - No build tags required. Use mocks for external dependencies (YouTube client, database).
- **Database tests** (`//go:build database`) - Test repository layer against real PostgreSQL using testcontainers. Each test runs in a transaction that gets rolled back.
- **Integration tests** (`//go:build integration`) - Full end-to-end tests that spin up the HTTP server and PostgreSQL container, testing complete API flows.
- **Third-party tests** (`//go:build thirdparty`) - Hit real external APIs (YouTube). Require valid API credentials in environment.

### Test Patterns

- Repository tests extend `BaseChannelRepoTestSuite` which handles PostgreSQL container lifecycle and transaction isolation
- Integration tests use `helpers.CreatePostgresContainer()` for database setup
- YouTube client is mocked in integration tests via `MockYoutubeClient`

## Secrets Management

Secrets are managed via [Bitnami SealedSecrets](https://github.com/bitnami-labs/sealed-secrets). Each environment (default, prod) has its own set of sealed secrets with different credentials. The SealedSecrets controller is installed automatically by `apply-k8s.sh`.

- `seal-secrets.sh` - Helper script to encrypt secrets using `kubeseal`
- `k8s/secrets-plain/*.example` - Shared plain secret templates (committed)
- `k8s/secrets-plain/default/` - Plain secrets for dev environment (gitignored)
- `k8s/secrets-plain/prod/` - Plain secrets for prod environment (gitignored)
- `k8s/sealed-secrets/default/` - Encrypted secrets for dev (safe to commit)
- `k8s/sealed-secrets/prod/` - Encrypted secrets for prod (safe to commit)

### Workflow

1. Copy `.example` templates into the target environment directory and fill in values:
   ```bash
   cp k8s/secrets-plain/postgres-secrets.yaml.example k8s/secrets-plain/default/postgres-secrets.yaml
   # Edit the file with real base64-encoded values
   ```
2. Run `./seal-secrets.sh` (or `./seal-secrets.sh prod` for prod)
3. Commit the generated files in `k8s/sealed-secrets/<env>/`
4. Deploy with `make k8s` or `./apply-k8s.sh [default|prod]`

## Environment Requirements
- Go 1.24+
- k8s installed and running (for integration tests)

## Custom Agents

### Verify Agent (`/verify`)

Use the verify agent to check if your changes are valid before committing:

```
/verify
```

This agent will:
1. Run `make generate` to ensure definitions are current
2. Run `go build ./...` to verify compilation
3. Run `go test ./internal/...` to execute tests

The agent reports success/failure for each step and suggests fixes for any issues.

Always run the verify agent after completing code changes to validate the build and tests.

## Misc
Don't implement tests when you are asked to implement code.
Don't implement code when you are asked to implement tests.

Always use Context7 MCP when I need library/API documentation, code generation, setup or configuration steps without me having to explicitly ask.
Add files to git if they should be under version control.
