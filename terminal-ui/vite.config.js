import { defineConfig } from 'vite'

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api/terminal': {
        target: 'ws://localhost:8080',
        ws: true,
        // rewrite: (path) => path.replace(/^\/api/, ''),
      }
    },
  }
})

