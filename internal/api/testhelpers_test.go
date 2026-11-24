package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sebrandon1/ztp-dashboard/internal/ai"
	"github.com/sebrandon1/ztp-dashboard/internal/config"
	"github.com/sebrandon1/ztp-dashboard/internal/hub"
	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
	"github.com/sebrandon1/ztp-dashboard/internal/spoke"
	"github.com/sebrandon1/ztp-dashboard/internal/store"
	"github.com/sebrandon1/ztp-dashboard/internal/ws"
)

var testGVRKinds = map[schema.GroupVersionResource]string{
	k8s.GVRManagedCluster:      "ManagedClusterList",
	k8s.GVRClusterDeployment:   "ClusterDeploymentList",
	k8s.GVRClusterInstance:     "ClusterInstanceList",
	k8s.GVRInfraEnv:            "InfraEnvList",
	k8s.GVRBareMetalHost:       "BareMetalHostList",
	k8s.GVRAgentClusterInstall: "AgentClusterInstallList",
	k8s.GVRAgent:               "AgentList",
	k8s.GVRPolicy:              "PolicyList",
	k8s.GVRArgoApplication:     "ApplicationList",
}

// newFakeK8sClient builds a k8s.Client on fake typed and dynamic clients.
func newFakeK8sClient(objs ...runtime.Object) *k8s.Client {
	scheme := runtime.NewScheme()
	for gvr, listKind := range testGVRKinds {
		kind := strings.TrimSuffix(listKind, "List")
		scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind(kind), &unstructured.Unstructured{})
		scheme.AddKnownTypeWithName(gvr.GroupVersion().WithKind(listKind), &unstructured.UnstructuredList{})
	}
	return &k8s.Client{
		Clientset:     fake.NewClientset(),
		Dynamic:       dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, testGVRKinds, objs...),
		ServerVersion: "v1.30.0",
	}
}

func unstructuredObj(gvr schema.GroupVersionResource, kind, namespace, name string) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": gvr.GroupVersion().String(),
		"kind":       kind,
		"metadata":   map[string]interface{}{"name": name},
	}
	if namespace != "" {
		obj["metadata"].(map[string]interface{})["namespace"] = namespace
	}
	return &unstructured.Unstructured{Object: obj}
}

func managedClusterObj(name string) *unstructured.Unstructured {
	obj := unstructuredObj(k8s.GVRManagedCluster, "ManagedCluster", "", name)
	_ = unstructured.SetNestedSlice(obj.Object, []interface{}{
		map[string]interface{}{"type": "ManagedClusterConditionAvailable", "status": "True"},
	}, "status", "conditions")
	return obj
}

func policyObj(namespace, name, compliant string) *unstructured.Unstructured {
	obj := unstructuredObj(k8s.GVRPolicy, "Policy", namespace, name)
	_ = unstructured.SetNestedField(obj.Object, compliant, "status", "compliant")
	return obj
}

func argoAppObj(name, health, sync string) *unstructured.Unstructured {
	obj := unstructuredObj(k8s.GVRArgoApplication, "Application", "openshift-gitops", name)
	_ = unstructured.SetNestedMap(obj.Object, map[string]interface{}{"targetRevision": "main"}, "spec", "source")
	_ = unstructured.SetNestedField(obj.Object, health, "status", "health", "status")
	_ = unstructured.SetNestedField(obj.Object, sync, "status", "sync", "status")
	return obj
}

// newFakeOllama serves the Ollama endpoints the AI client uses.
func newFakeOllama(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.1"}]}`))
	})
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"response":"hello","done":false}` + "\n" + `{"response":" world","done":true}` + "\n"))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// newTestServer wires a Server against fake Kubernetes, a temp SQLite store and
// a fake Ollama endpoint.
func newTestServer(t *testing.T, objs ...runtime.Object) *Server {
	t.Helper()
	hubClient := newFakeK8sClient(objs...)

	eventStore, err := store.NewEventStore(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}
	t.Cleanup(func() { _ = eventStore.Close() })

	ollama := newFakeOllama(t)
	cfg := config.Config{
		OllamaEndpoint: ollama.URL,
		OllamaModel:    "llama3.1",
	}
	return NewServer(
		cfg,
		hubClient,
		hub.NewManager(hubClient),
		ai.NewClient(ollama.URL, "llama3.1"),
		ws.NewHub(),
		spoke.NewService(hubClient, k8s.NewClientPool(5*time.Minute, 2)),
		eventStore,
	)
}

// do issues a request against the server's full handler chain.
func do(t *testing.T, s *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}
