import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    rollupOptions: {
      output: {
        // 只改分包粒度，不改任何运行时代码：入口静态依赖仍是同一批模块，
        // 打出的字节总量与加载时机不变，但三方库与应用代码分离后，
        // 发版时 element-plus / vue 等长缓存不被失效，且多个小 chunk 可并行下载。
        // 用函数形式而不是对象形式，避免对象前缀匹配把 vue-router / pinia
        // 误归到 'vue' 之外，或将同一包拆散导致运行时重复初始化。
        manualChunks(id) {
          if (!id.includes('node_modules')) return
          if (id.includes('@element-plus/icons-vue')) return 'vendor-el-icons'
        },
      },
    },
  },
  server: {
    allowedHosts: ['inkpot.cn'],
    host: '0.0.0.0',
    port: 5176,
    proxy: {
      '/api': {
        target: 'http://localhost:8004',
        changeOrigin: true,
      },
    },
  },
})
