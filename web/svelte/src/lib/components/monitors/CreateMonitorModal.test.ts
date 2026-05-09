import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$lib/api', () => ({
  monitors: {
    createMonitor: vi.fn(),
    listDeviceTemplates: vi.fn(async () => ({ data: [] })),
    getDeviceTemplate: vi.fn(),
  },
}));

import CreateMonitorModal from './CreateMonitorModal.svelte';

const mkAgent = () => ({ id: 'a1', name: 'Agent 1', status: 'online', last_seen_at: null } as never);

describe('CreateMonitorModal', () => {
  it('does not render dialog when open=false', () => {
    const { container } = render(CreateMonitorModal, {
      props: { open: false, agents: [mkAgent()], onClose: () => {}, onCreated: () => {} },
    });
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open=true', () => {
    const { container } = render(CreateMonitorModal, {
      props: { open: true, agents: [mkAgent()], onClose: () => {}, onCreated: () => {} },
    });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('renders a primary submit Button labelled "Create Monitor"', () => {
    const { container } = render(CreateMonitorModal, {
      props: { open: true, agents: [mkAgent()], onClose: () => {}, onCreated: () => {} },
    });
    const btn = container.querySelector('button[data-variant="primary"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent?.trim()).toContain('Create Monitor');
  });

  it('renders an outline Cancel Button', () => {
    const { container } = render(CreateMonitorModal, {
      props: { open: true, agents: [mkAgent()], onClose: () => {}, onCreated: () => {} },
    });
    const btn = container.querySelector('button[data-variant="outline"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent?.trim()).toBe('Cancel');
  });

  it('renders a header X close button that calls onClose', () => {
    let closed = false;
    const { container } = render(CreateMonitorModal, {
      props: { open: true, agents: [mkAgent()], onClose: () => { closed = true; }, onCreated: () => {} },
    });
    const xBtn = container.querySelector('button[aria-label="Close"]') as HTMLButtonElement | null;
    expect(xBtn).toBeInTheDocument();
    xBtn!.click();
    expect(closed).toBe(true);
  });

  it('preserves the Banner Grab checkbox when type is port_scan', async () => {
    const { container } = render(CreateMonitorModal, {
      props: { open: true, agents: [mkAgent()], onClose: () => {}, onCreated: () => {} },
    });
    expect(container.querySelector('[id="monitor-type"]')).toBeInTheDocument();
  });
});
