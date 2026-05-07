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
});
