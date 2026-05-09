import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import IncidentCard from './IncidentCard.svelte';

const mkIncident = (over: Record<string, unknown> = {}) => ({
  id: 'i1',
  monitor_id: 'm1',
  status: 'open',
  started_at: new Date().toISOString(),
  ...over,
});

const mkMonitor = () => ({
  id: 'm1',
  name: 'API',
  type: 'http',
  target: '',
  status: 'down',
  total: 1,
  uptimeUp: 0,
  uptimeDown: 1,
  latencies: [],
  interval_seconds: 60,
});

describe('IncidentCard', () => {
  it('renders a Pill per displayed incident', () => {
    const incidents = [mkIncident(), mkIncident({ id: 'i2', status: 'acknowledged' })] as never;
    const monitors = new Map([['m1', mkMonitor()]]) as never;
    const { container } = render(IncidentCard, { props: { incidents, monitors } });
    expect(container.querySelectorAll('.ui-pill').length).toBe(2);
  });

  it('Pill tone="down" for open incidents', () => {
    const incidents = [mkIncident({ status: 'open' })] as never;
    const monitors = new Map([['m1', mkMonitor()]]) as never;
    const { container } = render(IncidentCard, { props: { incidents, monitors } });
    expect(container.querySelector('.ui-pill[data-tone="down"]')).toBeInTheDocument();
  });

  it('Pill tone="warn" for acknowledged incidents', () => {
    const incidents = [mkIncident({ status: 'acknowledged' })] as never;
    const monitors = new Map([['m1', mkMonitor()]]) as never;
    const { container } = render(IncidentCard, { props: { incidents, monitors } });
    expect(container.querySelector('.ui-pill[data-tone="warn"]')).toBeInTheDocument();
  });

  it('renders empty state when no incidents', () => {
    const { container } = render(IncidentCard, { props: { incidents: [], monitors: new Map() as never } });
    expect(container.textContent).toContain('No active incidents');
  });
});
