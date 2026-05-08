import { describe, it, expect } from 'vitest';
import { chartTheme } from './chart-theme';

describe('chartTheme', () => {
  it('returns an object with the expected keys', () => {
    const t = chartTheme();
    expect(t).toHaveProperty('accent');
    expect(t).toHaveProperty('statusUp');
    expect(t).toHaveProperty('statusDown');
    expect(t).toHaveProperty('statusWarn');
    expect(t).toHaveProperty('text');
    expect(t).toHaveProperty('grid');
    expect(t).toHaveProperty('fontMono');
  });

  it('returns the active brand accent color', () => {
    const t = chartTheme();
    expect(t.accent).toBe('#3b82f6');
  });

  it('returns mono-family font string for fontMono', () => {
    const t = chartTheme();
    expect(t.fontMono).toMatch(/JetBrains Mono|monospace/);
  });
});
