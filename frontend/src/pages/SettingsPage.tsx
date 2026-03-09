import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Settings, Server, Bot, CheckCircle2, XCircle, Loader2 } from 'lucide-react';
import { useDashboardStore } from '../lib/store';
import { aiAPI } from '../lib/api';
import type { AIStatus, AIModel } from '../types/api';

export default function SettingsPage() {
  const { hubStatus, wsConnected } = useDashboardStore();
  const [aiStatus, setAiStatus] = useState<AIStatus | null>(null);
  const [models, setModels] = useState<AIModel[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      aiAPI.getStatus().catch(() => null),
      aiAPI.getModels().catch(() => []),
    ]).then(([status, m]) => {
      if (status) setAiStatus(status);
      setModels(m as AIModel[]);
      setLoading(false);
    });
  }, []);

  const formatSize = (bytes: number) => {
    if (bytes > 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
    if (bytes > 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
    return `${bytes} B`;
  };

  return (
    <div className="space-y-8 max-w-3xl">
      <div>
        <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
          <Settings className="w-6 h-6 text-primary-400" />
          Settings
        </h1>
        <p className="text-sm text-text-muted mt-1">Connection status and configuration</p>
      </div>

      {/* Hub connection */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="card"
      >
        <div className="flex items-center gap-2 px-5 py-4 border-b border-border-subtle">
          <Server className="w-4 h-4 text-text-muted" />
          <h2 className="text-sm font-semibold text-text-primary">Hub Cluster Connection</h2>
        </div>
        <div className="p-5 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-text-secondary">Status</span>
            <span className="flex items-center gap-2 text-sm">
              {hubStatus?.connected ? (
                <><CheckCircle2 className="w-4 h-4 text-emerald-400" /> Connected</>
              ) : (
                <><XCircle className="w-4 h-4 text-red-400" /> Disconnected</>
              )}
            </span>
          </div>
          {hubStatus?.serverVersion && (
            <div className="flex items-center justify-between">
              <span className="text-sm text-text-secondary">Server Version</span>
              <span className="text-sm text-text-primary font-mono">{hubStatus.serverVersion}</span>
            </div>
          )}
          <div className="flex items-center justify-between">
            <span className="text-sm text-text-secondary">WebSocket</span>
            <span className={`flex items-center gap-2 text-sm ${wsConnected ? 'text-emerald-400' : 'text-amber-400'}`}>
              {wsConnected ? (
                <><CheckCircle2 className="w-4 h-4" /> Connected</>
              ) : (
                <><Loader2 className="w-4 h-4 animate-spin" /> Reconnecting</>
              )}
            </span>
          </div>
        </div>
      </motion.div>

      {/* AI / Ollama */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.1 }}
        className="card"
      >
        <div className="flex items-center gap-2 px-5 py-4 border-b border-border-subtle">
          <Bot className="w-4 h-4 text-text-muted" />
          <h2 className="text-sm font-semibold text-text-primary">Ollama AI Configuration</h2>
        </div>
        <div className="p-5 space-y-3">
          {loading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="w-5 h-5 text-primary-400 animate-spin" />
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between">
                <span className="text-sm text-text-secondary">Status</span>
                <span className="flex items-center gap-2 text-sm">
                  {aiStatus?.connected ? (
                    <><CheckCircle2 className="w-4 h-4 text-emerald-400" /> Connected</>
                  ) : (
                    <><XCircle className="w-4 h-4 text-red-400" /> Not Available</>
                  )}
                </span>
              </div>
              {aiStatus && (
                <>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-text-secondary">Endpoint</span>
                    <span className="text-sm text-text-primary font-mono">{aiStatus.endpoint}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-text-secondary">Default Model</span>
                    <span className="text-sm text-text-primary font-mono">{aiStatus.defaultModel}</span>
                  </div>
                </>
              )}
              {models.length > 0 && (
                <div className="mt-4">
                  <p className="text-xs font-medium text-text-muted uppercase tracking-wider mb-2">Available Models</p>
                  <div className="space-y-2">
                    {models.map((m) => (
                      <div key={m.name} className="flex items-center justify-between p-3 rounded-lg bg-surface/50">
                        <span className="text-sm font-medium text-text-primary">{m.name}</span>
                        <span className="text-xs text-text-muted">{formatSize(m.size)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {aiStatus?.error && (
                <div className="mt-3 p-3 rounded-lg bg-red-500/10 border border-red-500/20">
                  <p className="text-xs text-red-400">{aiStatus.error}</p>
                </div>
              )}
            </>
          )}
        </div>
      </motion.div>
    </div>
  );
}
