import { useEffect, useRef } from 'react';
import { useDashboardStore } from '../lib/store';
import { hubAPI, clusterAPI, aiAPI } from '../lib/api';

const AI_STATUS_POLL_MS = 60000;

function refreshAIStatus(setAIConnected: (connected: boolean) => void): void {
  aiAPI.getStatus()
    .then((s) => setAIConnected(s.connected))
    .catch(() => setAIConnected(false));
}

export function useHub() {
  const { setHubStatus, setClusters, setAIConnected, updateCounter } = useDashboardStore();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    hubAPI.getStatus().then(setHubStatus).catch(console.error);
    clusterAPI.list().then(setClusters).catch(console.error);
    refreshAIStatus(setAIConnected);
  }, [setHubStatus, setClusters, setAIConnected]);

  // Poll ollama so the UI recovers when it comes back online (or goes away).
  useEffect(() => {
    const interval = setInterval(() => refreshAIStatus(setAIConnected), AI_STATUS_POLL_MS);
    return () => clearInterval(interval);
  }, [setAIConnected]);

  useEffect(() => {
    if (updateCounter === 0) return;

    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      hubAPI.getStatus().then(setHubStatus).catch(console.error);
      clusterAPI.list().then(setClusters).catch(console.error);
    }, 2000);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [updateCounter, setHubStatus, setClusters]);
}
