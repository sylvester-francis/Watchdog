import type { Config } from 'tailwindcss';
import { watchdogTokens as t } from '@sylvester-francis/watchdog-ui/tokens';

const config: Config = {
  darkMode: 'class',
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        background: t.color.bg,
        foreground: t.color.text,
        card: { DEFAULT: t.color.bgElev, foreground: t.color.text },
        'card-elevated': t.color.bgOverlay,
        muted: { DEFAULT: t.color.border, foreground: t.color.textMuted },
        accent: { DEFAULT: t.color.accent, foreground: t.color.text },
        border: t.color.borderStrong,
        input: t.color.borderStrong,
        ring: t.color.accent,
        destructive: { DEFAULT: t.color.statusDown, foreground: t.color.text },
        success: t.color.statusUp,
        warning: t.color.statusWarn,
        'status-healthy': '#10b981',
        'status-critical': t.color.statusDown,
        'status-warning': '#f59e0b',
        'status-info': t.color.accent,
        'surface-primary': t.color.bgElev,
        'surface-elevated': t.color.bgOverlay,
        'surface-sunken': t.color.bg,
      },
      borderRadius: {
        sm: `${t.radius.sm}px`,
        md: `${t.radius.base}px`,
        lg: `${t.radius.lg}px`,
      },
      fontFamily: {
        sans: t.font.body.split(',').map((s) => s.trim().replace(/^"|"$/g, '')),
        mono: t.font.mono.split(',').map((s) => s.trim().replace(/^"|"$/g, '')),
      },
      fontSize: {
        stat: ['2rem', { lineHeight: '1', fontWeight: '700' }],
        'stat-sm': ['1.5rem', { lineHeight: '1', fontWeight: '700' }],
      },
      height: {
        '55dvh': '55dvh',
        '92dvh': '92dvh',
      },
      transitionTimingFunction: {
        sheet: 'cubic-bezier(0.32, 0.72, 0, 1)',
      },
    },
  },
  plugins: [],
};

export default config;
