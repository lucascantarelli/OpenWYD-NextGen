# Contrato gRPC do web-api (`web.proto`)

Contrato do web-api Go arquivado (2046 linhas, pacote `web.v1`), consumido por
`wyd-portal/src/server/grpc.ts` via `WEB_API_PROTO` e copiado para a imagem
do portal (`docker/Dockerfile.portal`).

## Estado (2026-11-07, revisado)

`web.proto` foi **restaurado do histórico git** (`b0a0394^`, pré-arquivamento):
o commit de arquivamento do Go esvaziou este diretório e o portal quebrou em
silêncio (teste `route.test.ts` + `COPY` do Dockerfile + runtime gRPC). Com o
arquivo de volta, `npx vitest run` passa 118/118 e o build da imagem volta a
levar o contrato.

Regra: este arquivo é o contrato vigente até o transporte Rust (FASE-07-4) o
substituir. Mudança de contrato exige regenerar e rodar `portal-test`.

## Como validar

```bash
cd portal && npx vitest run src/app/api/search/accounts/route.test.ts
docker compose build portal   # requer daemon + rede p/ node:22-alpine
```
