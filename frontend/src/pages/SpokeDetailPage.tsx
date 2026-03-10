import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ArrowLeft, GitBranch, Shield, Sparkles, Loader2, Code, Trash2, Unlink } from 'lucide-react';
import { toast } from 'sonner';
import { clusterAPI, resourceAPI } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import StatusBadge from '../components/StatusBadge';
import PipelineVisualizer from '../components/PipelineVisualizer';
import PolicyComplianceBar from '../components/PolicyComplianceBar';
import AIAssistantPanel from '../components/AIAssistantPanel';
import YamlViewer from '../components/YamlViewer';
import ConfirmationModal from '../components/ConfirmationModal';
import type { PipelineStatus, PolicyInfo } from '../types/api';

type Tab = 'pipeline' | 'policies' | 'ai';

export default function SpokeDetailPage() {
  const { clusterName } = useParams<{ clusterName: string }>();
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>('pipeline');
  const [pipeline, setPipeline] = useState<PipelineStatus | null>(null);
  const [policies, setPolicies] = useState<PolicyInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [aiContext, setAiContext] = useState<Record<string, unknown> | undefined>(undefined);
  const [yamlData, setYamlData] = useState<Record<string, unknown> | null>(null);
  const [yamlTitle, setYamlTitle] = useState('');
  const [togglingPolicy, setTogglingPolicy] = useState<string | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showDetachModal, setShowDetachModal] = useState(false);
  const { updateCounter } = useDashboardStore();
  const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  useEffect(() => {
    if (!clusterName) return;
    let cancelled = false;
    Promise.all([
      clusterAPI.getPipeline(clusterName).catch(() => null),
      clusterAPI.getPolicies(clusterName).catch(() => []),
    ]).then(([p, pol]) => {
      if (cancelled) return;
      if (p) setPipeline(p);
      setPolicies(pol as PolicyInfo[]);
      setLoading(false);
    });
    return () => { cancelled = true; };
  }, [clusterName]);

  useEffect(() => {
    if (updateCounter === 0 || !clusterName) return;
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      clusterAPI.getPipeline(clusterName).then(setPipeline).catch(() => {});
      clusterAPI.getPolicies(clusterName).then(p => setPolicies(p as PolicyInfo[])).catch(() => {});
    }, 2000);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [updateCounter, clusterName]);

  const handleAskAI = (context: Record<string, unknown>) => {
    setAiContext({ ...context, clusterName });
    setTab('ai');
  };

  const handleTogglePolicy = async (policy: PolicyInfo) => {
    if (!clusterName) return;
    const currentDisabled = policy.data?.disabled as boolean || false;
    const newDisabled = !currentDisabled;
    setTogglingPolicy(policy.name);
    try {
      await clusterAPI.setPolicyState(clusterName, policy.name, newDisabled);
      setPolicies(prev => prev.map(p =>
        p.name === policy.name
          ? { ...p, data: { ...p.data, disabled: newDisabled } }
          : p
      ));
      toast.success(`Policy ${policy.name} ${newDisabled ? 'disabled' : 'enabled'}`);
    } catch {
      toast.error(`Failed to ${newDisabled ? 'disable' : 'enable'} policy`);
    } finally {
      setTogglingPolicy(null);
    }
  };

  const handleViewYaml = async (group: string, version: string, resource: string, namespace: string, name: string) => {
    try {
      const data = await resourceAPI.get(group, version, resource, namespace, name);
      setYamlData(data);
      setYamlTitle(`${resource}/${name}`);
    } catch {
      toast.error('Failed to load resource YAML');
    }
  };

  const handleDeleteCluster = async () => {
    if (!clusterName) return;
    try {
      await clusterAPI.delete(clusterName, clusterName);
      toast.success(`Cluster ${clusterName} deleted`);
      navigate('/clusters');
    } catch {
      toast.error('Failed to delete cluster');
    }
  };

  const handleDetachCluster = async () => {
    if (!clusterName) return;
    try {
      await clusterAPI.detach(clusterName, clusterName);
      toast.success(`Cluster ${clusterName} detached`);
      navigate('/clusters');
    } catch {
      toast.error('Failed to detach cluster');
    }
  };

  const tabs: { key: Tab; label: string; icon: typeof GitBranch }[] = [
    { key: 'pipeline', label: 'Pipeline', icon: GitBranch },
    { key: 'policies', label: 'Policies', icon: Shield },
    { key: 'ai', label: 'AI Diagnostics', icon: Sparkles },
  ];

  const cluster = useDashboardStore(s => s.clusters.find(c => c.name === clusterName));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button onClick={() => navigate('/clusters')} className="btn btn-ghost p-2">
          <ArrowLeft className="w-4 h-4" />
        </button>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-text-primary">{clusterName}</h1>
            {cluster && <StatusBadge status={cluster.available === 'True' ? 'True' : 'False'} size="md" />}
          </div>
          {cluster?.openshiftVersion && (
            <p className="text-sm text-text-muted mt-0.5">OpenShift {cluster.openshiftVersion}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowDetachModal(true)}
            className="btn btn-ghost text-amber-400 hover:bg-amber-500/10 gap-1.5 text-xs"
          >
            <Unlink className="w-3.5 h-3.5" />
            Detach
          </button>
          <button
            onClick={() => setShowDeleteModal(true)}
            className="btn btn-ghost text-red-400 hover:bg-red-500/10 gap-1.5 text-xs"
          >
            <Trash2 className="w-3.5 h-3.5" />
            Delete
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 bg-surface-raised rounded-xl border border-border-subtle p-1">
        {tabs.map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-all ${
              tab === key
                ? 'bg-primary-600 text-white shadow-sm'
                : 'text-text-muted hover:text-text-secondary hover:bg-surface-overlay/50'
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <Loader2 className="w-6 h-6 text-primary-400 animate-spin" />
        </div>
      ) : (
        <motion.div
          key={tab}
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.2 }}
        >
          {tab === 'pipeline' && pipeline && (
            <PipelineVisualizer pipeline={pipeline} onAskAI={handleAskAI} />
          )}

          {tab === 'policies' && (
            <div className="space-y-4">
              <PolicyComplianceBar policies={policies} />
              <div className="card">
                <div className="divide-y divide-border-subtle">
                  {policies.length === 0 ? (
                    <div className="p-8 text-center text-sm text-text-muted">No policies found for this cluster</div>
                  ) : (
                    policies.map((policy) => {
                      const isDisabled = policy.data?.disabled as boolean || false;
                      const remediationAction = policy.data?.remediationAction as string || '';
                      return (
                        <div
                          key={policy.name}
                          className={`flex items-center justify-between px-4 py-3 ${isDisabled ? 'opacity-60' : ''}`}
                        >
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <p className="text-sm font-medium text-text-primary">{policy.name}</p>
                              {isDisabled && (
                                <span className="badge badge-neutral text-[10px]">Disabled</span>
                              )}
                              {remediationAction && (
                                <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${
                                  remediationAction === 'enforce'
                                    ? 'bg-blue-500/10 text-blue-400 ring-1 ring-blue-500/20'
                                    : 'bg-slate-500/10 text-slate-400 ring-1 ring-slate-500/20'
                                }`}>
                                  {remediationAction}
                                </span>
                              )}
                            </div>
                            {policy.namespace && (
                              <p className="text-xs text-text-muted">{policy.namespace}</p>
                            )}
                          </div>
                          <div className="flex items-center gap-3">
                            <StatusBadge status={isDisabled ? 'Unknown' : (policy.data?.compliant as string || 'Unknown')} />
                            <button
                              onClick={() => handleViewYaml(
                                'policy.open-cluster-management.io', 'v1', 'policies',
                                policy.namespace || clusterName || '', policy.name
                              )}
                              className="btn btn-ghost p-1.5"
                              title="View YAML"
                            >
                              <Code className="w-3.5 h-3.5 text-text-muted" />
                            </button>
                            <button
                              onClick={() => handleTogglePolicy(policy)}
                              disabled={togglingPolicy === policy.name}
                              className="relative w-9 h-5 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
                              style={{ backgroundColor: isDisabled ? 'var(--color-surface-overlay)' : 'var(--color-primary-600)' }}
                              title={isDisabled ? 'Enable policy' : 'Disable policy'}
                            >
                              {togglingPolicy === policy.name ? (
                                <Loader2 className="w-3 h-3 text-text-muted animate-spin absolute top-1 left-3" />
                              ) : (
                                <span
                                  className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${
                                    isDisabled ? 'left-0.5' : 'left-[18px]'
                                  }`}
                                />
                              )}
                            </button>
                          </div>
                        </div>
                      );
                    })
                  )}
                </div>
              </div>
            </div>
          )}

          {tab === 'ai' && (
            <AIAssistantPanel
              initialContext={aiContext || { clusterName }}
              diagnoseType={aiContext ? 'provisioning' : undefined}
            />
          )}
        </motion.div>
      )}

      {/* YAML Viewer Modal */}
      {yamlData && (
        <YamlViewer
          data={yamlData}
          title={yamlTitle}
          onClose={() => setYamlData(null)}
        />
      )}

      {/* Cluster Action Modals */}
      <ConfirmationModal
        open={showDeleteModal}
        onClose={() => setShowDeleteModal(false)}
        onConfirm={handleDeleteCluster}
        title="Delete Cluster"
        message={`This will permanently delete the ManagedCluster "${clusterName}" and all associated namespace resources. This action cannot be undone.`}
        confirmText={clusterName || ''}
        variant="danger"
      />
      <ConfirmationModal
        open={showDetachModal}
        onClose={() => setShowDetachModal(false)}
        onConfirm={handleDetachCluster}
        title="Detach Cluster"
        message={`This will detach the spoke cluster "${clusterName}" from this hub. The spoke cluster itself will continue running but will no longer be managed.`}
        confirmText={clusterName || ''}
        variant="warning"
      />
    </div>
  );
}
