import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const apiTarget =
  process.env.AREAFLOW_API_URL ??
  process.env.AREFLOW_API_URL ??
  "http://127.0.0.1:3847";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
      },
    },
  },
});
