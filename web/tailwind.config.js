/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        roboto: ['Roboto', 'sans-serif'],
      },
      colors: {
        // Qualys color palette
        qualys: {
          bg: '#f7f7f7',
          card: '#ffffff',
          border: '#dae7f4',
          'border-light': '#e7ecee',
          sidebar: '#0e1215',
          'sidebar-hover': '#364750',
          'sidebar-active': '#1a2328',
          accent: '#9dbfe1',
          'accent-hover': '#b5d1eb',
          'text-primary': '#333333',
          'text-secondary': '#56707e',
          'text-muted': '#8a9ba5',
        },
        primary: {
          50: '#e8f4fc',
          100: '#d1e9f9',
          200: '#a3d3f3',
          300: '#75bded',
          400: '#47a7e7',
          500: '#1991e1',
          600: '#1474b4',
          700: '#0f5787',
          800: '#0a3a5a',
          900: '#051d2d',
        },
        severity: {
          critical: '#c41230',
          high: '#e85d04',
          medium: '#f4a100',
          low: '#2e7d32',
          info: '#56707e',
        },
      },
      boxShadow: {
        'qualys': '0 0 8px 2px #e7ecee',
        'qualys-sm': '0 1px 3px rgba(0,0,0,0.08)',
      },
    },
  },
  plugins: [],
}
