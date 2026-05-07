import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Tooltip from './Tooltip.svelte';

describe('Tooltip', () => {
  it('renders with the label as title attribute', () => {
    const { container } = render(Tooltip, { props: { label: 'Click to refresh' } });
    expect(container.querySelector('span')).toHaveAttribute('title', 'Click to refresh');
  });
});
