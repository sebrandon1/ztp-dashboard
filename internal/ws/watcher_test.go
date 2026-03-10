package ws

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// helper to build an unstructured object with a name, namespace, and optional nested fields
func newUnstructured(name, namespace string, fields map[string]interface{}) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
		},
	}}
	for k, v := range fields {
		obj.Object[k] = v
	}
	return obj
}

func makeConditions(items ...map[string]interface{}) []interface{} {
	out := make([]interface{}, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

// ---------- classifyEvent ----------

func TestClassifyEvent(t *testing.T) {
	tests := []struct {
		name             string
		resourceType     string
		eventType        WatchEventType
		data             map[string]interface{}
		expectedSeverity string
		expectedInsight  string
	}{
		// Deleted events always return warning
		{
			name:             "deleted event returns warning",
			resourceType:     "ManagedCluster",
			eventType:        WatchEventDeleted,
			data:             map[string]interface{}{},
			expectedSeverity: "warning",
			expectedInsight:  "Resource removed from cluster",
		},

		// Added events
		{
			name:             "added ManagedCluster",
			resourceType:     "ManagedCluster",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{},
			expectedSeverity: "good",
			expectedInsight:  "New spoke cluster registered",
		},
		{
			name:             "added Policy compliant",
			resourceType:     "Policy",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{"compliant": "Compliant"},
			expectedSeverity: "good",
			expectedInsight:  "Policy applied and compliant",
		},
		{
			name:             "added Policy non-compliant",
			resourceType:     "Policy",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{"compliant": "NonCompliant"},
			expectedSeverity: "bad",
			expectedInsight:  "Policy not yet enforced on cluster",
		},
		{
			name:             "added Policy no compliance status",
			resourceType:     "Policy",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Policy created, awaiting evaluation",
		},
		{
			name:             "added BareMetalHost",
			resourceType:     "BareMetalHost",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Bare metal host discovered",
		},
		{
			name:             "added Agent",
			resourceType:     "Agent",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Agent registered for provisioning",
		},
		{
			name:             "added unknown resource",
			resourceType:     "InfraEnv",
			eventType:        WatchEventAdded,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Resource created",
		},

		// Modified ManagedCluster
		{
			name:         "modified ManagedCluster available",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "True",
				}),
			},
			expectedSeverity: "good",
			expectedInsight:  "Cluster is healthy and reporting",
		},
		{
			name:         "modified ManagedCluster unknown",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "Unknown",
				}),
			},
			expectedSeverity: "warning",
			expectedInsight:  "Cluster stopped heartbeating — may be unreachable",
		},
		{
			name:         "modified ManagedCluster not available",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "False",
				}),
			},
			expectedSeverity: "bad",
			expectedInsight:  "Cluster is not available",
		},
		{
			name:             "modified ManagedCluster no conditions",
			resourceType:     "ManagedCluster",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Cluster metadata updated",
		},

		// Modified ClusterDeployment
		{
			name:             "modified ClusterDeployment installed",
			resourceType:     "ClusterDeployment",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"installed": true},
			expectedSeverity: "good",
			expectedInsight:  "Cluster installation complete",
		},
		{
			name:         "modified ClusterDeployment failed condition",
			resourceType: "ClusterDeployment",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"installed": false,
				"conditions": makeConditions(map[string]interface{}{
					"type": "Failed", "status": "True",
				}),
			},
			expectedSeverity: "bad",
			expectedInsight:  "Deployment error — check conditions",
		},
		{
			name:             "modified ClusterDeployment progressing",
			resourceType:     "ClusterDeployment",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"installed": false},
			expectedSeverity: "info",
			expectedInsight:  "Deployment progressing",
		},

		// Modified ClusterInstance
		{
			name:         "modified ClusterInstance failed",
			resourceType: "ClusterInstance",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Failed", "status": "True",
				}),
			},
			expectedSeverity: "bad",
			expectedInsight:  "ClusterInstance hit an error",
		},
		{
			name:         "modified ClusterInstance all true",
			resourceType: "ClusterInstance",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Validated", "status": "True"},
					map[string]interface{}{"type": "Ready", "status": "True"},
				),
			},
			expectedSeverity: "good",
			expectedInsight:  "All validations and templates applied",
		},
		{
			name:             "modified ClusterInstance progressing",
			resourceType:     "ClusterInstance",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "ClusterInstance progressing",
		},

		// Modified BareMetalHost
		{
			name:             "modified BMH with error",
			resourceType:     "BareMetalHost",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"errorMessage": "BMC connection failed"},
			expectedSeverity: "bad",
			expectedInsight:  "Hardware error reported",
		},
		{
			name:             "modified BMH powered off",
			resourceType:     "BareMetalHost",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"poweredOn": false},
			expectedSeverity: "warning",
			expectedInsight:  "Host is powered off",
		},
		{
			name:             "modified BMH powered on",
			resourceType:     "BareMetalHost",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"poweredOn": true},
			expectedSeverity: "good",
			expectedInsight:  "Host is powered on and operational",
		},
		{
			name:             "modified BMH no power info",
			resourceType:     "BareMetalHost",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "BMH state updated",
		},

		// Modified AgentClusterInstall
		{
			name:         "modified ACI failed",
			resourceType: "AgentClusterInstall",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Failed", "status": "True",
				}),
			},
			expectedSeverity: "bad",
			expectedInsight:  "Installation error — may need intervention",
		},
		{
			name:         "modified ACI all true",
			resourceType: "AgentClusterInstall",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Completed", "status": "True",
				}),
			},
			expectedSeverity: "good",
			expectedInsight:  "Cluster install completed successfully",
		},
		{
			name:             "modified ACI progressing",
			resourceType:     "AgentClusterInstall",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Install progressing",
		},

		// Modified Agent
		{
			name:         "modified Agent failed",
			resourceType: "Agent",
			eventType:    WatchEventModified,
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Failed", "status": "True",
				}),
			},
			expectedSeverity: "bad",
			expectedInsight:  "Agent hit a validation or binding error",
		},
		{
			name:             "modified Agent updated",
			resourceType:     "Agent",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Agent status updated",
		},

		// Modified InfraEnv
		{
			name:             "modified InfraEnv has ISO",
			resourceType:     "InfraEnv",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"hasISO": true},
			expectedSeverity: "good",
			expectedInsight:  "Discovery ISO is ready",
		},
		{
			name:             "modified InfraEnv no ISO",
			resourceType:     "InfraEnv",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "InfraEnv updated",
		},

		// Modified Policy
		{
			name:             "modified Policy compliant",
			resourceType:     "Policy",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"compliant": "Compliant"},
			expectedSeverity: "good",
			expectedInsight:  "All policy rules are satisfied",
		},
		{
			name:             "modified Policy non-compliant",
			resourceType:     "Policy",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"compliant": "NonCompliant"},
			expectedSeverity: "bad",
			expectedInsight:  "Policy violations detected — remediation needed",
		},
		{
			name:             "modified Policy pending",
			resourceType:     "Policy",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"compliant": ""},
			expectedSeverity: "info",
			expectedInsight:  "Policy status pending evaluation",
		},

		// Modified Application
		{
			name:             "modified Application degraded",
			resourceType:     "Application",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"health": "Degraded"},
			expectedSeverity: "bad",
			expectedInsight:  "ArgoCD app is degraded",
		},
		{
			name:             "modified Application missing health",
			resourceType:     "Application",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"health": "Missing"},
			expectedSeverity: "bad",
			expectedInsight:  "ArgoCD app is degraded",
		},
		{
			name:             "modified Application out of sync",
			resourceType:     "Application",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"sync": "OutOfSync"},
			expectedSeverity: "warning",
			expectedInsight:  "App drifted from Git — sync may be needed",
		},
		{
			name:             "modified Application healthy and synced",
			resourceType:     "Application",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{"health": "Healthy", "sync": "Synced"},
			expectedSeverity: "good",
			expectedInsight:  "App is healthy and in sync with Git",
		},
		{
			name:             "modified Application unknown state",
			resourceType:     "Application",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "ArgoCD app state changed",
		},

		// Unknown resource type on modified
		{
			name:             "modified unknown resource type",
			resourceType:     "SomethingElse",
			eventType:        WatchEventModified,
			data:             map[string]interface{}{},
			expectedSeverity: "info",
			expectedInsight:  "Resource updated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity, insight := classifyEvent(tt.resourceType, tt.eventType, tt.data)
			if severity != tt.expectedSeverity {
				t.Errorf("severity = %q, want %q", severity, tt.expectedSeverity)
			}
			if insight != tt.expectedInsight {
				t.Errorf("insight = %q, want %q", insight, tt.expectedInsight)
			}
		})
	}
}

