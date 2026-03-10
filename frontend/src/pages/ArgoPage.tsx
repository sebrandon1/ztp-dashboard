import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { GitBranch, RefreshCw, Play, Loader2, AlertTriangle, X } from 'lucide-react';
import { toast } from 'sonner';
import { argoAPI } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import StatusBadge from '../components/StatusBadge';
import type { ArgoApplication } from '../types/api';

export default function ArgoPage() {
  const navigate = useNavigate();
  const [apps, setApps] = useState<ArgoApplication[]>([]);
  const [loading, setLoading] = useState(true);
  const [syncingApp, setSyncingApp] = useState<string | null>(null);
  const [refreshingApp, setRefreshingApp] = useState<string | null>(null);

  // Sync modal state
  const [syncModalOpen, setSyncModalOpen] = useState(false);
  const [selectedApp, setSelectedApp] = useState('');
  const [syncConfirmInput, setSyncConfirmInput] = useState('');
  const [syncPrune, setSyncPrune] = useState(false);
  const [syncForce, setSyncForce] = useState(false);

  const { updateCounter } = useDashboardStore();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const fetchApps = useCallback(() => {
    argoAPI.getApplications()
      .then(setApps)
      .catch(() => {
        toast.error('Failed to fetch ArgoCD applications');
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    fetchApps();
  }, [fetchApps]);

  // Debounced refetch on updateCounter changes
  useEffect(() => {
    if (updateCounter === 0) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      argoAPI.getApplications().then(setApps).catch(() => {});
    }, 2000);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [updateCounter]);

  const openSyncModal = (appName: string) => {
    setSelectedApp(appName);
    setSyncConfirmInput('');
    setSyncPrune(false);
    setSyncForce(false);
    setSyncModalOpen(true);
  };

  const handleSyncConfirm = async () => {
    setSyncModalOpen(false);
    setSyncingApp(selectedApp);
    try {
      await argoAPI.sync(selectedApp, { prune: syncPrune, force: syncForce });
      toast.success(`Sync triggered for ${selectedApp}`);
      fetchApps();
    } catch {
      toast.error(`Failed to sync ${selectedApp}`);
    } finally {
      setSyncingApp(null);
    }
  };

  const handleRefresh = async (appName: string) => {
    setRefreshingApp(appName);
    try {
      await argoAPI.refresh(appName);
      toast.success(`Refreshed ${appName}`);
      fetchApps();
    } catch {
      toast.error(`Failed to refresh ${appName}`);
    } finally {
      setRefreshingApp(null);
    }
  };

  const syncConfirmMatch = syncConfirmInput === selectedApp;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
          <GitBranch className="w-6 h-6 text-primary-400" />
          ArgoCD Applications
        </h1>
        <p className="text-sm text-text-muted mt-1">
          {apps.length} application{apps.length !== 1 ? 's' : ''} managed by ArgoCD
        </p>
      </div>

      {/* Content */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-6 h-6 text-primary-400 animate-spin" />
        </div>
      ) : apps.length === 0 ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-center py-20"
        >
          <GitBranch className="w-12 h-12 text-text-muted mx-auto mb-3 opacity-30" />
          <p className="text-sm text-text-muted">No ArgoCD applications found</p>
        </motion.div>
      ) : (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
          className="card overflow-hidden"
        >
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border-subtle">
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wider">Name</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wider">Health</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wider">Sync Status</th>
                  <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wider">Revision</th>
                  <th className="text-right px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border-subtle">
                {apps.map((app, i) => (
                  <motion.tr
                    key={app.name}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: i * 0.03 }}
                    className="hover:bg-surface-overlay/50 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <button
                        onClick={() => navigate(`/argocd/${app.name}`)}
                        className="text-sm font-medium text-primary-400 hover:text-primary-300 hover:underline transition-colors"
                      >
                        {app.name}
                      </button>
                      {app.namespace && (
                        <p className="text-xs text-text-muted">{app.namespace}</p>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={app.data?.health || 'Unknown'} />
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={app.data?.sync || 'Unknown'} />
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs font-mono text-text-muted">
                        {app.status ? app.status.substring(0, 8) : '--'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          onClick={() => openSyncModal(app.name)}
                          disabled={syncingApp === app.name}
                          className="btn btn-ghost p-1.5"
                          title="Sync"
                        >
                          {syncingApp === app.name ? (
                            <Loader2 className="w-3.5 h-3.5 animate-spin" />
                          ) : (
                            <Play className="w-3.5 h-3.5" />
                          )}
                        </button>
                        <button
                          onClick={() => handleRefresh(app.name)}
                          disabled={refreshingApp === app.name}
                          className="btn btn-ghost p-1.5"
                          title="Refresh"
                        >
                          {refreshingApp === app.name ? (
                            <Loader2 className="w-3.5 h-3.5 animate-spin" />
                          ) : (
                            <RefreshCw className="w-3.5 h-3.5" />
                          )}
                        </button>
                      </div>
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </div>
        </motion.div>
      )}

      {/* Sync Confirmation Modal with Options */}
      <AnimatePresence>
        {syncModalOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="fixed inset-0 z-50 flex items-center justify-center p-4"
          >
            <div
              className="absolute inset-0 bg-black/60 backdrop-blur-sm"
              onClick={() => setSyncModalOpen(false)}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 10 }}
              transition={{ duration: 0.15 }}
              className="relative w-full max-w-md bg-surface-raised border border-amber-500/30 rounded-xl shadow-2xl"
            >
              {/* Header */}
              <div className="flex items-start gap-3 px-5 pt-5 pb-3">
                <div className="mt-0.5 text-amber-400">
                  <AlertTriangle className="w-5 h-5" />
                </div>
                <div className="flex-1">
                  <h3 className="text-base font-semibold text-text-primary">Sync {selectedApp}</h3>
                  <p className="text-sm text-text-muted mt-1">
                    This will trigger a sync operation for this application. Type the application name to confirm.
                  </p>
                </div>
                <button onClick={() => setSyncModalOpen(false)} className="btn btn-ghost p-1">
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Sync options */}
              <div className="px-5 pb-3">
                <p className="text-xs font-medium text-text-secondary mb-2">Sync Options</p>
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 text-xs text-text-muted cursor-pointer">
                    <input
                      type="checkbox"
                      checked={syncPrune}
                      onChange={(e) => setSyncPrune(e.target.checked)}
                      className="rounded border-border-default bg-surface-overlay"
                    />
                    Prune resources
                  </label>
                  <label className="flex items-center gap-2 text-xs text-text-muted cursor-pointer">
                    <input
                      type="checkbox"
                      checked={syncForce}
                      onChange={(e) => setSyncForce(e.target.checked)}
                      className="rounded border-border-default bg-surface-overlay"
                    />
                    Force sync
                  </label>
                </div>
              </div>

              {/* Confirmation input */}
              <div className="px-5 pb-4">
                <label className="block text-xs text-text-muted mb-1.5">
                  Type <span className="font-mono font-semibold text-text-secondary">{selectedApp}</span> to confirm
                </label>
                <input
                  type="text"
                  value={syncConfirmInput}
                  onChange={(e) => setSyncConfirmInput(e.target.value)}
                  placeholder={selectedApp}
                  className="w-full px-3 py-2 bg-surface-overlay border border-border-default rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  autoFocus
                />
              </div>

              {/* Actions */}
              <div className="flex items-center justify-end gap-2 px-5 pb-5">
                <button onClick={() => setSyncModalOpen(false)} className="btn btn-secondary">
                  Cancel
                </button>
                <button
                  onClick={handleSyncConfirm}
                  disabled={!syncConfirmMatch}
                  className={`btn text-white transition-colors ${
                    syncConfirmMatch
                      ? 'bg-amber-600 hover:bg-amber-700 focus:ring-amber-500'
                      : 'bg-amber-600/40 cursor-not-allowed'
                  }`}
                >
                  Sync
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
