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
});
