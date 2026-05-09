import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Checkbox from './Checkbox.svelte';

describe('Checkbox', () => {
  it('renders a checkbox input', () => {
    const { container } = render(Checkbox);
    const input = container.querySelector('input');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('type', 'checkbox');
  });

  it('reflects disabled prop', () => {
    const { container } = render(Checkbox, { props: { disabled: true } });
    expect(container.querySelector('input')).toBeDisabled();
  });

  it('applies focus ring class', () => {
    const { container } = render(Checkbox);
    expect(container.querySelector('input')!.className).toContain('focus:ring-2');
  });

  it('applies error styling', () => {
    const { container } = render(Checkbox, { props: { error: true } });
    expect(container.querySelector('input')!.className).toContain('border-destructive');
  });

  it('reflects error onto aria-invalid', () => {
    const { container } = render(Checkbox, { props: { error: true } });
    expect(container.querySelector('input')!.getAttribute('aria-invalid')).toBe('true');
  });
});
