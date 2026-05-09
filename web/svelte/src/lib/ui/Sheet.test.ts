import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Sheet from './Sheet.svelte';

describe('Sheet', () => {
  it('does not render dialog when closed', () => {
    const { container } = render(Sheet);
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open', () => {
    const { container } = render(Sheet, { props: { open: true } });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('reflects side prop onto data-side attribute', () => {
    const { container } = render(Sheet, { props: { open: true, side: 'left' } });
    expect(container.querySelector('[role="dialog"]')).toHaveAttribute('data-side', 'left');
  });

  it('calls onclose on Escape', async () => {
    let closed = false;
    render(Sheet, { props: { open: true, onclose: () => { closed = true; } } });
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(closed).toBe(true);
  });

  it('calls onclose on overlay click', async () => {
    let closed = false;
    const { container } = render(Sheet, { props: { open: true, onclose: () => { closed = true; } } });
    await fireEvent.click(container.querySelector('[data-sheet-overlay]')!);
    expect(closed).toBe(true);
  });

  it('does not close when content is clicked', async () => {
    let closed = false;
    const { container } = render(Sheet, { props: { open: true, onclose: () => { closed = true; } } });
    await fireEvent.click(container.querySelector('[role="dialog"]')!);
    expect(closed).toBe(false);
  });

  it('positions on the right by default', () => {
    const { container } = render(Sheet, { props: { open: true } });
    const dialog = container.querySelector('[role="dialog"]')!;
    expect(dialog.className).toContain('right-0');
  });

  it('positions on the left when side=left', () => {
    const { container } = render(Sheet, { props: { open: true, side: 'left' } });
    expect(container.querySelector('[role="dialog"]')!.className).toContain('left-0');
  });

  it('positions on the bottom when side=bottom', () => {
    const { container } = render(Sheet, { props: { open: true, side: 'bottom' } });
    expect(container.querySelector('[role="dialog"]')!.className).toContain('bottom-0');
  });
});
