import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Radio, Pause, Play, Trash2, Sparkles } from 'lucide-react';
import { useDashboardStore } from '../lib/store';
import { eventsAPI } from '../lib/api';
import EventFeed from '../components/EventFeed';

export default function EventsPage() {
  const { events, setEvents } = useDashboardStore();
  const [paused, setPaused] = useState(false);
  const [filter, setFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [autoAI, setAutoAI] = useState(false);

  useEffect(() => {
    eventsAPI.getRecent().then(setEvents).catch(console.error);
  }, [setEvents]);

  const resourceTypes = ['all', ...new Set(events.map(e => e.resource_type))];

  const filtered = events
    .filter(e => {
      if (typeFilter !== 'all' && e.resource_type !== typeFilter) return false;
      if (filter && !e.name.toLowerCase().includes(filter.toLowerCase()) &&
          !e.resource_type.toLowerCase().includes(filter.toLowerCase()) &&
          !e.summary?.toLowerCase().includes(filter.toLowerCase())) return false;
      return true;
    });

  const displayEvents = paused ? [] : filtered;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
            <Radio className="w-6 h-6 text-primary-400" />
            Live Events
          </h1>
          <p className="text-sm text-text-muted mt-1">
            Real-time chronological event log of cluster and ZTP resource changes
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* Auto-AI toggle */}
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
            onClick={() => setEvents([])}
            className="btn btn-ghost p-2"
            title="Clear events"
          >
            <Trash2 className="w-4 h-4" />
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

      {/* Filters */}
      <div className="flex items-center gap-3">
        <input
          type="text"
          placeholder="Filter events..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="flex-1 max-w-sm px-4 py-2 bg-surface-raised border border-border-default rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
        <select
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          className="px-3 py-2 bg-surface-raised border border-border-default rounded-lg text-sm text-text-secondary focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          {resourceTypes.map(t => (
            <option key={t} value={t}>{t === 'all' ? 'All Resources' : t}</option>
          ))}
        </select>
        <div className="flex items-center gap-2 text-xs text-text-muted">
          <motion.div
            className="w-2 h-2 rounded-full bg-emerald-400"
            animate={!paused ? { opacity: [1, 0.3, 1] } : {}}
            transition={{ duration: 1.5, repeat: Infinity }}
          />
          {paused ? 'Paused' : `${filtered.length} events`}
        </div>
      </div>

      {/* Event stream */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="card"
      >
        <div className="p-2 max-h-[calc(100vh-280px)] overflow-y-auto">
          {paused ? (
            <div className="text-center py-12">
              <Pause className="w-8 h-8 text-text-muted mx-auto mb-2 opacity-30" />
              <p className="text-sm text-text-muted">Event stream paused</p>
              <p className="text-xs text-text-muted mt-1">Events are still being collected in the background</p>
            </div>
          ) : (
            <EventFeed events={displayEvents} maxItems={200} autoAI={autoAI} />
          )}
        </div>
      </motion.div>
    </div>
  );
}
