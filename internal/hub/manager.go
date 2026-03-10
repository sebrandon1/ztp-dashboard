package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

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

		disabled, _, _ := unstructured.NestedBool(item.Object, "spec", "disabled")
		rs.Data["disabled"] = disabled

		remediationAction, _, _ := unstructured.NestedString(item.Object, "spec", "remediationAction")
		rs.Data["remediationAction"] = remediationAction

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

type ArgoApplicationDetail struct {
	Name           string                   `json:"name"`
	Namespace      string                   `json:"namespace"`
	Health         string                   `json:"health"`
	SyncStatus     string                   `json:"syncStatus"`
	SyncRevision   string                   `json:"syncRevision,omitempty"`
	Source         map[string]interface{}   `json:"source,omitempty"`
	Conditions     []map[string]interface{} `json:"conditions,omitempty"`
	Resources      []map[string]interface{} `json:"resources,omitempty"`
	OperationState map[string]interface{}   `json:"operationState,omitempty"`
}

func (m *Manager) GetArgoApplication(ctx context.Context, name string) (*ArgoApplicationDetail, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	obj, err := m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace("openshift-gitops").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting argo application %s: %w", name, err)
	}

	detail := &ArgoApplicationDetail{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	detail.Health, _, _ = unstructured.NestedString(obj.Object, "status", "health", "status")
	detail.SyncStatus, _, _ = unstructured.NestedString(obj.Object, "status", "sync", "status")
	detail.SyncRevision, _, _ = unstructured.NestedString(obj.Object, "status", "sync", "revision")

	if source, found, _ := unstructured.NestedMap(obj.Object, "spec", "source"); found {
		detail.Source = source
	}

	if conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions"); found {
		detail.Conditions = parseConditions(conditions)
	}

	if resources, found, _ := unstructured.NestedSlice(obj.Object, "status", "resources"); found {
		for _, r := range resources {
			if rm, ok := r.(map[string]interface{}); ok {
				detail.Resources = append(detail.Resources, rm)
			}
		}
	}

	if opState, found, _ := unstructured.NestedMap(obj.Object, "status", "operationState"); found {
		detail.OperationState = opState
	}

	return detail, nil
}

type SyncOptions struct {
	Prune bool `json:"prune"`
	Force bool `json:"force"`
}

func (m *Manager) SyncArgoApplication(ctx context.Context, name string, opts SyncOptions) error {
	if m.client == nil {
		return fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Get the current application to read the target revision
	obj, err := m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace("openshift-gitops").Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting argo application %s: %w", name, err)
	}

	revision, _, _ := unstructured.NestedString(obj.Object, "spec", "source", "targetRevision")

	operation := map[string]interface{}{
		"initiatedBy": map[string]interface{}{
			"username":  "ztp-dashboard",
			"automated": false,
		},
		"sync": map[string]interface{}{
			"revision": revision,
			"prune":    opts.Prune,
		},
	}

	if opts.Force {
		syncOpts := []interface{}{"Replace=true"}
		_ = unstructured.SetNestedSlice(operation, syncOpts, "sync", "syncOptions")
	}

	_ = unstructured.SetNestedField(obj.Object, operation, "operation")

	_, err = m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace("openshift-gitops").Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("triggering sync for %s: %w", name, err)
	}

	slog.Info("argo sync triggered", "application", name, "prune", opts.Prune, "force", opts.Force)
	return nil
}

func (m *Manager) RefreshArgoApplication(ctx context.Context, name string) error {
	if m.client == nil {
		return fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	patch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"argocd.argoproj.io/refresh": "normal",
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling refresh patch: %w", err)
	}

	_, err = m.client.Dynamic.Resource(k8s.GVRArgoApplication).Namespace("openshift-gitops").Patch(ctx, name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("refreshing application %s: %w", name, err)
	}

	slog.Info("argo refresh triggered", "application", name)
	return nil
}

func (m *Manager) SetPolicyDisabled(ctx context.Context, namespace, policyName string, disabled bool) error {
	if m.client == nil {
		return fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"disabled": disabled,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshaling policy patch: %w", err)
	}

	_, err = m.client.Dynamic.Resource(k8s.GVRPolicy).Namespace(namespace).Patch(ctx, policyName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("patching policy %s/%s: %w", namespace, policyName, err)
	}

	action := "enabled"
	if disabled {
		action = "disabled"
	}
	slog.Info("policy state changed", "policy", policyName, "namespace", namespace, "action", action)
	return nil
}

