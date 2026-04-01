/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      fontFamily: { sans: ['Inter', 'sans-serif'] },
      colors: {
        sidebar: '#141b2d',
        primary: '#4361ee',
        'primary-dark': '#3451d1',
      },
    },
  },
  plugins: [],
}
