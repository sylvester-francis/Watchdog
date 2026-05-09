import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import FleetBanner from './FleetBanner.svelte';

const mkStats = (over: Record<string, number> = {}) => ({
  total_monitors: 10,
  monitors_up: 8,
  monitors_down: 2,
  active_incidents: 0,
  online_agents: 1,
  total_agents: 1,
  ...over,
});

describe('FleetBanner', () => {
  it('renders an "up" pill when monitors exist', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats(), uptimePercent: 90 } });
    expect(container.querySelector('.ui-pill[data-tone="up"]')).toBeInTheDocument();
  });

  it('renders a "down" pill when monitors_down > 0', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats({ monitors_down: 3 }), uptimePercent: 70 } });
    expect(container.querySelector('.ui-pill[data-tone="down"]')).toBeInTheDocument();
  });

  it('does not render down pill when monitors_down is 0', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats({ monitors_down: 0 }), uptimePercent: 100 } });
    expect(container.querySelector('.ui-pill[data-tone="down"]')).not.toBeInTheDocument();
  });

  it('renders empty-state when no monitors exist', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats({ total_monitors: 0, monitors_up: 0, monitors_down: 0 }), uptimePercent: 0 } });
    expect(container.textContent).toContain('No monitors configured');
  });
});
