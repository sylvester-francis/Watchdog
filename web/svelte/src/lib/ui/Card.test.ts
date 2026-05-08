import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Card from './Card.svelte';

describe('Card', () => {
  it('renders a card container', () => {
    const { container } = render(Card);
    expect(container.querySelector('div')).toBeInTheDocument();
  });

  it('reflects variant prop onto data-variant attribute', () => {
    const { container } = render(Card, { props: { variant: 'elevated' } });
    expect(container.querySelector('div')).toHaveAttribute('data-variant', 'elevated');
  });

  it('defaults variant to "default"', () => {
    const { container } = render(Card);
    expect(container.querySelector('div')).toHaveAttribute('data-variant', 'default');
  });
});
