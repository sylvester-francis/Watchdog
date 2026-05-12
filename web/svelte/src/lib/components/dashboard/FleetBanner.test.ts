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
  it('shows "All systems operational" when no monitors are down', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats({ monitors_down: 0 }), uptimePercent: 100 } });
    expect(container.textContent).toContain('All systems operational');
  });

  it('shows degraded count when monitors are down', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats({ monitors_down: 3 }), uptimePercent: 70 } });
    expect(container.textContent).toContain('3 systems degraded');
  });

  it('renders the uptime percent when monitors exist', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats(), uptimePercent: 99.5 } });
    expect(container.textContent).toContain('99.5');
  });

  it('renders empty-state when no monitors exist', () => {
    const { container } = render(FleetBanner, { props: { stats: mkStats({ total_monitors: 0, monitors_up: 0, monitors_down: 0 }), uptimePercent: 0 } });
    expect(container.textContent).toContain('No monitors configured');
  });
});
