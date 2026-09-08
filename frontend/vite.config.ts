import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from "path"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    // Proxy the API in development so the dev server is the same origin as the
    // API, exactly as Caddy makes it in production. Uploaded files are protected
    // by a cookie scoped to /api/uploads, and a browser will not attach that
    // cookie to a cross-origin <img> request — without this proxy, images would
    // load in production but silently 404 in development.
    proxy: {
      // The backend port is configurable so a developer can run the API
      // beside another service without editing this file.
      '/api': {
        target: process.env.VITE_API_PROXY || 'http://localhost:8080',
        changeOrigin: false,
        ws: true,
      },
    },
  },
  preview: {
    host: '0.0.0.0',
    port: 4173,
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    minify: 'oxc',
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})
