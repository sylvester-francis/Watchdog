import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Modal from './Modal.svelte';

describe('Modal', () => {
  it('does not render when closed', () => {
    const { container } = render(Modal);
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();
  });

  it('renders dialog when open', () => {
    const { container } = render(Modal, { props: { open: true } });
    expect(container.querySelector('[role="dialog"]')).toBeInTheDocument();
  });

  it('calls onclose when Escape key is pressed', async () => {
    let closed = false;
    render(Modal, { props: { open: true, onclose: () => { closed = true; } } });
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(closed).toBe(true);
  });

  it('calls onclose when overlay is clicked', async () => {
    let closed = false;
    const { container } = render(Modal, { props: { open: true, onclose: () => { closed = true; } } });
    const overlay = container.querySelector('[role="dialog"]')!;
    await fireEvent.click(overlay);
    expect(closed).toBe(true);
  });

  it('does not call onclose when content (non-overlay) is clicked', async () => {
    let closed = false;
    const { container } = render(Modal, { props: { open: true, onclose: () => { closed = true; } } });
    const content = container.querySelector('[data-modal-content]')!;
    await fireEvent.click(content);
    expect(closed).toBe(false);
  });

  it('renders content with data-modal-content attribute', () => {
    const { container } = render(Modal, { props: { open: true } });
    expect(container.querySelector('[data-modal-content]')).toBeInTheDocument();
  });

  it('defaults to max-w-md size', () => {
    const { container } = render(Modal, { props: { open: true } });
    expect(container.querySelector('[data-modal-content]')!.className).toContain('max-w-md');
  });

  it('applies size="lg" -> max-w-lg', () => {
    const { container } = render(Modal, { props: { open: true, size: 'lg' } });
    expect(container.querySelector('[data-modal-content]')!.className).toContain('max-w-lg');
  });

  it('applies size="sm" -> max-w-sm', () => {
    const { container } = render(Modal, { props: { open: true, size: 'sm' } });
    expect(container.querySelector('[data-modal-content]')!.className).toContain('max-w-sm');
  });

  it('applies size="xl" -> max-w-xl', () => {
    const { container } = render(Modal, { props: { open: true, size: 'xl' } });
    expect(container.querySelector('[data-modal-content]')!.className).toContain('max-w-xl');
  });

  it('content has scroll wrapper (max-h + overflow-y-auto) so tall content is reachable', () => {
    const { container } = render(Modal, { props: { open: true } });
    const content = container.querySelector('[data-modal-content]')!;
    expect(content.className).toContain('overflow-y-auto');
    expect(content.className).toContain('max-h-[90vh]');
  });
});
