package ws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/sebrandon1/ztp-dashboard/internal/k8s"
)

var watchedResources = []struct {
	GVR          schema.GroupVersionResource
	ResourceType string
	MessageType  MessageType
	ClusterScope bool
}{
	{GVR: k8s.GVRManagedCluster, ResourceType: "ManagedCluster", MessageType: MessageTypeManagedCluster, ClusterScope: true},
	{GVR: k8s.GVRClusterDeployment, ResourceType: "ClusterDeployment", MessageType: MessageTypeClusterDeployment},
	{GVR: k8s.GVRClusterInstance, ResourceType: "ClusterInstance", MessageType: MessageTypeClusterInstance},
	{GVR: k8s.GVRInfraEnv, ResourceType: "InfraEnv", MessageType: MessageTypeInfraEnv},
	{GVR: k8s.GVRBareMetalHost, ResourceType: "BareMetalHost", MessageType: MessageTypeBareMetalHost},
	{GVR: k8s.GVRAgentClusterInstall, ResourceType: "AgentClusterInstall", MessageType: MessageTypeAgentClusterInstall},
	{GVR: k8s.GVRAgent, ResourceType: "Agent", MessageType: MessageTypeAgent},
	{GVR: k8s.GVRPolicy, ResourceType: "Policy", MessageType: MessageTypePolicy},
	{GVR: k8s.GVRArgoApplication, ResourceType: "Application", MessageType: MessageTypeArgoApplication},
}

type Watcher struct {
	client  *k8s.Client
	hub     *Hub
	OnEvent func(WatchEvent)
}

func NewWatcher(client *k8s.Client, hub *Hub) *Watcher {
	return &Watcher{
		client: client,
		hub:    hub,
	}
}

func (w *Watcher) Start(ctx context.Context) {
	for _, res := range watchedResources {
		go w.watchResource(ctx, res)
	}
}

func (w *Watcher) watchResource(ctx context.Context, res struct {
	GVR          schema.GroupVersionResource
	ResourceType string
	MessageType  MessageType
	ClusterScope bool
}) {
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		resource := w.client.Dynamic.Resource(res.GVR)
		var watcher watch.Interface
		var err error
		if res.ClusterScope {
			watcher, err = resource.Watch(ctx, metav1.ListOptions{})
		} else {
			watcher, err = resource.Namespace("").Watch(ctx, metav1.ListOptions{})
		}

		if err != nil {
			if isCRDNotFound(err) {
				slog.Debug("CRD not found, will retry", "resource", res.ResourceType)
				backoff = 60 * time.Second
			} else {
				slog.Warn("watch error", "resource", res.ResourceType, "error", err, "retry", backoff)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			continue
		}

		backoff = time.Second
		w.processEvents(ctx, watcher, res.ResourceType, res.MessageType)
		watcher.Stop()
	}
}

func (w *Watcher) processEvents(ctx context.Context, watcher watch.Interface, resourceType string, msgType MessageType) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			obj, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			var eventType WatchEventType
			switch event.Type {
			case watch.Added:
				eventType = WatchEventAdded
			case watch.Modified:
				eventType = WatchEventModified
			case watch.Deleted:
				eventType = WatchEventDeleted
			default:
				continue
			}

			data := extractRelevantData(resourceType, obj)
			summary := buildSummary(resourceType, eventType, obj, data)
			severity, insight := classifyEvent(resourceType, eventType, data)

			watchEvent := WatchEvent{
				EventType:    eventType,
				ResourceType: resourceType,
				Name:         obj.GetName(),
				Namespace:    obj.GetNamespace(),
				Summary:      summary,
				Severity:     severity,
				Insight:      insight,
				Data:         data,
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
			}

			w.hub.Broadcast(Message{
				Type:    msgType,
				Payload: watchEvent,
			})

			// Also broadcast as a generic event for the Events page
			w.hub.Broadcast(Message{
				Type:    MessageTypeEvent,
				Payload: watchEvent,
			})

			if w.OnEvent != nil {
				w.OnEvent(watchEvent)
			}
		}
	}
}

