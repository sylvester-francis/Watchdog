import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import FormField from './FormField.svelte';

describe('FormField', () => {
  it('renders the label when provided', () => {
    render(FormField, { props: { label: 'Email' } });
    expect(screen.getByText('Email')).toBeInTheDocument();
  });

  it('renders the error message when provided', () => {
    render(FormField, { props: { error: 'Required' } });
    expect(screen.getByText('Required')).toBeInTheDocument();
  });

  it('does not render error region when error is null', () => {
    const { container } = render(FormField, { props: { label: 'Email', error: null } });
    expect(container.querySelector('p')).not.toBeInTheDocument();
  });
});
