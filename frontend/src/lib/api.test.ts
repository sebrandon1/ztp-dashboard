import { afterEach, describe, expect, it, vi } from 'vitest';
import { UNAUTHORIZED_EVENT, eventsExportPath, getAuthToken, storeAuthToken } from './api';

describe('auth token storage', () => {
  afterEach(() => {
    window.localStorage.clear();
    storeAuthToken('');
    vi.restoreAllMocks();
  });

  it('round-trips a token through localStorage', () => {
    storeAuthToken('s3cr3t');
    expect(window.localStorage.getItem('ztp-auth-token')).toBe('s3cr3t');
    expect(getAuthToken()).toBe('s3cr3t');
  });

  it('clears the token when given an empty string', () => {
    storeAuthToken('s3cr3t');
    storeAuthToken('');
    expect(window.localStorage.getItem('ztp-auth-token')).toBeNull();
    expect(getAuthToken()).toBe('');
  });

  it('falls back to the in-memory token when localStorage throws', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('unavailable');
    });
    storeAuthToken('memory-only');
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('unavailable');
    });
    expect(getAuthToken()).toBe('memory-only');
  });

  it('exposes a stable unauthorized event name', () => {
    expect(UNAUTHORIZED_EVENT).toBe('ztp:unauthorized');
  });
});

describe('eventsExportPath', () => {
  it('includes filters and the format', () => {
    expect(eventsExportPath({ q: 'cluster', severity: 'bad' }, 'csv')).toBe(
      '/api/events/export?q=cluster&severity=bad&format=csv',
    );
  });

  it('works without filters', () => {
    expect(eventsExportPath(undefined, 'json')).toBe('/api/events/export?format=json');
  });
});
