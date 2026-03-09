import { motion } from 'framer-motion';
import type { PolicyInfo } from '../types/api';

interface PolicyComplianceBarProps {
  policies: PolicyInfo[];
}

export default function PolicyComplianceBar({ policies }: PolicyComplianceBarProps) {
  const compliant = policies.filter(p => p.data?.compliant === 'Compliant').length;
  const nonCompliant = policies.filter(p => p.data?.compliant === 'NonCompliant').length;
  const unknown = policies.length - compliant - nonCompliant;
  const total = policies.length;

  if (total === 0) {
    return <p className="text-xs text-text-muted">No policies found</p>;
  }

  const compliantPct = (compliant / total) * 100;
  const nonCompliantPct = (nonCompliant / total) * 100;

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-xs">
        <span className="text-text-secondary">Policy Compliance</span>
        <span className="text-text-muted">{compliant}/{total} compliant</span>
      </div>
      <div className="h-2 bg-surface-overlay rounded-full overflow-hidden flex">
        {compliantPct > 0 && (
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: `${compliantPct}%` }}
            transition={{ duration: 0.5 }}
            className="bg-emerald-500 h-full"
          />
        )}
        {nonCompliantPct > 0 && (
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: `${nonCompliantPct}%` }}
            transition={{ duration: 0.5, delay: 0.1 }}
            className="bg-red-500 h-full"
          />
        )}
        {unknown > 0 && (
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: `${(unknown / total) * 100}%` }}
            transition={{ duration: 0.5, delay: 0.2 }}
            className="bg-slate-500 h-full"
          />
        )}
      </div>
      <div className="flex gap-4 text-xs">
        <span className="flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-emerald-500" />
          <span className="text-text-muted">{compliant} Compliant</span>
        </span>
        <span className="flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-red-500" />
          <span className="text-text-muted">{nonCompliant} Non-Compliant</span>
        </span>
        {unknown > 0 && (
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-slate-500" />
            <span className="text-text-muted">{unknown} Unknown</span>
          </span>
        )}
      </div>
    </div>
  );
}
