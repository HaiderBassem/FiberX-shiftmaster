import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from "path"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: '192.168.16.138',
    port: 5173,
  },
  preview: {
    host: '192.168.16.138',
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
