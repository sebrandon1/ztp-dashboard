package spoke

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
)

var (
	gvrClusterVersion = schema.GroupVersionResource{
		Group: "config.openshift.io", Version: "v1", Resource: "clusterversions",
	}
	gvrClusterOperator = schema.GroupVersionResource{
		Group: "config.openshift.io", Version: "v1", Resource: "clusteroperators",
	}
)

// ClusterHealth is the aggregated health response for a spoke cluster.
type ClusterHealth struct {
	ClusterVersion *ClusterVersionInfo `json:"clusterVersion,omitempty"`
	NodeCount      int                 `json:"nodeCount"`
	NodesReady     int                 `json:"nodesReady"`
	OperatorCount  int                 `json:"operatorCount"`
	DegradedCount  int                 `json:"degradedCount"`
}

// ClusterVersionInfo holds cluster version details.
type ClusterVersionInfo struct {
	Version string `json:"version"`
	Channel string `json:"channel,omitempty"`
	ClusterID string `json:"clusterID,omitempty"`
}

// NodeInfo describes a spoke cluster node.
type NodeInfo struct {
	Name           string `json:"name"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	KubeletVersion string `json:"kubeletVersion"`
	Age            string `json:"age"`
}

// OperatorInfo describes a spoke cluster operator.
type OperatorInfo struct {
	Name        string `json:"name"`
	Available   bool   `json:"available"`
	Degraded    bool   `json:"degraded"`
	Progressing bool   `json:"progressing"`
	Message     string `json:"message,omitempty"`
	Version     string `json:"version,omitempty"`
}

// Service handles health queries to spoke clusters.
type Service struct {
	hubClient  *k8s.Client
	clientPool *k8s.ClientPool
}

// NewService creates a spoke health service.
func NewService(hubClient *k8s.Client, clientPool *k8s.ClientPool) *Service {
	return &Service{
		hubClient:  hubClient,
		clientPool: clientPool,
	}
}

// getSpokeClient extracts the kubeconfig from the hub secret and returns a spoke client.
func (s *Service) getSpokeClient(ctx context.Context, clusterName string) (*k8s.SpokeClient, error) {
	secretName := clusterName + "-admin-kubeconfig"
	secret, err := s.hubClient.Clientset.CoreV1().Secrets(clusterName).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting kubeconfig secret for %s: %w", clusterName, err)
	}

	kubeconfig, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, fmt.Errorf("kubeconfig key not found in secret %s/%s", clusterName, secretName)
	}

	return s.clientPool.GetOrCreate(clusterName, kubeconfig)
}

// GetHealth returns aggregated health for a spoke cluster.
func (s *Service) GetHealth(ctx context.Context, clusterName string) (*ClusterHealth, error) {
	spokeClient, err := s.getSpokeClient(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	health := &ClusterHealth{}

	// ClusterVersion
	cv, err := s.getClusterVersion(ctx, spokeClient)
	if err != nil {
		slog.Debug("could not get cluster version", "cluster", clusterName, "error", err)
	} else {
		health.ClusterVersion = cv
	}

	// Nodes
	nodes, err := s.GetNodes(ctx, clusterName)
	if err != nil {
		slog.Debug("could not get nodes", "cluster", clusterName, "error", err)
	} else {
		health.NodeCount = len(nodes)
		for _, n := range nodes {
			if n.Status == "Ready" {
				health.NodesReady++
			}
		}
	}

	// Operators
	operators, err := s.GetOperators(ctx, clusterName)
	if err != nil {
		slog.Debug("could not get operators", "cluster", clusterName, "error", err)
	} else {
		health.OperatorCount = len(operators)
		for _, op := range operators {
			if op.Degraded {
				health.DegradedCount++
			}
		}
	}

	return health, nil
}

func (s *Service) getClusterVersion(ctx context.Context, spokeClient *k8s.SpokeClient) (*ClusterVersionInfo, error) {
	obj, err := spokeClient.Dynamic.Resource(gvrClusterVersion).Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	info := &ClusterVersionInfo{}
	info.Channel, _, _ = unstructured.NestedString(obj.Object, "spec", "channel")
	info.ClusterID, _, _ = unstructured.NestedString(obj.Object, "spec", "clusterID")

	// Get the current version from status.history[0].version
	history, found, _ := unstructured.NestedSlice(obj.Object, "status", "history")
	if found && len(history) > 0 {
		if entry, ok := history[0].(map[string]any); ok {
			info.Version, _, _ = unstructured.NestedString(entry, "version")
		}
	}

	return info, nil
}

// GetNodes returns the list of nodes on the spoke cluster.
func (s *Service) GetNodes(ctx context.Context, clusterName string) ([]NodeInfo, error) {
	spokeClient, err := s.getSpokeClient(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	nodeList, err := spokeClient.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes on spoke %s: %w", clusterName, err)
	}

	nodes := make([]NodeInfo, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		info := NodeInfo{
			Name:           node.Name,
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
			Age:            formatAge(node.CreationTimestamp.Time),
		}

		// Determine role
		roles := []string{}
		for label := range node.Labels {
			if role, found := strings.CutPrefix(label, "node-role.kubernetes.io/"); found {
				if role != "" {
					roles = append(roles, role)
				}
			}
		}
		if len(roles) > 0 {
			info.Role = strings.Join(roles, ",")
		} else {
			info.Role = "worker"
		}

		// Determine status from conditions
		info.Status = "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				info.Status = "Ready"
				break
			}
		}

		nodes = append(nodes, info)
	}

	return nodes, nil
}

// GetOperators returns the list of cluster operators on the spoke cluster.
func (s *Service) GetOperators(ctx context.Context, clusterName string) ([]OperatorInfo, error) {
	spokeClient, err := s.getSpokeClient(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	list, err := spokeClient.Dynamic.Resource(gvrClusterOperator).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing cluster operators on spoke %s: %w", clusterName, err)
	}

	operators := make([]OperatorInfo, 0, len(list.Items))
	for _, item := range list.Items {
		info := OperatorInfo{
			Name: item.GetName(),
		}

		// Parse conditions
		conditions, found, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
		if found {
			for _, c := range conditions {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				condType, _ := cm["type"].(string)
				status, _ := cm["status"].(string)
				message, _ := cm["message"].(string)

				switch condType {
				case "Available":
					info.Available = status == "True"
				case "Degraded":
					info.Degraded = status == "True"
					if info.Degraded && message != "" {
						info.Message = message
					}
				case "Progressing":
					info.Progressing = status == "True"
				}
			}
		}

		// Get version from status.versions
		versions, found, _ := unstructured.NestedSlice(item.Object, "status", "versions")
		if found {
			for _, v := range versions {
				vm, ok := v.(map[string]any)
				if !ok {
					continue
				}
				vName, _ := vm["name"].(string)
				if vName == "operator" {
					info.Version, _ = vm["version"].(string)
					break
				}
			}
		}

		operators = append(operators, info)
	}

	return operators, nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
