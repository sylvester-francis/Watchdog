import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Tabs from './Tabs.svelte';

describe('Tabs', () => {
  it('renders a tablist role element', () => {
    const { container } = render(Tabs);
    expect(container.querySelector('[role="tablist"]')).toBeInTheDocument();
  });

  it('reflects value prop onto data-active attribute', () => {
    const { container } = render(Tabs, { props: { value: 'overview' } });
    expect(container.querySelector('[role="tablist"]')).toHaveAttribute('data-active', 'overview');
  });
});
