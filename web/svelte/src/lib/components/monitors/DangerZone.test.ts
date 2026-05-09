import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$lib/api', () => ({
  monitors: { deleteMonitor: vi.fn() },
}));

vi.mock('$lib/stores/toast.svelte', () => ({
  getToasts: () => ({ success: vi.fn(), error: vi.fn() }),
}));

vi.mock('$app/navigation', () => ({
  goto: vi.fn(),
}));

import DangerZone from './DangerZone.svelte';

describe('DangerZone', () => {
  it('renders a destructive Button for delete', () => {
    const { container } = render(DangerZone, { props: { monitorId: 'm1' } });
    const btn = container.querySelector('button[data-variant="destructive"]');
    expect(btn).toBeInTheDocument();
    expect(btn?.textContent).toContain('Delete Monitor');
  });
});
