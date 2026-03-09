import { motion } from 'framer-motion';

interface StatusBadgeProps {
  status: string;
  size?: 'sm' | 'md';
}

const statusConfig: Record<string, { class: string; label?: string }> = {
  True: { class: 'badge-success', label: 'Available' },
  Completed: { class: 'badge-success' },
  Available: { class: 'badge-success' },
  Healthy: { class: 'badge-success' },
  Synced: { class: 'badge-success' },
  Compliant: { class: 'badge-success' },
  False: { class: 'badge-danger', label: 'Unavailable' },
  Error: { class: 'badge-danger' },
  Failed: { class: 'badge-danger' },
  Degraded: { class: 'badge-warning' },
  NonCompliant: { class: 'badge-warning', label: 'Non-Compliant' },
  InProgress: { class: 'badge-info', label: 'In Progress' },
  Pending: { class: 'badge-neutral' },
  Unknown: { class: 'badge-neutral' },
};

export default function StatusBadge({ status, size = 'sm' }: StatusBadgeProps) {
  const config = statusConfig[status] || { class: 'badge-neutral' };
  const label = config.label || status;
  const isAnimated = status === 'InProgress';

  return (
    <motion.span
      className={`badge ${config.class} ${size === 'md' ? 'px-3 py-1 text-sm' : ''}`}
      initial={{ scale: 0.9, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      transition={{ duration: 0.2 }}
    >
      {isAnimated && (
        <span className="relative flex h-2 w-2 mr-1.5">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
          <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500" />
        </span>
      )}
      {label}
    </motion.span>
  );
}
