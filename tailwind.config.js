/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './web/templates/**/*.html',
    './web/static/js/**/*.js',
  ],
  theme: {
    extend: {
      colors: {
        background: '#09090b',
        foreground: '#fafafa',
        card: { DEFAULT: '#111113', foreground: '#fafafa' },
        'card-elevated': '#18181b',
        muted: { DEFAULT: '#27272a', foreground: '#a1a1aa' },
        accent: { DEFAULT: '#3b82f6', foreground: '#fafafa' },
        border: '#27272a',
        input: '#27272a',
        ring: '#3b82f6',
        destructive: { DEFAULT: '#ef4444', foreground: '#fafafa' },
        success: '#22c55e',
        warning: '#eab308',
        // Semantic status tokens
        'status-healthy': '#10b981',
        'status-critical': '#ef4444',
        'status-warning': '#f59e0b',
        'status-info': '#3b82f6',
        // Surface hierarchy
        'surface-primary': '#111113',
        'surface-elevated': '#18181b',
        'surface-sunken': '#09090b',
      },
      borderRadius: {
        lg: '0.5rem',
        md: 'calc(0.5rem - 2px)',
        sm: 'calc(0.5rem - 4px)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'monospace'],
      },
      fontSize: {
        'stat': ['2rem', { lineHeight: '1', fontWeight: '700' }],
        'stat-sm': ['1.5rem', { lineHeight: '1', fontWeight: '700' }],
      },
    },
  },
  plugins: [],
};
