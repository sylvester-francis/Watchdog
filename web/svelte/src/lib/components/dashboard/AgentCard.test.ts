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

  it('renders a Pill per agent', () => {
    const agents = [
      { id: 'a', name: 'A', status: 'online', last_seen_at: null },
      { id: 'b', name: 'B', status: 'offline', last_seen_at: null },
    ] as never;
    const { container } = render(AgentCard, { props: { agents, stats: mkStats({ total_agents: 2, online_agents: 1 }), onCreateAgent: () => {} } });
    expect(container.querySelectorAll('.ui-pill').length).toBe(2);
  });

  it('Pill has tone="up" when agent is online', () => {
    const agents = [{ id: 'a', name: 'A', status: 'online', last_seen_at: null }] as never;
    const { container } = render(AgentCard, { props: { agents, stats: mkStats({ total_agents: 1, online_agents: 1 }), onCreateAgent: () => {} } });
    expect(container.querySelector('.ui-pill[data-tone="up"]')).toBeInTheDocument();
  });

  it('calls onCreateAgent when New Agent button is clicked', async () => {
    let called = false;
    const { container } = render(AgentCard, { props: { agents: [], stats: mkStats(), onCreateAgent: () => { called = true; } } });
    await fireEvent.click(container.querySelector('button[data-variant="primary"]')!);
    expect(called).toBe(true);
  });
});
