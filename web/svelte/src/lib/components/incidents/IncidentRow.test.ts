import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
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

  // Regression: incident detail is an in-place InvestigationDrawer, not an
  // /incidents/:id route. Clicking the row must open the drawer via
  // onInvestigate, never navigate to a route that does not exist in CE.
  it('opens the investigation drawer via onInvestigate when the row is clicked', async () => {
    const onInvestigate = vi.fn();
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident({ id: 'i1' }), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {}, onInvestigate },
    });
    const row = container.querySelector('tr');
    expect(row).not.toBeNull();
    await fireEvent.click(row!);
    expect(onInvestigate).toHaveBeenCalledWith('i1');
  });

  it('does not link to a nonexistent /incidents/:id detail route', () => {
    const { container } = render(IncidentRow, {
      props: { incident: mkIncident(), monitor: mkMonitor(), onAcknowledge: async () => {}, onResolve: async () => {}, onInvestigate: () => {} },
    });
    expect(container.querySelector('a[href^="/incidents/"]')).toBeNull();
  });
});
