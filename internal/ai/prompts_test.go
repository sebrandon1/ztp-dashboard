package ai

import (
	"strings"
	"testing"
)

func TestProvisioningErrorPrompt(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		conditions  interface{}
		keywords    []string
	}{
		{
			name:        "basic cluster with string conditions",
			clusterName: "spoke-cluster-1",
			conditions:  "ImagePullBackOff on node-0",
			keywords:    []string{"spoke-cluster-1", "provisioning", "root cause", "remediation"},
		},
		{
			name:        "cluster with map conditions",
			clusterName: "edge-site-42",
			conditions: map[string]string{
				"type":    "Failed",
				"message": "BMC unreachable",
			},
			keywords: []string{"edge-site-42", "provisioning", "went wrong"},
		},
		{
			name:        "cluster with nil conditions",
			clusterName: "test-cluster",
			conditions:  nil,
			keywords:    []string{"test-cluster", "provisioning"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProvisioningErrorPrompt(tt.clusterName, tt.conditions)
			if result == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, kw := range tt.keywords {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(kw)) {
					t.Errorf("prompt missing keyword %q", kw)
				}
			}
		})
	}
}

func TestClusterHealthPrompt(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		healthData  interface{}
		keywords    []string
	}{
		{
			name:        "basic health check",
			clusterName: "prod-spoke-1",
			healthData:  "all nodes ready",
			keywords:    []string{"prod-spoke-1", "health", "assessment", "degraded"},
		},
		{
			name:        "health with map data",
			clusterName: "staging-cluster",
			healthData: map[string]interface{}{
				"nodes":      3,
				"podsReady":  true,
				"conditions": []string{"Ready"},
			},
			keywords: []string{"staging-cluster", "health", "recommended actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClusterHealthPrompt(tt.clusterName, tt.healthData)
			if result == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, kw := range tt.keywords {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(kw)) {
					t.Errorf("prompt missing keyword %q", kw)
				}
			}
		})
	}
}

func TestPolicyCompliancePrompt(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		policies    interface{}
		keywords    []string
	}{
		{
			name:        "basic compliance check",
			clusterName: "spoke-01",
			policies:    "NonCompliant",
			keywords:    []string{"spoke-01", "compliance", "non-compliant", "remediation"},
		},
		{
			name:        "compliance with list of policies",
			clusterName: "dc-east-cluster",
			policies: []map[string]string{
				{"name": "policy-sriov", "status": "Compliant"},
				{"name": "policy-ptp", "status": "NonCompliant"},
			},
			keywords: []string{"dc-east-cluster", "RHACM", "compliance", "best practices"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PolicyCompliancePrompt(tt.clusterName, tt.policies)
			if result == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, kw := range tt.keywords {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(kw)) {
					t.Errorf("prompt missing keyword %q", kw)
				}
			}
		})
	}
}

func TestBMCErrorPrompt(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		bmcData     interface{}
		keywords    []string
	}{
		{
			name:        "basic BMC error",
			clusterName: "bare-metal-site",
			bmcData:     "connection refused to BMC endpoint",
			keywords:    []string{"bare-metal-site", "BMC", "hardware", "troubleshooting"},
		},
		{
			name:        "BMC with structured data",
			clusterName: "edge-node-5",
			bmcData: map[string]interface{}{
				"host":         "worker-0",
				"errorMessage": "Redfish timeout",
				"poweredOn":    false,
			},
			keywords: []string{"edge-node-5", "BMC", "Redfish", "errors"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BMCErrorPrompt(tt.clusterName, tt.bmcData)
			if result == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, kw := range tt.keywords {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(kw)) {
					t.Errorf("prompt missing keyword %q", kw)
				}
			}
		})
	}
}

func TestGeneralDiagnosePrompt(t *testing.T) {
	tests := []struct {
		name     string
		context  map[string]interface{}
		keywords []string
	}{
		{
			name: "basic context",
			context: map[string]interface{}{
				"cluster": "spoke-1",
				"issue":   "pods crashing",
			},
			keywords: []string{"ZTP", "diagnostic", "root cause", "remediation"},
		},
		{
			name:     "empty context",
			context:  map[string]interface{}{},
			keywords: []string{"ZTP", "diagnostic", "anomalies"},
		},
		{
			name: "context with nested data",
			context: map[string]interface{}{
				"cluster":    "prod-edge",
				"conditions": []string{"Failed", "Degraded"},
				"nodeCount":  5,
			},
			keywords: []string{"OpenShift", "diagnostic", "remediation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneralDiagnosePrompt(tt.context)
			if result == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, kw := range tt.keywords {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(kw)) {
					t.Errorf("prompt missing keyword %q", kw)
				}
			}
		})
	}
}
