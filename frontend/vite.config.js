import path from 'path'
import { readFileSync } from 'fs'
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

let appVersion = ''
try {
  const pkg = JSON.parse(readFileSync(path.resolve(__dirname, 'package.json'), 'utf-8'))
  appVersion = pkg?.version ?? ''
} catch {}

export default defineConfig({
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
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
})
