import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api/terminal": { target: "ws://localhost:8080", ws: true },
      "/api": { target: "http://localhost:8080", changeOrigin: false },
      "/version": { target: "http://localhost:8080", changeOrigin: false },
      "/debug": { target: "http://localhost:8080", changeOrigin: false },
    },
  },
  build: {
    outDir: "../embed/public_html",
    emptyOutDir: true,
    sourcemap: false,
  },
});
