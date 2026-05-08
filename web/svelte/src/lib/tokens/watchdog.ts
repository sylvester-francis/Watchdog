import type { Tokens } from './types';

export const watchdogTokens: Tokens = {
  font: {
    display: 'Inter, system-ui, -apple-system, sans-serif',
    body: 'Inter, system-ui, -apple-system, sans-serif',
    prose: 'Inter, system-ui, -apple-system, sans-serif',
    mono: '"JetBrains Mono", Menlo, monospace',
  },
  color: {
    bg: '#09090b',
    bgElev: '#111113',
    bgOverlay: '#18181b',
    border: '#27272a',
    borderStrong: '#27272a',
    text: '#fafafa',
    textSecondary: '#d4d4d8',
    textMuted: '#a1a1aa',
    textFaint: '#52525b',
    accent: '#3b82f6',
    accentFade: 'rgb(59 130 246 / 0.15)',
    accentGlow: 'rgb(59 130 246 / 0.4)',
    statusUp: '#22c55e',
    statusWarn: '#eab308',
    statusDown: '#ef4444',
  },
  radius: {
    sm: 4,
    base: 6,
    lg: 8,
  },
  motion: {
    fast: 120,
    base: 160,
    ease: 'cubic-bezier(0.2, 0.8, 0.2, 1)',
  },
  tracking: {
    label: 0.04,
    display: 0.02,
  },
  shadow: {
    sm: 'none',
    base: '0 1px 2px rgb(0 0 0 / 0.3)',
  },
};

export function watchdogTokensToCSS(t: Tokens = watchdogTokens): string {
  return `:root {
  --font-display: ${t.font.display};
  --font-body: ${t.font.body};
  --font-prose: ${t.font.prose};
  --font-mono: ${t.font.mono};

  --bg: ${t.color.bg};
  --bg-elev: ${t.color.bgElev};
  --bg-overlay: ${t.color.bgOverlay};
  --border: ${t.color.border};
  --border-strong: ${t.color.borderStrong};

  --text: ${t.color.text};
  --text-secondary: ${t.color.textSecondary};
  --text-muted: ${t.color.textMuted};
  --text-faint: ${t.color.textFaint};

  --accent: ${t.color.accent};
  --accent-fade: ${t.color.accentFade};
  --accent-glow: ${t.color.accentGlow};

  --status-up: ${t.color.statusUp};
  --status-warn: ${t.color.statusWarn};
  --status-down: ${t.color.statusDown};

  --radius-sm: ${t.radius.sm}px;
  --radius: ${t.radius.base}px;
  --radius-lg: ${t.radius.lg}px;

  --shadow-sm: ${t.shadow.sm};
  --shadow: ${t.shadow.base};

  --tracking-label: ${t.tracking.label}em;
  --tracking-display: ${t.tracking.display}em;

  --motion-fast: ${t.motion.fast}ms;
  --motion-base: ${t.motion.base}ms;
  --ease: ${t.motion.ease};
}
`;
}
