# Portal de administração (WYD2)

Next.js 15 (App Router) como **BFF puro** sobre o `wyd-admin` via **HTTP/1.1 +
JSON em loopback** (`:7600`, ADR-0008). O browser nunca fala com o backend nem
vê o token de staff.

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
| `WEB_API_ADDR` | `http://127.0.0.1:7600` | URL do `wyd-admin` (loopback na v1) |
| `WEB_API_TOKEN` | — | token de staff (`authorization: Bearer`), processo→processo |
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
import { npcAdmin } from "@/server/admin-client";

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

**Não** chame o cliente admin direto de uma page ou action: sem `adminCall`
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

Falha de transporte (conexão recusada, timeout de 10 s ou 5xx) → **502**
"serviço de administração indisponível". Stack, DSN e caminho nunca vão para o
browser: `toResponse` só devolve `{ error }` com `cache-control: no-store`.

## Estrutura

```
src/server/admin-client.ts cliente HTTP/JSON do wyd-admin (server-only) + tipos
src/server/promisify.ts    fachada da chamada admin (deadline no cliente)
src/server/admin.ts        adminCall (sessão + moderator_id + resultado + auditoria)
src/server/admin-result.ts AdminResult → HTTP/mensagem (puro)
src/server/errors.ts       AdminError + toResponse
src/server/audit.ts        trilha de auditoria (console + ndjson opcional)
src/server/session.ts      cookie httpOnly HS256, getSession/requireStaff
src/server/catalog.ts      cache do catálogo de itens
src/server/guias.ts        leitura de /guias (JSON gerado a partir de docs/)
scripts/build-guias.mjs    gerador do conteúdo de /guias (npm run guias)
```

## Rodando no Docker (stack completa)

O compose da raiz sobe o portal junto com o web-api:

```bash
docker compose up -d --build portal   # http://localhost:3000
```

A imagem (`docker/Dockerfile.portal`) usa o output `standalone` do Next. O contexto de build é a
**raiz do repositório** (compose: `context: .`). No container, `WEB_API_ADDR=http://wyd-admin:7600`
e `AUDIT_LOG_DIR` vêm do compose;
`SESSION_SECRET` tem default apenas de desenvolvimento (>= 32 caracteres em produção).

### Guias públicos (`/guias`)

`/guias` e `/guias/<slug>` publicam a documentação escrita em `docs/guias/` (jogador e
administrador), `docs/sistemas/` e `docs/lore/` (história, guerras, reinos) — **os mesmos arquivos**
que alimentam o site mkdocs, sem cópia paralela. A rota é pública de propósito: não entra no
`matcher` do middleware nem no menu administrativo.

- `npm run guias` (ou `make guias`) gera `src/content/guias.json`, que é **versionado**. A imagem
  copia só `wyd-portal/`, então `docs/` não existe dentro do container — gerar no build do Next não
  funcionaria.
- O gerador reescreve link relativo para `/guias/<slug>`, converte `!!! tipo` (admonition do mkdocs)
  em citação — o `marked` não conhece essa sintaxe — e mantém como texto o link que sai do conjunto
  publicado, porque link morto é pior que texto.
- `src/server/guias.ts` expõe `listGuias`/`getGuia`/`groupGuias`; as páginas são estáticas
  (`generateStaticParams`), então o conteúdo só muda quando o JSON é regenerado.
- CI: `_portal.yml` roda `npm run guias` e `git diff --exit-code`; `docs/` alterado sem regenerar
  reprova o pipeline (`make guias-check` reproduz local).

### Logs ao vivo (`/logs`, issue 40)

O portal **não** monta o socket do Docker. No compose, quem fala com a Engine API é o sidecar
**`logpipe`** (`logpipe/`, Go stdlib): monta `/var/run/docker.sock` **read-only** e expõe só
`/status` e `/logs/<serviço>` na rede interna — sem escrita, sem `exec`, sem acesso a arquivos,
com `read_only`, `no-new-privileges` e `group_add: 0` para ler o socket rodando nonroot. O portal
autentica a sessão e repassa o stream em SSE (`/api/logs/<serviço>`, `/api/logs/status`), com
`redact` de ANSI/DSN/segredo antes de qualquer linha chegar ao browser.

Serviços aceitos: `db`, `portal` e o alvo `all` (o servidor de jogo entra com a FASE-07).
Nada vem cru do browser: whitelist fixa, `tail` 1..2000 e `since` só no formato `10m`. Sem
`LOG_PIPE_URL` (ex.: `make portal-dev` no host), o BFF volta ao `docker compose logs -f` local.

### Contas, papéis e banimento (`/contas`, issue 32)

Lista com cursor keyset (`apos`), filtro por prefixo de login, papel e estado de bloqueio; o detalhe
traz a conta, o motivo do bloqueio e a **auditoria** (`account_admin_audit`: quem, quando, antes →
depois e o motivo digitado). Criar conta já com papel é coisa de `admin`.

Toda escrita é `action → adminCall → admin-client`, com `moderator_id` vindo da sessão. As
regras **não** moram na UI nem no transporte: `crates/wyd-admin/src/api/accounts.rs` as aplica —
ninguém altera o próprio papel/bloqueio, a última conta admin ativa é protegida e `moderator` só
bloqueia `player`. O backend responde `INVALID` com uma explicação ("esta é a última conta admin
ativa — promova outra antes") e o `adminCall` repassa essa mensagem à tela.

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

A escolha de qual compose usar é `W2PP_COMPOSE_DIR` (default: o diretório pai de `wyd-portal/`).

A tela exige sessão de `moderator`/`admin` — `player` recebe 403, e sem sessão a resposta é 401
antes de qualquer conexão com o Docker (logpipe ou CLI).
