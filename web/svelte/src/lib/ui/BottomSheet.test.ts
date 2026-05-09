import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import BottomSheet from './BottomSheet.svelte';

describe('BottomSheet', () => {
  it('does not render dialog when closed', () => {
    const { container } = render(BottomSheet);
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open', () => {
    const { container } = render(BottomSheet, { props: { open: true } });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('renders a drag handle', () => {
    const { container } = render(BottomSheet, { props: { open: true } });
    expect(container.querySelector('[data-bottom-sheet-handle]')).toBeInTheDocument();
  });

  it('renders an overlay backdrop', () => {
    const { container } = render(BottomSheet, { props: { open: true } });
    expect(container.querySelector('[data-bottom-sheet-overlay]')).toBeInTheDocument();
  });

  it('starts at half height by default', () => {
    const { container } = render(BottomSheet, { props: { open: true } });
    const dialog = container.querySelector('[role="dialog"]')!;
    expect(dialog.className).toContain('h-55dvh');
  });

  it('starts at full height when height="full"', () => {
    const { container } = render(BottomSheet, { props: { open: true, height: 'full' } });
    const dialog = container.querySelector('[role="dialog"]')!;
    expect(dialog.className).toContain('h-92dvh');
  });

  it('calls onclose on Escape', async () => {
    let closed = false;
    render(BottomSheet, { props: { open: true, onclose: () => { closed = true; } } });
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(closed).toBe(true);
  });

  it('reflects height onto data-height attribute', () => {
    const { container } = render(BottomSheet, { props: { open: true, height: 'full' } });
    expect(container.querySelector('[role="dialog"]')!.getAttribute('data-height')).toBe('full');
  });
});
