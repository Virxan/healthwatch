import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// Dev: Vite serves the SPA itself and proxies /api/* straight through to
// the Go backend on :8080 (same path, no rewrite - the backend mounts
// the exact same handlers under /api/ for this reason, see
// backend/handlers.go).
//
// Prod: there is no Vite server at all - `npm run build` writes directly
// into backend/web/dist, which the Go binary embeds and serves itself
// (see backend/web.go). outDir points there so `task build-frontend` is
// the only step needed; nothing else has to copy files around.
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: "../backend/web/dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
