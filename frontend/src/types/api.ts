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
    disabled?: boolean;
    remediationAction?: string;
  };
}

export interface ArgoApplication extends ResourceStatus {
  data?: {
    health?: string;
    sync?: string;
  };
}

export interface ArgoApplicationDetail {
  name: string;
  namespace: string;
  health: string;
  syncStatus: string;
  syncRevision?: string;
  source?: Record<string, unknown>;
  conditions?: Condition[];
  resources?: ArgoResource[];
  operationState?: Record<string, unknown>;
}

export interface ArgoResource {
  group?: string;
  version: string;
  kind: string;
  namespace?: string;
  name: string;
  status?: string;
  health?: { status: string; message?: string };
}

export interface PolicySummary {
  total: number;
  compliant: number;
  nonCompliant: number;
  unknown: number;
  clusters: ClusterPolicySummary[];
  nonCompliantPolicies: NonCompliantPolicy[];
}

export interface ClusterPolicySummary {
  clusterName: string;
  total: number;
  compliant: number;
  nonCompliant: number;
}

export interface NonCompliantPolicy {
  name: string;
  namespace: string;
  clusterName: string;
}

export interface ArgoSummary {
  total: number;
  healthy: number;
  degraded: number;
  other: number;
  synced: number;
  outOfSync: number;
}

export interface ClusterHealth {
  clusterVersion?: ClusterVersionInfo;
  nodeCount: number;
  nodesReady: number;
  operatorCount: number;
  degradedCount: number;
}

export interface ClusterVersionInfo {
  version: string;
  channel?: string;
  clusterID?: string;
}

export interface NodeInfo {
  name: string;
  role: string;
  status: string;
  kubeletVersion: string;
  age: string;
}

export interface OperatorInfo {
  name: string;
  available: boolean;
  degraded: boolean;
  progressing: boolean;
  message?: string;
  version?: string;
}

export interface ResourceSummaryItem {
  kind: string;
  name: string;
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

export interface EventQueryParams {
  q?: string;
  severity?: string;
  resource_type?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

export interface EventsResponse {
  events: WatchEvent[];
  total: number;
}

export interface EventStats {
  bySeverity: Record<string, number>;
  byResourceType: Record<string, number>;
  total: number;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}
