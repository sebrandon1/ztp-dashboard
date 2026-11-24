import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import EventsPage from './EventsPage';
import { useDashboardStore } from '../lib/store';

vi.mock('../lib/api', () => ({
  UNAUTHORIZED_EVENT: 'ztp:unauthorized',
  getAuthToken: () => '',
  storeAuthToken: () => {},
  eventsAPI: {
    query: vi.fn().mockResolvedValue({ events: [], total: 0 }),
    getStats: vi.fn().mockResolvedValue({ bySeverity: {}, byResourceType: {}, total: 0 }),
    exportEvents: vi.fn(),
  },
  aiAPI: { getStatus: vi.fn(), diagnose: vi.fn(), getModels: vi.fn() },
}));

function renderPage() {
  return render(
    <MemoryRouter>
      <EventsPage />
    </MemoryRouter>,
  );
}

describe('EventsPage Auto-AI toggle', () => {
  afterEach(() => {
    useDashboardStore.setState({ aiConnected: null });
  });

  it('is disabled with a tooltip when ollama is offline', () => {
    useDashboardStore.setState({ aiConnected: false });
    renderPage();

    const toggle = screen.getByRole('button', { name: /auto-ai/i });
    expect(toggle).toBeDisabled();
    expect(toggle).toHaveAttribute('title', expect.stringMatching(/AI is offline/));
  });

  it('is enabled when ollama is connected', () => {
    useDashboardStore.setState({ aiConnected: true });
    renderPage();

    expect(screen.getByRole('button', { name: /auto-ai/i })).toBeEnabled();
  });

  it('is enabled while the AI status is still unknown', () => {
    useDashboardStore.setState({ aiConnected: null });
    renderPage();

    expect(screen.getByRole('button', { name: /auto-ai/i })).toBeEnabled();
  });
});
