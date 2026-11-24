import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import PolicyComplianceBar from './PolicyComplianceBar';
import type { PolicyInfo } from '../types/api';

function policy(name: string, compliant: string): PolicyInfo {
  return { name, status: compliant, data: { compliant } };
}

describe('PolicyComplianceBar', () => {
  it('shows an empty state when there are no policies', () => {
    render(<PolicyComplianceBar policies={[]} />);
    expect(screen.getByText('No policies found')).toBeInTheDocument();
  });

  it('summarizes compliance counts', () => {
    render(
      <PolicyComplianceBar
        policies={[
          policy('p1', 'Compliant'),
          policy('p2', 'Compliant'),
          policy('p3', 'NonCompliant'),
          policy('p4', 'Unknown'),
        ]}
      />,
    );

    expect(screen.getByText('2/4 compliant')).toBeInTheDocument();
    expect(screen.getByText('2 Compliant')).toBeInTheDocument();
    expect(screen.getByText('1 Non-Compliant')).toBeInTheDocument();
    expect(screen.getByText('1 Unknown')).toBeInTheDocument();
  });

  it('omits the unknown legend when nothing is unknown', () => {
    render(<PolicyComplianceBar policies={[policy('p1', 'Compliant')]} />);
    expect(screen.getByText('1/1 compliant')).toBeInTheDocument();
    expect(screen.queryByText(/Unknown$/)).not.toBeInTheDocument();
  });
});
