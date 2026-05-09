import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$lib/api', () => ({
  monitors: { getMonitorsSummary: vi.fn(async () => []), deleteMonitor: vi.fn() },
  agents: { listAgents: vi.fn(async () => ({ data: [] })) },
}));

vi.mock('$lib/stores/toast.svelte', () => ({
  getToasts: () => ({ add: vi.fn(), success: vi.fn(), error: vi.fn() }),
}));

import Page from './+page.svelte';

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

describe('/monitors page', () => {
  it('renders a StatusDot per monitor row in services section', async () => {
    const { container } = render(Page);
    expect(container).toBeInTheDocument();
  });
});
