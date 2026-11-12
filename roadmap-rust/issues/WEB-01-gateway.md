# WEB-01 — Gateway Next.js ↔ `wyd-admin` (transporte + contrato)

> **ID canônico:** `WEB-01`
> **ID histórico:** `FASE-07-1`
> **Estado 2026-11-12:** 🚧 · ver `roadmap-rust/index.md`.

**Fase:** 7 · **Depende de:** PROC-16 · **Bloqueia:** WEB-02, WEB-03, WEB-04 · **Esforço:** M

## Objetivo

Implementar o transporte e o contrato mínimo do `wyd-admin` que o portal consome,
com auth de staff e auditoria — **HTTP/1.1 + JSON em loopback**, conforme
**ADR-0008** (gRPC/proto fora da árvore).

## Contexto / evidência

- `pre-release-rust.md` FASE 7 §7.1 planejava "gRPC/REST" sem escolher; a escolha
  está em `docs/arquitetura/adr/0008-transporte-cold-path-http-json.md`.
- O portal **não tinha** backend: o antigo `grpc.ts` apontava para um
  `admin.proto` inexistente; 16 serviços exportados, todos unários; `admin.ts`
  já impõe sessão, `moderator_id` da sessão, deadline, auditoria e `AdminResult`.
  Desde 2026-11-08 o BFF fala HTTP/JSON por `src/server/admin-client.ts`.

## Passos

1. [x] `src/transport/` (árvore §13): `mod.rs` (rota + envelope + auth),
   `http.rs` (HTTP/1.1 mínimo em std), `json.rs` (subconjunto do contrato).
2. [x] Rotas `POST /admin/v1/{servico}/{metodo}` para os handlers existentes
   (`accounts/find`, `accounts/create`, `grants/request`), autor de `StaffSession`.
3. [x] `tests/contract.rs` (401/403/404/405/413/envelope/replay + socket loopback).
4. [x] Portal: `admin-client.ts` (fetch) mantendo as exportações de `grpc.ts`;
   `promisify.ts` sem `@grpc/grpc-js`; `@grpc/*` fora do `package.json`/lock.
5. [x] Expor os demais serviços do portal (13 restantes) sobre o mesmo roteador —
   como rotas declaradas (`SERVICOS`), com `501` explícito e auditoria `admin.pending`
   enquanto o handler de domínio de cada uma não existe (ver resultado abaixo).

## Arquivos

- Criados: `crates/wyd-admin/src/transport/{mod,http,json}.rs`,
  `crates/wyd-admin/tests/contract.rs`, `wyd-portal/src/server/admin-client.ts`.
- Alterados: `crates/wyd-admin/src/lib.rs` (fachada + reexports); no portal,
  `promisify.ts`, `trace.ts` (headers HTTP), `errors.ts` (`AdminUnavailableError`),
  25 importadores (`@/server/grpc` → `@/server/admin-client`), `trace.test.ts`,
  `admin-moderator.test.ts`, `package.json`/lock, `Dockerfile.portal` e `.env.example`.
- Removido: `wyd-portal/src/server/grpc.ts`.
- Proibido tocar: estado de personagem vivo, segredos, `moderator_id` do corpo,
  `crates/wyd-core`/`wyd-net` (transporte é Cold Path).

## Testes / verificação

```bash
cargo test -p wyd-admin        # unitários do transporte + 9 do contrato
cargo fmt --all -- --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace
npm --prefix wyd-portal run typecheck
npm --prefix wyd-portal run test -- --run
```

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - BFF, sessão, papéis, `adminCall`.
- .agents/skills/wyd-rust-db - leitura/auditoria.
- .agents/skills/software-architecture - fronteira Cold Path (ADR-0008).

## Documentação de referencia

- `docs/arquitetura/adr/0008-transporte-cold-path-http-json.md` - contrato e regras.
- `docs/arquitetura/estrutura-e-padroes.md` §13 - árvore do `wyd-admin`.
- `wyd-portal/README.md` - BFF e sessão; `AGENTS.md` §3.7 - web não escreve estado vivo.

## Subtasks

- [x] Codificar issue (núcleo Rust: auth/auditoria/grants)
- [x] Testar issue (núcleo Rust)
- [x] Revisar issue
- [x] Codificar transporte HTTP/JSON + contrato (ADR-0008)
- [x] Testar transporte (`tests/contract.rs`)
- [x] Documentar issue (fiação do portal)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

**Status:** 🚧 (2026-11-08) — transporte HTTP/JSON entregue, portal fiado em
`admin-client.ts` (sem `@grpc/*`, vitest/typecheck verdes) e **paridade de rotas fechada**
(16 serviços / 72 métodos do portal conhecidos pelo roteador, sem `404` acidental). O que
resta não é transporte: são os **handlers de domínio** dos 13 serviços de conteúdo/painel,
que hoje respondem `501` auditado.

## Resultados (transporte, ADR-0008)

