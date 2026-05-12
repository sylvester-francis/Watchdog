import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import StatsGrid from './StatsGrid.svelte';

const mkStats = (over: Partial<{ total_monitors: number; monitors_up: number; monitors_down: number; active_incidents: number; online_agents: number; total_agents: number }> = {}) => ({
  total_monitors: 10,
  monitors_up: 8,
  monitors_down: 2,
  active_incidents: 1,
  online_agents: 3,
  total_agents: 4,
  ...over,
});

describe('StatsGrid', () => {
  it('renders 4 stat cells with stat-key data attributes', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats(), uptimePercent: 99.5 } });
    expect(container.querySelector('[data-stat-key="monitors"]')).toBeInTheDocument();
    expect(container.querySelector('[data-stat-key="healthy"]')).toBeInTheDocument();
    expect(container.querySelector('[data-stat-key="down"]')).toBeInTheDocument();
    expect(container.querySelector('[data-stat-key="incidents"]')).toBeInTheDocument();
  });

  it('renders uptime percent on the Healthy cell when monitors exist', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats(), uptimePercent: 99.5 } });
    expect(container.textContent).toContain('99.5');
  });

  it('renders "No checks yet" on Healthy cell when no monitors exist', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats({ total_monitors: 0, monitors_up: 0 }), uptimePercent: 0 } });
    expect(container.textContent).toContain('No checks yet');
  });

  it('shows "Requires attention" hint when monitors are down', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats({ monitors_down: 3 }), uptimePercent: 90 } });
    expect(container.textContent).toContain('Requires attention');
  });
});
