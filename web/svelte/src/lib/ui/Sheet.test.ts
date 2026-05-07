import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Sheet from './Sheet.svelte';

describe('Sheet', () => {
  it('does not render dialog when closed', () => {
    const { container } = render(Sheet);
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open', () => {
    const { container } = render(Sheet, { props: { open: true } });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('reflects side prop onto data-side attribute', () => {
    const { container } = render(Sheet, { props: { open: true, side: 'left' } });
    expect(container.querySelector('[role="dialog"]')).toHaveAttribute('data-side', 'left');
  });
});