func extractRelevantData(resourceType string, obj *unstructured.Unstructured) map[string]interface{} {
	data := make(map[string]interface{})

	switch resourceType {
	case "ManagedCluster":
		conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if found {
			data["conditions"] = conditions
		}
		labels := obj.GetLabels()
		if v, ok := labels["vendor"]; ok {
			data["vendor"] = v
		}
		if v, ok := labels["openshiftVersion"]; ok {
			data["openshiftVersion"] = v
		}

	case "ClusterDeployment":
		installed, _, _ := unstructured.NestedBool(obj.Object, "spec", "installed")
		data["installed"] = installed
		platform, _, _ := unstructured.NestedString(obj.Object, "status", "conditions")
		data["platform"] = platform
		conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if found {
			data["conditions"] = conditions
		}

	case "ClusterInstance":
		conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if found {
			data["conditions"] = conditions
		}

	case "BareMetalHost":
		provisioning, _, _ := unstructured.NestedMap(obj.Object, "status", "provisioning")
		data["provisioning"] = provisioning
		poweredOn, _, _ := unstructured.NestedBool(obj.Object, "status", "poweredOn")
		data["poweredOn"] = poweredOn
		errorMessage, _, _ := unstructured.NestedString(obj.Object, "status", "errorMessage")
		if errorMessage != "" {
			data["errorMessage"] = errorMessage
		}

	case "AgentClusterInstall":
		conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if found {
			data["conditions"] = conditions
		}
		progress, _, _ := unstructured.NestedMap(obj.Object, "status", "progress")
		data["progress"] = progress

	case "Agent":
		inventory, found, _ := unstructured.NestedMap(obj.Object, "status", "inventory")
		if found {
			hostname, _, _ := unstructured.NestedString(obj.Object, "status", "inventory", "hostname")
			data["hostname"] = hostname
		} else {
			_ = inventory
		}
		role, _, _ := unstructured.NestedString(obj.Object, "spec", "role")
		data["role"] = role
		conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if found {
			data["conditions"] = conditions
		}

	case "InfraEnv":
		conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
		if found {
			data["conditions"] = conditions
		}
		isoURL, _, _ := unstructured.NestedString(obj.Object, "status", "isoDownloadURL")
		if isoURL != "" {
			data["hasISO"] = true
		}

	case "Policy":
		compliant, _, _ := unstructured.NestedString(obj.Object, "status", "compliant")
		data["compliant"] = compliant

	case "Application":
		health, _, _ := unstructured.NestedString(obj.Object, "status", "health", "status")
		data["health"] = health
		sync, _, _ := unstructured.NestedString(obj.Object, "status", "sync", "status")
		data["sync"] = sync
	}

	return data
}

