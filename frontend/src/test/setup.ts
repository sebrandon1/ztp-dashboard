import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// Testing Library's automatic cleanup relies on a global afterEach, which is
// not registered when Vitest globals are disabled.
afterEach(() => {
  cleanup();
});
