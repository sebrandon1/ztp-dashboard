import { useEffect, useRef } from 'react';
import { useDashboardStore } from '../lib/store';
import type { WSMessage } from '../types/api';

const MAX_RECONNECT_DELAY_MS = 30000;

/** Exponential backoff for reconnects: 1s, 2s, 4s… capped at 30s. */
export function reconnectDelay(attempt: number): number {
  return Math.min(1000 * Math.pow(2, attempt), MAX_RECONNECT_DELAY_MS);
}

/** WebSocket URL for live updates; browsers cannot set handshake headers, so
 *  the auth token travels as a query parameter. */
export function buildWebSocketURL(token: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const tokenQuery = token ? `?token=${encodeURIComponent(token)}` : '';
  return `${protocol}//${window.location.host}/ws/watch${tokenQuery}`;
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const reconnectAttempts = useRef(0);
  const { setWSConnected, handleWSMessage, authToken } = useDashboardStore();

  useEffect(() => {
    function connect() {
      if (wsRef.current?.readyState === WebSocket.OPEN) return;

      const ws = new WebSocket(buildWebSocketURL(authToken));
      wsRef.current = ws;

      ws.onopen = () => {
        setWSConnected(true);
        reconnectAttempts.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const messages = event.data.split('\n');
          for (const msgStr of messages) {
            if (!msgStr.trim()) continue;
            const msg: WSMessage = JSON.parse(msgStr);
            handleWSMessage(msg);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.onclose = () => {
        setWSConnected(false);
        wsRef.current = null;

        const delay = reconnectDelay(reconnectAttempts.current);
        reconnectAttempts.current++;
        reconnectTimeoutRef.current = setTimeout(connect, delay);
      };

      ws.onerror = () => {
        ws.close();
      };
    }

    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [setWSConnected, handleWSMessage, authToken]);
}
