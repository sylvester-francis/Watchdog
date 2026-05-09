import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import StatBlock from './StatBlock.svelte';

describe('StatBlock', () => {
  it('renders the label and value', () => {
    render(StatBlock, { props: { label: 'Monitors up', value: 42 } });
    expect(screen.getByText('Monitors up')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('renders the delta when provided', () => {
    render(StatBlock, { props: { label: 'Latency', value: '14ms', delta: '−2ms' } });
    expect(screen.getByText('−2ms')).toBeInTheDocument();
  });

  it('does not render delta when not provided', () => {
    const { container } = render(StatBlock, { props: { label: 'Foo', value: 1 } });
    const spans = container.querySelectorAll('span');
    expect(spans.length).toBe(2);
  });

  it('renders delta with up direction class', () => {
    const { container } = render(StatBlock, { props: { label: 'L', value: 1, delta: '+5%', deltaDirection: 'up' } });
    const deltaEl = container.querySelector('[data-delta="up"]')!;
    expect(deltaEl).toBeInTheDocument();
    expect(deltaEl.className).toContain('text-status-up');
  });

  it('renders delta with down direction class', () => {
    const { container } = render(StatBlock, { props: { label: 'L', value: 1, delta: '-5%', deltaDirection: 'down' } });
    const deltaEl = container.querySelector('[data-delta="down"]')!;
    expect(deltaEl.className).toContain('text-destructive');
  });

  it('uses text-stat typography on the value', () => {
    const { container } = render(StatBlock, { props: { label: 'Foo', value: 42 } });
    const valueEl = Array.from(container.querySelectorAll('span')).find(s => s.textContent === '42')!;
    expect(valueEl.className).toContain('text-stat');
  });
});
