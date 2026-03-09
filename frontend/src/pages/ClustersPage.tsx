import { useState } from 'react';
import { motion } from 'framer-motion';
import { Search, Server } from 'lucide-react';
import { useDashboardStore } from '../lib/store';
import ClusterCard from '../components/ClusterCard';

export default function ClustersPage() {
  const { clusters } = useDashboardStore();
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<'all' | 'available' | 'unavailable'>('all');

  const filtered = clusters
    .filter(c => c.name.toLowerCase().includes(search.toLowerCase()))
    .filter(c => {
      if (filter === 'available') return c.available === 'True';
      if (filter === 'unavailable') return c.available !== 'True';
      return true;
    });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
            <Server className="w-6 h-6 text-primary-400" />
            Managed Clusters
          </h1>
          <p className="text-sm text-text-muted mt-1">{clusters.length} spoke clusters managed by this hub</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <input
            type="text"
            placeholder="Search clusters..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-2 bg-surface-raised border border-border-default rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>
        <div className="flex items-center bg-surface-raised rounded-lg border border-border-default p-0.5">
          {(['all', 'available', 'unavailable'] as const).map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1.5 rounded-md text-xs font-medium transition-all capitalize ${
                filter === f
                  ? 'bg-primary-600 text-white shadow-sm'
                  : 'text-text-muted hover:text-text-secondary'
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {/* Cluster grid */}
      {filtered.length === 0 ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-center py-20"
        >
          <Server className="w-12 h-12 text-text-muted mx-auto mb-3 opacity-30" />
          <p className="text-sm text-text-muted">
            {search ? 'No clusters match your search' : 'No managed clusters found'}
          </p>
        </motion.div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((cluster, i) => (
            <ClusterCard key={cluster.name} cluster={cluster} index={i} />
          ))}
        </div>
      )}
    </div>
  );
}
