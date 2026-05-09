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
  it('renders 4 stat tiles', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats(), uptimePercent: 99.5 } });
    expect(container.querySelectorAll('[data-accent]').length).toBe(4);
  });

  it('applies accent="warn" to the Down tile when monitors_down > 0', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats({ monitors_down: 3 }), uptimePercent: 90 } });
    const tiles = Array.from(container.querySelectorAll('[data-accent]'));
    const downTile = tiles.find(el => el.querySelector('[data-stat-key="down"]'));
    expect(downTile?.getAttribute('data-accent')).toBe('warn');
  });

  it('applies accent="up" to the Healthy tile', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats(), uptimePercent: 99.5 } });
    const tiles = Array.from(container.querySelectorAll('[data-accent]'));
    const healthyTile = tiles.find(el => el.querySelector('[data-stat-key="healthy"]'));
    expect(healthyTile?.getAttribute('data-accent')).toBe('up');
  });

  it('renders uptime percent on the Healthy tile when monitors exist', () => {
    const { container } = render(StatsGrid, { props: { stats: mkStats(), uptimePercent: 99.5 } });
    expect(container.textContent).toContain('99.5');
  });
});
