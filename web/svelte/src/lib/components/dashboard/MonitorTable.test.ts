import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import { Activity } from 'lucide-svelte';
import MonitorTable from './MonitorTable.svelte';

const mkMonitor = (over: Record<string, unknown> = {}) => ({
  id: 'm1',
  name: 'API',
  type: 'http',
  target: 'https://api.example.com',
  status: 'up',
  total: 100,
  uptimeUp: 95,
  uptimeDown: 5,
  latencies: [10, 12, 11],
  interval_seconds: 60,
  ...over,
});

describe('MonitorTable', () => {
  it('renders one StatusDot per monitor', () => {
    const monitors = [mkMonitor(), mkMonitor({ id: 'm2', status: 'down' })];
    const { container } = render(MonitorTable, { props: { monitors, title: 'Services', icon: Activity, variant: 'service' } });
    expect(container.querySelectorAll('.ui-status-dot').length).toBe(2);
  });

  it('renders one Pill per monitor for the type chip', () => {
    const monitors = [mkMonitor(), mkMonitor({ id: 'm2' })];
    const { container } = render(MonitorTable, { props: { monitors, title: 'Services', icon: Activity, variant: 'service' } });
    expect(container.querySelectorAll('.ui-pill').length).toBe(2);
  });

  it('marks down monitors with data-status="down" on the dot', () => {
    const monitors = [mkMonitor({ status: 'down' })];
    const { container } = render(MonitorTable, { props: { monitors, title: 'Services', icon: Activity, variant: 'service' } });
    expect(container.querySelector('.ui-status-dot[data-status="down"]')).toBeInTheDocument();
  });

  it('renders an svg sparkline when latencies exist (service variant)', () => {
    const monitors = [mkMonitor({ latencies: [10, 12, 11, 13] })];
    const { container } = render(MonitorTable, { props: { monitors, title: 'Services', icon: Activity, variant: 'service' } });
    expect(container.querySelector('svg polyline')).toBeInTheDocument();
  });
});
