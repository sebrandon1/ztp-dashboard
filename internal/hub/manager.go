package hub

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
)

type Manager struct {
	client *k8s.Client
}

func NewManager(client *k8s.Client) *Manager {
	return &Manager{client: client}
}

type HubStatus struct {
	Connected     bool   `json:"connected"`
	ServerVersion string `json:"serverVersion,omitempty"`
}

type ManagedClusterInfo struct {
	Name              string                   `json:"name"`
	Labels            map[string]string        `json:"labels,omitempty"`
	Conditions        []map[string]interface{} `json:"conditions,omitempty"`
	Available         string                   `json:"available"`
	HubAccepted       string                   `json:"hubAccepted"`
	Joined            string                   `json:"joined"`
	OpenShiftVersion  string                   `json:"openshiftVersion,omitempty"`
	CreationTimestamp string                   `json:"creationTimestamp"`
}

type PipelineStatus struct {
	ClusterName         string           `json:"clusterName"`
	ClusterInstance     *ResourceStatus  `json:"clusterInstance,omitempty"`
	ClusterDeployment   *ResourceStatus  `json:"clusterDeployment,omitempty"`
	InfraEnv            *ResourceStatus  `json:"infraEnv,omitempty"`
	BareMetalHosts      []ResourceStatus `json:"bareMetalHosts,omitempty"`
	Agents              []ResourceStatus `json:"agents,omitempty"`
	AgentClusterInstall *ResourceStatus  `json:"agentClusterInstall,omitempty"`
	ManagedCluster      *ResourceStatus  `json:"managedCluster,omitempty"`
}

type ResourceStatus struct {
	Name       string                   `json:"name"`
	Namespace  string                   `json:"namespace,omitempty"`
	Status     string                   `json:"status"`
	Conditions []map[string]interface{} `json:"conditions,omitempty"`
	Data       map[string]interface{}   `json:"data,omitempty"`
}

func (m *Manager) GetStatus() HubStatus {
	if m.client == nil {
		return HubStatus{Connected: false}
	}
	return HubStatus{
		Connected:     true,
		ServerVersion: m.client.ServerVersion,
	}
}

