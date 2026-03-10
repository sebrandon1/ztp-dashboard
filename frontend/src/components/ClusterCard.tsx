import { useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Server, Clock, ChevronRight } from 'lucide-react';
import StatusBadge from './StatusBadge';
import type { ManagedClusterInfo } from '../types/api';

function timeAgo(timestamp: string) {
  const diff = Date.now() - new Date(timestamp).getTime();
  const days = Math.floor(diff / 86400000);
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diff / 3600000);
  if (hours > 0) return `${hours}h ago`;
  const minutes = Math.floor(diff / 60000);
  return `${minutes}m ago`;
}

interface ClusterCardProps {
  cluster: ManagedClusterInfo;
  index: number;
}

export default function ClusterCard({ cluster, index }: ClusterCardProps) {
  const navigate = useNavigate();
  const age = useMemo(() => timeAgo(cluster.creationTimestamp), [cluster.creationTimestamp]);

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, delay: index * 0.05 }}
      onClick={() => navigate(`/clusters/${cluster.name}`)}
      className="card p-5 cursor-pointer group hover:border-primary-500/30 hover:shadow-lg hover:shadow-primary-500/5 transition-all duration-300"
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${
            cluster.available === 'True'
              ? 'bg-emerald-500/10 text-emerald-400'
              : 'bg-red-500/10 text-red-400'
          }`}>
            <Server className="w-5 h-5" />
          </div>
          <div>
            <h3 className="font-semibold text-text-primary group-hover:text-primary-400 transition-colors">
              {cluster.name}
            </h3>
            {cluster.openshiftVersion && (
              <p className="text-xs text-text-muted">OCP {cluster.openshiftVersion}</p>
            )}
          </div>
        </div>
        <ChevronRight className="w-4 h-4 text-text-muted group-hover:text-primary-400 transition-colors" />
      </div>

      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <StatusBadge status={cluster.available === 'True' ? 'True' : 'False'} />
          {cluster.joined === 'True' && (
            <span className="badge badge-info">Joined</span>
          )}
        </div>
        <div className="flex items-center gap-1 text-xs text-text-muted">
          <Clock className="w-3 h-3" />
          {age}
        </div>
      </div>
    </motion.div>
  );
}
