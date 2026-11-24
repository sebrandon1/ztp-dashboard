package spoke

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
)

const testCluster = "cluster-a"

// newHubClient builds a hub client whose typed clientset holds the given secrets.
func newHubClient(secrets ...*corev1.Secret) *k8s.Client {
	objs := make([]runtime.Object, len(secrets))
	for i, s := range secrets {
		objs[i] = s
	}
	return &k8s.Client{Clientset: fake.NewClientset(objs...)}
}

func adminSecret(clusterName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: clusterName + "-admin-kubeconfig", Namespace: clusterName},
		Data:       data,
	}
}

func kubeconfigFor(server string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: %s
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: fake-token
`, server))
}

// newFakeAPIServer serves the handful of endpoints the spoke service reads.
func newFakeAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONBody(w, `{
  "apiVersion": "v1",
  "kind": "NodeList",
  "items": [
    {
      "metadata": {
        "name": "node-1",
        "creationTimestamp": "2025-01-01T00:00:00Z",
        "labels": {"node-role.kubernetes.io/control-plane": "", "node-role.kubernetes.io/master": ""}
      },
      "status": {"nodeInfo": {"kubeletVersion": "v1.29.1"}, "conditions": [{"type": "Ready", "status": "True"}]}
    },
    {
      "metadata": {"name": "node-2", "creationTimestamp": "2025-01-01T00:00:00Z", "labels": {}},
      "status": {"nodeInfo": {"kubeletVersion": "v1.29.1"}, "conditions": [{"type": "Ready", "status": "False"}]}
    }
  ]
}`)
	})

	mux.HandleFunc("/apis/config.openshift.io/v1/clusterversions/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONBody(w, `{
  "apiVersion": "config.openshift.io/v1",
  "kind": "ClusterVersion",
  "metadata": {"name": "version"},
  "spec": {"channel": "stable-4.16", "clusterID": "abc-123"},
  "status": {"history": [{"version": "4.16.2"}]}
}`)
	})

	mux.HandleFunc("/apis/config.openshift.io/v1/clusteroperators", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONBody(w, `{
  "apiVersion": "config.openshift.io/v1",
  "kind": "ClusterOperatorList",
  "items": [
    {
      "metadata": {"name": "dns"},
      "status": {
        "conditions": [
          {"type": "Available", "status": "True"},
          {"type": "Degraded", "status": "False"},
          {"type": "Progressing", "status": "False"}
        ],
        "versions": [{"name": "operator", "version": "4.16.2"}]
      }
    },
    {
      "metadata": {"name": "network"},
      "status": {
        "conditions": [
          {"type": "Available", "status": "True"},
          {"type": "Degraded", "status": "True", "message": "pod restarting"}
        ]
      }
    }
  ]
}`)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSONBody(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	srv := newFakeAPIServer(t)
	hubClient := newHubClient(adminSecret(testCluster, map[string][]byte{
		"kubeconfig": kubeconfigFor(srv.URL),
	}))
	return NewService(hubClient, k8s.NewClientPool(10*time.Minute, 5))
}

func TestGetSpokeClientNilHub(t *testing.T) {
	svc := NewService(nil, k8s.NewClientPool(time.Minute, 1))
	if _, err := svc.getSpokeClient(context.Background(), testCluster); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("getSpokeClient() error = %v, want ErrNotConnected", err)
	}
}

func TestGetHealthNilHub(t *testing.T) {
	svc := NewService(nil, k8s.NewClientPool(time.Minute, 1))
	if _, err := svc.GetHealth(context.Background(), testCluster); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("GetHealth() error = %v, want ErrNotConnected", err)
	}
}

func TestGetNodesNilHub(t *testing.T) {
	svc := NewService(nil, k8s.NewClientPool(time.Minute, 1))
	if _, err := svc.GetNodes(context.Background(), testCluster); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("GetNodes() error = %v, want ErrNotConnected", err)
	}
}

func TestGetOperatorsNilHub(t *testing.T) {
	svc := NewService(nil, k8s.NewClientPool(time.Minute, 1))
	if _, err := svc.GetOperators(context.Background(), testCluster); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("GetOperators() error = %v, want ErrNotConnected", err)
	}
}

func TestGetSpokeClientMissingSecret(t *testing.T) {
	svc := NewService(newHubClient(), k8s.NewClientPool(time.Minute, 1))
	if _, err := svc.getSpokeClient(context.Background(), testCluster); err == nil {
		t.Fatal("expected error when kubeconfig secret is missing")
	}
}

func TestGetSpokeClientMissingKey(t *testing.T) {
	svc := NewService(
		newHubClient(adminSecret(testCluster, map[string][]byte{"other": []byte("x")})),
		k8s.NewClientPool(time.Minute, 1),
	)
	_, err := svc.getSpokeClient(context.Background(), testCluster)
	if err == nil {
		t.Fatal("expected error when kubeconfig key is missing")
	}
}

func TestGetSpokeClientSuccess(t *testing.T) {
	svc := newTestService(t)
	client, err := svc.getSpokeClient(context.Background(), testCluster)
	if err != nil {
		t.Fatalf("getSpokeClient: %v", err)
	}
	if client == nil || client.Clientset == nil {
		t.Fatal("expected a populated spoke client")
	}

	// A second call must hit the pool cache and succeed the same way.
	if _, err := svc.getSpokeClient(context.Background(), testCluster); err != nil {
		t.Fatalf("cached getSpokeClient: %v", err)
	}
}

func TestGetNodes(t *testing.T) {
	svc := newTestService(t)
	nodes, err := svc.GetNodes(context.Background(), testCluster)
	if err != nil {
		t.Fatalf("GetNodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].Name != "node-1" || nodes[0].Status != "Ready" {
		t.Errorf("node-1 = %+v", nodes[0])
	}
	if nodes[0].Role != "control-plane,master" {
		t.Errorf("node-1 role = %q, want control-plane,master", nodes[0].Role)
	}
	if nodes[0].KubeletVersion != "v1.29.1" {
		t.Errorf("kubelet version = %q", nodes[0].KubeletVersion)
	}
	if nodes[1].Role != "worker" {
		t.Errorf("node-2 role = %q, want worker", nodes[1].Role)
	}
	if nodes[1].Status != "NotReady" {
		t.Errorf("node-2 status = %q, want NotReady", nodes[1].Status)
	}
}

func TestGetOperators(t *testing.T) {
	svc := newTestService(t)
	operators, err := svc.GetOperators(context.Background(), testCluster)
	if err != nil {
		t.Fatalf("GetOperators: %v", err)
	}
	if len(operators) != 2 {
		t.Fatalf("got %d operators, want 2", len(operators))
	}
	dns := operators[0]
	if dns.Name != "dns" || !dns.Available || dns.Degraded || dns.Version != "4.16.2" {
		t.Errorf("dns operator = %+v", dns)
	}
	network := operators[1]
	if !network.Degraded || network.Message != "pod restarting" {
		t.Errorf("network operator = %+v", network)
	}
}

func TestGetHealth(t *testing.T) {
	svc := newTestService(t)
	health, err := svc.GetHealth(context.Background(), testCluster)
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if health.ClusterVersion == nil || health.ClusterVersion.Version != "4.16.2" {
		t.Errorf("ClusterVersion = %+v", health.ClusterVersion)
	}
	if health.ClusterVersion.Channel != "stable-4.16" || health.ClusterVersion.ClusterID != "abc-123" {
		t.Errorf("ClusterVersion = %+v", health.ClusterVersion)
	}
	if health.NodeCount != 2 || health.NodesReady != 1 {
		t.Errorf("nodes total/ready = %d/%d, want 2/1", health.NodeCount, health.NodesReady)
	}
	if health.OperatorCount != 2 || health.DegradedCount != 1 {
		t.Errorf("operators total/degraded = %d/%d, want 2/1", health.OperatorCount, health.DegradedCount)
	}
}

func TestGetHealthMissingSecret(t *testing.T) {
	svc := NewService(newHubClient(), k8s.NewClientPool(time.Minute, 1))
	if _, err := svc.GetHealth(context.Background(), testCluster); err == nil {
		t.Fatal("expected error when kubeconfig secret is missing")
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"minutes", time.Now().Add(-5 * time.Minute), "5m"},
		{"hours", time.Now().Add(-3 * time.Hour), "3h"},
		{"days", time.Now().Add(-50 * time.Hour), "2d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatAge(tt.at); got != tt.want {
				t.Errorf("formatAge() = %q, want %q", got, tt.want)
			}
		})
	}
}
