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

  it('renders an area-filled polygon when fill=true', () => {
    const { container } = render(Sparkline, { props: { data: [1, 2, 3], fill: true } });
    expect(container.querySelector('polygon')).toBeInTheDocument();
  });

  it('does not render polygon when fill is default (false)', () => {
    const { container } = render(Sparkline, { props: { data: [1, 2, 3] } });
    expect(container.querySelector('polygon')).not.toBeInTheDocument();
  });

  it('reflects status prop onto data-status attribute on svg', () => {
    const { container } = render(Sparkline, { props: { data: [1, 2, 3], status: 'down' } });
    expect(container.querySelector('svg')!.getAttribute('data-status')).toBe('down');
  });

  it('uses status-coded stroke when status is set', () => {
    const { container } = render(Sparkline, { props: { data: [1, 2, 3], status: 'down' } });
    expect(container.querySelector('polyline')!.getAttribute('stroke')).toBe('var(--status-down)');
  });
});