// GetResource returns the raw JSON of a watched resource.
// Only resources matching the 9 watched GVRs are allowed.
func (m *Manager) GetResource(ctx context.Context, group, version, resource, namespace, name string) (map[string]interface{}, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	if !isWatchedGVR(gvr) {
		return nil, fmt.Errorf("resource type %s/%s/%s is not allowed", group, version, resource)
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var obj *unstructured.Unstructured
	var err error
	if namespace == "" || namespace == "_" {
		obj, err = m.client.Dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = m.client.Dynamic.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("getting resource %s/%s: %w", namespace, name, err)
	}

	return obj.Object, nil
}

func isWatchedGVR(gvr schema.GroupVersionResource) bool {
	watched := []schema.GroupVersionResource{
		k8s.GVRManagedCluster, k8s.GVRClusterDeployment, k8s.GVRClusterInstance,
		k8s.GVRInfraEnv, k8s.GVRBareMetalHost, k8s.GVRAgentClusterInstall,
		k8s.GVRAgent, k8s.GVRPolicy, k8s.GVRArgoApplication,
	}
	for _, w := range watched {
		if gvr == w {
			return true
		}
	}
	return false
}

// GetPolicySummary returns aggregate policy compliance across all clusters.
func (m *Manager) GetPolicySummary(ctx context.Context) (*PolicySummary, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	clusters, err := m.ListManagedClusters(ctx)
	if err != nil {
		return nil, err
	}

	summary := &PolicySummary{
		Clusters: make([]ClusterPolicySummary, 0, len(clusters)),
	}

	for _, cluster := range clusters {
		policies, err := m.GetClusterPolicies(ctx, cluster.Name)
		if err != nil {
			slog.Debug("could not list policies for cluster", "cluster", cluster.Name, "error", err)
			continue
		}

		clusterSummary := ClusterPolicySummary{ClusterName: cluster.Name}
		for _, p := range policies {
			clusterSummary.Total++
			summary.Total++
			compliant, _ := p.Data["compliant"].(string)
			switch compliant {
			case "Compliant":
				clusterSummary.Compliant++
				summary.Compliant++
			case "NonCompliant":
				clusterSummary.NonCompliant++
				summary.NonCompliant++
				summary.NonCompliantPolicies = append(summary.NonCompliantPolicies, NonCompliantPolicy{
					Name:        p.Name,
					Namespace:   p.Namespace,
					ClusterName: cluster.Name,
				})
			default:
				summary.Unknown++
			}
		}
		summary.Clusters = append(summary.Clusters, clusterSummary)
	}

	return summary, nil
}

type PolicySummary struct {
	Total               int                    `json:"total"`
	Compliant           int                    `json:"compliant"`
	NonCompliant        int                    `json:"nonCompliant"`
	Unknown             int                    `json:"unknown"`
	Clusters            []ClusterPolicySummary `json:"clusters"`
	NonCompliantPolicies []NonCompliantPolicy  `json:"nonCompliantPolicies"`
}

type ClusterPolicySummary struct {
	ClusterName  string `json:"clusterName"`
	Total        int    `json:"total"`
	Compliant    int    `json:"compliant"`
	NonCompliant int    `json:"nonCompliant"`
}

type NonCompliantPolicy struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	ClusterName string `json:"clusterName"`
}

// GetArgoSummary returns aggregate ArgoCD app health/sync status.
func (m *Manager) GetArgoSummary(ctx context.Context) (*ArgoSummary, error) {
	apps, err := m.GetArgoApplications(ctx)
	if err != nil {
		return nil, err
	}

	summary := &ArgoSummary{Total: len(apps)}
	for _, app := range apps {
		health, _ := app.Data["health"].(string)
		syncStatus, _ := app.Data["sync"].(string)
		switch health {
		case "Healthy":
			summary.Healthy++
		case "Degraded":
			summary.Degraded++
		default:
			summary.Other++
		}
		switch syncStatus {
		case "OutOfSync":
			summary.OutOfSync++
		case "Synced":
			summary.Synced++
		}
	}

	return summary, nil
}

type ArgoSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Degraded  int `json:"degraded"`
	Other     int `json:"other"`
	Synced    int `json:"synced"`
	OutOfSync int `json:"outOfSync"`
}

// DeleteManagedCluster removes a ManagedCluster CR from the hub.
func (m *Manager) DeleteManagedCluster(ctx context.Context, name string) error {
	if m.client == nil {
		return fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := m.client.Dynamic.Resource(k8s.GVRManagedCluster).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("deleting managed cluster %s: %w", name, err)
	}

	slog.Info("managed cluster deleted", "cluster", name)
	return nil
}

// DetachManagedCluster removes a ManagedCluster CR without destroying resources.
// This is the same operation as delete on the CR level; the distinction is semantic.
func (m *Manager) DetachManagedCluster(ctx context.Context, name string) error {
	return m.DeleteManagedCluster(ctx, name)
}

// ListClusterResources lists resources in a cluster namespace for pre-delete review.
func (m *Manager) ListClusterResources(ctx context.Context, clusterName string) ([]ResourceSummary, error) {
	if m.client == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	namespacedGVRs := []struct {
		GVR  schema.GroupVersionResource
		Kind string
	}{
		{k8s.GVRClusterDeployment, "ClusterDeployment"},
		{k8s.GVRClusterInstance, "ClusterInstance"},
		{k8s.GVRInfraEnv, "InfraEnv"},
		{k8s.GVRBareMetalHost, "BareMetalHost"},
		{k8s.GVRAgent, "Agent"},
		{k8s.GVRAgentClusterInstall, "AgentClusterInstall"},
		{k8s.GVRPolicy, "Policy"},
	}

	var resources []ResourceSummary
	for _, g := range namespacedGVRs {
		list, err := m.client.Dynamic.Resource(g.GVR).Namespace(clusterName).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}
		for _, item := range list.Items {
			resources = append(resources, ResourceSummary{
				Kind: g.Kind,
				Name: item.GetName(),
			})
		}
	}

	return resources, nil
}

type ResourceSummary struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}
