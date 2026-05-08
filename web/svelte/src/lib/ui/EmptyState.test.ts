import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import EmptyState from './EmptyState.svelte';

describe('EmptyState', () => {
  it('renders the title when provided', () => {
    render(EmptyState, { props: { title: 'No monitors yet' } });
    expect(screen.getByText('No monitors yet')).toBeInTheDocument();
  });

  it('renders the description when provided', () => {
    render(EmptyState, { props: { description: 'Add your first monitor to get started.' } });
    expect(screen.getByText('Add your first monitor to get started.')).toBeInTheDocument();
  });

  it('renders nothing visible when no props provided', () => {
    const { container } = render(EmptyState);
    expect(container.querySelector('h3')).not.toBeInTheDocument();
    expect(container.querySelector('p')).not.toBeInTheDocument();
  });

  // Snippet rendering tests deferred to Phase 1c — `createRawSnippet` client-mode API
  // diverges from SSR; needs a wrapper-component pattern. Snippet props are type-checked
  // at compile time, and EmptyState is consumed in Phase 1b+.
  it.skip('renders the icon snippet when provided', () => {});
  it.skip('renders the cta snippet when provided', () => {});
});
