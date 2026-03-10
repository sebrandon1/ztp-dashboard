import { useState, useEffect, useCallback } from 'react';
import { motion } from 'framer-motion';
import { Radio, Pause, Play, Sparkles, Search, Download, ChevronLeft, ChevronRight } from 'lucide-react';
import { useDashboardStore } from '../lib/store';
import { eventsAPI } from '../lib/api';
import EventFeed from '../components/EventFeed';
import type { WatchEvent, EventStats } from '../types/api';

const SEVERITIES = ['all', 'good', 'bad', 'warning', 'neutral', 'info'] as const;
const PAGE_SIZE = 50;

export default function EventsPage() {
  const { updateCounter } = useDashboardStore();
  const [paused, setPaused] = useState(false);
  const [query, setQuery] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>('all');
  const [autoAI, setAutoAI] = useState(false);
  const [events, setEvents] = useState<WatchEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [stats, setStats] = useState<EventStats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    try {
      const result = await eventsAPI.query({
        q: query || undefined,
        severity: severityFilter !== 'all' ? severityFilter : undefined,
        limit: PAGE_SIZE,
        offset,
      });
      setEvents(result.events || []);
      setTotal(result.total);
    } catch {
      // keep existing events on error
    } finally {
      setLoading(false);
    }
  }, [query, severityFilter, offset]);

  // Load events on mount and when filters/pagination change
  useEffect(() => {
    fetchEvents();
  }, [fetchEvents]);

  // Reload on new WS events (unless paused)
  useEffect(() => {
    if (updateCounter === 0 || paused) return;
    const timer = setTimeout(fetchEvents, 2000);
    return () => clearTimeout(timer);
  }, [updateCounter, paused, fetchEvents]);

  // Load stats on mount
  useEffect(() => {
    eventsAPI.getStats().then(setStats).catch(() => {});
  }, []);

  // Refresh stats periodically
  useEffect(() => {
    if (updateCounter === 0) return;
    const timer = setTimeout(() => {
      eventsAPI.getStats().then(setStats).catch(() => {});
    }, 5000);
    return () => clearTimeout(timer);
  }, [updateCounter]);

  // Reset to first page when filters change
  useEffect(() => {
    setOffset(0);
  }, [query, severityFilter]);

  const totalPages = Math.ceil(total / PAGE_SIZE);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  const handleExportCSV = async () => {
    try {
      const result = await eventsAPI.query({
        q: query || undefined,
        severity: severityFilter !== 'all' ? severityFilter : undefined,
        limit: 500,
      });
      const rows = (result.events || []).map(e => [
        e.timestamp || '',
        e.event_type,
        e.resource_type,
        e.name,
        e.namespace,
        e.severity,
        `"${(e.summary || '').replace(/"/g, '""')}"`,
      ]);
      const csv = ['Timestamp,Event Type,Resource Type,Name,Namespace,Severity,Summary', ...rows.map(r => r.join(','))].join('\n');
      const blob = new Blob([csv], { type: 'text/csv' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `ztp-events-${new Date().toISOString().slice(0, 10)}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // ignore export errors
    }
  };

  const severityColor: Record<string, string> = {
    good: 'text-emerald-400',
    bad: 'text-red-400',
    warning: 'text-amber-400',
    neutral: 'text-slate-400',
    info: 'text-blue-400',
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
            <Radio className="w-6 h-6 text-primary-400" />
            Events
          </h1>
          <p className="text-sm text-text-muted mt-1">
            Persistent event log with search and filtering
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setAutoAI(!autoAI)}
            className={`btn gap-2 ${autoAI ? 'bg-primary-600/20 text-primary-400 ring-1 ring-primary-500/30' : 'btn-secondary'}`}
            title="Auto-analyze bad/warning events with AI"
          >
            <Sparkles className={`w-4 h-4 ${autoAI ? 'text-primary-400' : ''}`} />
            Auto-AI
            <span className={`w-2 h-2 rounded-full ${autoAI ? 'bg-primary-400 animate-pulse' : 'bg-text-muted'}`} />
          </button>
          <button
            onClick={() => setPaused(!paused)}
            className={`btn ${paused ? 'btn-primary' : 'btn-secondary'} gap-2`}
          >
            {paused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
            {paused ? 'Resume' : 'Pause'}
          </button>
          <button
            onClick={handleExportCSV}
            className="btn btn-secondary gap-2"
            title="Export filtered events as CSV"
          >
            <Download className="w-4 h-4" />
            CSV
          </button>
        </div>
      </div>

      {/* Auto-AI info banner */}
      {autoAI && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          exit={{ opacity: 0, height: 0 }}
          className="flex items-center gap-3 px-4 py-3 rounded-xl bg-primary-500/5 border border-primary-500/10"
        >
          <Sparkles className="w-4 h-4 text-primary-400 shrink-0" />
          <p className="text-xs text-text-secondary">
            <span className="font-medium text-primary-400">Auto-AI is on</span> — Bad and warning events will be automatically analyzed by ollama. This may use significant compute on your ollama instance.
          </p>
        </motion.div>
      )}

      {/* Stats bar */}
      {stats && stats.total > 0 && (
        <div className="flex items-center gap-4 text-xs">
          <span className="text-text-muted">Last 24h:</span>
          {Object.entries(stats.bySeverity).map(([sev, count]) => (
            <span key={sev} className={`flex items-center gap-1.5 ${severityColor[sev] || 'text-text-muted'}`}>
              <span className={`w-2 h-2 rounded-full ${
                sev === 'good' ? 'bg-emerald-400' :
                sev === 'bad' ? 'bg-red-400' :
                sev === 'warning' ? 'bg-amber-400' :
                sev === 'info' ? 'bg-blue-400' : 'bg-slate-400'
              }`} />
              {sev}: {count}
            </span>
          ))}
          <span className="text-text-muted ml-auto">{stats.total} total</span>
        </div>
      )}

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <input
            type="text"
            placeholder="Search events..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full pl-9 pr-4 py-2 bg-surface-raised border border-border-default rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <select
          value={severityFilter}
          onChange={(e) => setSeverityFilter(e.target.value)}
          className="px-3 py-2 bg-surface-raised border border-border-default rounded-lg text-sm text-text-secondary focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          {SEVERITIES.map(s => (
            <option key={s} value={s}>{s === 'all' ? 'All Severities' : s.charAt(0).toUpperCase() + s.slice(1)}</option>
          ))}
        </select>
        <div className="flex items-center gap-2 text-xs text-text-muted">
          <motion.div
            className="w-2 h-2 rounded-full bg-emerald-400"
            animate={!paused ? { opacity: [1, 0.3, 1] } : {}}
            transition={{ duration: 1.5, repeat: Infinity }}
          />
          {loading ? 'Loading...' : paused ? 'Paused' : `${total} events`}
        </div>
      </div>

      {/* Event stream */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="card"
      >
        <div className="p-2 max-h-[calc(100vh-360px)] overflow-y-auto">
          {paused ? (
            <div className="text-center py-12">
              <Pause className="w-8 h-8 text-text-muted mx-auto mb-2 opacity-30" />
              <p className="text-sm text-text-muted">Event stream paused</p>
              <p className="text-xs text-text-muted mt-1">Events are still being collected in the background</p>
            </div>
          ) : events.length === 0 ? (
            <div className="text-center py-12">
              <Radio className="w-8 h-8 text-text-muted mx-auto mb-2 opacity-30" />
              <p className="text-sm text-text-muted">No events found</p>
            </div>
          ) : (
            <EventFeed events={events} maxItems={PAGE_SIZE} autoAI={autoAI} />
          )}
        </div>
      </motion.div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-text-muted">
            Showing {offset + 1}–{Math.min(offset + PAGE_SIZE, total)} of {total}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
              disabled={offset === 0}
              className="btn btn-secondary p-2 disabled:opacity-30"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <span className="text-text-secondary">
              Page {currentPage} of {totalPages}
            </span>
            <button
              onClick={() => setOffset(offset + PAGE_SIZE)}
              disabled={currentPage >= totalPages}
              className="btn btn-secondary p-2 disabled:opacity-30"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
