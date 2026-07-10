# CLAUDE.md

## Project Overview

ZTP Hub/Spoke Manager Dashboard - a purpose-built React + Go dashboard for managing ZTP provisioning pipelines. Single Go binary with embedded React SPA, WebSocket-driven updates, dynamic K8s client for ZTP CRDs, AI-powered diagnostics via ollama.

## Common Commands

```bash
# Build everything (frontend + backend)
make build

# Development
make frontend-dev          # Start Vite dev server (port 5173, proxies to :8080)
make run                   # Build and run the Go binary
make stop                  # Kill any running dashboard server processes

# Testing
make test                  # Run Go unit tests
make lint                  # Run golangci-lint
make frontend-lint         # Run frontend ESLint

# Individual steps
make frontend-install      # npm install in frontend/
make frontend-build        # Build frontend to frontend/dist/
make embed-frontend        # Copy frontend dist to embed location
make clean                 # Remove build artifacts

# Container
make docker-build          # Build container image (ghcr.io/sebrandon1/ztp-dashboard)

# Help
make help                  # Show all available targets
```

## Architecture

- `main.go` + `cmd/` - Cobra CLI with `serve` subcommand
- `internal/config/` - Configuration struct, flag/env loading
- `internal/k8s/` - Kubernetes client (typed + dynamic), GVR definitions for ZTP CRDs, ClientPool for spoke connections (LRU eviction, TTL-based expiry)
- `internal/hub/` - Hub lifecycle, ManagedCluster operations, pipeline status aggregation
- `internal/spoke/` - Spoke cluster service layer; connects to spoke K8s APIs via hub kubeconfig secrets, queries ClusterVersion, nodes, and ClusterOperators for health status
- `internal/store/` - SQLite-backed event persistence (modernc.org/sqlite, WAL mode); searchable/filterable event log with background purge loop (7-day retention, hourly sweep)
- `internal/ai/` - Ollama HTTP client, ZTP-specific prompt templates, SSE streaming
- `internal/api/` - HTTP server with go:embed, REST handlers, middleware
- `internal/ws/` - WebSocket hub, client, K8s watch bridge for ~9 CRD types
- `internal/argocd/` - ArgoCD health monitoring (placeholder, not yet implemented)
- `internal/bmc/` - BMC event management (placeholder, not yet implemented)
- `internal/ztp/` - ZTP-specific logic (placeholder, not yet implemented)
- `frontend/` - React 19 + TypeScript + Vite + Tailwind v4 + Zustand + framer-motion

## Key Patterns

- All K8s operations use `context.Context` with timeouts
- Dynamic client for ZTP CRDs (ManagedCluster, ClusterDeployment, ClusterInstance, BMH, Agent, ACI, InfraEnv, Policy, ArgoCD Application)
- WebSocket hub broadcasts K8s watch events to all connected browsers
- Events are persisted to SQLite (ztp-events.db) with WAL mode; background purge loop removes events older than 7 days
- Spoke health queries use a ClientPool (max 20 clients, 10-min TTL) that builds K8s clients from hub kubeconfig secrets
- AI diagnostics use SSE streaming from Go backend to browser via ollama
- Frontend uses Zustand for state, axios for API calls, custom WebSocket hook
- Single binary: `go:embed all:frontend_dist` serves the React SPA
- Dark theme with Tailwind v4 custom theme tokens

## Flags

- `--kubeconfig` / `$KUBECONFIG` / `~/.kube/config`
- `--port` / `8080`
- `--ollama-endpoint` / `$OLLAMA_ENDPOINT` / `http://localhost:11434`
- `--ollama-model` / `$OLLAMA_MODEL` / `llama3.1`
- `--log-format` / `$LOG_FORMAT` / `text`
