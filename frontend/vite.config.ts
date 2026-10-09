/// <reference types="vitest" />
import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const apiTarget = process.env.E2E_API_URL || `http://localhost:${process.env.BB_API_PORT || 8080}`

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    // vitest loads ../scripts/lib guard tests from outside frontend/; keep the dev server's default fs
    // sandbox (do NOT widen it to the repo root: backend/.env would be readable via /@fs/)
    ...(process.env.VITEST ? { fs: { allow: ['..'] } } : {}),
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          query: ['@tanstack/react-query'],
          ui: ['@radix-ui/react-dialog'],
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
    // scripts/lib guard tests live outside frontend/ so the Docker build (frontend/-only context) never type-checks them
    include: ['src/**/*.{test,spec}.{ts,tsx}', '../scripts/lib/**/*.test.ts'],
    exclude: ['**/node_modules/**', '**/dist/**', '**/e2e/**'],
    // minimum of 1 lets `--maxWorkers=N` (N below core count) work without a min/max conflict
    poolOptions: { threads: { minThreads: 1 }, forks: { minForks: 1 } },
  },
})
