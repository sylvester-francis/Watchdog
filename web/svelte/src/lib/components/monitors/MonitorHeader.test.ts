import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import MonitorHeader from './MonitorHeader.svelte';

const mkMonitor = (over: Record<string, unknown> = {}) => ({
  id: 'm1',
  name: 'My API',
  type: 'http',
  target: 'https://example.com',
  status: 'up',
  interval_seconds: 60,
  timeout_seconds: 5,
  ...over,
} as never);

describe('MonitorHeader', () => {
  it('renders a Pill for the monitor type', () => {
    const { container } = render(MonitorHeader, { props: { monitor: mkMonitor() } });
    expect(container.querySelector('.ui-pill')).toBeInTheDocument();
    expect(container.querySelector('.ui-pill')?.textContent?.trim()).toContain('http');
  });

  it('renders a secondary Button when onEdit is provided', () => {
    const { container } = render(MonitorHeader, { props: { monitor: mkMonitor(), onEdit: () => {} } });
    const btn = container.querySelector('button[data-variant="secondary"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent).toContain('Edit');
  });

  it('does not render an Edit button when onEdit is omitted', () => {
    const { container } = render(MonitorHeader, { props: { monitor: mkMonitor() } });
    expect(container.querySelector('button[data-variant="secondary"]')).not.toBeInTheDocument();
  });
});
