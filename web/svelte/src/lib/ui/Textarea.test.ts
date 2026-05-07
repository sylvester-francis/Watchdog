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
});