// ---------- buildSummary ----------

func TestBuildSummary(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		eventType    WatchEventType
		obj          *unstructured.Unstructured
		data         map[string]interface{}
		expected     string
	}{
		// Added / Deleted use generic format
		{
			name:         "added event",
			resourceType: "ManagedCluster",
			eventType:    WatchEventAdded,
			obj:          newUnstructured("spoke1", "", nil),
			data:         map[string]interface{}{},
			expected:     "ManagedCluster spoke1 created",
		},
		{
			name:         "deleted event",
			resourceType: "Policy",
			eventType:    WatchEventDeleted,
			obj:          newUnstructured("my-policy", "default", nil),
			data:         map[string]interface{}{},
			expected:     "Policy my-policy deleted",
		},

		// Modified ManagedCluster
		{
			name:         "modified ManagedCluster available",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			obj:          newUnstructured("spoke1", "", nil),
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "True",
				}),
			},
			expected: "Cluster spoke1 is now available",
		},
		{
			name:         "modified ManagedCluster unavailable with reason",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			obj:          newUnstructured("spoke1", "", nil),
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "False", "reason": "Unreachable",
				}),
			},
			expected: "Cluster spoke1 unavailable: Unreachable",
		},
		{
			name:         "modified ManagedCluster unavailable with message only",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			obj:          newUnstructured("spoke1", "", nil),
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "False", "message": "Connection lost",
				}),
			},
			expected: "Cluster spoke1 unavailable — Connection lost",
		},
		{
			name:         "modified ManagedCluster unavailable no reason or message",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			obj:          newUnstructured("spoke1", "", nil),
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "ManagedClusterConditionAvailable", "status": "False",
				}),
			},
			expected: "Cluster spoke1 is unavailable",
		},
		{
			name:         "modified ManagedCluster no relevant condition",
			resourceType: "ManagedCluster",
			eventType:    WatchEventModified,
			obj:          newUnstructured("spoke1", "", nil),
			data:         map[string]interface{}{},
			expected:     "ManagedCluster spoke1 updated",
		},

		// Modified ClusterDeployment
		{
			name:         "modified ClusterDeployment installed",
			resourceType: "ClusterDeployment",
			eventType:    WatchEventModified,
			obj:          newUnstructured("cd1", "default", nil),
			data:         map[string]interface{}{"installed": true},
			expected:     "Cluster cd1 is installed",
		},
		{
			name:         "modified ClusterDeployment with condition message",
			resourceType: "ClusterDeployment",
			eventType:    WatchEventModified,
			obj:          newUnstructured("cd1", "default", nil),
			data: map[string]interface{}{
				"installed": false,
				"conditions": makeConditions(map[string]interface{}{
					"type": "Ready", "status": "False", "message": "Provisioning in progress",
				}),
			},
			expected: "ClusterDeployment cd1 — Provisioning in progress",
		},
		{
			name:         "modified ClusterDeployment no conditions",
			resourceType: "ClusterDeployment",
			eventType:    WatchEventModified,
			obj:          newUnstructured("cd1", "default", nil),
			data:         map[string]interface{}{"installed": false},
			expected:     "ClusterDeployment cd1 updated",
		},

		// Modified BareMetalHost
		{
			name:         "modified BMH powered on with provisioning state",
			resourceType: "BareMetalHost",
			eventType:    WatchEventModified,
			obj:          newUnstructured("bmh1", "openshift", nil),
			data: map[string]interface{}{
				"poweredOn":    true,
				"provisioning": map[string]interface{}{"state": "provisioned"},
			},
			expected: "BMH bmh1 — powered on, state: provisioned",
		},
		{
			name:         "modified BMH with error",
			resourceType: "BareMetalHost",
			eventType:    WatchEventModified,
			obj:          newUnstructured("bmh1", "openshift", nil),
			data: map[string]interface{}{
				"errorMessage": "BMC connection refused",
			},
			expected: "BMH bmh1 — error: BMC connection refused",
		},
		{
			name:         "modified BMH no data",
			resourceType: "BareMetalHost",
			eventType:    WatchEventModified,
			obj:          newUnstructured("bmh1", "openshift", nil),
			data:         map[string]interface{}{},
			expected:     "BareMetalHost bmh1 updated",
		},

		// Modified Policy
		{
			name:         "modified Policy compliant",
			resourceType: "Policy",
			eventType:    WatchEventModified,
			obj:          newUnstructured("policy1", "default", nil),
			data:         map[string]interface{}{"compliant": "Compliant"},
			expected:     "Policy policy1 is now compliant",
		},
		{
			name:         "modified Policy non-compliant",
			resourceType: "Policy",
			eventType:    WatchEventModified,
			obj:          newUnstructured("policy1", "default", nil),
			data:         map[string]interface{}{"compliant": "NonCompliant"},
			expected:     "Policy policy1 is non-compliant",
		},
		{
			name:         "modified Policy other status",
			resourceType: "Policy",
			eventType:    WatchEventModified,
			obj:          newUnstructured("policy1", "default", nil),
			data:         map[string]interface{}{"compliant": "Pending"},
			expected:     "Policy policy1 compliance: Pending",
		},
		{
			name:         "modified Policy no status",
			resourceType: "Policy",
			eventType:    WatchEventModified,
			obj:          newUnstructured("policy1", "default", nil),
			data:         map[string]interface{}{},
			expected:     "Policy policy1 updated",
		},

		// Modified Application
		{
			name:         "modified Application with health and sync",
			resourceType: "Application",
			eventType:    WatchEventModified,
			obj:          newUnstructured("app1", "argocd", nil),
			data:         map[string]interface{}{"health": "Healthy", "sync": "Synced"},
			expected:     "ArgoCD app app1 — health: Healthy, sync: Synced",
		},
		{
			name:         "modified Application no data",
			resourceType: "Application",
			eventType:    WatchEventModified,
			obj:          newUnstructured("app1", "argocd", nil),
			data:         map[string]interface{}{},
			expected:     "Application app1 updated",
		},

		// Modified Agent
		{
			name:         "modified Agent with hostname and role",
			resourceType: "Agent",
			eventType:    WatchEventModified,
			obj:          newUnstructured("agent1", "default", nil),
			data:         map[string]interface{}{"hostname": "worker-0", "role": "worker"},
			expected:     "Agent agent1 — worker-0, role: worker",
		},
		{
			name:         "modified Agent no data",
			resourceType: "Agent",
			eventType:    WatchEventModified,
			obj:          newUnstructured("agent1", "default", nil),
			data:         map[string]interface{}{},
			expected:     "Agent agent1 updated",
		},

		// Modified InfraEnv
		{
			name:         "modified InfraEnv has ISO",
			resourceType: "InfraEnv",
			eventType:    WatchEventModified,
			obj:          newUnstructured("infra1", "default", nil),
			data:         map[string]interface{}{"hasISO": true},
			expected:     "InfraEnv infra1 — ISO image available",
		},
		{
			name:         "modified InfraEnv no ISO",
			resourceType: "InfraEnv",
			eventType:    WatchEventModified,
			obj:          newUnstructured("infra1", "default", nil),
			data:         map[string]interface{}{},
			expected:     "InfraEnv infra1 updated",
		},

		// Modified ClusterInstance
		{
			name:         "modified ClusterInstance with condition",
			resourceType: "ClusterInstance",
			eventType:    WatchEventModified,
			obj:          newUnstructured("ci1", "default", nil),
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Ready", "status": "False", "message": "Waiting for templates",
				}),
			},
			expected: "ClusterInstance ci1 — Waiting for templates",
		},
		{
			name:         "modified ClusterInstance no conditions",
			resourceType: "ClusterInstance",
			eventType:    WatchEventModified,
			obj:          newUnstructured("ci1", "default", nil),
			data:         map[string]interface{}{},
			expected:     "ClusterInstance ci1 updated",
		},

		// Modified AgentClusterInstall
		{
			name:         "modified ACI with condition",
			resourceType: "AgentClusterInstall",
			eventType:    WatchEventModified,
			obj:          newUnstructured("aci1", "default", nil),
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Completed", "status": "False", "reason": "Installing",
				}),
			},
			expected: "AgentClusterInstall aci1 — Installing",
		},
		{
			name:         "modified ACI no conditions",
			resourceType: "AgentClusterInstall",
			eventType:    WatchEventModified,
			obj:          newUnstructured("aci1", "default", nil),
			data:         map[string]interface{}{},
			expected:     "AgentClusterInstall aci1 updated",
		},

		// Unknown resource type
		{
			name:         "modified unknown type",
			resourceType: "Unknown",
			eventType:    WatchEventModified,
			obj:          newUnstructured("obj1", "default", nil),
			data:         map[string]interface{}{},
			expected:     "Unknown obj1 updated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSummary(tt.resourceType, tt.eventType, tt.obj, tt.data)
			if got != tt.expected {
				t.Errorf("buildSummary() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// ---------- extractRelevantData ----------

func TestExtractRelevantData(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		obj          *unstructured.Unstructured
		checkFunc    func(t *testing.T, data map[string]interface{})
	}{
		{
			name:         "ManagedCluster with conditions and labels",
			resourceType: "ManagedCluster",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("mc1", "", nil)
				obj.SetLabels(map[string]string{
					"vendor":           "OpenShift",
					"openshiftVersion": "4.14",
				})
				obj.Object["status"] = map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"type": "Available", "status": "True"},
					},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["vendor"] != "OpenShift" {
					t.Errorf("expected vendor=OpenShift, got %v", data["vendor"])
				}
				if data["openshiftVersion"] != "4.14" {
					t.Errorf("expected openshiftVersion=4.14, got %v", data["openshiftVersion"])
				}
				if _, ok := data["conditions"]; !ok {
					t.Error("expected conditions to be present")
				}
			},
		},
		{
			name:         "ManagedCluster without labels",
			resourceType: "ManagedCluster",
			obj:          newUnstructured("mc2", "", nil),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["vendor"]; ok {
					t.Error("expected no vendor key")
				}
			},
		},
		{
			name:         "ClusterDeployment with installed and conditions",
			resourceType: "ClusterDeployment",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("cd1", "default", nil)
				obj.Object["spec"] = map[string]interface{}{
					"installed": true,
				}
				obj.Object["status"] = map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"type": "Ready", "status": "True", "message": "Done"},
					},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["installed"] != true {
					t.Errorf("expected installed=true, got %v", data["installed"])
				}
				if _, ok := data["conditions"]; !ok {
					t.Error("expected conditions to be present")
				}
			},
		},
		{
			name:         "BareMetalHost with provisioning and error",
			resourceType: "BareMetalHost",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("bmh1", "openshift", nil)
				obj.Object["status"] = map[string]interface{}{
					"provisioning": map[string]interface{}{"state": "provisioned"},
					"poweredOn":    true,
					"errorMessage": "some error",
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["poweredOn"] != true {
					t.Errorf("expected poweredOn=true, got %v", data["poweredOn"])
				}
				if data["errorMessage"] != "some error" {
					t.Errorf("expected errorMessage, got %v", data["errorMessage"])
				}
				prov, ok := data["provisioning"].(map[string]interface{})
				if !ok {
					t.Fatal("expected provisioning map")
				}
				if prov["state"] != "provisioned" {
					t.Errorf("expected provisioning state=provisioned, got %v", prov["state"])
				}
			},
		},
		{
			name:         "BareMetalHost no error message omitted",
			resourceType: "BareMetalHost",
			obj:          newUnstructured("bmh2", "openshift", nil),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["errorMessage"]; ok {
					t.Error("expected errorMessage to be absent when empty")
				}
			},
		},
		{
			name:         "AgentClusterInstall with conditions and progress",
			resourceType: "AgentClusterInstall",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("aci1", "default", nil)
				obj.Object["status"] = map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"type": "Completed", "status": "True"},
					},
					"progress": map[string]interface{}{"totalPercentage": int64(100)},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["conditions"]; !ok {
					t.Error("expected conditions")
				}
				if _, ok := data["progress"]; !ok {
					t.Error("expected progress")
				}
			},
		},
		{
			name:         "Agent with inventory and role",
			resourceType: "Agent",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("agent1", "default", nil)
				obj.Object["spec"] = map[string]interface{}{"role": "worker"}
				obj.Object["status"] = map[string]interface{}{
					"inventory": map[string]interface{}{"hostname": "node-0"},
					"conditions": []interface{}{
						map[string]interface{}{"type": "Validated", "status": "True"},
					},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["hostname"] != "node-0" {
					t.Errorf("expected hostname=node-0, got %v", data["hostname"])
				}
				if data["role"] != "worker" {
					t.Errorf("expected role=worker, got %v", data["role"])
				}
				if _, ok := data["conditions"]; !ok {
					t.Error("expected conditions")
				}
			},
		},
		{
			name:         "InfraEnv with ISO URL",
			resourceType: "InfraEnv",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("infra1", "default", nil)
				obj.Object["status"] = map[string]interface{}{
					"isoDownloadURL": "https://example.com/iso",
					"conditions": []interface{}{
						map[string]interface{}{"type": "Ready", "status": "True"},
					},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["hasISO"] != true {
					t.Errorf("expected hasISO=true, got %v", data["hasISO"])
				}
				if _, ok := data["conditions"]; !ok {
					t.Error("expected conditions")
				}
			},
		},
		{
			name:         "InfraEnv without ISO URL",
			resourceType: "InfraEnv",
			obj:          newUnstructured("infra2", "default", nil),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["hasISO"]; ok {
					t.Error("expected hasISO to be absent")
				}
			},
		},
		{
			name:         "Policy with compliant status",
			resourceType: "Policy",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("p1", "default", nil)
				obj.Object["status"] = map[string]interface{}{
					"compliant": "Compliant",
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["compliant"] != "Compliant" {
					t.Errorf("expected compliant=Compliant, got %v", data["compliant"])
				}
			},
		},
		{
			name:         "Application with health and sync",
			resourceType: "Application",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("app1", "argocd", nil)
				obj.Object["status"] = map[string]interface{}{
					"health": map[string]interface{}{"status": "Healthy"},
					"sync":   map[string]interface{}{"status": "Synced"},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if data["health"] != "Healthy" {
					t.Errorf("expected health=Healthy, got %v", data["health"])
				}
				if data["sync"] != "Synced" {
					t.Errorf("expected sync=Synced, got %v", data["sync"])
				}
			},
		},
		{
			name:         "ClusterInstance with conditions",
			resourceType: "ClusterInstance",
			obj: func() *unstructured.Unstructured {
				obj := newUnstructured("ci1", "default", nil)
				obj.Object["status"] = map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{"type": "Validated", "status": "True"},
					},
				}
				return obj
			}(),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if _, ok := data["conditions"]; !ok {
					t.Error("expected conditions")
				}
			},
		},
		{
			name:         "unknown resource type returns empty map",
			resourceType: "Unknown",
			obj:          newUnstructured("obj1", "default", nil),
			checkFunc: func(t *testing.T, data map[string]interface{}) {
				if len(data) != 0 {
					t.Errorf("expected empty map for unknown resource, got %v", data)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := extractRelevantData(tt.resourceType, tt.obj)
			tt.checkFunc(t, data)
		})
	}
}

