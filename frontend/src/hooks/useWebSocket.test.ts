import { describe, expect, it } from 'vitest';
import { buildWebSocketURL, reconnectDelay } from './useWebSocket';

describe('reconnectDelay', () => {
  it('starts at one second and doubles each attempt', () => {
    expect(reconnectDelay(0)).toBe(1000);
    expect(reconnectDelay(1)).toBe(2000);
    expect(reconnectDelay(2)).toBe(4000);
    expect(reconnectDelay(3)).toBe(8000);
  });

  it('caps the delay at 30 seconds', () => {
    expect(reconnectDelay(5)).toBe(30000);
    expect(reconnectDelay(10)).toBe(30000);
    expect(reconnectDelay(100)).toBe(30000);
  });
});

describe('buildWebSocketURL', () => {
  it('omits the token query when no token is configured', () => {
    expect(buildWebSocketURL('')).toBe(`ws://${window.location.host}/ws/watch`);
  });

  it('appends an encoded token query parameter', () => {
    expect(buildWebSocketURL('a b/c')).toBe(
      `ws://${window.location.host}/ws/watch?token=${encodeURIComponent('a b/c')}`,
    );
  });
});
