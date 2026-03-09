import { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Plus, Pencil, Trash2, Server, HardDrive, Shield, GitBranch, Cpu, FileCode, Network, Box, Sparkles, Loader2, ChevronDown, ChevronUp, CheckCircle2, XCircle, AlertTriangle, Info } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import type { WatchEvent } from '../types/api';
import { aiAPI } from '../lib/api';

interface EventFeedProps {
  events: WatchEvent[];
  maxItems?: number;
  autoAI?: boolean;
}

const resourceIcons: Record<string, typeof Server> = {
  ManagedCluster: Server,
  ClusterDeployment: FileCode,
  ClusterInstance: Box,
  BareMetalHost: HardDrive,
  AgentClusterInstall: Cpu,
  Agent: Cpu,
  InfraEnv: Network,
  Policy: Shield,
  Application: GitBranch,
};

const eventTypeConfig = {
  ADDED: { icon: Plus, color: 'text-emerald-400', bg: 'bg-emerald-500/10', label: 'Created' },
  MODIFIED: { icon: Pencil, color: 'text-blue-400', bg: 'bg-blue-500/10', label: 'Updated' },
  DELETED: { icon: Trash2, color: 'text-red-400', bg: 'bg-red-500/10', label: 'Deleted' },
};

const severityConfig = {
  good: { icon: CheckCircle2, color: 'text-emerald-400', bg: 'bg-emerald-500/10', ring: 'ring-emerald-500/20' },
  bad: { icon: XCircle, color: 'text-red-400', bg: 'bg-red-500/10', ring: 'ring-red-500/20' },
  warning: { icon: AlertTriangle, color: 'text-amber-400', bg: 'bg-amber-500/10', ring: 'ring-amber-500/20' },
  info: { icon: Info, color: 'text-blue-400', bg: 'bg-blue-500/10', ring: 'ring-blue-500/20' },
  neutral: { icon: Info, color: 'text-slate-400', bg: 'bg-slate-500/10', ring: 'ring-slate-500/20' },
};

function formatTimestamp(ts?: string) {
  if (!ts) return '';
  const d = new Date(ts);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();

  if (diffMs < 60000) {
    const secs = Math.floor(diffMs / 1000);
    return `${secs}s ago`;
  }
  if (d.toDateString() === now.toDateString()) {
    return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) + ' ' +
    d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' });
}

export default function EventFeed({ events, maxItems = 50, autoAI = false }: EventFeedProps) {
  const displayed = events.slice(0, maxItems);

  if (displayed.length === 0) {
    return (
      <div className="text-center py-12">
        <Server className="w-10 h-10 text-text-muted mx-auto mb-3 opacity-30" />
        <p className="text-sm text-text-muted">Waiting for cluster events...</p>
        <p className="text-xs text-text-muted mt-1">Events will appear here as resources change</p>
      </div>
    );
  }

  return (
    <div className="space-y-0.5">
      <AnimatePresence initial={false}>
        {displayed.map((event, i) => (
          <EventRow
            key={`${event.timestamp}-${event.resource_type}-${event.name}-${i}`}
            event={event}
            autoAI={autoAI}
          />
        ))}
      </AnimatePresence>
    </div>
  );
}

