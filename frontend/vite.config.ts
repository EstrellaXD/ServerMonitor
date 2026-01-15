import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 4829,
    proxy: {
      '/api': {
        target: 'http://localhost:8742',
        changeOrigin: true
      },
      '/ws': {
        target: 'ws://localhost:8742',
        ws: true
      }
    }
  }
})
