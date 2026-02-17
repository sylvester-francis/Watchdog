/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: ['./web/templates/**/*.html'],
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
    },
  },
  plugins: [],
};
