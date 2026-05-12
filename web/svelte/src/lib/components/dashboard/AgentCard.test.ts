import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import AgentCard from './AgentCard.svelte';

const mkStats = (over: Record<string, number> = {}) => ({
  total_monitors: 0,
  monitors_up: 0,
  monitors_down: 0,
  active_incidents: 0,
  total_agents: 0,
  online_agents: 0,
  ...over,
});

describe('AgentCard', () => {
  it('renders a primary Button for "New Agent" CTA', () => {
    const { container } = render(AgentCard, { props: { agents: [], stats: mkStats(), onCreateAgent: () => {} } });
    const btn = container.querySelector('button[data-variant="primary"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent?.trim()).toBe('New Agent');
  });

  it('lists each agent name', () => {
    const agents = [
      { id: 'a', name: 'agent-one', status: 'online', last_seen_at: null },
      { id: 'b', name: 'agent-two', status: 'offline', last_seen_at: null },
    ] as never;
    const { container } = render(AgentCard, { props: { agents, stats: mkStats({ total_agents: 2, online_agents: 1 }), onCreateAgent: () => {} } });
    expect(container.textContent).toContain('agent-one');
    expect(container.textContent).toContain('agent-two');
  });

  it('shows the online/total count', () => {
    const agents = [{ id: 'a', name: 'A', status: 'online', last_seen_at: null }] as never;
    const { container } = render(AgentCard, { props: { agents, stats: mkStats({ total_agents: 1, online_agents: 1 }), onCreateAgent: () => {} } });
    expect(container.textContent).toContain('1/1 online');
  });

  it('calls onCreateAgent when New Agent button is clicked', async () => {
    let called = false;
    const { container } = render(AgentCard, { props: { agents: [], stats: mkStats(), onCreateAgent: () => { called = true; } } });
    await fireEvent.click(container.querySelector('button[data-variant="primary"]')!);
    expect(called).toBe(true);
  });
});