- `transport/json.rs`: parse/render do subconjunto (objeto/array/string/número/
  bool/null), escapes `\uXXXX` com surrogate, `MAX_DEPTH` 32, zero `unsafe`;
  rejeita `01`, lixo após o documento e aninhamento profundo sem `panic!`.
- `transport/http.rs`: `POST` obrigatório, `Content-Length` obrigatório, recusa
  `Transfer-Encoding` (sem chunked), head ≤ 16 KiB, corpo ≤ 32 MiB, resposta com
  `cache-control: no-store`, `x-trace-id`, `connection: close`; `bind_loopback`
  só em `127.0.0.1` e `serve_conn` recusa peer não-loopback.
- `transport/mod.rs`: `POST /admin/v1/{servico}/{metodo}` → `api/*`; token
  `Bearer` via `Authenticator` (`StaticTokens` em dev/teste); `401` sem token
  (não audita), `403` de papel vira `200` + `result: 2` (recusa de negócio) com
  `auth.denied` auditado; `404` método desconhecido, `400` JSON inválido,
  `405`/`413` de transporte; envelope `result` 1..5 + `trace_id`; futuro pendente
  responde `500` (runtime async segue na ADR-0006).
- `tests/contract.rs`: 9 casos — 401 sem auditoria, player 403 auditado, staff
  `accounts.find` com envelope/trace, moderador não cria staff, grant idempotente
  (replay devolve mesmo `grant_id`), params inválidos → `result: 3` sem auditar,
  `404`/`400`, `405`/`413`, e round-trip real em socket loopback.
- Gate: `fmt` 0, `clippy -D warnings` 0, `cargo test --workspace` 22 suites ok.

## Resultados (fiação do portal, 2026-11-08)

- `admin-client.ts`: `fetch` para `WEB_API_ADDR` (default `http://127.0.0.1:7600`),
  `POST /admin/v1/{servico}/{metodo}`, `authorization: Bearer $WEB_API_TOKEN`,
  `x-trace-id`, `AbortSignal.timeout(10 s)`, `cache: no-store`; mantém as 16
  exportações e todos os tipos de `grpc.ts`, agora com `svc<T>()` por serviço.
  Remove `moderator_id`/`moderatorId` do corpo antes de enviar (autor é da sessão).
- Erros: `AdminUnavailableError` (conexão recusada/timeout/5xx/resposta inválida)
  vira 502 em `toResponse`; 4xx de transporte vira `AdminError` com o status.
- `promisify.ts`: fachada `(req, traceId) => Promise<Res>`, sem `@grpc/grpc-js`;
  deadline mora no cliente. `trace.ts`: `traceHeaders()` devolve header puro.
- `package.json` sem `@grpc/grpc-js`/`@grpc/proto-loader`; lockfile atualizado;
  `Dockerfile.portal` não copia mais `.proto`; `.env.example` documenta
  `WEB_API_ADDR` (URL) + `WEB_API_TOKEN`.
- Prova: `npm run typecheck` 0; `npm run test -- --run` 118/118 (16 arquivos).
- Pendência honesta: `StaticTokens` não valida HMAC — o portal manda o token de
  dev até a fiação cripto do JWT; rotas ainda ausentes respondem 404 de transporte.

## Resultados (paridade de rotas, 2026-11-08)

O roteador passou a conhecer **toda** a superfície que o portal declara, não só as rotas
que já tinham handler:

- `transport/mod.rs` ganhou `SERVICOS` (16 serviços / 72 métodos) e `conhecido()`; a
  tabela é a lista do `admin-client.ts` já convertida para `snake_case` (o `toSnake` do
  cliente). Era o ponto cego: método declarado no portal e ausente do roteador virava
  `404` de transporte, indistinguível de nome inválido.
- Método **declarado sem handler** → `501` com a rota no corpo (`"contrato
  npc_admin.list_npcs existe no portal e ainda não tem handler no control plane"`) e
  auditoria `admin.pending` com autor da sessão. `404` continua reservado a contrato
  violado (serviço/método fora da tabela). Não inventei código novo em `AdminResult`
  (o portal mapeia valor desconhecido para 500): `501` é falha de transporte e o BFF já a
  traduz em `AdminUnavailableError` → `502` na tela.
- `tests/contract.rs` ganhou dois testes de paridade:
  - `cada_metodo_do_portal_tem_rota_no_control_plane` — extrai a superfície do
    TypeScript com `include_str!` (se o arquivo mudar de lugar, o build quebra) e falha
    se o roteador não conhecer um método, se sobrar rota que o portal não declara mais,
    ou se a contagem divergir (16 serviços / 72 métodos).
  - `rota_do_portal_sem_handler_e_501_auditado_e_nunca_404` — despacha as 72 rotas com
    sessão de staff: cada uma responde `501` (ou `200` quando o handler existir), a
    mensagem nomeia a rota e as 72 pendências ficam auditadas com o autor da sessão.
- Prova: `cargo test -p wyd-admin` → 28 unitários + 11 (contrato) + 6 (painel) + 4
  (unauthorized), 0 falhas; `cargo fmt --check` 0; `cargo clippy --workspace
  --all-targets -D warnings` 0; `cargo test --workspace` 0 (37 alvos, 429 testes).
