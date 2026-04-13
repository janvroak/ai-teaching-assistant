/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['DM Sans', 'sans-serif'],
        display: ['Syne', 'sans-serif'],
      },
      colors: {
        brand: {
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
        violet: {
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',
        },
        cyan: {
          400: '#22d3ee',
          500: '#06b6d4',
        },
        dark: {
          background: '#04050a',
          surface: '#0b1020',
          surfaceAlt: '#111827',
          panel: 'rgba(255, 255, 255, 0.025)',
          border: 'rgba(255, 255, 255, 0.08)',
          text: '#f3f4f6',
          secondary: '#94a3b8',
          muted: '#64748b',
        },
      },
      animation: {
        'fade-up': 'fadeUp 0.7s cubic-bezier(0.16, 1, 0.3, 1) both',
        'pulse-slow': 'pulseSlow 7s ease-in-out infinite',
        'gradient-shift': 'gradientShift 8s ease infinite',
        'scanline': 'scanline 14s linear infinite',
        'float-ambient': 'floatAmbient 26s ease-in-out infinite',
        'drift-particles': 'driftParticles 46s linear infinite',
      },
      keyframes: {
        fadeUp: {
          'from': { opacity: '0', transform: 'translateY(24px) scale(0.98)' },
          'to': { opacity: '1', transform: 'translateY(0) scale(1)' },
        },
        pulseSlow: {
          '0%, 100%': { transform: 'scale(1)', opacity: '0.7' },
          '50%': { transform: 'scale(1.15)', opacity: '1' },
        },
        gradientShift: {
          '0%, 100%': { backgroundPosition: '0% 50%' },
          '50%': { backgroundPosition: '100% 50%' },
        },
        scanline: {
          'to': { top: '110%' }
        },
        floatAmbient: {
          '0%, 100%': { transform: 'translate3d(0, 0, 0) scale(1)' },
          '50%': { transform: 'translate3d(2%, -3%, 0) scale(1.04)' },
        },
        driftParticles: {
          '0%': { transform: 'translate3d(0, 0, 0)' },
          '50%': { transform: 'translate3d(-1.5%, -2%, 0)' },
          '100%': { transform: 'translate3d(0, 0, 0)' },
        },
      }
    },
  },
  plugins: [],
}
