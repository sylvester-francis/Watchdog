import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import StatusBadge from './StatusBadge.svelte';

describe('StatusBadge', () => {
  it('reflects status prop onto data-status attribute', () => {
    const { container } = render(StatusBadge, { props: { status: 'up' } });
    expect(container.querySelector('span')).toHaveAttribute('data-status', 'up');
  });

  it('renders the up glyph when status=up', () => {
    const { container } = render(StatusBadge, { props: { status: 'up' } });
    expect(container.textContent).toContain('●');
  });

  it('renders the down glyph when status=down', () => {
    const { container } = render(StatusBadge, { props: { status: 'down' } });
    expect(container.textContent).toContain('✕');
  });

  it('renders the warn glyph when status=warn', () => {
    const { container } = render(StatusBadge, { props: { status: 'warn' } });
    expect(container.textContent).toContain('⚠');
  });

  it('has the ui-status-badge class for styling hook', () => {
    const { container } = render(StatusBadge, { props: { status: 'up' } });
    expect(container.querySelector('span')!.className).toContain('ui-status-badge');
  });
});
