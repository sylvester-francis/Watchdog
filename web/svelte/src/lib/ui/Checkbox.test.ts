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
});
