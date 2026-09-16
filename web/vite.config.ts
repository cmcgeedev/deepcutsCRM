/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      manifest: {
        name: "Deep Cuts Driver",
        short_name: "Deep Cuts",
        start_url: "/driver/route",
        scope: "/",
        display: "standalone",
        background_color: "#111111",
        theme_color: "#111111",
        icons: [{ src: "/icon.svg", sizes: "any", type: "image/svg+xml" }],
      },
      workbox: {
        navigateFallback: "/index.html",
        navigateFallbackDenylist: [/^\/api\//],
        globPatterns: ["**/*.{js,css,html,svg}"],
      },
    }),
  ],
  server: { proxy: { "/api": "http://localhost:8080" } },
  build: { outDir: "dist", emptyOutDir: true },
  test: { environment: "jsdom", setupFiles: ["src/test/setup.ts"], globals: true },
});
