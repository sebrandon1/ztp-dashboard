import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusBadge from './StatusBadge';

describe('StatusBadge', () => {
  it('maps known statuses to friendly labels', () => {
    render(<StatusBadge status="True" />);
    expect(screen.getByText('Available')).toBeInTheDocument();
  });

  it('renders the non-compliant label', () => {
    render(<StatusBadge status="NonCompliant" />);
    expect(screen.getByText('Non-Compliant')).toBeInTheDocument();
  });

  it('shows the animated indicator for InProgress', () => {
    render(<StatusBadge status="InProgress" />);
    expect(screen.getByText('In Progress')).toBeInTheDocument();
  });

  it('falls back to the raw status for unknown values', () => {
    render(<StatusBadge status="SomethingElse" />);
    expect(screen.getByText('SomethingElse')).toBeInTheDocument();
  });

  it('applies the large size classes when requested', () => {
    render(<StatusBadge status="Healthy" size="md" />);
    expect(screen.getByText('Healthy')).toHaveClass('px-3', 'py-1', 'text-sm');
  });
});