- **Pendência declarada:** os 13 serviços (npc, conteúdo/rates, drop, daily, donate,
  revenue, world event/state, mob, attribute map, teleport, item effect, account admin,
  character admin) têm **rota e resposta determinística**, mas não têm leitura/escrita no
  Postgres — cada um exige `api/<dominio>.rs` + porta de repositório. Nada disso é
  pré-requisito do transporte e nenhum deles escreve estado vivo (invariante AGENTS §3.7).

## Critérios de aceite

- [x] Contrato mínimo decidido e registrado (ADR-0008, rota/params/resposta).
- [x] Transporte HTTP/JSON implementado com `tests/contract.rs` verde.
- [x] Portal opera sem Go, usando `admin-client.ts` (nenhum `@grpc/*`).
- [x] Auditoria com autor e timestamp em toda ação.

## Riscos / rollback

- Risco: paridade incompleta entre rota e método do portal → mitigação:
  `tests/contract.rs` cobre cada rota implementada; 13 serviços restantes
  entram um a um com o mesmo teste.
- Risco: token verificado por registro explícito em vez de HMAC (sem dep de
  cripto) → mitigação: `Authenticator` é porta; a fiação do portal liga a
  verificação real do JWT sem mudar o transporte.
- Rollback: `git checkout -- crates/wyd-admin`; o portal continua no cliente
  gRPC lazy até a fiação terminar.

## Resultados (binário `wyd-gateway` em Docker, 2026-11-08 — ADR-0022)

O transporte já existia; faltava o **serviço**. Agora existe binário e ele roda no compose:

- `crates/wyd-admin/src/main.rs`: config por env (`WYD_GATEWAY_BIND`, `DATABASE_URL`,
  `WEB_API_TOKEN` — os dois últimos obrigatórios), pool `sqlx` com `PoolConfig::local()`,
  `SELECT 1` no boot, `GET /health` (200/503 conforme banco), `GET /metrics` (texto Prometheus
  0.0.4), `POST /admin/v1/{servico}/{metodo}` pelo roteador existente e `--healthcheck`
  (TCP + `GET /health`, exit 0 só em 200) para imagem distroless.
- `crates/wyd-admin/src/metrics.rs`: `wyd_gateway_requests_total{result="ok|business|transport|unauthorized"}`
  e `wyd_gateway_uptime_seconds`.
- `crates/wyd-admin/src/api/catalog.rs` + `Router::with_catalog`: `item_catalog/list_items`
  responde da foto real do Postgres (3.220 itens) — rota que `/api/health` do portal usa.
  **Lacuna:** `icon_key/slot_mask/slots/grade/mesh/texture/icon_url` não existem em
  `item_catalog`; a resposta devolve vazio/0 e a lacuna está no ADR-0022.
- `crates/wyd-db`: `PgBackend::ping()` (health real) e `catalog_snapshot()` (leitura única),
  sem SQL saindo do adapter.
- `docker/Dockerfile.gateway` + serviço `gateway` no compose (db saudável, sem porta na base),
  `portal` com `depends_on: gateway` e `WEB_API_TOKEN`.
- O schema entra no boot do próprio gateway/canal (ADR-0024): o `db` sobe vazio de propósito e
  o runner `wyd_db::migrate` aplica a série antes de abrir o pool (ver
  `docs/rust/05-stack-docker.md` §5).

**Ainda aberto nesta issue:** handlers de domínio das 13 rotas de conteúdo/painel (seguem
`501` auditado). A escrita de auditoria em `audit_log` foi fechada na WEB-03 (`PersistentSink`).

**Revisão 2026-11-09:** o que resta não é transporte nem uma issue só — é conteúdo/operação,
e foi **dividido** para virar entrega pequena: `WEB-05` (rotas de leitura de conteúdo) e
`WEB-06` (rotas de operação com escrita auditada). A WEB-01 fecha quando as duas estiverem
verdes e o `admin.pending` não for mais alcançado por nenhuma rota conhecida.

## Revisão de prova (2026-11-12)

- **Provado:** os 44 métodos que o portal declara têm handler no roteador; o teste
  `rota_do_portal_sem_handler_e_501_auditado_e_nunca_404` encontra **0** `admin.pending`.
  `WEB-05` e `WEB-06` estão concluídas e o gateway roda no compose.
- **Residual:** a rota **legada** `daily_reward_admin/set_reward_item_enabled`, que o portal
  não declara mais, continua em `SERVICOS` e responde `501` auditado. Ela existe porque
  `daily_reward_item` não tem coluna de liga/desliga e a série proíbe `ALTER` no `up`.
- **Decisão necessária:** remover a rota legada do contrato (vira `404` para chamadas antigas)
  ou criar a coluna em migration nova. Nenhuma das duas é transporte; por isso a WEB-01 segue
  aberta, sem trabalho de handler pendente.
