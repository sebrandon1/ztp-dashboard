import { motion } from 'framer-motion';
import { ArrowRight, CheckCircle2, Circle, AlertCircle, Loader2, ChevronDown, ChevronUp, Sparkles } from 'lucide-react';
import { useState } from 'react';
import type { PipelineStatus, ResourceStatus } from '../types/api';

interface PipelineVisualizerProps {
  pipeline: PipelineStatus;
  onAskAI?: (context: Record<string, unknown>) => void;
}

interface PipelineStage {
  label: string;
  resource: ResourceStatus | ResourceStatus[] | undefined;
  key: string;
}

function getStageStatus(resource: ResourceStatus | ResourceStatus[] | undefined): string {
  if (!resource) return 'Pending';
  if (Array.isArray(resource)) {
    if (resource.length === 0) return 'Pending';
    const hasError = resource.some(r => r.status === 'Error' || r.status === 'Failed');
    if (hasError) return 'Error';
    const allComplete = resource.every(r => r.status === 'Completed' || r.status === 'True');
    if (allComplete) return 'Completed';
    return 'InProgress';
  }
  return resource.status;
}

function StatusIcon({ status }: { status: string }) {
  switch (status) {
    case 'Completed':
    case 'True':
      return <CheckCircle2 className="w-5 h-5 text-emerald-400" />;
    case 'Error':
    case 'Failed':
      return <AlertCircle className="w-5 h-5 text-red-400" />;
    case 'InProgress':
      return <Loader2 className="w-5 h-5 text-primary-400 animate-spin" />;
    default:
      return <Circle className="w-5 h-5 text-slate-500" />;
  }
}

function nodeColor(status: string): string {
  switch (status) {
    case 'Completed':
    case 'True':
      return 'border-emerald-500/30 bg-emerald-500/5';
    case 'Error':
    case 'Failed':
      return 'border-red-500/30 bg-red-500/5';
    case 'InProgress':
      return 'border-primary-500/30 bg-primary-500/5 pulse-glow';
    default:
      return 'border-border-default bg-surface-raised';
  }
}

export default function PipelineVisualizer({ pipeline, onAskAI }: PipelineVisualizerProps) {
  const [expandedStage, setExpandedStage] = useState<string | null>(null);

  const stages: PipelineStage[] = [
    { label: 'ClusterInstance', resource: pipeline.clusterInstance, key: 'ci' },
    { label: 'ClusterDeployment', resource: pipeline.clusterDeployment, key: 'cd' },
    { label: 'InfraEnv', resource: pipeline.infraEnv, key: 'ie' },
    { label: 'BareMetalHosts', resource: pipeline.bareMetalHosts, key: 'bmh' },
    { label: 'Agents', resource: pipeline.agents, key: 'agents' },
    { label: 'AgentClusterInstall', resource: pipeline.agentClusterInstall, key: 'aci' },
    { label: 'ManagedCluster', resource: pipeline.managedCluster, key: 'mc' },
  ];

  return (
    <div className="space-y-3">
      {/* Pipeline flow */}
      <div className="flex items-center gap-2 overflow-x-auto pb-4">
        {stages.map((stage, i) => {
          const status = getStageStatus(stage.resource);
          return (
            <div key={stage.key} className="flex items-center gap-2 shrink-0">
              <motion.button
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: i * 0.08 }}
                onClick={() => setExpandedStage(expandedStage === stage.key ? null : stage.key)}
                className={`relative flex items-center gap-2 px-4 py-3 rounded-xl border-2 transition-all duration-200 cursor-pointer hover:scale-105 ${nodeColor(status)}`}
              >
                <StatusIcon status={status} />
                <span className="text-sm font-medium text-text-primary whitespace-nowrap">{stage.label}</span>
                {Array.isArray(stage.resource) && stage.resource.length > 0 && (
                  <span className="text-xs text-text-muted">({stage.resource.length})</span>
                )}
                {stage.resource && (
                  expandedStage === stage.key ? <ChevronUp className="w-3 h-3 text-text-muted" /> : <ChevronDown className="w-3 h-3 text-text-muted" />
                )}
              </motion.button>
              {i < stages.length - 1 && (
                <ArrowRight className={`w-4 h-4 shrink-0 ${status === 'Completed' || status === 'True' ? 'text-emerald-400' : 'text-text-muted'}`} />
              )}
            </div>
          );
        })}
      </div>

      {/* Expanded stage details */}
      {expandedStage && (() => {
        const stage = stages.find(s => s.key === expandedStage);
        if (!stage?.resource) return null;
        const resources = Array.isArray(stage.resource) ? stage.resource : [stage.resource];
        const status = getStageStatus(stage.resource);
        const hasError = status === 'Error' || status === 'Failed';

        return (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="card p-4"
          >
            <div className="flex items-center justify-between mb-3">
              <h4 className="text-sm font-semibold text-text-primary">{stage.label} Details</h4>
              {hasError && onAskAI && (
                <button
                  onClick={() => onAskAI({
                    clusterName: pipeline.clusterName,
                    resource: stage.label,
                    conditions: resources.flatMap(r => r.conditions || []),
                  })}
                  className="btn btn-ghost text-xs gap-1.5"
                >
                  <Sparkles className="w-3 h-3" />
                  Ask AI
                </button>
              )}
            </div>
            {resources.map((r) => (
              <div key={r.name} className="mb-3 last:mb-0">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-sm font-medium text-text-primary">{r.name}</span>
                  <span className="text-xs text-text-muted">{r.namespace}</span>
                </div>
                {r.conditions && r.conditions.length > 0 && (
                  <div className="space-y-1">
                    {r.conditions.map((c, idx) => (
                      <div key={idx} className="flex items-start gap-2 text-xs p-2 rounded-lg bg-surface/50">
                        <span className={`shrink-0 mt-0.5 w-1.5 h-1.5 rounded-full ${
                          c.status === 'True' ? 'bg-emerald-400' : c.status === 'False' ? 'bg-red-400' : 'bg-amber-400'
                        }`} />
                        <div>
                          <span className="font-medium text-text-secondary">{String(c.type)}</span>
                          {c.message && (
                            <p className="text-text-muted mt-0.5">{String(c.message)}</p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </motion.div>
        );
      })()}
    </div>
  );
}
