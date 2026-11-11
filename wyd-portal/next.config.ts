import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // A imagem Docker roda `.next/standalone/server.js` (wyd-portal/Dockerfile).
  output: "standalone",
  // Sem esta lista o Next bloqueia fontes e HMR como cross-origin e a tela cai
  // no fallback sem tipografia. Para testar do celular na mesma rede, suba com
  // `PORTAL_LAN_HOST=<ip-do-host> npm run dev` (ex.: 192.168.100.7) e abra
  // `http://<ip-do-host>:3000` no celular.
  allowedDevOrigins: [
    "127.0.0.1",
    "localhost",
    ...(process.env.PORTAL_LAN_HOST ? [process.env.PORTAL_LAN_HOST] : []),
  ],
  experimental: {
    serverActions: { bodySizeLimit: "2mb" },
  },
};

export default nextConfig;
