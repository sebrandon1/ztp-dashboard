# Configuration

## CLI Flags and Environment Variables

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--kubeconfig` | `KUBECONFIG` | `~/.kube/config` | Path to hub cluster kubeconfig |
| `--port` | — | `8080` | HTTP server port |
| `--ollama-endpoint` | `OLLAMA_ENDPOINT` | `http://localhost:11434` | Ollama API endpoint |
| `--ollama-model` | `OLLAMA_MODEL` | `llama3.1` | Default ollama model for AI diagnostics |
| `--log-format` | `LOG_FORMAT` | `text` | Log format (`text` or `json`) |
| `--db-path` | `ZTP_DB_PATH` | `ztp-events.db` | Path to the SQLite event database; parent directories are created if missing |
| `--allowed-origins` | `ZTP_ALLOWED_ORIGINS` | — (same-origin only) | Comma-separated extra origins allowed to call the API and open the WebSocket |

## Running with Ollama AI

```bash
# Start ollama (if not already running)
ollama serve

# Pull a model
ollama pull llama3.1

# Run the dashboard with AI enabled
./bin/ztp-dashboard serve \
  --kubeconfig=~/.kube/config \
  --ollama-endpoint=http://localhost:11434 \
  --ollama-model=llama3.1
```

## Docker

```bash
# Build container image
make docker-build

# Run with a named volume so events survive restarts
docker run -p 8080:8080 \
  -v ~/.kube/config:/kubeconfig:ro \
  -v ztp-data:/data \
  ztp-dashboard:latest \
  serve --kubeconfig=/kubeconfig
```

The image runs as the nonroot user (uid 65532) and writes the event database to
`/data/ztp-events.db` (`ZTP_DB_PATH`). Mount a volume at `/data` for persistence;
without one, events are lost when the container is replaced. To store the database
elsewhere, override `--db-path` or `ZTP_DB_PATH` and mount that directory.

## Cross-Origin Access

The API and WebSocket accept requests from the dashboard's own origin (the `Host`
the request arrives on) and reject other browser origins with `403`. Requests
without an `Origin` header, such as `curl`, are allowed.

Add origins to `--allowed-origins` when the UI is served from a different origin
than the backend API, for example behind a reverse proxy or Route whose public
host differs from the backend `Host` header:

```bash
./bin/ztp-dashboard serve \
  --allowed-origins=https://dashboard.example.com,http://localhost:5173
```

`make frontend-dev` proxies through Vite with `changeOrigin`, so the backend sees
`Origin: http://localhost:5173` while `Host` is `localhost:8080`; include
`http://localhost:5173` to use the dev server against a locally running backend.
