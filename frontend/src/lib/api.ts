import axios from 'axios';
import type {
  HubStatus,
  ManagedClusterInfo,
  PipelineStatus,
  PolicyInfo,
  ArgoApplication,
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
};

export const argoAPI = {
  getApplications: () => unwrap<ArgoApplication[]>(api.get('/argocd/applications')),
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
