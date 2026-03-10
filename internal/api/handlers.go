package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sebrandon1/ztp-dashboard/internal/ai"
	"github.com/sebrandon1/ztp-dashboard/internal/hub"
	"github.com/sebrandon1/ztp-dashboard/internal/ws"
)

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.k8sClient == nil {
		writeError(w, http.StatusServiceUnavailable, "not connected to cluster")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) handleHubStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.hubManager.GetStatus())
}

func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	clusters, err := s.hubManager.ListManagedClusters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clusters)
}

func (s *Server) handleGetCluster(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cluster, err := s.hubManager.GetManagedCluster(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cluster)
}

func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pipeline, err := s.hubManager.GetPipelineStatus(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pipeline)
}

func (s *Server) handleGetPolicies(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	policies, err := s.hubManager.GetClusterPolicies(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (s *Server) handleGetArgoApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := s.hubManager.GetArgoApplications(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) handleGetArgoApplication(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	app, err := s.hubManager.GetArgoApplication(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) handleSyncArgoApplication(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var opts hub.SyncOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		// Body is optional; default options are fine
		opts = hub.SyncOptions{}
	}

	if err := s.hubManager.SyncArgoApplication(r.Context(), name, opts); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sync triggered", "application": name})
}

func (s *Server) handleRefreshArgoApplication(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.hubManager.RefreshArgoApplication(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "refresh triggered", "application": name})
}

func (s *Server) handleSetPolicyState(w http.ResponseWriter, r *http.Request) {
	clusterName := r.PathValue("name")
	policyName := r.PathValue("policyName")

	var body struct {
		Disabled bool `json:"disabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.hubManager.SetPolicyDisabled(r.Context(), clusterName, policyName, body.Disabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	action := "enabled"
	if body.Disabled {
		action = "disabled"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": action, "policy": policyName})
}

func (s *Server) handleGetResource(w http.ResponseWriter, r *http.Request) {
	group := r.PathValue("group")
	version := r.PathValue("version")
	resource := r.PathValue("resource")
	namespace := r.PathValue("namespace")
	name := r.PathValue("name")

	obj, err := s.hubManager.GetResource(r.Context(), group, version, resource, namespace, name)
	if err != nil {
		if err.Error() == fmt.Sprintf("resource type %s/%s/%s is not allowed", group, version, resource) {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *Server) handleGetPolicySummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.hubManager.GetPolicySummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleGetArgoSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.hubManager.GetArgoSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		ConfirmName string `json:"confirmName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.ConfirmName != name {
		writeError(w, http.StatusBadRequest, "cluster name confirmation does not match")
		return
	}

	if err := s.hubManager.DeleteManagedCluster(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "cluster": name})
}

func (s *Server) handleDetachCluster(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		ConfirmName string `json:"confirmName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.ConfirmName != name {
		writeError(w, http.StatusBadRequest, "cluster name confirmation does not match")
		return
	}

	if err := s.hubManager.DetachManagedCluster(r.Context(), name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "detached", "cluster": name})
}

func (s *Server) handleListClusterResources(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	resources, err := s.hubManager.ListClusterResources(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resources)
}

func (s *Server) handleGetClusterHealth(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	health, err := s.spokeService.GetHealth(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, health)
}

func (s *Server) handleGetClusterNodes(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	nodes, err := s.spokeService.GetNodes(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleGetClusterOperators(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	operators, err := s.spokeService.GetOperators(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, operators)
}

func (s *Server) handleAIStatus(w http.ResponseWriter, _ *http.Request) {
	connected, err := s.aiClient.GetStatus()
	result := map[string]interface{}{
		"connected":    connected,
		"endpoint":     s.cfg.OllamaEndpoint,
		"defaultModel": s.aiClient.DefaultModel(),
	}
	if err != nil {
		result["error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAIModels(w http.ResponseWriter, _ *http.Request) {
	models, err := s.aiClient.ListModels()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

type diagnoseRequest struct {
	Context map[string]interface{} `json:"context"`
	Model   string                 `json:"model"`
	Type    string                 `json:"type"`
}

func (s *Server) handleAIDiagnose(w http.ResponseWriter, r *http.Request) {
	var req diagnoseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	clusterName, _ := req.Context["clusterName"].(string)
	var prompt string
	switch req.Type {
	case "provisioning":
		prompt = ai.ProvisioningErrorPrompt(clusterName, req.Context["conditions"])
	case "health":
		prompt = ai.ClusterHealthPrompt(clusterName, req.Context["healthData"])
	case "policy":
		prompt = ai.PolicyCompliancePrompt(clusterName, req.Context["policies"])
	case "bmc":
		prompt = ai.BMCErrorPrompt(clusterName, req.Context["bmcData"])
	default:
		prompt = ai.GeneralDiagnosePrompt(req.Context)
	}

	reader, err := s.aiClient.GenerateStream(req.Model, prompt)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("ollama error: %v", err))
		return
	}

	// SSE streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	err = ai.StreamTokens(reader, func(token string, done bool) error {
		data := map[string]interface{}{
			"token": token,
			"done":  done,
		}
		jsonData, _ := json.Marshal(data)
		_, writeErr := fmt.Fprintf(w, "data: %s\n\n", jsonData)
		if writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})
	if err != nil {
		slog.Error("streaming error", "error", err)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws.ServeWS(s.wsHub, w, r)
}

// Event log - stores recent events in memory
var (
	recentEvents   []ws.WatchEvent
	recentEventsMu sync.RWMutex
	maxEvents      = 500
)

func init() {
	recentEvents = make([]ws.WatchEvent, 0, maxEvents)
}

// RecordEvent stores an event in the recent events buffer
func RecordEvent(event ws.WatchEvent) {
	recentEventsMu.Lock()
	defer recentEventsMu.Unlock()
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	recentEvents = append(recentEvents, event)
	if len(recentEvents) > maxEvents {
		recentEvents = recentEvents[len(recentEvents)-maxEvents:]
	}
}

func (s *Server) handleGetEvents(w http.ResponseWriter, _ *http.Request) {
	recentEventsMu.RLock()
	defer recentEventsMu.RUnlock()
	// Return in reverse chronological order
	events := make([]ws.WatchEvent, len(recentEvents))
	for i, e := range recentEvents {
		events[len(recentEvents)-1-i] = e
	}
	writeJSON(w, http.StatusOK, events)
}
