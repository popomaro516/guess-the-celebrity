import { defineConfig } from "astro/config";
import react from "@astrojs/react";

const apiTarget = process.env.API_PROXY_TARGET ?? "http://localhost:8080";

export default defineConfig({
  integrations: [react()],
  output: "static",
  vite: {
    server: {
      proxy: {
        "/api": {
          target: apiTarget,
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ""),
        },
      },
    },
  },
});
