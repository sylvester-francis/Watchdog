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

  it('renders an asterisk when required', () => {
    const { container } = render(FormField, { props: { label: 'Email', required: true } });
    expect(container.textContent).toContain('*');
  });

  it('does not render an asterisk when not required', () => {
    const { container } = render(FormField, { props: { label: 'Email' } });
    expect(container.querySelector('label')!.textContent).not.toContain('*');
  });

  it('connects label to input via htmlFor', () => {
    render(FormField, { props: { label: 'Email', htmlFor: 'email-input' } });
    expect(screen.getByText('Email').getAttribute('for')).toBe('email-input');
  });

  it('renders error with role="alert" for screen reader announcement', () => {
    render(FormField, { props: { error: 'Required' } });
    const alert = screen.getByRole('alert');
    expect(alert.textContent).toBe('Required');
  });
});
