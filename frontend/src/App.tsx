import { Routes, Route, Navigate } from 'react-router-dom';
import Layout from './components/Layout';
import DashboardPage from './pages/DashboardPage';
import ClustersPage from './pages/ClustersPage';
import SpokeDetailPage from './pages/SpokeDetailPage';
import EventsPage from './pages/EventsPage';
import SettingsPage from './pages/SettingsPage';
import { useWebSocket } from './hooks/useWebSocket';
import { useHub } from './hooks/useHub';

function App() {
  useWebSocket();
  useHub();

  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Navigate to="/dashboard" />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="clusters" element={<ClustersPage />} />
        <Route path="clusters/:clusterName" element={<SpokeDetailPage />} />
        <Route path="events" element={<EventsPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  );
}

export default App;
