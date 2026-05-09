import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Select from './Select.svelte';

describe('Select', () => {
  it('renders a select element', () => {
    const { container } = render(Select);
    expect(container.querySelector('select')).toBeInTheDocument();
  });

  it('reflects disabled prop', () => {
    const { container } = render(Select, { props: { disabled: true } });
    expect(container.querySelector('select')).toBeDisabled();
  });

  it('applies error styling', () => {
    const { container } = render(Select, { props: { error: true } });
    expect(container.querySelector('select')!.className).toContain('border-destructive');
  });

  it('reflects error onto aria-invalid', () => {
    const { container } = render(Select, { props: { error: true } });
    expect(container.querySelector('select')!.getAttribute('aria-invalid')).toBe('true');
  });

  it('exposes focus ring class', () => {
    const { container } = render(Select);
    expect(container.querySelector('select')!.className).toContain('focus:ring-2');
  });
});
