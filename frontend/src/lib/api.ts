import axios from 'axios';
import type {
  HubStatus,
  ManagedClusterInfo,
  PipelineStatus,
  PolicyInfo,
  ArgoApplication,
  ArgoApplicationDetail,
  PolicySummary,
  ArgoSummary,
  ResourceSummaryItem,
  ClusterHealth,
  NodeInfo,
  OperatorInfo,
  AIStatus,
  AIModel,
  WatchEvent,
  APIResponse,
} from '../types/api';

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

async function unwrap<T>(promise: Promise<{ data: APIResponse<T> }>): Promise<T> {
  const { data } = await promise;
  if (!data.success) throw new Error(data.error || 'Unknown error');
  return data.data as T;
}

export const hubAPI = {
  getStatus: () => unwrap<HubStatus>(api.get('/status')),
};

export const clusterAPI = {
  list: () => unwrap<ManagedClusterInfo[]>(api.get('/clusters')),
  get: (name: string) => unwrap<ManagedClusterInfo>(api.get(`/clusters/${name}`)),
  getPipeline: (name: string) => unwrap<PipelineStatus>(api.get(`/clusters/${name}/pipeline`)),
  getPolicies: (name: string) => unwrap<PolicyInfo[]>(api.get(`/clusters/${name}/policies`)),
  getResources: (name: string) => unwrap<ResourceSummaryItem[]>(api.get(`/clusters/${name}/resources`)),
  setPolicyState: (clusterName: string, policyName: string, disabled: boolean) =>
    unwrap<{ status: string }>(api.post(`/clusters/${clusterName}/policies/${policyName}/state`, { disabled })),
  delete: (name: string, confirmName: string) =>
    unwrap<{ status: string }>(api.delete(`/clusters/${name}`, { data: { confirmName } })),
  detach: (name: string, confirmName: string) =>
    unwrap<{ status: string }>(api.post(`/clusters/${name}/detach`, { confirmName })),
  getHealth: (name: string) => unwrap<ClusterHealth>(api.get(`/clusters/${name}/health`)),
  getNodes: (name: string) => unwrap<NodeInfo[]>(api.get(`/clusters/${name}/health/nodes`)),
  getOperators: (name: string) => unwrap<OperatorInfo[]>(api.get(`/clusters/${name}/health/operators`)),
};

export const argoAPI = {
  getApplications: () => unwrap<ArgoApplication[]>(api.get('/argocd/applications')),
  getApplication: (name: string) => unwrap<ArgoApplicationDetail>(api.get(`/argocd/applications/${name}`)),
  sync: (name: string, opts?: { prune?: boolean; force?: boolean }) =>
    unwrap<{ status: string }>(api.post(`/argocd/applications/${name}/sync`, opts || {})),
  refresh: (name: string) =>
    unwrap<{ status: string }>(api.post(`/argocd/applications/${name}/refresh`, {})),
  getSummary: () => unwrap<ArgoSummary>(api.get('/argocd/summary')),
};

export const policyAPI = {
  getSummary: () => unwrap<PolicySummary>(api.get('/policies/summary')),
};

export const resourceAPI = {
  get: (group: string, version: string, resource: string, namespace: string, name: string) =>
    unwrap<Record<string, unknown>>(api.get(`/resources/${group}/${version}/${resource}/${namespace}/${name}`)),
};

export const aiAPI = {
  getStatus: () => unwrap<AIStatus>(api.get('/ai/status')),
  getModels: () => unwrap<AIModel[]>(api.get('/ai/models')),
  diagnose: (context: Record<string, unknown>, model?: string, type?: string) => {
    return fetch('/api/ai/diagnose', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ context, model, type }),
    });
  },
};

export const eventsAPI = {
  getRecent: () => unwrap<WatchEvent[]>(api.get('/events')),
};
