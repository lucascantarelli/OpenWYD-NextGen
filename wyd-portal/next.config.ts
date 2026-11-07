import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // A imagem Docker roda `.next/standalone/server.js` (wyd-portal/Dockerfile).
  output: "standalone",
  // O BFF roda no servidor; o bundle do cliente nunca deve conter grpc-js.
  serverExternalPackages: ["@grpc/grpc-js", "@grpc/proto-loader"],
  experimental: {
    serverActions: { bodySizeLimit: "2mb" },
  },
};

export default nextConfig;
