/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    fontFamily: {
      mono: ['"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
    },
    extend: {
      colors: {
        bg: 'var(--t-bg)',
        fg: 'var(--t-fg)',
        dim: 'var(--t-dim)',
        line: 'var(--t-line)',
        ok: 'var(--ok)',
        warn: 'var(--warn)',
        crit: 'var(--crit)',
        off: 'var(--off)',
        cy: 'var(--cy)',
        mg: 'var(--mg)',
      },
    },
  },
  plugins: [],
}
