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

### Logs ao vivo (`/logs`, issue 40)

O portal **não** monta o socket do Docker. No compose, quem fala com a Engine API é o sidecar
**`logpipe`** (`logpipe/`, Go stdlib): monta `/var/run/docker.sock` **read-only** e expõe só
`/status` e `/logs/<serviço>` na rede interna — sem escrita, sem `exec`, sem acesso a arquivos,
com `read_only`, `no-new-privileges` e `group_add: 0` para ler o socket rodando nonroot. O portal
autentica a sessão e repassa o stream em SSE (`/api/logs/<serviço>`, `/api/logs/status`), com
`redact` de ANSI/DSN/segredo antes de qualquer linha chegar ao browser.

Serviços aceitos: `tmserver`, `dbserver`, `binserver`, `webserver`, `db`, `portal` e o alvo `all`.
Nada vem cru do browser: whitelist fixa, `tail` 1..2000 e `since` só no formato `10m`. Sem
`LOG_PIPE_URL` (ex.: `make portal-dev` no host), o BFF volta ao `docker compose logs -f` local.

### Contas, papéis e banimento (`/contas`, issue 32)

Lista com cursor keyset (`apos`), filtro por prefixo de login, papel e estado de bloqueio; o detalhe
traz a conta, o motivo do bloqueio e a **auditoria** (`account_admin_audit`: quem, quando, antes →
depois e o motivo digitado). Criar conta já com papel é coisa de `admin`.

Toda escrita é `action → adminCall → AccountAdminService`, com `moderator_id` vindo da sessão. As
regras **não** moram na UI nem no gRPC: `internal/store/account_admin.go` as aplica dentro da
transação — ninguém altera o próprio papel/bloqueio, a última conta admin ativa é protegida e
`moderator` só bloqueia `player`. O web-api responde `INVALID` com uma explicação ("esta é a última
conta admin ativa — promova outra antes") e o `adminCall` repassa essa mensagem à tela.

Nada aqui toca personagem: `account.role` e `account.is_blocked` são lidos no **próximo login**.
A conta nunca é apagada e senha/PIN não são redefinidos por esta tela (fora do escopo da issue).

### Faturamento (`/faturamento`, issue 31)

Leitura pura do `DonateRevenueAdminService` — nada escreve, nem linha de auditoria no banco — e
restrita a `admin`: `moderator` cai em `/sem-permissao`, porque receita e dados de pagador são
informação financeira. A janela (7/30/90 dias ou intervalo digitado) é interpretada em
`America/Sao_Paulo` e enviada em segundos Unix; a tela exibe a janela que o **servidor** aplicou.
Dinheiro (`gross_cents`, BRL) e donate (`credits`) nunca se somam, e `pending_cents` tem cartão
próprio com o aviso de que não é receita. Pedidos saem em CSV client-side (100 linhas por carga) e
a busca de conta usa `/api/search/accounts`, a mesma do `Combobox` do crédito manual em `/loja`.

A escolha de qual compose usar é `W2PP_COMPOSE_DIR` (default: o diretório pai de `portal/`).

A tela exige sessão de `moderator`/`admin` — `player` recebe 403, e sem sessão a resposta é 401
antes de qualquer conexão com o Docker (logpipe ou CLI).
