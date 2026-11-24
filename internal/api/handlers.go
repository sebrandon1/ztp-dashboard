package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sebrandon1/ztp-dashboard/internal/ai"
	"github.com/sebrandon1/ztp-dashboard/internal/hub"
	"github.com/sebrandon1/ztp-dashboard/internal/store"
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
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, clusters)
}

func (s *Server) handleGetCluster(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	cluster, err := s.hubManager.GetManagedCluster(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cluster)
}

func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pipeline, err := s.hubManager.GetPipelineStatus(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pipeline)
}

func (s *Server) handleGetPolicies(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	policies, err := s.hubManager.GetClusterPolicies(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (s *Server) handleGetArgoApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := s.hubManager.GetArgoApplications(r.Context())
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) handleGetArgoApplication(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	app, err := s.hubManager.GetArgoApplication(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (s *Server) handleSyncArgoApplication(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var opts hub.SyncOptions
	// Body is optional; only an empty body falls back to default options.
	if err := decodeJSON(w, r, &opts); err != nil && !errors.Is(err, io.EOF) {
		writeDecodeError(w, err)
		return
	}

	if err := s.hubManager.SyncArgoApplication(r.Context(), name, opts); err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sync triggered", "application": name})
}

func (s *Server) handleRefreshArgoApplication(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.hubManager.RefreshArgoApplication(r.Context(), name); err != nil {
		writeError(w, statusForError(err), err.Error())
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
	if err := decodeJSON(w, r, &body); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := s.hubManager.SetPolicyDisabled(r.Context(), clusterName, policyName, body.Disabled); err != nil {
		writeError(w, statusForError(err), err.Error())
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
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *Server) handleGetPolicySummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.hubManager.GetPolicySummary(r.Context())
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleGetArgoSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.hubManager.GetArgoSummary(r.Context())
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		ConfirmName string `json:"confirmName"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeDecodeError(w, err)
		return
	}

	if body.ConfirmName != name {
		writeError(w, http.StatusBadRequest, "cluster name confirmation does not match")
		return
	}

	if err := s.hubManager.DeleteManagedCluster(r.Context(), name); err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "cluster": name})
}

func (s *Server) handleListClusterResources(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	resources, err := s.hubManager.ListClusterResources(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resources)
}

func (s *Server) handleGetClusterHealth(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	health, err := s.spokeService.GetHealth(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, health)
}

func (s *Server) handleGetClusterNodes(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	nodes, err := s.spokeService.GetNodes(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (s *Server) handleGetClusterOperators(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	operators, err := s.spokeService.GetOperators(r.Context(), name)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, operators)
}

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	connected, err := s.aiClient.GetStatus(r.Context())
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

func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.aiClient.ListModels(r.Context())
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
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
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

	if !s.acquireAISlot() {
		writeError(w, http.StatusTooManyRequests, "AI busy, retry shortly")
		return
	}
	defer s.releaseAISlot()

	// Check streaming support before writing any headers or status.
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// r.Context() is cancelled when the browser disconnects, which aborts the
	// upstream ollama request instead of letting it generate to completion.
	reader, err := s.aiClient.GenerateStream(r.Context(), req.Model, prompt)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("ollama error: %v", err))
		return
	}

	// SSE streaming response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

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
	ws.ServeWS(s.wsHub, w, r, s.originAllowed)
}

// RecordEvent stores an event using the provided EventStore.
func RecordEvent(eventStore *store.EventStore, event ws.WatchEvent) {
	if err := eventStore.Insert(event); err != nil {
		slog.Error("failed to record event", "error", err)
	}
}

// eventQueryFromRequest builds an EventQuery from the request's URL parameters.
func eventQueryFromRequest(r *http.Request) store.EventQuery {
	v := r.URL.Query()
	q := store.EventQuery{
		Query:        v.Get("q"),
		Severity:     v.Get("severity"),
		ResourceType: v.Get("resource_type"),
		From:         v.Get("from"),
		To:           v.Get("to"),
	}
	if v := v.Get("limit"); v != "" {
		q.Limit, _ = strconv.Atoi(v)
	}
	if v := v.Get("offset"); v != "" {
		q.Offset, _ = strconv.Atoi(v)
	}
	return q
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	q := eventQueryFromRequest(r)

	events, total, err := s.eventStore.Query(q)
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	if events == nil {
		events = []ws.WatchEvent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events": events,
		"total":  total,
	})
}

// handleExportEvents streams every event matching the same filters as
// handleGetEvents as CSV or newline-delimited JSON.
func (s *Server) handleExportEvents(w http.ResponseWriter, r *http.Request) {
	q := eventQueryFromRequest(r)

	var contentType, filename string
	switch format := strings.ToLower(r.URL.Query().Get("format")); format {
	case "", "json":
		contentType, filename = "application/x-ndjson", "ztp-events.ndjson"
	case "csv":
		contentType, filename = "text/csv", "ztp-events.csv"
	default:
		writeError(w, http.StatusBadRequest, "format must be csv or json")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	if strings.HasSuffix(filename, ".csv") {
		s.writeEventsCSV(w, q)
		return
	}

	enc := json.NewEncoder(w)
	if err := s.eventStore.Iterate(q, func(e ws.WatchEvent) error {
		return enc.Encode(e)
	}); err != nil {
		slog.Error("exporting events", "format", "json", "error", err)
	}
}

func (s *Server) writeEventsCSV(w http.ResponseWriter, q store.EventQuery) {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"timestamp", "severity", "event_type", "resource_type", "name", "namespace", "summary", "insight"})
	err := s.eventStore.Iterate(q, func(e ws.WatchEvent) error {
		return cw.Write([]string{
			e.Timestamp, e.Severity, string(e.EventType), e.ResourceType,
			e.Name, e.Namespace, e.Summary, e.Insight,
		})
	})
	cw.Flush()
	if err != nil {
		slog.Error("exporting events", "format", "csv", "error", err)
	}
	if err := cw.Error(); err != nil {
		slog.Error("flushing event CSV", "error", err)
	}
}

func (s *Server) handleGetEventStats(w http.ResponseWriter, _ *http.Request) {
	stats, err := s.eventStore.GetStats()
	if err != nil {
		writeError(w, statusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
