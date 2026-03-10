import { useEffect, useRef } from 'react';
import { useDashboardStore } from '../lib/store';
import { hubAPI, clusterAPI, aiAPI } from '../lib/api';

export function useHub() {
  const { setHubStatus, setClusters, setAIConnected, updateCounter } = useDashboardStore();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    hubAPI.getStatus().then(setHubStatus).catch(console.error);
    clusterAPI.list().then(setClusters).catch(console.error);
    aiAPI.getStatus().then(s => setAIConnected(s.connected)).catch(() => setAIConnected(false));
  }, [setHubStatus, setClusters, setAIConnected]);

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
