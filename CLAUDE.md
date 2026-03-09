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

# Testing
make test                  # Run Go unit tests
make lint                  # Run golangci-lint

# Individual steps
make frontend-install      # npm install in frontend/
make frontend-build        # Build frontend to frontend/dist/
make clean                 # Remove build artifacts
```

## Architecture

- `main.go` + `cmd/` - Cobra CLI with `serve` subcommand
- `internal/config/` - Configuration struct, flag/env loading
- `internal/k8s/` - Kubernetes client (typed + dynamic), GVR definitions for ZTP CRDs
- `internal/hub/` - Hub lifecycle, ManagedCluster operations, pipeline status aggregation
- `internal/ai/` - Ollama HTTP client, ZTP-specific prompt templates, SSE streaming
- `internal/api/` - HTTP server with go:embed, REST handlers, middleware
- `internal/ws/` - WebSocket hub, client, K8s watch bridge for ~9 CRD types
- `frontend/` - React 19 + TypeScript + Vite + Tailwind v4 + Zustand + framer-motion

## Key Patterns

- All K8s operations use `context.Context` with timeouts
- Dynamic client for ZTP CRDs (ManagedCluster, ClusterDeployment, ClusterInstance, BMH, Agent, ACI, InfraEnv, Policy, ArgoCD Application)
- WebSocket hub broadcasts K8s watch events to all connected browsers
- Events are also recorded in-memory for the Events page REST endpoint
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
