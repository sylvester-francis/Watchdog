import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import InvestigationPanel from './InvestigationPanel.svelte';

const mkInvestigation = (over: Record<string, unknown> = {}) => ({
  recurrence_pattern: 'first_time',
  mttr_seconds: 300,
  agent_summary: { name: 'Agent 1', status: 'online' },
  previous_incidents: [],
  timeline: [],
  sibling_monitors: [],
  system_metrics: [],
  ...over,
} as never);

describe('InvestigationPanel', () => {
  it('renders a Pill for the recurrence pattern', () => {
    const { container } = render(InvestigationPanel, { props: { investigation: mkInvestigation() } });
    expect(container.querySelector('.ui-pill')).toBeInTheDocument();
  });

  it('renders Pills for sibling monitor types when siblings exist', () => {
    const sibling_monitors = [{ id: 'm2', name: 'Sib', type: 'http', target: '', status: 'up', has_incident: false }];
    const { container } = render(InvestigationPanel, { props: { investigation: mkInvestigation({ sibling_monitors }) } });
    expect(container.querySelectorAll('.ui-pill').length).toBeGreaterThan(1);
  });

  it('renders a StatusDot per sibling monitor', () => {
    const sibling_monitors = [
      { id: 'm2', name: 'Sib', type: 'http', target: '', status: 'up', has_incident: false },
      { id: 'm3', name: 'Sib2', type: 'tcp', target: '', status: 'down', has_incident: true },
    ];
    const { container } = render(InvestigationPanel, { props: { investigation: mkInvestigation({ sibling_monitors }) } });
    expect(container.querySelectorAll('.ui-status-dot').length).toBeGreaterThanOrEqual(2);
  });

  it('renders "incident" Pill (tone=down) when sibling has_incident=true', () => {
    const sibling_monitors = [{ id: 'm2', name: 'Sib', type: 'http', target: '', status: 'down', has_incident: true }];
    const { container } = render(InvestigationPanel, { props: { investigation: mkInvestigation({ sibling_monitors }) } });
    const downPills = container.querySelectorAll('.ui-pill[data-tone="down"]');
    expect(downPills.length).toBeGreaterThan(0);
  });
});
