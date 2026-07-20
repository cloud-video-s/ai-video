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
    proxy: {
      '/admin': process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
      '/api': process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
      '/uploads': process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
    },
  },
  build: {
    outDir: 'dist',
  },
})
