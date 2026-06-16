# Configuration

## CLI Flags and Environment Variables

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--kubeconfig` | `KUBECONFIG` | `~/.kube/config` | Path to hub cluster kubeconfig |
| `--port` | — | `8080` | HTTP server port |
| `--ollama-endpoint` | `OLLAMA_ENDPOINT` | `http://localhost:11434` | Ollama API endpoint |
| `--ollama-model` | `OLLAMA_MODEL` | `llama3.1` | Default ollama model for AI diagnostics |
| `--log-format` | `LOG_FORMAT` | `text` | Log format (`text` or `json`) |

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

# Run
docker run -p 8080:8080 \
  -v ~/.kube/config:/kubeconfig:ro \
  ztp-dashboard:latest \
  serve --kubeconfig=/kubeconfig
```
