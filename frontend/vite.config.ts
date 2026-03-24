import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    host: "0.0.0.0",
    allowedHosts: ["scid.my", "auth.scid.my", "dev.scid.my", "dev-auth.scid.my", "localhost", "127.0.0.1", "frontend"],
    proxy: {
      "/api": {
        target: process.env.VITE_API_BASE ?? "http://localhost:8080",
        changeOrigin: true,
      },
      "/config.js": {
        target: process.env.VITE_API_BASE ?? "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
