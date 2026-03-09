package ws

type MessageType string

const (
	MessageTypeManagedCluster      MessageType = "managed_cluster"
	MessageTypeClusterDeployment   MessageType = "cluster_deployment"
	MessageTypeClusterInstance     MessageType = "cluster_instance"
	MessageTypeBareMetalHost       MessageType = "baremetal_host"
	MessageTypeAgentClusterInstall MessageType = "agent_cluster_install"
	MessageTypeAgent               MessageType = "agent"
	MessageTypeInfraEnv            MessageType = "infraenv"
	MessageTypePolicy              MessageType = "policy"
	MessageTypeArgoApplication     MessageType = "argo_application"
	MessageTypeEvent               MessageType = "event"
	MessageTypeError               MessageType = "error"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type WatchEventType string

const (
	WatchEventAdded    WatchEventType = "ADDED"
	WatchEventModified WatchEventType = "MODIFIED"
	WatchEventDeleted  WatchEventType = "DELETED"
)

type WatchEvent struct {
	EventType    WatchEventType `json:"event_type"`
	ResourceType string         `json:"resource_type"`
	Name         string         `json:"name"`
	Namespace    string         `json:"namespace"`
	Summary      string         `json:"summary"`
	Severity     string         `json:"severity"`
	Insight      string         `json:"insight"`
	Data         interface{}    `json:"data,omitempty"`
	Timestamp    string         `json:"timestamp,omitempty"`
}
