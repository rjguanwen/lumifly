/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  // 关闭 preflight，避免重置 Element Plus 组件默认样式
  corePlugins: { preflight: false },
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#eef2ff',
          100: '#e0e7ff',
          400: '#5B7CFF',
          500: '#4A6CF7',
          600: '#3b5bdb',
          700: '#2b47b8',
        },
      },
    },
  },
  plugins: [],
}
