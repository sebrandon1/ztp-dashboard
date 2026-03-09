import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ArrowLeft, GitBranch, Shield, Sparkles, Loader2 } from 'lucide-react';
import { clusterAPI } from '../lib/api';
import { useDashboardStore } from '../lib/store';
import StatusBadge from '../components/StatusBadge';
import PipelineVisualizer from '../components/PipelineVisualizer';
import PolicyComplianceBar from '../components/PolicyComplianceBar';
import AIAssistantPanel from '../components/AIAssistantPanel';
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

  // Debounced refetch on updateCounter changes
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
                    policies.map((policy) => (
                      <div key={policy.name} className="flex items-center justify-between px-4 py-3">
                        <div>
                          <p className="text-sm font-medium text-text-primary">{policy.name}</p>
                          {policy.namespace && (
                            <p className="text-xs text-text-muted">{policy.namespace}</p>
                          )}
                        </div>
                        <StatusBadge status={policy.data?.compliant || 'Unknown'} />
                      </div>
                    ))
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
    </div>
  );
}
