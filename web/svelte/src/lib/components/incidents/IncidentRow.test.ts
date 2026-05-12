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
  it('renders the incident status as uppercase text', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ status: 'open' }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.textContent?.toLowerCase()).toContain('open');
  });

  it('shows the monitor name', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident(), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.textContent).toContain('API');
  });

  it('shows the monitor type metadata', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident(), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.textContent?.toLowerCase()).toContain('http');
  });

  it('shows TTR for resolved incidents', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ status: 'resolved', ttr_seconds: 90 }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {} },
    });
    expect(container.textContent).toContain('TTR');
  });

  it('hides ack/resolve buttons when canWrite is false', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ status: 'open' }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {}, canWrite: false },
    });
    expect(container.textContent).not.toContain('Ack');
    expect(container.textContent).not.toContain('Resolve');
  });
});
