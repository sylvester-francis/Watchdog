import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import ChartFrame from './ChartFrame.svelte';

describe('ChartFrame', () => {
  it('renders the title when provided', () => {
    render(ChartFrame, { props: { title: 'Latency p50' } });
    expect(screen.getByText('Latency p50')).toBeInTheDocument();
  });

  it('shows skeleton when loading is true', () => {
    const { container } = render(ChartFrame, { props: { loading: true } });
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
  });

  it('does not show skeleton when loading is false', () => {
    const { container } = render(ChartFrame);
    expect(container.querySelector('.animate-pulse')).not.toBeInTheDocument();
  });
});
