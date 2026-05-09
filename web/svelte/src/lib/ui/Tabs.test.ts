import { render, fireEvent, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Tabs from './Tabs.svelte';

describe('Tabs', () => {
  it('renders a tablist role element', () => {
    const { container } = render(Tabs, { props: { options: [], value: '' } });
    expect(container.querySelector('[role="tablist"]')).toBeInTheDocument();
  });

  it('reflects value prop onto the tablist data-active attribute', () => {
    const options = [{ value: 'a', label: 'A' }];
    const { container } = render(Tabs, { props: { options, value: 'a' } });
    expect(container.querySelector('[role="tablist"]')!.getAttribute('data-active')).toBe('a');
  });

  it('renders one button per option', () => {
    const options = [{ value: 'a', label: 'A' }, { value: 'b', label: 'B' }];
    const { container } = render(Tabs, { props: { options, value: 'a' } });
    expect(container.querySelectorAll('button[role="tab"]').length).toBe(2);
  });

  it('marks the active tab with aria-selected="true"', () => {
    const options = [{ value: 'a', label: 'A' }, { value: 'b', label: 'B' }];
    render(Tabs, { props: { options, value: 'b' } });
    const buttons = screen.getAllByRole('tab');
    expect(buttons[0]).toHaveAttribute('aria-selected', 'false');
    expect(buttons[1]).toHaveAttribute('aria-selected', 'true');
  });

  it('calls onchange when a tab is clicked', async () => {
    let changed: string | null = null;
    const options = [{ value: 'a', label: 'A' }, { value: 'b', label: 'B' }];
    render(Tabs, { props: { options, value: 'a', onchange: (v: string) => { changed = v; } } });
    await fireEvent.click(screen.getAllByRole('tab')[1]);
    expect(changed).toBe('b');
  });
});
