import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import fs from 'fs'
import path from 'path'

const host = process.env.TAURI_DEV_HOST;

// https://vite.dev/config/
export default defineConfig(async () => ({
  plugins: [react()],

  // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
  //
  // 1. prevent Vite from obscuring rust errors
  clearScreen: false,
  // 2. tauri expects a fixed port, fail if that port is not available
  server: {
    port: 3001,
    strictPort: true,
    host: '0.0.0.0',
    // HTTPS 配置
    https: {
      key: fs.readFileSync(path.resolve(__dirname, 'frontend-key.pem')),
      cert: fs.readFileSync(path.resolve(__dirname, 'frontend-cert.pem'))
    },
    hmr: host
      ? {
          protocol: "ws",
          host,
          port: 1421,
        }
      : undefined,
    watch: {
      // 3. tell Vite to ignore watching `src-tauri`
      ignored: ["**/src-tauri/**"],
    },
  },
}));