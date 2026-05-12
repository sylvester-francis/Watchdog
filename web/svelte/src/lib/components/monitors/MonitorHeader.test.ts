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
  it('shows the monitor type and status in the meta row', () => {
    const { container } = render(MonitorHeader, { props: { monitor: mkMonitor() } });
    const meta = container.querySelector('header > div > div');
    expect(meta?.textContent).toContain('Operational');
    expect(meta?.textContent).toContain('http');
  });

  it('shows the monitor name and target', () => {
    const { getByText } = render(MonitorHeader, { props: { monitor: mkMonitor() } });
    expect(getByText('My API')).toBeInTheDocument();
    expect(getByText('https://example.com')).toBeInTheDocument();
  });

  it('renders an Edit button when onEdit is provided', () => {
    const { container } = render(MonitorHeader, { props: { monitor: mkMonitor(), onEdit: () => {} } });
    const btn = container.querySelector('button');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent?.trim()).toBe('Edit');
  });

  it('does not render an Edit button when onEdit is omitted', () => {
    const { container } = render(MonitorHeader, { props: { monitor: mkMonitor() } });
    expect(container.querySelector('button')).not.toBeInTheDocument();
  });
});
