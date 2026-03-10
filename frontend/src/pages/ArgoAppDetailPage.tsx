import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ArrowLeft, GitBranch, RefreshCw, Play, Loader2, Code, ExternalLink, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import { argoAPI } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import StatusBadge from '../components/StatusBadge';
import ConfirmationModal from '../components/ConfirmationModal';
import YamlViewer from '../components/YamlViewer';
import type { ArgoApplicationDetail } from '../types/api';

export default function ArgoAppDetailPage() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [detail, setDetail] = useState<ArgoApplicationDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [syncModalOpen, setSyncModalOpen] = useState(false);
  const [yamlOpen, setYamlOpen] = useState(false);
  const { updateCounter } = useDashboardStore();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    if (!name) return;
    let cancelled = false;
    argoAPI.getApplication(name)
      .then((d) => { if (!cancelled) setDetail(d); })
      .catch(() => { toast.error(`Failed to load application ${name}`); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [name]);

  // Debounced refetch on updateCounter changes
  useEffect(() => {
    if (updateCounter === 0 || !name) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      argoAPI.getApplication(name).then(setDetail).catch(() => {});
    }, 2000);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [updateCounter, name]);

  const handleSync = async () => {
    if (!name) return;
    setSyncing(true);
    try {
      await argoAPI.sync(name);
      toast.success(`Sync triggered for ${name}`);
      argoAPI.getApplication(name).then(setDetail).catch(() => {});
    } catch {
      toast.error(`Failed to sync ${name}`);
    } finally {
      setSyncing(false);
    }
  };

  const handleRefresh = async () => {
    if (!name) return;
    setRefreshing(true);
    try {
      await argoAPI.refresh(name);
      toast.success(`Refreshed ${name}`);
      argoAPI.getApplication(name).then(setDetail).catch(() => {});
    } catch {
      toast.error(`Failed to refresh ${name}`);
    } finally {
      setRefreshing(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="w-6 h-6 text-primary-400 animate-spin" />
      </div>
    );
  }

  if (!detail) {
    return (
      <div className="text-center py-20">
        <AlertCircle className="w-12 h-12 text-text-muted mx-auto mb-3 opacity-30" />
        <p className="text-sm text-text-muted">Application not found</p>
        <button onClick={() => navigate('/argocd')} className="btn btn-primary mt-4">
          Back to Applications
        </button>
      </div>
    );
  }

  const source = detail.source as Record<string, string> | undefined;
  const operationState = detail.operationState as Record<string, string> | undefined;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button onClick={() => navigate('/argocd')} className="btn btn-ghost p-2">
          <ArrowLeft className="w-4 h-4" />
        </button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-text-primary">{detail.name}</h1>
            <StatusBadge status={detail.health} size="md" />
            <StatusBadge status={detail.syncStatus} size="md" />
          </div>
          {detail.syncRevision && (
            <p className="text-xs font-mono text-text-muted mt-1">
              Revision: {detail.syncRevision}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setYamlOpen(true)}
            className="btn btn-secondary flex items-center gap-2"
          >
            <Code className="w-4 h-4" />
            View YAML
          </button>
          <button
            onClick={() => setSyncModalOpen(true)}
            disabled={syncing}
            className="btn btn-primary flex items-center gap-2"
          >
            {syncing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
            Sync
          </button>
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="btn btn-secondary flex items-center gap-2"
          >
            {refreshing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            Refresh
          </button>
        </div>
      </div>

      {/* Source */}
      {source && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="card"
        >
          <div className="px-4 py-3 border-b border-border-subtle">
            <h2 className="text-sm font-semibold text-text-primary flex items-center gap-2">
              <GitBranch className="w-4 h-4 text-primary-400" />
              Source
            </h2>
          </div>
          <div className="px-4 py-3 grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <p className="text-xs text-text-muted mb-1">Repository URL</p>
              <p className="text-sm text-text-primary flex items-center gap-1.5 break-all">
                {source.repoURL || '--'}
                {source.repoURL && (
                  <a
                    href={source.repoURL as string}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary-400 hover:text-primary-300 shrink-0"
                  >
                    <ExternalLink className="w-3.5 h-3.5" />
                  </a>
                )}
              </p>
            </div>
            <div>
              <p className="text-xs text-text-muted mb-1">Path</p>
              <p className="text-sm text-text-primary font-mono">{source.path || '--'}</p>
            </div>
            <div>
              <p className="text-xs text-text-muted mb-1">Target Revision</p>
              <p className="text-sm text-text-primary font-mono">{source.targetRevision || '--'}</p>
            </div>
          </div>
        </motion.div>
      )}

      {/* Resources */}
      {detail.resources && detail.resources.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="card overflow-hidden"
        >
          <div className="px-4 py-3 border-b border-border-subtle">
            <h2 className="text-sm font-semibold text-text-primary">
              Resources ({detail.resources.length})
            </h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border-subtle">
                  <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted uppercase tracking-wider">Kind</th>
                  <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted uppercase tracking-wider">Name</th>
                  <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted uppercase tracking-wider">Namespace</th>
                  <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted uppercase tracking-wider">Status</th>
                  <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted uppercase tracking-wider">Health</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border-subtle">
                {detail.resources.map((res, i) => (
                  <tr key={`${res.kind}-${res.name}-${i}`} className="hover:bg-surface-overlay/50 transition-colors">
                    <td className="px-4 py-2">
                      <span className="text-xs font-mono text-text-secondary">{res.kind}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-sm text-text-primary">{res.name}</span>
                    </td>
                    <td className="px-4 py-2">
                      <span className="text-xs text-text-muted">{res.namespace || '--'}</span>
                    </td>
                    <td className="px-4 py-2">
                      {res.status ? <StatusBadge status={res.status} /> : <span className="text-xs text-text-muted">--</span>}
                    </td>
                    <td className="px-4 py-2">
                      {res.health?.status ? (
                        <StatusBadge status={res.health.status} />
                      ) : (
                        <span className="text-xs text-text-muted">--</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </motion.div>
      )}

      {/* Operation State */}
      {operationState && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="card"
        >
          <div className="px-4 py-3 border-b border-border-subtle">
            <h2 className="text-sm font-semibold text-text-primary">Last Operation</h2>
          </div>
          <div className="px-4 py-3 space-y-2">
            {operationState.phase && (
              <div className="flex items-center gap-2">
                <span className="text-xs text-text-muted">Phase:</span>
                <StatusBadge status={operationState.phase as string} />
              </div>
            )}
            {operationState.message && (
              <div>
                <span className="text-xs text-text-muted">Message:</span>
                <p className="text-sm text-text-secondary mt-0.5">{operationState.message}</p>
              </div>
            )}
            {operationState.startedAt && (
              <div className="flex items-center gap-2">
                <span className="text-xs text-text-muted">Started:</span>
                <span className="text-xs text-text-secondary">{operationState.startedAt}</span>
              </div>
            )}
            {operationState.finishedAt && (
              <div className="flex items-center gap-2">
                <span className="text-xs text-text-muted">Finished:</span>
                <span className="text-xs text-text-secondary">{operationState.finishedAt}</span>
              </div>
            )}
          </div>
        </motion.div>
      )}

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="card"
        >
          <div className="px-4 py-3 border-b border-border-subtle">
            <h2 className="text-sm font-semibold text-text-primary">Conditions</h2>
          </div>
          <div className="divide-y divide-border-subtle">
            {detail.conditions.map((cond, i) => (
              <div key={`${cond.type}-${i}`} className="px-4 py-3">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-text-primary">{cond.type}</span>
                  <StatusBadge status={cond.status} />
                </div>
                {cond.message && (
                  <p className="text-xs text-text-muted">{cond.message}</p>
                )}
                {cond.lastTransitionTime && (
                  <p className="text-xs text-text-muted mt-0.5">{cond.lastTransitionTime}</p>
                )}
              </div>
            ))}
          </div>
        </motion.div>
      )}

      {/* Sync Confirmation Modal */}
      <ConfirmationModal
        open={syncModalOpen}
        onClose={() => setSyncModalOpen(false)}
        onConfirm={handleSync}
        title={`Sync ${name}`}
        message="This will trigger a sync operation for this application. Type the application name to confirm."
        confirmText={name || ''}
        variant="warning"
      />

      {/* YAML Viewer */}
      {yamlOpen && (
        <YamlViewer
          data={detail as unknown as Record<string, unknown>}
          title={`${detail.name} - Application YAML`}
          onClose={() => setYamlOpen(false)}
        />
      )}
    </div>
  );
}
