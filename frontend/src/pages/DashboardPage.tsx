import { motion } from 'framer-motion';
import { Server, CheckCircle2, AlertCircle, Clock, Zap, Wifi } from 'lucide-react';
import { useDashboardStore } from '../lib/store';
import EventFeed from '../components/EventFeed';

export default function DashboardPage() {
  const { hubStatus, clusters, wsConnected, events } = useDashboardStore();

  const available = clusters.filter(c => c.available === 'True').length;
  const unavailable = clusters.filter(c => c.available !== 'True').length;

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
