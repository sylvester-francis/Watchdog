import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Textarea from './Textarea.svelte';

describe('Textarea', () => {
  it('renders a textarea element', () => {
    const { container } = render(Textarea);
    expect(container.querySelector('textarea')).toBeInTheDocument();
  });

  it('defaults to rows=3', () => {
    const { container } = render(Textarea);
    expect(container.querySelector('textarea')).toHaveAttribute('rows', '3');
  });

  it('respects rows prop', () => {
    const { container } = render(Textarea, { props: { rows: 5 } });
    expect(container.querySelector('textarea')).toHaveAttribute('rows', '5');
  });

  it('applies error styling when error prop is true', () => {
    const { container } = render(Textarea, { props: { error: true } });
    expect(container.querySelector('textarea')!.className).toContain('border-destructive');
  });

  it('reflects error prop onto aria-invalid', () => {
    const { container } = render(Textarea, { props: { error: true } });
    expect(container.querySelector('textarea')!.getAttribute('aria-invalid')).toBe('true');
  });

  it('exposes focus ring class', () => {
    const { container } = render(Textarea);
    expect(container.querySelector('textarea')!.className).toContain('focus:ring-2');
  });

  it('applies disabled styling', () => {
    const { container } = render(Textarea, { props: { disabled: true } });
    expect(container.querySelector('textarea')).toBeDisabled();
  });
});
