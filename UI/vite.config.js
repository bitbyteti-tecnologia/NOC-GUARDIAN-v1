import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Permitir hosts externos (produção via Nginx)
export default defineConfig({
  plugins: [react()],
  server: {
    host: true,                // escuta em todas interfaces
    port: 5173,
    strictPort: true,
    allowedHosts: [
      "nocguardian.bitbyteti.tec.br",
      "localhost",
      "127.0.0.1"
    ],
    cors: true
  }
});
