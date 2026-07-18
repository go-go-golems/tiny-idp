import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: "/static/",
  plugins: [react()],
  build: {
    outDir: "dist",
    emptyOutDir: true
  },
  server: {
    proxy: {
      "/api": "https://127.0.0.1:19443",
      "/auth": "https://127.0.0.1:19443",
      "/idp": "https://127.0.0.1:19443"
    }
  }
});
