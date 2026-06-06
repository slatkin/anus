import path from 'path'
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

const apiMode = process.env.VITE_API ?? 'web'

export default defineConfig({
  plugins: [svelte()],
  test: {
    environment: 'node',
  },
  build: {
    outDir: 'dist',
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
