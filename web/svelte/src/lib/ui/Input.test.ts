import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Input from './Input.svelte';

describe('Input', () => {
  it('renders an input element', () => {
    const { container } = render(Input);
    expect(container.querySelector('input')).toBeInTheDocument();
  });

  it('defaults to type="text"', () => {
    const { container } = render(Input);
    expect(container.querySelector('input')).toHaveAttribute('type', 'text');
  });

  it('respects type prop', () => {
    const { container } = render(Input, { props: { type: 'email' } });
    expect(container.querySelector('input')).toHaveAttribute('type', 'email');
  });

  it('reflects placeholder prop', () => {
    const { container } = render(Input, { props: { placeholder: 'enter email' } });
    expect(container.querySelector('input')).toHaveAttribute('placeholder', 'enter email');
  });

  it('applies error styling when error prop is true', () => {
    const { container } = render(Input, { props: { error: true } });
    expect(container.querySelector('input')!.className).toContain('border-destructive');
  });

  it('does not apply error styling by default', () => {
    const { container } = render(Input);
    expect(container.querySelector('input')!.className).not.toContain('border-destructive');
  });

  it('reflects error prop onto aria-invalid', () => {
    const { container } = render(Input, { props: { error: true } });
    expect(container.querySelector('input')!.getAttribute('aria-invalid')).toBe('true');
  });

  it('applies disabled styling', () => {
    const { container } = render(Input, { props: { disabled: true } });
    const input = container.querySelector('input')!;
    expect(input).toBeDisabled();
    expect(input.className).toContain('disabled:opacity-50');
  });

  it('exposes focus ring class on the input', () => {
    const { container } = render(Input);
    expect(container.querySelector('input')!.className).toContain('focus:ring-2');
  });

  it('forwards min, max, step attributes for number inputs', () => {
    const { container } = render(Input, { props: { type: 'number', min: 5, max: 3600, step: 1 } });
    const input = container.querySelector('input')!;
    expect(input.getAttribute('min')).toBe('5');
    expect(input.getAttribute('max')).toBe('3600');
    expect(input.getAttribute('step')).toBe('1');
  });

  it('omits min, max, step when not provided', () => {
    const { container } = render(Input);
    const input = container.querySelector('input')!;
    expect(input.hasAttribute('min')).toBe(false);
    expect(input.hasAttribute('max')).toBe(false);
    expect(input.hasAttribute('step')).toBe(false);
  });

  it('accepts numeric value', () => {
    const { container } = render(Input, { props: { type: 'number', value: 42 } });
    const input = container.querySelector('input') as HTMLInputElement;
    expect(input.value).toBe('42');
  });
});
