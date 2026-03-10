import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Server, CheckCircle2, AlertCircle, Clock, Zap, Wifi, Shield, GitBranch } from 'lucide-react';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';
import { useDashboardStore } from '../lib/store';
import { policyAPI, argoAPI } from '../lib/api';
import EventFeed from '../components/EventFeed';
import type { PolicySummary, ArgoSummary, NonCompliantPolicy } from '../types/api';

export default function DashboardPage() {
  const navigate = useNavigate();
  const { hubStatus, clusters, wsConnected, events, updateCounter } = useDashboardStore();
  const [policySummary, setPolicySummary] = useState<PolicySummary | null>(null);
  const [argoSummary, setArgoSummary] = useState<ArgoSummary | null>(null);

  const available = clusters.filter(c => c.available === 'True').length;
  const unavailable = clusters.filter(c => c.available !== 'True').length;

  useEffect(() => {
    policyAPI.getSummary().then(setPolicySummary).catch(() => {});
    argoAPI.getSummary().then(setArgoSummary).catch(() => {});
  }, []);

  useEffect(() => {
    if (updateCounter === 0) return;
    const timer = setTimeout(() => {
      policyAPI.getSummary().then(setPolicySummary).catch(() => {});
      argoAPI.getSummary().then(setArgoSummary).catch(() => {});
    }, 3000);
    return () => clearTimeout(timer);
  }, [updateCounter]);

  const stats = [
    {
      label: 'Total Clusters',
      value: clusters.length,
      icon: Server,
      color: 'text-primary-400',
      bg: 'bg-primary-500/10',
    },
    {
      label: 'Available',
      value: available,
      icon: CheckCircle2,
      color: 'text-emerald-400',
      bg: 'bg-emerald-500/10',
    },
    {
      label: 'Unavailable',
      value: unavailable,
      icon: AlertCircle,
      color: unavailable > 0 ? 'text-red-400' : 'text-text-muted',
      bg: unavailable > 0 ? 'bg-red-500/10' : 'bg-surface-overlay',
    },
    {
      label: 'WebSocket',
      value: wsConnected ? 'Live' : 'Down',
      icon: Wifi,
      color: wsConnected ? 'text-emerald-400' : 'text-amber-400',
      bg: wsConnected ? 'bg-emerald-500/10' : 'bg-amber-500/10',
    },
  ];

  const policyChartData = policySummary ? [
    { name: 'Compliant', value: policySummary.compliant, color: '#10b981' },
    { name: 'Non-Compliant', value: policySummary.nonCompliant, color: '#ef4444' },
    { name: 'Unknown', value: policySummary.unknown, color: '#64748b' },
  ].filter(d => d.value > 0) : [];

  const argoChartData = argoSummary ? [
    { name: 'Healthy', value: argoSummary.healthy, color: '#10b981' },
    { name: 'Degraded', value: argoSummary.degraded, color: '#ef4444' },
    { name: 'Other', value: argoSummary.other, color: '#64748b' },
  ].filter(d => d.value > 0) : [];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
          <Zap className="w-6 h-6 text-primary-400" />
          ZTP Hub Overview
        </h1>
        <p className="text-sm text-text-muted mt-1">
          {hubStatus?.connected
            ? `Connected to hub cluster ${hubStatus.serverVersion ? `(${hubStatus.serverVersion})` : ''}`
            : 'Not connected to hub cluster'}
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, i) => (
          <motion.div
            key={stat.label}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: i * 0.08 }}
            className="card p-5"
          >
            <div className="flex items-center justify-between mb-3">
              <span className="text-xs font-medium text-text-muted uppercase tracking-wider">{stat.label}</span>
              <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${stat.bg}`}>
                <stat.icon className={`w-4 h-4 ${stat.color}`} />
              </div>
            </div>
            <p className={`text-3xl font-bold ${stat.color}`}>{stat.value}</p>
          </motion.div>
        ))}
      </div>

      {/* Policy + ArgoCD summary row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Policy Compliance */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.35 }}
          className="card"
        >
          <div className="flex items-center justify-between px-5 py-4 border-b border-border-subtle">
            <div className="flex items-center gap-2">
              <Shield className="w-4 h-4 text-text-muted" />
              <h2 className="text-sm font-semibold text-text-primary">Policy Compliance</h2>
            </div>
            {policySummary && (
              <span className="text-xs text-text-muted">{policySummary.total} policies</span>
            )}
          </div>
          <div className="p-5">
            {policySummary ? (
              <div className="flex items-center gap-6">
                {policyChartData.length > 0 && (
                  <div className="w-24 h-24 shrink-0">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={policyChartData}
                          cx="50%"
                          cy="50%"
                          innerRadius={25}
                          outerRadius={40}
                          paddingAngle={2}
                          dataKey="value"
                          strokeWidth={0}
                        >
                          {policyChartData.map((entry, i) => (
                            <Cell key={i} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip
                          contentStyle={{ background: '#1e293b', border: 'none', borderRadius: '8px', fontSize: '12px' }}
                          itemStyle={{ color: '#e2e8f0' }}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                )}
                <div className="space-y-2 flex-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
                      <span className="text-text-secondary">Compliant</span>
                    </span>
                    <span className="font-semibold text-emerald-400">{policySummary.compliant}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-red-500" />
                      <span className="text-text-secondary">Non-Compliant</span>
                    </span>
                    <span className="font-semibold text-red-400">{policySummary.nonCompliant}</span>
                  </div>
                  {policySummary.unknown > 0 && (
                    <div className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="w-2.5 h-2.5 rounded-full bg-slate-500" />
                        <span className="text-text-secondary">Unknown</span>
                      </span>
                      <span className="font-semibold text-slate-400">{policySummary.unknown}</span>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <p className="text-sm text-text-muted text-center py-4">Loading...</p>
            )}

            {/* Non-compliant policies list */}
            {policySummary && policySummary.nonCompliantPolicies.length > 0 && (
              <div className="mt-4 pt-4 border-t border-border-subtle">
                <p className="text-xs font-medium text-text-muted uppercase tracking-wider mb-2">Non-Compliant Policies</p>
                <div className="space-y-1 max-h-32 overflow-y-auto">
                  {policySummary.nonCompliantPolicies.slice(0, 10).map((p: NonCompliantPolicy, i: number) => (
                    <button
                      key={i}
                      onClick={() => navigate(`/clusters/${p.clusterName}`)}
                      className="flex items-center justify-between w-full text-left px-2 py-1.5 rounded-md hover:bg-surface-overlay/50 transition-colors"
                    >
                      <span className="text-xs text-text-primary truncate">{p.name}</span>
                      <span className="text-[10px] text-text-muted shrink-0 ml-2">{p.clusterName}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        </motion.div>

        {/* ArgoCD Summary */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.4 }}
          className="card"
        >
          <div className="flex items-center justify-between px-5 py-4 border-b border-border-subtle">
            <div className="flex items-center gap-2">
              <GitBranch className="w-4 h-4 text-text-muted" />
              <h2 className="text-sm font-semibold text-text-primary">ArgoCD Applications</h2>
            </div>
            {argoSummary && (
              <span className="text-xs text-text-muted">{argoSummary.total} apps</span>
            )}
          </div>
          <div className="p-5">
            {argoSummary ? (
              <div className="flex items-center gap-6">
                {argoChartData.length > 0 && (
                  <div className="w-24 h-24 shrink-0">
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={argoChartData}
                          cx="50%"
                          cy="50%"
                          innerRadius={25}
                          outerRadius={40}
                          paddingAngle={2}
                          dataKey="value"
                          strokeWidth={0}
                        >
                          {argoChartData.map((entry, i) => (
                            <Cell key={i} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip
                          contentStyle={{ background: '#1e293b', border: 'none', borderRadius: '8px', fontSize: '12px' }}
                          itemStyle={{ color: '#e2e8f0' }}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>
                )}
                <div className="space-y-2 flex-1">
                  <div className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
                      <span className="text-text-secondary">Healthy</span>
                    </span>
                    <span className="font-semibold text-emerald-400">{argoSummary.healthy}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-red-500" />
                      <span className="text-text-secondary">Degraded</span>
                    </span>
                    <span className="font-semibold text-red-400">{argoSummary.degraded}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-amber-500" />
                      <span className="text-text-secondary">Out of Sync</span>
                    </span>
                    <span className="font-semibold text-amber-400">{argoSummary.outOfSync}</span>
                  </div>
                  <div className="flex items-center justify-between text-sm">
                    <span className="flex items-center gap-2">
                      <span className="w-2.5 h-2.5 rounded-full bg-blue-500" />
                      <span className="text-text-secondary">Synced</span>
                    </span>
                    <span className="font-semibold text-blue-400">{argoSummary.synced}</span>
                  </div>
                </div>
              </div>
            ) : (
              <p className="text-sm text-text-muted text-center py-4">Loading...</p>
            )}
          </div>
        </motion.div>
      </div>

      {/* Recent events */}
      <div className="card">
        <div className="flex items-center justify-between px-5 py-4 border-b border-border-subtle">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-text-muted" />
            <h2 className="text-sm font-semibold text-text-primary">Recent Events</h2>
          </div>
          <span className="text-xs text-text-muted">{events.length} events</span>
        </div>
        <div className="p-2 max-h-96 overflow-y-auto">
          <EventFeed events={events} maxItems={20} />
        </div>
      </div>
    </div>
  );
}
