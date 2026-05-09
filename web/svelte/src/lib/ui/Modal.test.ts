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
});