function EventRow({ event, autoAI }: { event: WatchEvent; autoAI: boolean }) {
  const [aiInsight, setAiInsight] = useState<string | null>(null);
  const [aiLoading, setAiLoading] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const autoAITriggered = useRef(false);

  const ResourceIcon = resourceIcons[event.resource_type] || Server;
  const evtConfig = eventTypeConfig[event.event_type] || eventTypeConfig.MODIFIED;
  const EventIcon = evtConfig.icon;
  const sevConfig = severityConfig[event.severity] || severityConfig.info;
  const SeverityIcon = sevConfig.icon;

  // Auto-AI: trigger on mount if enabled and event is bad/warning
  useEffect(() => {
    if (autoAI && !autoAITriggered.current && (event.severity === 'bad' || event.severity === 'warning')) {
      autoAITriggered.current = true;
      fetchAIInsight();
    }
  }, [autoAI]); // eslint-disable-line react-hooks/exhaustive-deps

  const fetchAIInsight = async () => {
    if (aiLoading) return;
    setAiLoading(true);
    setExpanded(true);

    try {
      const context = {
        resourceType: event.resource_type,
        name: event.name,
        namespace: event.namespace,
        eventType: event.event_type,
        summary: event.summary,
        severity: event.severity,
        insight: event.insight,
        data: event.data,
      };

      const res = await aiAPI.diagnose(context, undefined, undefined);
      if (!res.ok) {
        setAiInsight('Failed to get AI insight.');
        setAiLoading(false);
        return;
      }

      const reader = res.body?.getReader();
      if (!reader) {
        setAiInsight('No response from AI.');
        setAiLoading(false);
        return;
      }

      const decoder = new TextDecoder();
      let buffer = '';
      let fullResponse = '';

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
                fullResponse += data.token;
                setAiInsight(fullResponse);
              }
            } catch {
              // skip
            }
          }
        }
      }
    } catch {
      setAiInsight('AI service unavailable.');
    } finally {
      setAiLoading(false);
    }
  };

  const handleAIClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (aiInsight) {
      setExpanded(!expanded);
      return;
    }
    fetchAIInsight();
  };

  return (
    <motion.div
      initial={{ opacity: 0, x: -20, height: 0 }}
      animate={{ opacity: 1, x: 0, height: 'auto' }}
      exit={{ opacity: 0, height: 0 }}
      transition={{ duration: 0.2 }}
      className="rounded-lg hover:bg-surface-overlay/50 transition-colors group"
    >
      <div className="flex items-start gap-3 px-3 py-2.5">
        {/* Timestamp */}
        <span className="text-[11px] text-text-muted font-mono shrink-0 w-16 pt-0.5 text-right">
          {formatTimestamp(event.timestamp)}
        </span>

        {/* Severity indicator */}
        <div className={`w-6 h-6 rounded-md flex items-center justify-center shrink-0 mt-0.5 ${sevConfig.bg}`}>
          <SeverityIcon className={`w-3.5 h-3.5 ${sevConfig.color}`} />
        </div>

        {/* Event type icon */}
        <div className={`w-5 h-5 rounded flex items-center justify-center shrink-0 mt-0.5 ${evtConfig.bg}`}>
          <EventIcon className={`w-2.5 h-2.5 ${evtConfig.color}`} />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Summary */}
          <p className="text-sm text-text-primary leading-snug">
            {event.summary || `${event.resource_type} ${event.name} ${evtConfig.label.toLowerCase()}`}
          </p>

          {/* Insight + metadata */}
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            {/* Insight badge */}
            <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${sevConfig.bg} ${sevConfig.color} ring-1 ${sevConfig.ring}`}>
              {event.insight}
            </span>

            {/* Resource type */}
            <span className="flex items-center gap-1 text-[10px] text-text-muted">
              <ResourceIcon className="w-3 h-3" />
              {event.resource_type}
            </span>

            {event.namespace && event.namespace !== event.name && (
              <span className="text-[10px] text-text-muted">ns/{event.namespace}</span>
            )}

            <EventDataChips event={event} />
          </div>
        </div>

        {/* AI button */}
        <button
          onClick={handleAIClick}
          className={`shrink-0 flex items-center gap-1 px-2 py-1 rounded-md text-[10px] font-medium transition-all ${
            aiInsight
              ? 'bg-primary-500/10 text-primary-400 ring-1 ring-primary-500/20'
              : 'opacity-0 group-hover:opacity-100 bg-surface-overlay text-text-muted hover:text-primary-400 hover:bg-primary-500/10'
          }`}
          title="AI deep analysis"
        >
          {aiLoading ? (
            <Loader2 className="w-3 h-3 animate-spin" />
          ) : (
            <Sparkles className="w-3 h-3" />
          )}
          {aiInsight ? (expanded ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />) : 'AI'}
        </button>
      </div>

      {/* AI panel */}
      <AnimatePresence>
        {expanded && (aiInsight || aiLoading) && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.2 }}
            className="mx-3 mb-2 ml-[7.5rem]"
          >
            <div className="p-3 rounded-lg bg-primary-500/5 border border-primary-500/10">
              {aiLoading && !aiInsight && (
                <div className="flex items-center gap-2">
                  <Loader2 className="w-3.5 h-3.5 text-primary-400 animate-spin" />
                  <span className="text-xs text-text-muted">Analyzing event...</span>
                </div>
              )}
              {aiInsight && (
                <div className="prose prose-invert prose-xs max-w-none text-xs leading-relaxed">
                  <ReactMarkdown>{aiInsight}</ReactMarkdown>
                  {aiLoading && <span className="inline-block w-1.5 h-3 bg-primary-400 animate-pulse ml-0.5" />}
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}

function EventDataChips({ event }: { event: WatchEvent }) {
  const chips: { label: string; className: string }[] = [];

  if (!event.data) return null;

  if (event.resource_type === 'Policy') {
    const compliant = event.data.compliant as string;
    if (compliant === 'Compliant') {
      chips.push({ label: 'Compliant', className: 'bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20' });
    } else if (compliant === 'NonCompliant') {
      chips.push({ label: 'Non-Compliant', className: 'bg-red-500/10 text-red-400 ring-1 ring-red-500/20' });
    }
  }

  if (event.resource_type === 'BareMetalHost') {
    const poweredOn = event.data.poweredOn as boolean;
    if (poweredOn !== undefined) {
      chips.push({
        label: poweredOn ? 'Powered On' : 'Powered Off',
        className: poweredOn ? 'bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20' : 'bg-slate-500/10 text-slate-400 ring-1 ring-slate-500/20',
      });
    }
  }

  if (event.resource_type === 'Application') {
    const health = event.data.health as string;
    const sync = event.data.sync as string;
    if (health) {
      chips.push({
        label: health,
        className: health === 'Healthy' ? 'bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20' : 'bg-amber-500/10 text-amber-400 ring-1 ring-amber-500/20',
      });
    }
    if (sync) {
      chips.push({
        label: sync,
        className: sync === 'Synced' ? 'bg-blue-500/10 text-blue-400 ring-1 ring-blue-500/20' : 'bg-amber-500/10 text-amber-400 ring-1 ring-amber-500/20',
      });
    }
  }

  if (chips.length === 0) return null;

  return (
    <>
      {chips.map((chip, i) => (
        <span key={i} className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${chip.className}`}>
          {chip.label}
        </span>
      ))}
    </>
  );
}
