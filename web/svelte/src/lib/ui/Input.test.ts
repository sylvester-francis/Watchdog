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
});
