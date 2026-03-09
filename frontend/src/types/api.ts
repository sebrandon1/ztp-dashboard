export interface HubStatus {
  connected: boolean;
  serverVersion?: string;
}

export interface ManagedClusterInfo {
  name: string;
  labels?: Record<string, string>;
  conditions?: Condition[];
  available: string;
  hubAccepted: string;
  joined: string;
  openshiftVersion?: string;
  creationTimestamp: string;
}

export interface Condition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastTransitionTime?: string;
}

export interface ResourceStatus {
  name: string;
  namespace?: string;
  status: string;
  conditions?: Condition[];
  data?: Record<string, unknown>;
}

export interface PipelineStatus {
  clusterName: string;
  clusterInstance?: ResourceStatus;
  clusterDeployment?: ResourceStatus;
  infraEnv?: ResourceStatus;
  bareMetalHosts?: ResourceStatus[];
  agents?: ResourceStatus[];
  agentClusterInstall?: ResourceStatus;
  managedCluster?: ResourceStatus;
}

export interface PolicyInfo extends ResourceStatus {
  data?: {
    compliant?: string;
  };
}

export interface ArgoApplication extends ResourceStatus {
  data?: {
    health?: string;
    sync?: string;
  };
}

export interface AIStatus {
  connected: boolean;
  endpoint: string;
  defaultModel: string;
  error?: string;
}

export interface AIModel {
  name: string;
  modified_at: string;
  size: number;
}

export interface WatchEvent {
  event_type: 'ADDED' | 'MODIFIED' | 'DELETED';
  resource_type: string;
  name: string;
  namespace: string;
  summary: string;
  severity: 'good' | 'bad' | 'warning' | 'neutral' | 'info';
  insight: string;
  data?: Record<string, unknown>;
  timestamp?: string;
}

export interface WSMessage {
  type: string;
  payload: WatchEvent;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}
