import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$lib/api/monitors', () => ({
  getCertDetails: vi.fn(async () => ({
    data: {
      expiry_days: 60,
      issuer: "Let's Encrypt",
      algorithm: 'RSA',
      key_size: 2048,
      chain_valid: true,
      serial_number: 'abc123def456',
      sans: ['example.com', 'www.example.com', 'api.example.com'],
      last_checked_at: new Date().toISOString(),
    },
  })),
}));

import CertDetailsCard from './CertDetailsCard.svelte';

describe('CertDetailsCard', () => {
  it('renders one Pill per SAN entry', async () => {
    const { container } = render(CertDetailsCard, { props: { monitorId: 'm1' } });
    await new Promise(r => setTimeout(r, 50));
    expect(container.querySelectorAll('.ui-pill').length).toBe(3);
  });
});
