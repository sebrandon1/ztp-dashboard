# API Reference

## Authentication

When the server runs with `--auth-token` / `ZTP_AUTH_TOKEN`, all `/api/*` and
`/ws/*` routes require `Authorization: Bearer <token>`. Requests without a valid
token get `401` with `{"success":false,"error":"unauthorized"}`. `/healthz`,
`/readyz` and the SPA are always public.

```bash
curl -H "Authorization: Bearer $ZTP_AUTH_TOKEN" localhost:8080/api/status
```

Browsers cannot set headers on a WebSocket handshake, so `/ws/watch` also
accepts the token as a `token` query parameter:

```
ws://localhost:8080/ws/watch?token=<token>
```

## REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/status` | Hub connection status |
| `GET` | `/api/clusters` | List managed clusters |
| `GET` | `/api/clusters/:name` | Cluster detail |
| `GET` | `/api/clusters/:name/pipeline` | Pipeline status |
| `GET` | `/api/clusters/:name/policies` | Cluster policies |
| `DELETE` | `/api/clusters/:name` | Delete the ManagedCluster (detaches from ACM; body must confirm the name) |
| `GET` | `/api/argocd/applications` | ArgoCD applications |
| `GET` | `/api/events` | Recent events |
| `GET` | `/api/events/export` | Stream all matching events as `?format=csv` or `?format=json` (NDJSON) |
| `GET` | `/api/ai/status` | Ollama status |
| `GET` | `/api/ai/models` | Available models |
| `POST` | `/api/ai/diagnose` | AI diagnosis (SSE stream; `429` when the AI concurrency or per-minute limit is reached) |

## WebSocket

| Path | Description |
|------|-------------|
| `GET` | `/ws/watch` | WebSocket event stream — broadcasts K8s watch events to all connected clients |
