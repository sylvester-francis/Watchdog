import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Button from './Button.svelte';

describe('Button', () => {
  it('renders a button element', () => {
    render(Button);
    expect(screen.getByRole('button')).toBeInTheDocument();
  });

  it('defaults to type="button"', () => {
    render(Button);
    expect(screen.getByRole('button')).toHaveAttribute('type', 'button');
  });

  it('reflects variant prop onto data-variant attribute', () => {
    render(Button, { props: { variant: 'destructive' } });
    expect(screen.getByRole('button')).toHaveAttribute('data-variant', 'destructive');
  });

  it('reflects disabled prop onto disabled attribute', () => {
    render(Button, { props: { disabled: true } });
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('applies primary variant classes by default', () => {
    const { container } = render(Button);
    const btn = container.querySelector('button')!;
    expect(btn.className).toContain('bg-accent');
  });

  it('applies destructive variant classes', () => {
    const { container } = render(Button, { props: { variant: 'destructive' } });
    expect(container.querySelector('button')!.className).toContain('bg-destructive');
  });

  it('applies ghost variant: no background, hover only', () => {
    const { container } = render(Button, { props: { variant: 'ghost' } });
    const btn = container.querySelector('button')!;
    expect(btn.className).toContain('bg-transparent');
    expect(btn.className).toContain('hover:bg-card-elevated');
  });

  it('applies outline variant: bordered, transparent', () => {
    const { container } = render(Button, { props: { variant: 'outline' } });
    expect(container.querySelector('button')!.className).toContain('border');
  });

  it('applies size sm', () => {
    const { container } = render(Button, { props: { size: 'sm' } });
    expect(container.querySelector('button')!.className).toContain('text-sm');
  });

  it('applies size lg', () => {
    const { container } = render(Button, { props: { size: 'lg' } });
    expect(container.querySelector('button')!.className).toContain('text-lg');
  });

  it('reflects size onto data-size attribute', () => {
    const { container } = render(Button, { props: { size: 'lg' } });
    expect(container.querySelector('button')!.getAttribute('data-size')).toBe('lg');
  });
});
