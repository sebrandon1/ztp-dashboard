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
  EventsResponse,
  EventStats,
  EventQueryParams,
  APIResponse,
} from '../types/api';

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

const AUTH_TOKEN_KEY = 'ztp-auth-token';

/** Event dispatched when the API answers 401 so the UI can prompt for a token. */
export const UNAUTHORIZED_EVENT = 'ztp:unauthorized';

// Fallback for browsers where localStorage is unavailable (private mode).
let memoryToken = '';

/** Bearer token configured by the user in Settings, or ''. */
export function getAuthToken(): string {
  try {
    return window.localStorage.getItem(AUTH_TOKEN_KEY) || memoryToken;
  } catch {
    return memoryToken;
  }
}

/** Persist the bearer token; an empty string clears it. */
export function storeAuthToken(token: string): void {
  memoryToken = token;
  try {
    if (token) {
      window.localStorage.setItem(AUTH_TOKEN_KEY, token);
    } else {
      window.localStorage.removeItem(AUTH_TOKEN_KEY);
    }
  } catch {
    // localStorage unavailable; the in-memory copy still applies this session.
  }
}

api.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401) {
      window.dispatchEvent(new Event(UNAUTHORIZED_EVENT));
    }
    return Promise.reject(error);
  },
);

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
    const token = getAuthToken();
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (token) headers.Authorization = `Bearer ${token}`;
    return fetch('/api/ai/diagnose', {
      method: 'POST',
      headers,
      body: JSON.stringify({ context, model, type }),
    });
  },
};

export const eventsAPI = {
  query: (params?: EventQueryParams) => {
    const searchParams = new URLSearchParams();
    if (params?.q) searchParams.set('q', params.q);
    if (params?.severity) searchParams.set('severity', params.severity);
    if (params?.resource_type) searchParams.set('resource_type', params.resource_type);
    if (params?.from) searchParams.set('from', params.from);
    if (params?.to) searchParams.set('to', params.to);
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));
    const qs = searchParams.toString();
    return unwrap<EventsResponse>(api.get(`/events${qs ? `?${qs}` : ''}`));
  },
  getStats: () => unwrap<EventStats>(api.get('/events/stats')),
};
