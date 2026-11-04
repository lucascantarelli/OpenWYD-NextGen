# syntax=docker/dockerfile:1
# Portal de administração (Next.js, BFF gRPC). O contexto é a raiz do repo
# (docker compose: context: . / dockerfile: portal/Dockerfile) porque a imagem
# precisa de portal/ e de api/web/v1/web.proto.
FROM node:22-alpine AS deps
WORKDIR /app
COPY portal/package.json portal/package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY portal/ ./
# O build coleta dados das rotas e importa grpc.ts, que carrega o proto.
COPY api/web/v1/web.proto /app/api/web/v1/web.proto
ENV WEB_API_PROTO=/app/api/web/v1/web.proto
RUN npm run build

# Runtime: apenas o standalone + estáticos + o contrato gRPC.
FROM node:22-alpine AS run
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
COPY --chown=node:node --from=build /app/.next/standalone ./
COPY --chown=node:node --from=build /app/.next/static ./.next/static
COPY api/web/v1/web.proto /app/api/web/v1/web.proto
ENV WEB_API_PROTO=/app/api/web/v1/web.proto
# Auditoria em arquivo (AUDIT_LOG_DIR); o compose monta um volume aqui.
RUN mkdir -p /var/log/wyd-portal && chown node:node /var/log/wyd-portal
USER node
EXPOSE 3000
CMD ["node", "server.js"]
