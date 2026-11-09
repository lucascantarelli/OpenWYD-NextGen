import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // A imagem Docker roda `.next/standalone/server.js` (wyd-portal/Dockerfile).
  output: "standalone",
  // O dev server é aberto em 127.0.0.1:3100; sem esta lista o Next bloqueia
  // fontes e HMR como cross-origin e a tela cai no fallback sem tipografia.
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  experimental: {
    serverActions: { bodySizeLimit: "2mb" },
  },
};

export default nextConfig;