func buildSummary(resourceType string, eventType WatchEventType, obj *unstructured.Unstructured, data map[string]interface{}) string {
	name := obj.GetName()

	if eventType == WatchEventAdded {
		return fmt.Sprintf("%s %s created", resourceType, name)
	}
	if eventType == WatchEventDeleted {
		return fmt.Sprintf("%s %s deleted", resourceType, name)
	}

	// For MODIFIED events, provide context-specific summaries
	switch resourceType {
	case "ManagedCluster":
		// Find the most recent/relevant condition
		if conditions, ok := data["conditions"].([]interface{}); ok {
			for _, c := range conditions {
				cm, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				condType, _ := cm["type"].(string)
				status, _ := cm["status"].(string)
				reason, _ := cm["reason"].(string)
				message, _ := cm["message"].(string)

				switch condType {
				case "ManagedClusterConditionAvailable":
					if status == "True" {
						return fmt.Sprintf("Cluster %s is now available", name)
					}
					if reason != "" {
						return fmt.Sprintf("Cluster %s unavailable: %s", name, reason)
					}
					if message != "" {
						return fmt.Sprintf("Cluster %s unavailable — %s", name, truncate(message, 80))
					}
					return fmt.Sprintf("Cluster %s is unavailable", name)
				}
			}
		}
		return fmt.Sprintf("ManagedCluster %s updated", name)

	case "ClusterDeployment":
		if installed, ok := data["installed"].(bool); ok && installed {
			return fmt.Sprintf("Cluster %s is installed", name)
		}
		if conditions, ok := data["conditions"].([]interface{}); ok {
			summary := latestConditionSummary(conditions)
			if summary != "" {
				return fmt.Sprintf("ClusterDeployment %s — %s", name, summary)
			}
		}
		return fmt.Sprintf("ClusterDeployment %s updated", name)

	case "ClusterInstance":
		if conditions, ok := data["conditions"].([]interface{}); ok {
			summary := latestConditionSummary(conditions)
			if summary != "" {
				return fmt.Sprintf("ClusterInstance %s — %s", name, summary)
			}
		}
		return fmt.Sprintf("ClusterInstance %s updated", name)

	case "BareMetalHost":
		parts := []string{}
		if poweredOn, ok := data["poweredOn"].(bool); ok {
			if poweredOn {
				parts = append(parts, "powered on")
			} else {
				parts = append(parts, "powered off")
			}
		}
		if prov, ok := data["provisioning"].(map[string]interface{}); ok {
			if state, ok := prov["state"].(string); ok && state != "" {
				parts = append(parts, "state: "+state)
			}
		}
		if errMsg, ok := data["errorMessage"].(string); ok && errMsg != "" {
			parts = append(parts, "error: "+truncate(errMsg, 60))
		}
		if len(parts) > 0 {
			return fmt.Sprintf("BMH %s — %s", name, strings.Join(parts, ", "))
		}
		return fmt.Sprintf("BareMetalHost %s updated", name)

	case "AgentClusterInstall":
		if conditions, ok := data["conditions"].([]interface{}); ok {
			summary := latestConditionSummary(conditions)
			if summary != "" {
				return fmt.Sprintf("AgentClusterInstall %s — %s", name, summary)
			}
		}
		return fmt.Sprintf("AgentClusterInstall %s updated", name)

	case "Agent":
		parts := []string{}
		if hostname, ok := data["hostname"].(string); ok && hostname != "" {
			parts = append(parts, hostname)
		}
		if role, ok := data["role"].(string); ok && role != "" {
			parts = append(parts, "role: "+role)
		}
		if conditions, ok := data["conditions"].([]interface{}); ok {
			summary := latestConditionSummary(conditions)
			if summary != "" {
				parts = append(parts, summary)
			}
		}
		if len(parts) > 0 {
			return fmt.Sprintf("Agent %s — %s", name, strings.Join(parts, ", "))
		}
		return fmt.Sprintf("Agent %s updated", name)

	case "InfraEnv":
		if hasISO, ok := data["hasISO"].(bool); ok && hasISO {
			return fmt.Sprintf("InfraEnv %s — ISO image available", name)
		}
		if conditions, ok := data["conditions"].([]interface{}); ok {
			summary := latestConditionSummary(conditions)
			if summary != "" {
				return fmt.Sprintf("InfraEnv %s — %s", name, summary)
			}
		}
		return fmt.Sprintf("InfraEnv %s updated", name)

	case "Policy":
		if compliant, ok := data["compliant"].(string); ok && compliant != "" {
			switch compliant {
			case "Compliant":
				return fmt.Sprintf("Policy %s is now compliant", name)
			case "NonCompliant":
				return fmt.Sprintf("Policy %s is non-compliant", name)
			default:
				return fmt.Sprintf("Policy %s compliance: %s", name, compliant)
			}
		}
		return fmt.Sprintf("Policy %s updated", name)

	case "Application":
		parts := []string{}
		if health, ok := data["health"].(string); ok && health != "" {
			parts = append(parts, "health: "+health)
		}
		if syncStatus, ok := data["sync"].(string); ok && syncStatus != "" {
			parts = append(parts, "sync: "+syncStatus)
		}
		if len(parts) > 0 {
			return fmt.Sprintf("ArgoCD app %s — %s", name, strings.Join(parts, ", "))
		}
		return fmt.Sprintf("Application %s updated", name)
	}

	return fmt.Sprintf("%s %s updated", resourceType, name)
}

