import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Sparkline from './Sparkline.svelte';

describe('Sparkline', () => {
  it('renders an svg element', () => {
    const { container } = render(Sparkline, { props: { data: [1, 2, 3] } });
    expect(container.querySelector('svg')).toBeInTheDocument();
  });

  it('renders a polyline with N-1 segments for N data points', () => {
    const { container } = render(Sparkline, { props: { data: [1, 2, 3, 4] } });
    const points = container.querySelector('polyline')?.getAttribute('points') ?? '';
    expect(points.split(' ').length).toBe(4);
  });

  it('renders empty polyline when data is empty', () => {
    const { container } = render(Sparkline, { props: { data: [] } });
    expect(container.querySelector('polyline')?.getAttribute('points')).toBe('');
  });
});
