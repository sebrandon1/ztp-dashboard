import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDashboardStore } from './store';
import type { WatchEvent, WSMessage } from '../types/api';

function makeEvent(overrides: Partial<WatchEvent> = {}): WatchEvent {
  return {
    event_type: 'ADDED',
    resource_type: 'ManagedCluster',
    name: 'cluster-a',
    namespace: '',
    summary: 'summary',
    severity: 'info',
    insight: '',
    ...overrides,
  };
}

beforeEach(() => {
  useDashboardStore.setState({
    wsConnected: false,
    hubStatus: null,
    aiConnected: null,
    clusters: [],
    events: [],
    updateCounter: 0,
  });
});

describe('dashboard store', () => {
  it('tracks websocket connection state', () => {
    useDashboardStore.getState().setWSConnected(true);
    expect(useDashboardStore.getState().wsConnected).toBe(true);
  });

  it('increments updateCounter for watched resource messages', () => {
    useDashboardStore.getState().handleWSMessage({ type: 'managed_cluster', payload: makeEvent() });
    useDashboardStore.getState().handleWSMessage({ type: 'policy', payload: makeEvent() });
    expect(useDashboardStore.getState().updateCounter).toBe(2);
  });

  it('prepends event messages without touching updateCounter', () => {
    useDashboardStore.getState().handleWSMessage({ type: 'event', payload: makeEvent({ name: 'first' }) });
    useDashboardStore.getState().handleWSMessage({ type: 'event', payload: makeEvent({ name: 'second' }) });

    const state = useDashboardStore.getState();
    expect(state.events.map((e) => e.name)).toEqual(['second', 'first']);
    expect(state.updateCounter).toBe(0);
  });

  it('caps stored events at 500', () => {
    const existing = Array.from({ length: 500 }, (_, i) => makeEvent({ name: `e${i}` }));
    useDashboardStore.setState({ events: existing });

    useDashboardStore.getState().handleWSMessage({ type: 'event', payload: makeEvent({ name: 'newest' }) });

    const events = useDashboardStore.getState().events;
    expect(events).toHaveLength(500);
    expect(events[0].name).toBe('newest');
  });

  it('caps addEvent at 500 as well', () => {
    useDashboardStore.setState({ events: Array.from({ length: 500 }, (_, i) => makeEvent({ name: `e${i}` })) });
    useDashboardStore.getState().addEvent(makeEvent({ name: 'latest' }));

    const events = useDashboardStore.getState().events;
    expect(events).toHaveLength(500);
    expect(events[0].name).toBe('latest');
  });

  it('logs server error messages', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const msg: WSMessage = { type: 'error', payload: makeEvent() };

    useDashboardStore.getState().handleWSMessage(msg);

    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });

  it('ignores unknown message types', () => {
    useDashboardStore.getState().handleWSMessage({ type: 'unrelated', payload: makeEvent() });
    const state = useDashboardStore.getState();
    expect(state.events).toHaveLength(0);
    expect(state.updateCounter).toBe(0);
  });
});
