import path from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  // loadEnv exposes VITE_* vars (e.g. VITE_API_PROXY from .env.local).
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_API_PROXY || 'http://localhost:9090'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        '@': path.resolve(import.meta.dirname, './src'),
      },
    },
    server: {
      port: 5173,
      proxy: {
        // All API traffic is forwarded to the backend; the backend enforces CORS.
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
        '/swagger': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
    build: {
      target: 'es2022',
      sourcemap: false,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('node_modules/recharts')) return 'charts'
            if (id.includes('node_modules/lucide-react')) return 'icons'
            if (id.includes('node_modules/@tanstack/react-query')) return 'query'
            if (id.includes('node_modules/react')) return 'vendor'
            return undefined
          },
        },
      },
    },
  }
})
