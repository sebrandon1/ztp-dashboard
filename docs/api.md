# API Reference

## REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/status` | Hub connection status |
| `GET` | `/api/clusters` | List managed clusters |
| `GET` | `/api/clusters/:name` | Cluster detail |
| `GET` | `/api/clusters/:name/pipeline` | Pipeline status |
| `GET` | `/api/clusters/:name/policies` | Cluster policies |
| `GET` | `/api/argocd/applications` | ArgoCD applications |
| `GET` | `/api/events` | Recent events |
| `GET` | `/api/ai/status` | Ollama status |
| `GET` | `/api/ai/models` | Available models |
| `POST` | `/api/ai/diagnose` | AI diagnosis (SSE stream) |

## WebSocket

| Path | Description |
|------|-------------|
| `GET` | `/ws/watch` | WebSocket event stream — broadcasts K8s watch events to all connected clients |
