import { tokens } from '$lib/tokens';

export function chartTheme() {
  return {
    accent:     tokens.color.accent,
    statusUp:   tokens.color.statusUp,
    statusDown: tokens.color.statusDown,
    statusWarn: tokens.color.statusWarn,
    text:       tokens.color.textMuted,
    grid:       tokens.color.border,
    fontMono:   tokens.font.mono,
  };
}
