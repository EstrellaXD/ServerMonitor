/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        heading: ['Space Grotesk', 'sans-serif'],
        body: ['DM Sans', 'sans-serif'],
        mono: ['Fira Code', 'monospace'],
      },
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          200: '#bfdbfe',
          300: '#93c5fd',
          400: '#60a5fa',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
          800: '#1e40af',
          900: '#1e3a8a',
          950: '#172554',
        },
        surface: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          800: '#1e293b',
          850: '#162032',
          900: '#0f172a',
          950: '#020617',
        },
      },
      boxShadow: {
        'glow-sm': '0 0 10px -3px rgba(59, 130, 246, 0.3)',
        'glow': '0 0 20px -5px rgba(59, 130, 246, 0.4)',
        'glow-lg': '0 0 30px -5px rgba(59, 130, 246, 0.5)',
        'glow-green': '0 0 12px -2px rgba(34, 197, 94, 0.5)',
        'glow-amber': '0 0 12px -2px rgba(245, 158, 11, 0.5)',
        'glow-red': '0 0 12px -2px rgba(239, 68, 68, 0.5)',
      },
      backdropBlur: {
        xs: '2px',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'glow-pulse': 'glow-pulse 2s ease-in-out infinite',
      },
      keyframes: {
        'glow-pulse': {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.6' },
        },
      },
    },
  },
  plugins: [],
}
