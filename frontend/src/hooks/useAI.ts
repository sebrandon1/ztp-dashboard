import { useState, useCallback } from 'react';
import { aiAPI } from '../lib/api';

export function useAI() {
  const [response, setResponse] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const diagnose = useCallback(async (context: Record<string, unknown>, model?: string, type?: string) => {
    setResponse('');
    setIsStreaming(true);
    setError(null);

    try {
      const res = await aiAPI.diagnose(context, model, type);
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || 'AI diagnosis failed');
      }

      const reader = res.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n');
        buffer = lines.pop() || '';

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              if (data.token) {
                setResponse((prev) => prev + data.token);
              }
            } catch {
              // Skip malformed lines
            }
          }
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsStreaming(false);
    }
  }, []);

  const reset = useCallback(() => {
    setResponse('');
    setError(null);
    setIsStreaming(false);
  }, []);

  return { response, isStreaming, error, diagnose, reset };
}
