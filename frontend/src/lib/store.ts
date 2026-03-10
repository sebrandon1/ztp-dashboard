import { create } from 'zustand';
import type { HubStatus, ManagedClusterInfo, WatchEvent, WSMessage } from '../types/api';

interface DashboardState {
  wsConnected: boolean;
  setWSConnected: (connected: boolean) => void;

  hubStatus: HubStatus | null;
  setHubStatus: (status: HubStatus) => void;

  aiConnected: boolean | null;
  setAIConnected: (connected: boolean) => void;

  clusters: ManagedClusterInfo[];
  setClusters: (clusters: ManagedClusterInfo[]) => void;

  events: WatchEvent[];
  addEvent: (event: WatchEvent) => void;
  setEvents: (events: WatchEvent[]) => void;

  updateCounter: number;
  incrementUpdateCounter: () => void;

  handleWSMessage: (msg: WSMessage) => void;
}

const MAX_EVENTS = 500;

export const useDashboardStore = create<DashboardState>((set) => ({
  wsConnected: false,
  setWSConnected: (connected) => set({ wsConnected: connected }),

  hubStatus: null,
  setHubStatus: (status) => set({ hubStatus: status }),

  aiConnected: null,
  setAIConnected: (connected) => set({ aiConnected: connected }),

  clusters: [],
  setClusters: (clusters) => set({ clusters }),

  events: [],
  addEvent: (event) =>
    set((state) => {
      const events = [event, ...state.events].slice(0, MAX_EVENTS);
      return { events };
    }),
  setEvents: (events) => set({ events }),

  updateCounter: 0,
  incrementUpdateCounter: () =>
    set((state) => ({ updateCounter: state.updateCounter + 1 })),

  handleWSMessage: (msg: WSMessage) => {
    const event = msg.payload;

    if (msg.type === 'event' && event) {
      set((state) => {
        const events = [event, ...state.events].slice(0, MAX_EVENTS);
        return { events };
      });
    }

    switch (msg.type) {
      case 'managed_cluster':
      case 'cluster_deployment':
      case 'cluster_instance':
      case 'baremetal_host':
      case 'agent_cluster_install':
      case 'agent':
      case 'infraenv':
      case 'policy':
      case 'argo_application':
        set((state) => ({ updateCounter: state.updateCounter + 1 }));
        break;
      case 'error':
        console.error('WebSocket error:', msg.payload);
        break;
    }
  },
}));
