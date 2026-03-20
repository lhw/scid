import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    host: "0.0.0.0",
    allowedHosts: ["scid.my", "id.scid.my"],
    proxy: {
      "/api": {
        target: process.env.VITE_API_BASE ?? "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
