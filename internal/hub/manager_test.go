package hub

import (
	"context"
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
)

func TestGetStatus(t *testing.T) {
	t.Run("not connected", func(t *testing.T) {
		got := NewManager(nil).GetStatus()
		if got.Connected {
			t.Fatal("expected disconnected status")
		}
	})

	t.Run("connected", func(t *testing.T) {
		got := newFakeManager().GetStatus()
		if !got.Connected {
			t.Fatal("expected connected status")
		}
		if got.ServerVersion != "v1.30.0" {
			t.Errorf("ServerVersion = %q, want v1.30.0", got.ServerVersion)
		}
	})
}

func TestDeriveStatus(t *testing.T) {
	tests := []struct {
		name       string
		conditions []map[string]interface{}
		want       string
	}{
		{"empty", nil, "Pending"},
		{"completed true", []map[string]interface{}{condition("Completed", "True", "")}, "Completed"},
		{"ready true", []map[string]interface{}{condition("Ready", "True", "")}, "Completed"},
		{"in progress with reason", []map[string]interface{}{condition("Completed", "False", "Installing")}, "Installing"},
		{"in progress without reason", []map[string]interface{}{condition("Ready", "False", "")}, "InProgress"},
		{"failed true", []map[string]interface{}{condition("Failed", "True", "")}, "Error"},
		{"error condition", []map[string]interface{}{condition("Error", "True", "")}, "Error"},
		{"unrelated condition", []map[string]interface{}{condition("Synced", "True", "")}, "Pending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveStatus(tt.conditions); got != tt.want {
				t.Errorf("deriveStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseConditions(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{"type": "Ready", "status": "True"},
		"not-a-map",
		42,
		map[string]interface{}{"type": "Synced", "status": "False"},
	}
	got := parseConditions(raw)
	if len(got) != 2 {
		t.Fatalf("parseConditions returned %d entries, want 2", len(got))
	}
	if got[0]["type"] != "Ready" || got[1]["type"] != "Synced" {
		t.Errorf("unexpected conditions: %v", got)
	}
}

func TestParseManagedCluster(t *testing.T) {
	obj := managedCluster("cluster-a", []interface{}{
		condition("ManagedClusterConditionAvailable", "True", ""),
		condition("HubAcceptedManagedCluster", "True", ""),
		condition("ManagedClusterJoined", "False", ""),
	}, map[string]string{"openshiftVersion": "4.16.2"})

	got := parseManagedCluster(obj)
	if got.Name != "cluster-a" {
		t.Errorf("Name = %q, want cluster-a", got.Name)
	}
	if got.Available != "True" || got.HubAccepted != "True" || got.Joined != "False" {
		t.Errorf("condition summary wrong: %+v", got)
	}
	if got.OpenShiftVersion != "4.16.2" {
		t.Errorf("OpenShiftVersion = %q, want 4.16.2", got.OpenShiftVersion)
	}
	if len(got.Conditions) != 3 {
		t.Errorf("Conditions = %d, want 3", len(got.Conditions))
	}
}

func TestParseManagedClusterUnavailable(t *testing.T) {
	obj := managedCluster("cluster-b", []interface{}{
		condition("ManagedClusterConditionAvailable", "False", ""),
	}, nil)
	if got := parseManagedCluster(obj); got.Available != "False" {
		t.Errorf("Available = %q, want False", got.Available)
	}
}

func TestParseResourceStatus(t *testing.T) {
	withConditions := namespacedResource(k8s.GVRPolicy, "Policy", "ns", "p1",
		[]interface{}{condition("Ready", "True", "")}, nil)
	if got := parseResourceStatus(withConditions); got.Status != "Completed" || got.Namespace != "ns" {
		t.Errorf("parseResourceStatus() = %+v", got)
	}

	withoutConditions := namespacedResource(k8s.GVRPolicy, "Policy", "ns", "p2", nil, nil)
	if got := parseResourceStatus(withoutConditions); got.Status != "Unknown" {
		t.Errorf("Status = %q, want Unknown", got.Status)
	}
}

func TestIsWatchedGVR(t *testing.T) {
	if !isWatchedGVR(k8s.GVRManagedCluster) {
		t.Error("managedclusters should be watched")
	}
	if isWatchedGVR(schema.GroupVersionResource{Group: "v1", Version: "v1", Resource: "secrets"}) {
		t.Error("secrets should not be watched")
	}
}

func TestListManagedClusters(t *testing.T) {
	m := newFakeManager(
		managedCluster("cluster-a", nil, nil),
		managedCluster("cluster-b", nil, nil),
		managedCluster("local-cluster", nil, nil),
	)
	clusters, err := m.ListManagedClusters(context.Background())
	if err != nil {
		t.Fatalf("ListManagedClusters: %v", err)
	}
	if len(clusters) != 2 {
		t.Fatalf("got %d clusters, want 2 (local-cluster excluded)", len(clusters))
	}
}

func TestGetManagedCluster(t *testing.T) {
	m := newFakeManager(managedCluster("cluster-a", nil, nil))

	cluster, err := m.GetManagedCluster(context.Background(), "cluster-a")
	if err != nil {
		t.Fatalf("GetManagedCluster: %v", err)
	}
	if cluster.Name != "cluster-a" {
		t.Errorf("Name = %q", cluster.Name)
	}

	if _, err := m.GetManagedCluster(context.Background(), "missing"); err == nil {
		t.Error("expected error for missing cluster")
	}
}

func TestGetPipelineStatus(t *testing.T) {
	const ns = "cluster-a"
	m := newFakeManager(
		managedCluster(ns, []interface{}{condition("ManagedClusterConditionAvailable", "True", "")}, nil),
		namespacedResource(k8s.GVRClusterInstance, "ClusterInstance", ns, ns, []interface{}{condition("Completed", "True", "")}, nil),
		namespacedResource(k8s.GVRClusterDeployment, "ClusterDeployment", ns, ns, nil, nil),
		namespacedResource(k8s.GVRInfraEnv, "InfraEnv", ns, ns, nil, nil),
		namespacedResource(k8s.GVRAgentClusterInstall, "AgentClusterInstall", ns, ns, nil, nil),
		namespacedResource(k8s.GVRBareMetalHost, "BareMetalHost", ns, "bmh-0", nil, nil),
		namespacedResource(k8s.GVRAgent, "Agent", ns, "agent-0", nil, nil),
	)

	pipeline, err := m.GetPipelineStatus(context.Background(), ns)
	if err != nil {
		t.Fatalf("GetPipelineStatus: %v", err)
	}
	if pipeline.ClusterInstance == nil || pipeline.ClusterInstance.Status != "Completed" {
		t.Errorf("ClusterInstance = %+v", pipeline.ClusterInstance)
	}
	if pipeline.ClusterDeployment == nil || pipeline.InfraEnv == nil || pipeline.AgentClusterInstall == nil {
		t.Error("expected ClusterDeployment, InfraEnv and AgentClusterInstall to be populated")
	}
	if len(pipeline.BareMetalHosts) != 1 || len(pipeline.Agents) != 1 {
		t.Errorf("BMHs=%d Agents=%d, want 1/1", len(pipeline.BareMetalHosts), len(pipeline.Agents))
	}
	if pipeline.ManagedCluster == nil || pipeline.ManagedCluster.Status != "True" {
		t.Errorf("ManagedCluster = %+v", pipeline.ManagedCluster)
	}
}

func TestGetPipelineStatusMissingResources(t *testing.T) {
	// No CRs at all: the pipeline should still return without error.
	pipeline, err := newFakeManager().GetPipelineStatus(context.Background(), "empty")
	if err != nil {
		t.Fatalf("GetPipelineStatus: %v", err)
	}
	if pipeline.ClusterInstance != nil || pipeline.ManagedCluster != nil {
		t.Error("expected nil resources for an empty cluster")
	}
}

func TestGetClusterPolicies(t *testing.T) {
	m := newFakeManager(
		namespacedResource(k8s.GVRPolicy, "Policy", "cluster-a", "p1", nil, map[string]interface{}{
			"status.compliant":       "Compliant",
			"spec.disabled":          false,
			"spec.remediationAction": "inform",
		}),
	)
	policies, err := m.GetClusterPolicies(context.Background(), "cluster-a")
	if err != nil {
		t.Fatalf("GetClusterPolicies: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("got %d policies, want 1", len(policies))
	}
	p := policies[0]
	if p.Status != "Compliant" {
		t.Errorf("Status = %q, want Compliant", p.Status)
	}
	if p.Data["remediationAction"] != "inform" {
		t.Errorf("remediationAction = %v", p.Data["remediationAction"])
	}
	if p.Data["disabled"] != false {
		t.Errorf("disabled = %v", p.Data["disabled"])
	}
}

func TestSetPolicyDisabled(t *testing.T) {
	m := newFakeManager(namespacedResource(k8s.GVRPolicy, "Policy", "cluster-a", "p1", nil, nil))

	if err := m.SetPolicyDisabled(context.Background(), "cluster-a", "p1", true); err != nil {
		t.Fatalf("SetPolicyDisabled: %v", err)
	}

	obj, err := m.client.Dynamic.Resource(k8s.GVRPolicy).Namespace("cluster-a").Get(context.Background(), "p1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	disabled, _, _ := unstructured.NestedBool(obj.Object, "spec", "disabled")
	if !disabled {
		t.Error("spec.disabled = false, want true")
	}
}

func TestArgoApplicationLifecycle(t *testing.T) {
	const ns = "openshift-gitops"
	app := namespacedResource(k8s.GVRArgoApplication, "Application", ns, "app-1", nil, map[string]interface{}{
		"spec.source.targetRevision": "main",
		"status.health.status":       "Healthy",
		"status.sync.status":         "Synced",
		"status.sync.revision":       "abc123",
	})
	m := newFakeManager(app)

	t.Run("list", func(t *testing.T) {
		apps, err := m.GetArgoApplications(context.Background())
		if err != nil {
			t.Fatalf("GetArgoApplications: %v", err)
		}
		if len(apps) != 1 || apps[0].Status != "Healthy" {
			t.Errorf("apps = %+v", apps)
		}
	})

	t.Run("get", func(t *testing.T) {
		detail, err := m.GetArgoApplication(context.Background(), "app-1")
		if err != nil {
			t.Fatalf("GetArgoApplication: %v", err)
		}
		if detail.Health != "Healthy" || detail.SyncStatus != "Synced" || detail.SyncRevision != "abc123" {
			t.Errorf("detail = %+v", detail)
		}
		if detail.Source["targetRevision"] != "main" {
			t.Errorf("Source = %v", detail.Source)
		}
	})

	t.Run("sync sets operation", func(t *testing.T) {
		if err := m.SyncArgoApplication(context.Background(), "app-1", SyncOptions{Prune: true, Force: true}); err != nil {
			t.Fatalf("SyncArgoApplication: %v", err)
		}
		obj, err := m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace(ns).Get(context.Background(), "app-1", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get app: %v", err)
		}
		op, found, _ := unstructured.NestedMap(obj.Object, "operation")
		if !found {
			t.Fatal("operation field not set")
		}
		revision, _, _ := unstructured.NestedString(op, "sync", "revision")
		if revision != "main" {
			t.Errorf("operation.sync.revision = %q, want main", revision)
		}
		prune, _, _ := unstructured.NestedBool(op, "sync", "prune")
		if !prune {
			t.Error("operation.sync.prune = false, want true")
		}
		if _, found, _ := unstructured.NestedSlice(op, "sync", "syncOptions"); !found {
			t.Error("expected syncOptions when Force is set")
		}
	})

	t.Run("refresh sets annotation", func(t *testing.T) {
		if err := m.RefreshArgoApplication(context.Background(), "app-1"); err != nil {
			t.Fatalf("RefreshArgoApplication: %v", err)
		}
		obj, err := m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace(ns).Get(context.Background(), "app-1", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("get app: %v", err)
		}
		ann := obj.GetAnnotations()
		if ann["argocd.argoproj.io/refresh"] != "normal" {
			t.Errorf("refresh annotation = %q", ann["argocd.argoproj.io/refresh"])
		}
	})
}

func TestGetResource(t *testing.T) {
	m := newFakeManager(
		managedCluster("cluster-a", nil, nil),
		namespacedResource(k8s.GVRPolicy, "Policy", "cluster-a", "p1", nil, nil),
	)

	t.Run("cluster-scoped allowed", func(t *testing.T) {
		obj, err := m.GetResource(context.Background(),
			k8s.GVRManagedCluster.Group, k8s.GVRManagedCluster.Version, k8s.GVRManagedCluster.Resource, "_", "cluster-a")
		if err != nil {
			t.Fatalf("GetResource: %v", err)
		}
		if obj["kind"] != "ManagedCluster" {
			t.Errorf("kind = %v", obj["kind"])
		}
	})

	t.Run("namespaced allowed", func(t *testing.T) {
		if _, err := m.GetResource(context.Background(),
			k8s.GVRPolicy.Group, k8s.GVRPolicy.Version, k8s.GVRPolicy.Resource, "cluster-a", "p1"); err != nil {
			t.Fatalf("GetResource: %v", err)
		}
	})

	t.Run("not allowed", func(t *testing.T) {
		_, err := m.GetResource(context.Background(), "", "v1", "secrets", "default", "x")
		if !errors.Is(err, ErrResourceNotAllowed) {
			t.Errorf("expected ErrResourceNotAllowed, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := m.GetResource(context.Background(),
			k8s.GVRManagedCluster.Group, k8s.GVRManagedCluster.Version, k8s.GVRManagedCluster.Resource, "", "missing"); err == nil {
			t.Error("expected not-found error")
		}
	})
}

func TestGetPolicySummary(t *testing.T) {
	m := newFakeManager(
		managedCluster("cluster-a", nil, nil),
		managedCluster("cluster-b", nil, nil),
		namespacedResource(k8s.GVRPolicy, "Policy", "cluster-a", "p1", nil, map[string]interface{}{"status.compliant": "Compliant"}),
		namespacedResource(k8s.GVRPolicy, "Policy", "cluster-a", "p2", nil, map[string]interface{}{"status.compliant": "NonCompliant"}),
		namespacedResource(k8s.GVRPolicy, "Policy", "cluster-b", "p3", nil, map[string]interface{}{"status.compliant": "Unknown"}),
	)

	summary, err := m.GetPolicySummary(context.Background())
	if err != nil {
		t.Fatalf("GetPolicySummary: %v", err)
	}
	if summary.Total != 3 || summary.Compliant != 1 || summary.NonCompliant != 1 || summary.Unknown != 1 {
		t.Errorf("summary = %+v", summary)
	}
	if len(summary.NonCompliantPolicies) != 1 || summary.NonCompliantPolicies[0].Name != "p2" {
		t.Errorf("NonCompliantPolicies = %+v", summary.NonCompliantPolicies)
	}
	if len(summary.Clusters) != 2 {
		t.Errorf("Clusters = %d, want 2", len(summary.Clusters))
	}
}

func TestGetArgoSummary(t *testing.T) {
	const ns = "openshift-gitops"
	m := newFakeManager(
		namespacedResource(k8s.GVRArgoApplication, "Application", ns, "a1", nil, map[string]interface{}{
			"status.health.status": "Healthy", "status.sync.status": "Synced",
		}),
		namespacedResource(k8s.GVRArgoApplication, "Application", ns, "a2", nil, map[string]interface{}{
			"status.health.status": "Degraded", "status.sync.status": "OutOfSync",
		}),
		namespacedResource(k8s.GVRArgoApplication, "Application", ns, "a3", nil, map[string]interface{}{
			"status.health.status": "Progressing", "status.sync.status": "Unknown",
		}),
	)

	summary, err := m.GetArgoSummary(context.Background())
	if err != nil {
		t.Fatalf("GetArgoSummary: %v", err)
	}
	if summary.Total != 3 || summary.Healthy != 1 || summary.Degraded != 1 || summary.Other != 1 {
		t.Errorf("health summary = %+v", summary)
	}
	if summary.Synced != 1 || summary.OutOfSync != 1 {
		t.Errorf("sync summary = %+v", summary)
	}
}

func TestDeleteManagedCluster(t *testing.T) {
	m := newFakeManager(managedCluster("cluster-a", nil, nil))

	if err := m.DeleteManagedCluster(context.Background(), "cluster-a"); err != nil {
		t.Fatalf("DeleteManagedCluster: %v", err)
	}
	if _, err := m.GetManagedCluster(context.Background(), "cluster-a"); err == nil {
		t.Error("expected cluster to be gone after delete")
	}
}

func TestListClusterResources(t *testing.T) {
	const ns = "cluster-a"
	m := newFakeManager(
		namespacedResource(k8s.GVRClusterDeployment, "ClusterDeployment", ns, "cd", nil, nil),
		namespacedResource(k8s.GVRPolicy, "Policy", ns, "pol", nil, nil),
	)

	resources, err := m.ListClusterResources(context.Background(), ns)
	if err != nil {
		t.Fatalf("ListClusterResources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(resources))
	}
	kinds := map[string]bool{}
	for _, r := range resources {
		kinds[r.Kind] = true
	}
	if !kinds["ClusterDeployment"] || !kinds["Policy"] {
		t.Errorf("kinds = %v", kinds)
	}
}

func TestNotConnectedPaths(t *testing.T) {
	ctx := context.Background()
	m := NewManager(nil)

	tests := []struct {
		name string
		call func() error
	}{
		{"ListManagedClusters", func() error { _, err := m.ListManagedClusters(ctx); return err }},
		{"GetManagedCluster", func() error { _, err := m.GetManagedCluster(ctx, "x"); return err }},
		{"GetPipelineStatus", func() error { _, err := m.GetPipelineStatus(ctx, "x"); return err }},
		{"GetClusterPolicies", func() error { _, err := m.GetClusterPolicies(ctx, "x"); return err }},
		{"GetArgoApplications", func() error { _, err := m.GetArgoApplications(ctx); return err }},
		{"GetArgoApplication", func() error { _, err := m.GetArgoApplication(ctx, "x"); return err }},
		{"SyncArgoApplication", func() error { return m.SyncArgoApplication(ctx, "x", SyncOptions{}) }},
		{"RefreshArgoApplication", func() error { return m.RefreshArgoApplication(ctx, "x") }},
		{"SetPolicyDisabled", func() error { return m.SetPolicyDisabled(ctx, "ns", "x", true) }},
		{"GetResource", func() error { _, err := m.GetResource(ctx, "g", "v", "r", "", "x"); return err }},
		{"GetPolicySummary", func() error { _, err := m.GetPolicySummary(ctx); return err }},
		{"GetArgoSummary", func() error { _, err := m.GetArgoSummary(ctx); return err }},
		{"DeleteManagedCluster", func() error { return m.DeleteManagedCluster(ctx, "x") }},
		{"ListClusterResources", func() error { _, err := m.ListClusterResources(ctx, "x"); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertNotConnected(t, tt.call())
		})
	}
}

// assertNotConnected verifies err reports a missing Kubernetes client. The
// message is used rather than a sentinel so the test is independent of the
// error-typing change in #221.
func assertNotConnected(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "not connected to cluster") {
		t.Errorf("error = %v, want not-connected error", err)
	}
}
