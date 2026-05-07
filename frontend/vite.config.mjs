import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const frontendRoot = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  root: frontendRoot,
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/sync-dashboard": "http://localhost:8080",
      "/metrics": "http://localhost:8080",
      "/events": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
});
