/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: { sans: ['Inter', 'system-ui', 'sans-serif'] },
      colors: {
        navy: {
          950: '#0a1020',
          900: '#0f1623',
          800: '#141b2d',
          700: '#1a2336',
          600: '#1e2a3a',
          500: '#243044',
        },
        primary: { DEFAULT: '#4361ee', hover: '#3451d1', light: '#6b7ff0' },
        success: '#2ec4b6',
        warning: '#f7b731',
        danger:  '#fc5c65',
      },
    },
  },
  plugins: [],
}
