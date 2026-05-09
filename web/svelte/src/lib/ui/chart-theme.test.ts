import { describe, it, expect } from 'vitest';
import { chartTheme } from './chart-theme';
import { tokens } from '$lib/tokens';

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
    expect(t.accent).toBe(tokens.color.accent);
  });

  it('returns the active brand mono-family font string', () => {
    const t = chartTheme();
    expect(t.fontMono).toBe(tokens.font.mono);
  });
});
