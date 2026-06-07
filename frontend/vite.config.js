import path from 'path'
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

const apiMode = process.env.VITE_API ?? 'web'

export default defineConfig({
  plugins: [svelte({ onwarn: (warning, handler) => { if (warning.code.startsWith('a11y-')) return; handler(warning); } })],
  test: {
    environment: 'node',
  },
  build: {
    outDir: 'dist',
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  resolve: {
    alias: {
      './api.js': path.resolve(
        __dirname,
        apiMode === 'wails' ? 'src/api.wails.js' : 'src/api.web.js'
      ),
    },
  },
})
