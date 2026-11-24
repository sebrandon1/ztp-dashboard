package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
	"github.com/sebrandon1/ztp-dashboard/internal/ws"
	"golang.org/x/time/rate"
)

// decodeResponse parses the JSON envelope returned by writeJSON/writeError.
func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding response %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestHealthProbes(t *testing.T) {
	s := newTestServer(t)

	if rec := do(t, s, http.MethodGet, "/healthz", ""); rec.Code != http.StatusOK {
		t.Errorf("healthz = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodGet, "/readyz", ""); rec.Code != http.StatusOK {
		t.Errorf("readyz with client = %d, want 200", rec.Code)
	}

	// A server with no Kubernetes client is not ready.
	notReady := NewServer(s.cfg, nil, nil, s.aiClient, s.wsHub, s.spokeService, s.eventStore)
	if rec := do(t, notReady, http.MethodGet, "/readyz", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("readyz without client = %d, want 503", rec.Code)
	}
}

func TestHubStatus(t *testing.T) {
	rec := do(t, newTestServer(t), http.MethodGet, "/api/status", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	data := decodeResponse(t, rec)["data"].(map[string]interface{})
	if data["connected"] != true {
		t.Errorf("connected = %v, want true", data["connected"])
	}
}

func TestClusters(t *testing.T) {
	s := newTestServer(t, managedClusterObj("cluster-a"), managedClusterObj("local-cluster"))

	rec := do(t, s, http.MethodGet, "/api/clusters", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d, want 200", rec.Code)
	}
	clusters := decodeResponse(t, rec)["data"].([]interface{})
	if len(clusters) != 1 {
		t.Errorf("got %d clusters, want 1 (local-cluster excluded)", len(clusters))
	}

	if rec := do(t, s, http.MethodGet, "/api/clusters/cluster-a", ""); rec.Code != http.StatusOK {
		t.Errorf("get existing = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodGet, "/api/clusters/missing", ""); rec.Code != http.StatusNotFound {
		t.Errorf("get missing = %d, want 404", rec.Code)
	}
}

func TestPipeline(t *testing.T) {
	s := newTestServer(t, managedClusterObj("cluster-a"))
	rec := do(t, s, http.MethodGet, "/api/clusters/cluster-a/pipeline", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("pipeline = %d, want 200", rec.Code)
	}
	data := decodeResponse(t, rec)["data"].(map[string]interface{})
	if data["clusterName"] != "cluster-a" {
		t.Errorf("clusterName = %v", data["clusterName"])
	}
}

func TestPolicies(t *testing.T) {
	s := newTestServer(t, policyObj("cluster-a", "p1", "Compliant"))

	rec := do(t, s, http.MethodGet, "/api/clusters/cluster-a/policies", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("policies = %d, want 200", rec.Code)
	}
	policies := decodeResponse(t, rec)["data"].([]interface{})
	if len(policies) != 1 {
		t.Errorf("got %d policies, want 1", len(policies))
	}

	rec = do(t, s, http.MethodGet, "/api/policies/summary", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("summary = %d, want 200", rec.Code)
	}
}

func TestSetPolicyState(t *testing.T) {
	s := newTestServer(t, policyObj("cluster-a", "p1", "Compliant"))

	if rec := do(t, s, http.MethodPost, "/api/clusters/cluster-a/policies/p1/state", "not-json"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad body = %d, want 400", rec.Code)
	}

	rec := do(t, s, http.MethodPost, "/api/clusters/cluster-a/policies/p1/state", `{"disabled":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("set state = %d, want 200", rec.Code)
	}
	if data := decodeResponse(t, rec)["data"].(map[string]interface{}); data["status"] != "disabled" {
		t.Errorf("status = %v, want disabled", data["status"])
	}
}

func TestDeleteCluster(t *testing.T) {
	s := newTestServer(t, managedClusterObj("cluster-a"))

	if rec := do(t, s, http.MethodDelete, "/api/clusters/cluster-a", "not-json"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad body = %d, want 400", rec.Code)
	}
	if rec := do(t, s, http.MethodDelete, "/api/clusters/cluster-a", `{"confirmName":"wrong"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("mismatched confirm = %d, want 400", rec.Code)
	}
	if rec := do(t, s, http.MethodDelete, "/api/clusters/cluster-a", `{"confirmName":"cluster-a"}`); rec.Code != http.StatusOK {
		t.Errorf("delete = %d, want 200", rec.Code)
	}
}

func TestArgoEndpoints(t *testing.T) {
	s := newTestServer(t, argoAppObj("app-1", "Healthy", "Synced"))

	if rec := do(t, s, http.MethodGet, "/api/argocd/applications", ""); rec.Code != http.StatusOK {
		t.Errorf("list = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodGet, "/api/argocd/applications/app-1", ""); rec.Code != http.StatusOK {
		t.Errorf("get = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodGet, "/api/argocd/applications/missing", ""); rec.Code != http.StatusNotFound {
		t.Errorf("get missing = %d, want 404", rec.Code)
	}
	// Sync body is optional.
	if rec := do(t, s, http.MethodPost, "/api/argocd/applications/app-1/sync", ""); rec.Code != http.StatusOK {
		t.Errorf("sync no body = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodPost, "/api/argocd/applications/app-1/sync", `{"prune":true,"force":true}`); rec.Code != http.StatusOK {
		t.Errorf("sync with body = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodPost, "/api/argocd/applications/app-1/refresh", ""); rec.Code != http.StatusOK {
		t.Errorf("refresh = %d, want 200", rec.Code)
	}
	rec := do(t, s, http.MethodGet, "/api/argocd/summary", "")
	if rec.Code != http.StatusOK {
		t.Errorf("summary = %d, want 200", rec.Code)
	}
	data := decodeResponse(t, rec)["data"].(map[string]interface{})
	if data["total"] != float64(1) {
		t.Errorf("total = %v, want 1", data["total"])
	}
}

func TestGetResource(t *testing.T) {
	s := newTestServer(t, managedClusterObj("cluster-a"))

	ok := "/api/resources/cluster.open-cluster-management.io/v1/managedclusters/_/cluster-a"
	if rec := do(t, s, http.MethodGet, ok, ""); rec.Code != http.StatusOK {
		t.Errorf("allowed = %d, want 200", rec.Code)
	}

	notAllowed := "/api/resources/_/v1/secrets/default/x"
	if rec := do(t, s, http.MethodGet, notAllowed, ""); rec.Code != http.StatusForbidden {
		t.Errorf("not allowed = %d, want 403", rec.Code)
	}

	missing := "/api/resources/cluster.open-cluster-management.io/v1/managedclusters/_/missing"
	if rec := do(t, s, http.MethodGet, missing, ""); rec.Code != http.StatusNotFound {
		t.Errorf("missing = %d, want 404", rec.Code)
	}
}

func TestClusterHealthEndpoints(t *testing.T) {
	s := newTestServer(t)
	// A malformed kubeconfig makes the spoke service fail before any network call.
	_, err := s.k8sClient.Clientset.CoreV1().Secrets("cluster-a").Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster-a-admin-kubeconfig", Namespace: "cluster-a"},
		Data:       map[string][]byte{"kubeconfig": []byte("not-a-kubeconfig")},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create secret: %v", err)
	}

	for _, path := range []string{
		"/api/clusters/cluster-a/health",
		"/api/clusters/cluster-a/health/nodes",
		"/api/clusters/cluster-a/health/operators",
	} {
		rec := do(t, s, http.MethodGet, path, "")
		if rec.Code < 400 || rec.Code > 599 {
			t.Errorf("%s = %d, want an error status", path, rec.Code)
		}
	}
}

func TestAIEndpoints(t *testing.T) {
	s := newTestServer(t)

	rec := do(t, s, http.MethodGet, "/api/ai/status", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	data := decodeResponse(t, rec)["data"].(map[string]interface{})
	if data["connected"] != true || data["defaultModel"] != "llama3.1" {
		t.Errorf("ai status = %v", data)
	}

	if rec := do(t, s, http.MethodGet, "/api/ai/models", ""); rec.Code != http.StatusOK {
		t.Errorf("models = %d, want 200", rec.Code)
	}

	if rec := do(t, s, http.MethodPost, "/api/ai/diagnose", "not-json"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad body = %d, want 400", rec.Code)
	}
}

func TestAIDiagnoseStream(t *testing.T) {
	s := newTestServer(t)
	body := `{"type":"general","context":{"clusterName":"cluster-a"}}`
	rec := do(t, s, http.MethodPost, "/api/ai/diagnose", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("diagnose = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	out := rec.Body.String()
	if !strings.Contains(out, "data: ") {
		t.Errorf("stream missing data frames: %q", out)
	}
	if !strings.Contains(out, `"done":true`) {
		t.Errorf("stream did not terminate with done:true: %q", out)
	}
}

func TestAIDiagnoseConcurrencyLimit(t *testing.T) {
	s := newTestServer(t)
	s.aiSlots = make(chan struct{}, 1)
	s.aiSlots <- struct{}{} // occupy the only slot

	body := `{"type":"general","context":{"clusterName":"cluster-a"}}`
	rec := do(t, s, http.MethodPost, "/api/ai/diagnose", body)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("diagnose with full slot = %d, want 429", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "AI busy") {
		t.Errorf("body = %q, want AI busy message", rec.Body.String())
	}
}

func TestAIDiagnoseReleasesSlot(t *testing.T) {
	s := newTestServer(t)
	body := `{"type":"general","context":{"clusterName":"cluster-a"}}`
	if rec := do(t, s, http.MethodPost, "/api/ai/diagnose", body); rec.Code != http.StatusOK {
		t.Fatalf("diagnose = %d, want 200", rec.Code)
	}
	if len(s.aiSlots) != 0 {
		t.Errorf("slots in use after stream = %d, want 0", len(s.aiSlots))
	}
}

func TestAIDiagnoseRateLimit(t *testing.T) {
	s := newTestServer(t)
	// Allow a single generation for this window; the call below consumes it.
	s.aiLimiter = rate.NewLimiter(1, 1)
	body := `{"type":"general","context":{"clusterName":"cluster-a"}}`
	if rec := do(t, s, http.MethodPost, "/api/ai/diagnose", body); rec.Code != http.StatusOK {
		t.Fatalf("first diagnose = %d, want 200", rec.Code)
	}
	if rec := do(t, s, http.MethodPost, "/api/ai/diagnose", body); rec.Code != http.StatusTooManyRequests {
		t.Errorf("second diagnose = %d, want 429", rec.Code)
	}
}

func TestAIDiagnoseBodyTooLarge(t *testing.T) {
	s := newTestServer(t)
	body := `{"type":"general","context":{"blob":"` + strings.Repeat("a", maxJSONBodyBytes) + `"}}`
	rec := do(t, s, http.MethodPost, "/api/ai/diagnose", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("diagnose = %d, want 413", rec.Code)
	}
}

func TestSetPolicyStateInvalidBody(t *testing.T) {
	s := newTestServer(t, policyObj("cluster-a", "policy-a", "compliant"))
	rec := do(t, s, http.MethodPost, "/api/clusters/cluster-a/policies/policy-a/state", `{`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("set policy state = %d, want 400", rec.Code)
	}
}

func TestSyncArgoApplicationEmptyBody(t *testing.T) {
	s := newTestServer(t, argoAppObj("app-a", "Healthy", "Synced"))
	rec := do(t, s, http.MethodPost, "/api/argocd/applications/app-a/sync", "")
	if rec.Code != http.StatusOK {
		t.Errorf("sync with empty body = %d, want 200", rec.Code)
	}
}

func TestEvents(t *testing.T) {
	s := newTestServer(t)
	for _, sev := range []string{"info", "warning", "info"} {
		if err := s.eventStore.Insert(ws.WatchEvent{
			EventType: "MODIFIED", ResourceType: "ManagedCluster", Name: "c1", Severity: sev,
		}); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	rec := do(t, s, http.MethodGet, "/api/events?limit=1&offset=0&severity=info", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("events = %d, want 200", rec.Code)
	}
	data := decodeResponse(t, rec)["data"].(map[string]interface{})
	if data["total"] != float64(2) {
		t.Errorf("total = %v, want 2", data["total"])
	}
	if events := data["events"].([]interface{}); len(events) != 1 {
		t.Errorf("events = %d, want 1 (limit)", len(events))
	}

	if rec := do(t, s, http.MethodGet, "/api/events/stats", ""); rec.Code != http.StatusOK {
		t.Errorf("stats = %d, want 200", rec.Code)
	}
}

func TestSPAFallback(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/", "/clusters/cluster-a", "/settings"} {
		rec := do(t, s, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Errorf("%s = %d, want 200", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s Content-Type = %q, want text/html", path, ct)
		}
	}
}

func TestCORSOptions(t *testing.T) {
	rec := do(t, newTestServer(t), http.MethodOptions, "/api/clusters", "")
	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin = %q, want empty for requests without an Origin header", got)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	handler := recoveryMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("panic status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Internal server error") {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestListClusterResources(t *testing.T) {
	s := newTestServer(t,
		unstructuredObj(k8s.GVRClusterDeployment, "ClusterDeployment", "cluster-a", "cd"),
		policyObj("cluster-a", "p1", "Compliant"),
	)
	rec := do(t, s, http.MethodGet, "/api/clusters/cluster-a/resources", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("resources = %d, want 200", rec.Code)
	}
	resources := decodeResponse(t, rec)["data"].([]interface{})
	if len(resources) != 2 {
		t.Errorf("got %d resources, want 2", len(resources))
	}
}

func TestRecordEvent(t *testing.T) {
	s := newTestServer(t)
	RecordEvent(s.eventStore, ws.WatchEvent{
		EventType: "ADDED", ResourceType: "ManagedCluster", Name: "c1", Severity: "info",
	})
	stats, err := s.eventStore.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats == nil {
		t.Fatal("expected stats")
	}
}

func TestExportEvents(t *testing.T) {
	s := newTestServer(t)
	for _, e := range []ws.WatchEvent{
		{EventType: "ADDED", ResourceType: "ManagedCluster", Name: "cluster-a", Severity: "good", Summary: "joined", Timestamp: "2025-01-01T10:00:00Z"},
		{EventType: "MODIFIED", ResourceType: "Policy", Name: "policy-a", Severity: "bad", Summary: "non-compliant", Timestamp: "2025-01-01T11:00:00Z"},
	} {
		if err := s.eventStore.Insert(e); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("csv", func(t *testing.T) {
		rec := do(t, s, http.MethodGet, "/api/events/export?format=csv&severity=bad", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "text/csv" {
			t.Errorf("Content-Type = %q, want text/csv", got)
		}
		if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") {
			t.Errorf("Content-Disposition = %q, want attachment", got)
		}
		lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("csv lines = %d (%q), want header + one row", len(lines), rec.Body.String())
		}
		if !strings.HasPrefix(lines[0], "timestamp,severity,event_type") {
			t.Errorf("header = %q", lines[0])
		}
		if !strings.Contains(lines[1], "policy-a") {
			t.Errorf("row = %q, want policy-a", lines[1])
		}
	})

	t.Run("ndjson default", func(t *testing.T) {
		rec := do(t, s, http.MethodGet, "/api/events/export", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/x-ndjson" {
			t.Errorf("Content-Type = %q, want application/x-ndjson", got)
		}
		lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("ndjson lines = %d, want 2", len(lines))
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
			t.Fatalf("decoding ndjson line: %v", err)
		}
		if event["name"] != "policy-a" {
			t.Errorf("first exported event = %v, want policy-a", event["name"])
		}
	})

	t.Run("bad format", func(t *testing.T) {
		rec := do(t, s, http.MethodGet, "/api/events/export?format=xml", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}