func (m *Manager) ListManagedClusters(ctx context.Context) ([]ManagedClusterInfo, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	list, err := m.client.Dynamic.Resource(k8s.GVRManagedCluster).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing managed clusters: %w", err)
	}

	clusters := make([]ManagedClusterInfo, 0, len(list.Items))
	for _, item := range list.Items {
		cluster := parseManagedCluster(&item)
		if cluster.Name == "local-cluster" {
			continue
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

func (m *Manager) GetManagedCluster(ctx context.Context, name string) (*ManagedClusterInfo, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	obj, err := m.client.Dynamic.Resource(k8s.GVRManagedCluster).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting managed cluster %s: %w", name, err)
	}

	cluster := parseManagedCluster(obj)
	return &cluster, nil
}

func (m *Manager) GetPipelineStatus(ctx context.Context, clusterName string) (*PipelineStatus, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pipeline := &PipelineStatus{ClusterName: clusterName}

	if ci, err := m.getNamespacedResource(ctx, k8s.GVRClusterInstance, clusterName, clusterName); err == nil {
		pipeline.ClusterInstance = ci
	}
	if cd, err := m.getNamespacedResource(ctx, k8s.GVRClusterDeployment, clusterName, clusterName); err == nil {
		pipeline.ClusterDeployment = cd
	}
	if ie, err := m.getNamespacedResource(ctx, k8s.GVRInfraEnv, clusterName, clusterName); err == nil {
		pipeline.InfraEnv = ie
	}
	if aci, err := m.getNamespacedResource(ctx, k8s.GVRAgentClusterInstall, clusterName, clusterName); err == nil {
		pipeline.AgentClusterInstall = aci
	}
	if bmhs, err := m.listNamespacedResources(ctx, k8s.GVRBareMetalHost, clusterName); err == nil {
		pipeline.BareMetalHosts = bmhs
	}
	if agents, err := m.listNamespacedResources(ctx, k8s.GVRAgent, clusterName); err == nil {
		pipeline.Agents = agents
	}

	if mc, err := m.GetManagedCluster(ctx, clusterName); err == nil {
		pipeline.ManagedCluster = &ResourceStatus{
			Name:       mc.Name,
			Status:     mc.Available,
			Conditions: mc.Conditions,
		}
	}

	return pipeline, nil
}

func (m *Manager) getNamespacedResource(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*ResourceStatus, error) {
	obj, err := m.client.Dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return parseResourceStatus(obj), nil
}

func (m *Manager) listNamespacedResources(ctx context.Context, gvr schema.GroupVersionResource, namespace string) ([]ResourceStatus, error) {
	list, err := m.client.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	results := make([]ResourceStatus, 0, len(list.Items))
	for _, item := range list.Items {
		results = append(results, *parseResourceStatus(&item))
	}
	return results, nil
}

func parseManagedCluster(obj *unstructured.Unstructured) ManagedClusterInfo {
	info := ManagedClusterInfo{
		Name:              obj.GetName(),
		Labels:            obj.GetLabels(),
		CreationTimestamp: obj.GetCreationTimestamp().Format(time.RFC3339),
	}

	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found {
		info.Conditions = parseConditions(conditions)

		for _, c := range info.Conditions {
			condType, _ := c["type"].(string)
			status, _ := c["status"].(string)
			switch condType {
			case "ManagedClusterConditionAvailable":
				if status == "True" {
					info.Available = "True"
				} else {
					info.Available = "False"
				}
			case "HubAcceptedManagedCluster":
				info.HubAccepted = status
			case "ManagedClusterJoined":
				info.Joined = status
			}
		}
	}

	if v, ok := info.Labels["openshiftVersion"]; ok {
		info.OpenShiftVersion = v
	}

	return info
}

func parseResourceStatus(obj *unstructured.Unstructured) *ResourceStatus {
	rs := &ResourceStatus{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		Data:      make(map[string]interface{}),
	}

	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found {
		rs.Conditions = parseConditions(conditions)
		rs.Status = deriveStatus(rs.Conditions)
	} else {
		rs.Status = "Unknown"
	}

	return rs
}

func parseConditions(raw []interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(raw))
	for _, c := range raw {
		if cm, ok := c.(map[string]interface{}); ok {
			result = append(result, cm)
		}
	}
	return result
}

func deriveStatus(conditions []map[string]interface{}) string {
	for _, c := range conditions {
		condType, _ := c["type"].(string)
		status, _ := c["status"].(string)
		reason, _ := c["reason"].(string)

		if status == "False" && (condType == "Completed" || condType == "Ready" || condType == "Available") {
			if reason != "" {
				return reason
			}
			return "InProgress"
		}
		if status == "True" && (condType == "Failed" || condType == "Error") {
			return "Error"
		}
		if status == "True" && (condType == "Completed" || condType == "Ready" || condType == "Available") {
			return "Completed"
		}
	}
	return "Pending"
}

func (m *Manager) GetClusterPolicies(ctx context.Context, clusterName string) ([]ResourceStatus, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	list, err := m.client.Dynamic.Resource(k8s.GVRPolicy).Namespace(clusterName).List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Debug("could not list policies", "namespace", clusterName, "error", err)
		return nil, err
	}

	policies := make([]ResourceStatus, 0, len(list.Items))
	for _, item := range list.Items {
		rs := parseResourceStatus(&item)
		compliant, _, _ := unstructured.NestedString(item.Object, "status", "compliant")
		rs.Data["compliant"] = compliant
		rs.Status = compliant
		policies = append(policies, *rs)
	}
	return policies, nil
}

func (m *Manager) GetArgoApplications(ctx context.Context) ([]ResourceStatus, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	list, err := m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace("openshift-gitops").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing argo applications: %w", err)
	}

	apps := make([]ResourceStatus, 0, len(list.Items))
	for _, item := range list.Items {
		rs := parseResourceStatus(&item)
		health, _, _ := unstructured.NestedString(item.Object, "status", "health", "status")
		syncStatus, _, _ := unstructured.NestedString(item.Object, "status", "sync", "status")
		rs.Data["health"] = health
		rs.Data["sync"] = syncStatus
		rs.Status = health
		apps = append(apps, *rs)
	}
	return apps, nil
}
