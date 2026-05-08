import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import StatusDot from './StatusDot.svelte';

describe('StatusDot', () => {
  it('reflects status prop onto data-status attribute', () => {
    const { container } = render(StatusDot, { props: { status: 'up' } });
    expect(container.querySelector('span')).toHaveAttribute('data-status', 'up');
  });

  it('exposes a screen-reader label including the status', () => {
    const { container } = render(StatusDot, { props: { status: 'down' } });
    expect(container.querySelector('span')).toHaveAttribute('aria-label', expect.stringContaining('down'));
  });

  it('applies pulse class when pulse=true', () => {
    const { container } = render(StatusDot, { props: { status: 'up', pulse: true } });
    expect(container.querySelector('span')).toHaveClass('animate-pulse-dot');
  });
});
