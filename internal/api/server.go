package api

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/sebrandon1/ztp-dashboard/internal/ai"
	"github.com/sebrandon1/ztp-dashboard/internal/config"
	"github.com/sebrandon1/ztp-dashboard/internal/hub"
	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
	"github.com/sebrandon1/ztp-dashboard/internal/ws"
)

//go:embed all:frontend_dist
var frontendFS embed.FS

type Server struct {
	k8sClient  *k8s.Client
	hubManager *hub.Manager
	aiClient   *ai.Client
	wsHub      *ws.Hub
	cfg        config.Config
}

func NewServer(cfg config.Config, k8sClient *k8s.Client, hubManager *hub.Manager, aiClient *ai.Client, wsHub *ws.Hub) *Server {
	return &Server{
		k8sClient:  k8sClient,
		hubManager: hubManager,
		aiClient:   aiClient,
		wsHub:      wsHub,
		cfg:        cfg,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health probes
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)

	// Hub status
	mux.HandleFunc("GET /api/status", s.handleHubStatus)

	// Clusters
	mux.HandleFunc("GET /api/clusters", s.handleListClusters)
	mux.HandleFunc("GET /api/clusters/{name}", s.handleGetCluster)
	mux.HandleFunc("GET /api/clusters/{name}/pipeline", s.handleGetPipeline)

	// Policies
	mux.HandleFunc("GET /api/clusters/{name}/policies", s.handleGetPolicies)

	// ArgoCD
	mux.HandleFunc("GET /api/argocd/applications", s.handleGetArgoApplications)

	// AI
	mux.HandleFunc("POST /api/ai/diagnose", s.handleAIDiagnose)
	mux.HandleFunc("GET /api/ai/status", s.handleAIStatus)
	mux.HandleFunc("GET /api/ai/models", s.handleAIModels)

	// Events (returns recent events for the event log page)
	mux.HandleFunc("GET /api/events", s.handleGetEvents)

	// WebSocket
	mux.HandleFunc("GET /ws/watch", s.handleWebSocket)

	// SPA
	mux.Handle("/", spaHandler())

	handler := recoveryMiddleware(loggingMiddleware(corsMiddleware(mux)))
	return handler
}

func spaHandler() http.Handler {
	distFS, err := fs.Sub(frontendFS, "frontend_dist")
	if err != nil {
		slog.Warn("embedded frontend not available", "error", err)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body>
				<h1>ZTP Dashboard</h1>
				<p>Frontend not built. Run <code>make frontend-build</code> first.</p>
			</body></html>`))
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path != "/" && !strings.HasPrefix(path, "/api") && !strings.HasPrefix(path, "/ws") {
			f, err := distFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
