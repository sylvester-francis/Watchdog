import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import Pill from './Pill.svelte';

describe('Pill', () => {
  it('defaults tone to "neutral"', () => {
    const { container } = render(Pill);
    expect(container.querySelector('span')).toHaveAttribute('data-tone', 'neutral');
  });

  it('reflects tone prop onto data-tone attribute', () => {
    const { container } = render(Pill, { props: { tone: 'down' } });
    expect(container.querySelector('span')).toHaveAttribute('data-tone', 'down');
  });

  it('has the ui-pill class for styling hook', () => {
    const { container } = render(Pill);
    expect(container.querySelector('span')!.className).toContain('ui-pill');
  });
});
