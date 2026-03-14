import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
      '/v2': 'http://localhost:8080',
      '/npm': 'http://localhost:8080',
      '/pypi': 'http://localhost:8080',
      '/go': 'http://localhost:8080',
      '/cargo': 'http://localhost:8080',
      '/maven': 'http://localhost:8080',
      '/nuget': 'http://localhost:8080',
      '/helm': 'http://localhost:8080',
    },
  },
})
