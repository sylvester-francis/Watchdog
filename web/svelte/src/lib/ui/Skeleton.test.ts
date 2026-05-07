import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Skeleton from './Skeleton.svelte';

describe('Skeleton', () => {
  it('renders a div', () => {
    const { container } = render(Skeleton);
    expect(container.querySelector('div')).toBeInTheDocument();
  });

  it('defaults variant to "text"', () => {
    const { container } = render(Skeleton);
    expect(container.querySelector('div')).toHaveAttribute('data-variant', 'text');
  });

  it('reflects width and height props as inline styles', () => {
    const { container } = render(Skeleton, { props: { width: '120px', height: '24px' } });
    const el = container.querySelector('div') as HTMLElement;
    expect(el.style.width).toBe('120px');
    expect(el.style.height).toBe('24px');
  });
});
