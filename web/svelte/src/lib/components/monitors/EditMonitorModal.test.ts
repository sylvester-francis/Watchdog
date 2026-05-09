import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$lib/api', () => ({
  monitors: { updateMonitor: vi.fn() },
}));

import EditMonitorModal from './EditMonitorModal.svelte';

const mkMonitor = (over: Record<string, unknown> = {}) => ({
  id: 'm1',
  name: 'My API',
  type: 'http',
  target: 'https://api.example.com',
  agent_id: 'a1',
  interval_seconds: 60,
  timeout_seconds: 5,
  failure_threshold: 3,
  enabled: true,
  status: 'up',
  metadata: {},
  ...over,
} as never);

const mkAgent = () => ({ id: 'a1', name: 'Agent 1', status: 'online', last_seen_at: null } as never);

describe('EditMonitorModal', () => {
  it('does not render dialog when open=false', () => {
    const { container } = render(EditMonitorModal, {
      props: { open: false, monitor: mkMonitor(), agents: [mkAgent()], onClose: () => {}, onUpdated: () => {} },
    });
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open=true', () => {
    const { container } = render(EditMonitorModal, {
      props: { open: true, monitor: mkMonitor(), agents: [mkAgent()], onClose: () => {}, onUpdated: () => {} },
    });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('renders a primary submit Button labelled "Save Changes"', () => {
    const { container } = render(EditMonitorModal, {
      props: { open: true, monitor: mkMonitor(), agents: [mkAgent()], onClose: () => {}, onUpdated: () => {} },
    });
    const btn = container.querySelector('button[data-variant="primary"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent?.trim()).toContain('Save Changes');
  });

  it('renders an outline Cancel Button', () => {
    const { container } = render(EditMonitorModal, {
      props: { open: true, monitor: mkMonitor(), agents: [mkAgent()], onClose: () => {}, onUpdated: () => {} },
    });
    const btn = container.querySelector('button[data-variant="outline"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent?.trim()).toBe('Cancel');
  });

  it('renders a header X close button that calls onClose', () => {
    let closed = false;
    const { container } = render(EditMonitorModal, {
      props: { open: true, monitor: mkMonitor(), agents: [mkAgent()], onClose: () => { closed = true; }, onUpdated: () => {} },
    });
    const xBtn = container.querySelector('button[aria-label="Close"]') as HTMLButtonElement | null;
    expect(xBtn).toBeInTheDocument();
    xBtn!.click();
    expect(closed).toBe(true);
  });

  it('preserves the Enabled switch (role="switch")', () => {
    const { container } = render(EditMonitorModal, {
      props: { open: true, monitor: mkMonitor(), agents: [mkAgent()], onClose: () => {}, onUpdated: () => {} },
    });
    expect(container.querySelector('[role="switch"]')).toBeInTheDocument();
  });
});
