import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { resolve } from 'path'

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
      dts: false,
    }),
    Components({
      resolvers: [ElementPlusResolver()],
      dts: false,
    }),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    host: '0.0.0.0',   // 允许局域网访问
    port: 5173,         // 可选，默认就是 5173，这里显式写出来方便查看
    proxy: {
      '/admin': process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
      '/api': process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
      '/uploads': process.env.VITE_MEDIA_BASE_URL || 'https://balaai-cdn.zdrawai.com',
    },
  },
  build: {
    outDir: 'dist',
  },
})
