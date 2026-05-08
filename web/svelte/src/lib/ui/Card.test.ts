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

  it('applies default variant classes (bg-card, border)', () => {
    const { container } = render(Card);
    const div = container.querySelector('div')!;
    expect(div.className).toContain('bg-card');
    expect(div.className).toContain('border-border');
  });

  it('applies elevated variant (shadow + bg-card-elevated)', () => {
    const { container } = render(Card, { props: { variant: 'elevated' } });
    const div = container.querySelector('div')!;
    expect(div.className).toContain('bg-card-elevated');
    expect(div.className).toContain('shadow');
  });

  it('applies accent variant (border-accent)', () => {
    const { container } = render(Card, { props: { variant: 'accent' } });
    expect(container.querySelector('div')!.className).toContain('border-accent');
  });
});
