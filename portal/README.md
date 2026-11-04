# Portal de administração (WYD2)

Next.js 15 (App Router) como **BFF puro** sobre o `web-api` gRPC do
`webserver` (`:7600`). O browser nunca fala gRPC nem vê certificado.

```bash
cd portal
npm install
npm run dev          # http://127.0.0.1:3000  (login: conta com role moderator/admin)
npm run typecheck    # tsc --noEmit
npm test             # vitest
```

Variáveis de ambiente:

| Variável | Padrão | Uso |
|---|---|---|
| `WEB_API_ADDR` | `127.0.0.1:7600` | endereço do web-api |
| `WEB_API_PROTO` | `../api/web/v1/web.proto` | contrato gRPC |
| `WEB_API_TLS_{CA,CERT,KEY}` | — | mTLS; sem os três, canal insecure (dev) |
| `SESSION_SECRET` | — | assinatura HS256 do cookie `wyd_session` |
| `AUDIT_LOG_DIR` | — | opcional: grava `audit-YYYY-MM-DD.ndjson` além do log do processo |

## Regra: toda ação admin passa por `adminCall`

`src/server/admin.ts` é o único caminho para RPC de escrita/admin. Ele exige
sessão de staff, injeta `moderator_id` **da sessão** (nunca do corpo), aplica
deadline de 10s, mapeia `AdminResult` → HTTP e registra auditoria.

```ts
// src/app/api/npcs/[id]/route.ts
import { adminCall } from "@/server/admin";
import { toResponse } from "@/server/errors";
import { npcAdmin } from "@/server/grpc";

export async function DELETE(_req: Request, { params }: { params: Promise<{ id: string }> }) {
  try {
    const { id } = await params;
    const res = await adminCall(
      npcAdmin.deleteNpc.bind(npcAdmin),
      { npc_id: Number(id) },
      { action: "npc.delete", target: `npc:${id}` },
    );
    return Response.json({ ok: true, npcId: res.npc_id ?? Number(id) });
  } catch (err) {
    return toResponse(err);
  }
}
```

**Não** chame o cliente gRPC admin direto de uma page ou action: sem `adminCall`
não há `moderator_id` da sessão, nem mapeamento de `CONTENT_OWNED`, nem auditoria.

## `AdminResult` → HTTP

`src/server/admin-result.ts` (puro, testado em `admin.test.ts`):

| `AdminResult` | HTTP | Mensagem |
|---|---|---|
| `OK` (1) | 200 | — |
| `FORBIDDEN` (2) | 403 | sua conta não tem permissão para esta ação |
| `INVALID` (3) | 422 | dados inválidos para esta ação |
| `NOT_FOUND` (4) | 404 | o registro não existe mais |
| `CONTENT_OWNED` (5) | 409 | este item é conteúdo versionado: oculte em vez de apagar |
| `UNSPECIFIED` (0) / desconhecido | 500 | resultado desconhecido do web-api |

Falha de transporte (`UNAVAILABLE`, `DEADLINE_EXCEEDED`) → **502**
"serviço de administração indisponível". Stack, DSN e caminho nunca vão para o
browser: `toResponse` só devolve `{ error }` com `cache-control: no-store`.

## Estrutura

```
src/server/grpc.ts         cliente gRPC tipado (server-only) + tipos de resposta
src/server/promisify.ts    callback → Promise com deadline de 10s
src/server/admin.ts        adminCall (sessão + moderator_id + resultado + auditoria)
src/server/admin-result.ts AdminResult → HTTP/mensagem (puro)
src/server/errors.ts       AdminError + toResponse
src/server/audit.ts        trilha de auditoria (console + ndjson opcional)
src/server/session.ts      cookie httpOnly HS256, getSession/requireStaff
src/server/catalog.ts      cache do catálogo de itens
```

## Rodando no Docker (stack completa)

O compose da raiz sobe o portal junto com o web-api:

```bash
docker compose up -d --build portal   # http://localhost:3000
```

A imagem (`portal/Dockerfile`) usa o output `standalone` do Next. O contexto de build é a **raiz do
repositório** (não `portal/`) porque a imagem precisa de `api/web/v1/web.proto`. No container,
`WEB_API_ADDR=webserver:7600`, `W2PP_CONTENT=/Release` e `AUDIT_LOG_DIR` vêm do compose;
`SESSION_SECRET` tem default apenas de desenvolvimento (>= 32 caracteres em produção).