// ---------- hasFailedCondition ----------

func TestHasFailedCondition(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "no conditions key",
			data:     map[string]interface{}{},
			expected: false,
		},
		{
			name:     "conditions is not a slice",
			data:     map[string]interface{}{"conditions": "not-a-slice"},
			expected: false,
		},
		{
			name: "Failed condition with status True",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Failed", "status": "True",
				}),
			},
			expected: true,
		},
		{
			name: "Error condition with status True",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Error", "status": "True",
				}),
			},
			expected: true,
		},
		{
			name: "Ready False with InstallationFailed reason",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Ready", "status": "False", "reason": "InstallationFailed",
				}),
			},
			expected: true,
		},
		{
			name: "Completed False with Failed reason",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Completed", "status": "False", "reason": "Failed",
				}),
			},
			expected: true,
		},
		{
			name: "Available False with reason containing Error",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Available", "status": "False", "reason": "ConnectionError",
				}),
			},
			expected: true,
		},
		{
			name: "Ready False with non-failure reason",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Ready", "status": "False", "reason": "Installing",
				}),
			},
			expected: false,
		},
		{
			name: "Failed condition with status False (not a failure)",
			data: map[string]interface{}{
				"conditions": makeConditions(map[string]interface{}{
					"type": "Failed", "status": "False",
				}),
			},
			expected: false,
		},
		{
			name: "all conditions healthy",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready", "status": "True"},
					map[string]interface{}{"type": "Available", "status": "True"},
				),
			},
			expected: false,
		},
		{
			name: "empty conditions slice",
			data: map[string]interface{}{
				"conditions": makeConditions(),
			},
			expected: false,
		},
		{
			name: "condition that is not a map",
			data: map[string]interface{}{
				"conditions": []interface{}{"not-a-map"},
			},
			expected: false,
		},
		{
			name: "mixed conditions with one failure",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready", "status": "True"},
					map[string]interface{}{"type": "Failed", "status": "True"},
				),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasFailedCondition(tt.data)
			if got != tt.expected {
				t.Errorf("hasFailedCondition() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// ---------- allConditionsTrue ----------

func TestAllConditionsTrue(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name:     "no conditions key",
			data:     map[string]interface{}{},
			expected: false,
		},
		{
			name:     "conditions not a slice",
			data:     map[string]interface{}{"conditions": "string"},
			expected: false,
		},
		{
			name: "empty conditions slice",
			data: map[string]interface{}{
				"conditions": makeConditions(),
			},
			expected: false,
		},
		{
			name: "all conditions True",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready", "status": "True"},
					map[string]interface{}{"type": "Available", "status": "True"},
					map[string]interface{}{"type": "Validated", "status": "True"},
				),
			},
			expected: true,
		},
		{
			name: "single condition True",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready", "status": "True"},
				),
			},
			expected: true,
		},
		{
			name: "one condition False",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready", "status": "True"},
					map[string]interface{}{"type": "Available", "status": "False"},
				),
			},
			expected: false,
		},
		{
			name: "one condition Unknown",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready", "status": "True"},
					map[string]interface{}{"type": "Available", "status": "Unknown"},
				),
			},
			expected: false,
		},
		{
			name: "condition missing status field",
			data: map[string]interface{}{
				"conditions": makeConditions(
					map[string]interface{}{"type": "Ready"},
				),
			},
			expected: false,
		},
		{
			name: "non-map condition in slice skipped, remaining all true",
			data: map[string]interface{}{
				"conditions": []interface{}{
					"not-a-map",
					map[string]interface{}{"type": "Ready", "status": "True"},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allConditionsTrue(tt.data)
			if got != tt.expected {
				t.Errorf("allConditionsTrue() = %v, want %v", got, tt.expected)
			}
		})
	}
}
