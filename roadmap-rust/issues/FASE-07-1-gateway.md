# Issue 07-1 — Gateway Next.js ↔ `wyd-admin` (transporte + contrato)

**Fase:** 7 · **Depende de:** 04-3 · **Bloqueia:** 07-2, 07-3, 07-4 · **Esforço:** M

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
5. [ ] Expor os demais serviços do portal (13 restantes) sobre o mesmo roteador.

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

**Status:** 🚧 (2026-11-08) — transporte HTTP/JSON entregue e portal fiado em
`admin-client.ts` (sem `@grpc/*`, vitest/typecheck verdes); faltam os 13 serviços
restantes no roteador.

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