func latestConditionSummary(conditions []interface{}) string {
	// Return the message from the most informative condition
	for _, c := range conditions {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		status, _ := cm["status"].(string)
		reason, _ := cm["reason"].(string)
		message, _ := cm["message"].(string)

		// Prioritize conditions that indicate something noteworthy
		if status == "False" || status == "Unknown" {
			if message != "" {
				return truncate(message, 80)
			}
			if reason != "" {
				return reason
			}
		}
	}
	// Fall back to first condition with a message
	for _, c := range conditions {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		message, _ := cm["message"].(string)
		if message != "" {
			return truncate(message, 80)
		}
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// classifyEvent returns a severity ("good", "bad", "warning", "neutral", "info")
// and a short one-line insight for display next to the event.
func classifyEvent(resourceType string, eventType WatchEventType, data map[string]interface{}) (string, string) {
	if eventType == WatchEventDeleted {
		return "warning", "Resource removed from cluster"
	}
	if eventType == WatchEventAdded {
		switch resourceType {
		case "ManagedCluster":
			return "good", "New spoke cluster registered"
		case "Policy":
			compliant, _ := data["compliant"].(string)
			if compliant == "Compliant" {
				return "good", "Policy applied and compliant"
			}
			if compliant == "NonCompliant" {
				return "bad", "Policy not yet enforced on cluster"
			}
			return "info", "Policy created, awaiting evaluation"
		case "BareMetalHost":
			return "info", "Bare metal host discovered"
		case "Agent":
			return "info", "Agent registered for provisioning"
		default:
			return "info", "Resource created"
		}
	}

	// MODIFIED events — the interesting ones
	switch resourceType {
	case "ManagedCluster":
		if conditions, ok := data["conditions"].([]interface{}); ok {
			for _, c := range conditions {
				cm, ok := c.(map[string]interface{})
				if !ok {
					continue
				}
				condType, _ := cm["type"].(string)
				status, _ := cm["status"].(string)
				if condType == "ManagedClusterConditionAvailable" {
					if status == "True" {
						return "good", "Cluster is healthy and reporting"
					}
					if status == "Unknown" {
						return "warning", "Cluster stopped heartbeating — may be unreachable"
					}
					return "bad", "Cluster is not available"
				}
			}
		}
		return "info", "Cluster metadata updated"

	case "ClusterDeployment":
		if installed, ok := data["installed"].(bool); ok && installed {
			return "good", "Cluster installation complete"
		}
		if hasFailedCondition(data) {
			return "bad", "Deployment error — check conditions"
		}
		return "info", "Deployment progressing"

	case "ClusterInstance":
		if hasFailedCondition(data) {
			return "bad", "ClusterInstance hit an error"
		}
		if allConditionsTrue(data) {
			return "good", "All validations and templates applied"
		}
		return "info", "ClusterInstance progressing"

	case "BareMetalHost":
		if errMsg, ok := data["errorMessage"].(string); ok && errMsg != "" {
			return "bad", "Hardware error reported"
		}
		if poweredOn, ok := data["poweredOn"].(bool); ok {
			if !poweredOn {
				return "warning", "Host is powered off"
			}
			return "good", "Host is powered on and operational"
		}
		return "info", "BMH state updated"

	case "AgentClusterInstall":
		if hasFailedCondition(data) {
			return "bad", "Installation error — may need intervention"
		}
		if allConditionsTrue(data) {
			return "good", "Cluster install completed successfully"
		}
		return "info", "Install progressing"

	case "Agent":
		if hasFailedCondition(data) {
			return "bad", "Agent hit a validation or binding error"
		}
		return "info", "Agent status updated"

	case "InfraEnv":
		if hasISO, ok := data["hasISO"].(bool); ok && hasISO {
			return "good", "Discovery ISO is ready"
		}
		return "info", "InfraEnv updated"

	case "Policy":
		compliant, _ := data["compliant"].(string)
		switch compliant {
		case "Compliant":
			return "good", "All policy rules are satisfied"
		case "NonCompliant":
			return "bad", "Policy violations detected — remediation needed"
		default:
			return "info", "Policy status pending evaluation"
		}

	case "Application":
		health, _ := data["health"].(string)
		syncStatus, _ := data["sync"].(string)
		if health == "Degraded" || health == "Missing" {
			return "bad", "ArgoCD app is degraded"
		}
		if syncStatus == "OutOfSync" {
			return "warning", "App drifted from Git — sync may be needed"
		}
		if health == "Healthy" && syncStatus == "Synced" {
			return "good", "App is healthy and in sync with Git"
		}
		return "info", "ArgoCD app state changed"
	}

	return "info", "Resource updated"
}

func hasFailedCondition(data map[string]interface{}) bool {
	conditions, ok := data["conditions"].([]interface{})
	if !ok {
		return false
	}
	for _, c := range conditions {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := cm["type"].(string)
		status, _ := cm["status"].(string)
		if status == "True" && (condType == "Failed" || condType == "Error") {
			return true
		}
		if status == "False" && (condType == "Completed" || condType == "Ready" || condType == "Available") {
			reason, _ := cm["reason"].(string)
			if reason == "InstallationFailed" || reason == "Failed" || strings.Contains(reason, "Error") {
				return true
			}
		}
	}
	return false
}

func allConditionsTrue(data map[string]interface{}) bool {
	conditions, ok := data["conditions"].([]interface{})
	if !ok {
		return false
	}
	if len(conditions) == 0 {
		return false
	}
	for _, c := range conditions {
		cm, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		status, _ := cm["status"].(string)
		if status != "True" {
			return false
		}
	}
	return true
}

func isCRDNotFound(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "the server could not find the requested resource") ||
		strings.Contains(errStr, "no matches for kind")
}
