import { Outlet, NavLink } from 'react-router-dom';
import { LayoutDashboard, Server, Radio, Settings, Wifi, WifiOff, Zap, GitBranch } from 'lucide-react';
import { useDashboardStore } from '../lib/store';

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/clusters', icon: Server, label: 'Clusters' },
  { to: '/argocd', icon: GitBranch, label: 'ArgoCD' },
  { to: '/events', icon: Radio, label: 'Events' },
  { to: '/settings', icon: Settings, label: 'Settings' },
];

export default function Layout() {
  const { wsConnected, hubStatus } = useDashboardStore();

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="w-64 bg-surface-raised border-r border-border-subtle flex flex-col shrink-0">
        {/* Logo */}
        <div className="h-16 flex items-center gap-3 px-6 border-b border-border-subtle">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
            <Zap className="w-4 h-4 text-white" />
          </div>
          <div>
            <h1 className="text-sm font-bold text-text-primary tracking-tight">ZTP Dashboard</h1>
            <p className="text-[10px] text-text-muted uppercase tracking-widest">Hub Manager</p>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                  isActive
                    ? 'bg-primary-600/10 text-primary-400 shadow-sm'
                    : 'text-text-secondary hover:text-text-primary hover:bg-surface-overlay/50'
                }`
              }
            >
              <Icon className="w-4 h-4" />
              {label}
            </NavLink>
          ))}
        </nav>

        {/* Status footer */}
        <div className="px-4 py-3 border-t border-border-subtle space-y-2">
          <div className="flex items-center justify-between text-xs">
            <span className="text-text-muted">Hub</span>
            <span className={`flex items-center gap-1.5 ${hubStatus?.connected ? 'text-emerald-400' : 'text-red-400'}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${hubStatus?.connected ? 'bg-emerald-400' : 'bg-red-400'}`} />
              {hubStatus?.connected ? hubStatus.serverVersion || 'Connected' : 'Disconnected'}
            </span>
          </div>
          <div className="flex items-center justify-between text-xs">
            <span className="text-text-muted">WebSocket</span>
            <span className={`flex items-center gap-1.5 ${wsConnected ? 'text-emerald-400' : 'text-amber-400'}`}>
              {wsConnected ? <Wifi className="w-3 h-3" /> : <WifiOff className="w-3 h-3" />}
              {wsConnected ? 'Live' : 'Reconnecting'}
            </span>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-y-auto">
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
