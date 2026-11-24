import { useEffect } from 'react';
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { Toaster, toast } from 'sonner';
import Layout from './components/Layout';
import DashboardPage from './pages/DashboardPage';
import ClustersPage from './pages/ClustersPage';
import SpokeDetailPage from './pages/SpokeDetailPage';
import ArgoPage from './pages/ArgoPage';
import ArgoAppDetailPage from './pages/ArgoAppDetailPage';
import EventsPage from './pages/EventsPage';
import SettingsPage from './pages/SettingsPage';
import NotFoundPage from './pages/NotFoundPage';
import { useWebSocket } from './hooks/useWebSocket';
import { useHub } from './hooks/useHub';
import { UNAUTHORIZED_EVENT } from './lib/api';

function App() {
  useWebSocket();
  useHub();
  const navigate = useNavigate();

  // A rejected token sends the user to Settings to enter a valid one.
  useEffect(() => {
    const onUnauthorized = () => {
      toast.error('Authentication required. Enter your API token in Settings.');
      navigate('/settings');
    };
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, [navigate]);

  return (
    <>
      <Toaster theme="dark" position="bottom-right" richColors />
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Navigate to="/dashboard" />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="clusters" element={<ClustersPage />} />
          <Route path="clusters/:clusterName" element={<SpokeDetailPage />} />
          <Route path="argocd" element={<ArgoPage />} />
          <Route path="argocd/:name" element={<ArgoAppDetailPage />} />
          <Route path="events" element={<EventsPage />} />
          <Route path="settings" element={<SettingsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </>
  );
}

export default App;
