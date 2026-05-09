import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import IncidentRow from './IncidentRow.svelte';

const mkIncident = (over: Record<string, unknown> = {}) => ({
  id: 'i1',
  monitor_id: 'm1',
  monitor_name: 'API',
  status: 'open',
  started_at: new Date().toISOString(),
  ttr_seconds: null,
  ...over,
} as never);

const mkMonitor = () => ({ id: 'm1', name: 'API', type: 'http', target: 'https://example.com', status: 'down', total: 1, uptimeUp: 0, uptimeDown: 1, latencies: [], interval_seconds: 60 } as never);

describe('IncidentRow', () => {
  it('renders a StatusDot for the incident status', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident(), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.querySelector('.ui-status-dot')).toBeInTheDocument();
  });

  it('StatusDot has data-status="down" for open incidents', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ status: 'open' }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.querySelector('.ui-status-dot[data-status="down"]')).toBeInTheDocument();
  });

  it('renders a Pill for the incident status badge', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ status: 'open' }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.querySelector('.ui-pill[data-tone="down"]')).toBeInTheDocument();
  });

  it('Pill tone="up" for resolved incidents', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ status: 'resolved' }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.querySelector('.ui-pill[data-tone="up"]')).toBeInTheDocument();
  });

  it('renders a Pill for the monitor type chip', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident(), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    const pills = container.querySelectorAll('.ui-pill');
    expect(pills.length).toBe(2);
  });
});
