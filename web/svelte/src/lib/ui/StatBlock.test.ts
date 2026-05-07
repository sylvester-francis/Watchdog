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
});
