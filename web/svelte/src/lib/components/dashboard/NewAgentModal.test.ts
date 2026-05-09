import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$lib/api', () => ({
  agents: { createAgent: vi.fn() },
}));

import NewAgentModal from './NewAgentModal.svelte';

describe('NewAgentModal', () => {
  it('does not render dialog when open=false', () => {
    const { container } = render(NewAgentModal, { props: { open: false, onClose: () => {}, onCreated: () => {} } });
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open=true', () => {
    const { container } = render(NewAgentModal, { props: { open: true, onClose: () => {}, onCreated: () => {} } });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('renders a primary submit button', () => {
    const { container } = render(NewAgentModal, { props: { open: true, onClose: () => {}, onCreated: () => {} } });
    expect(container.querySelector('button[data-variant="primary"]')).toBeInTheDocument();
  });

  it('renders an outline cancel button', () => {
    const { container } = render(NewAgentModal, { props: { open: true, onClose: () => {}, onCreated: () => {} } });
    expect(container.querySelector('button[data-variant="outline"]')).toBeInTheDocument();
  });

  it('renders a header X close button that calls onClose when clicked', async () => {
    let closed = false;
    const { container } = render(NewAgentModal, { props: { open: true, onClose: () => { closed = true; }, onCreated: () => {} } });
    const xBtn = container.querySelector('button[aria-label="Close"]') as HTMLButtonElement | null;
    expect(xBtn).toBeInTheDocument();
    xBtn!.click();
    expect(closed).toBe(true);
  });
});
