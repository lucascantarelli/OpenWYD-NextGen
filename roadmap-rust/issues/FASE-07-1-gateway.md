# Issue 07-1 — Gateway Next.js ↔ `wyd-admin` (transporte + contrato)

**Fase:** 7 · **Depende de:** 04-3 · **Bloqueia:** 07-2, 07-3, 07-4 · **Esforço:** M

## Objetivo

Implementar o transporte e o contrato mínimo do `wyd-admin` que o portal consome,
com auth de staff e auditoria — **HTTP/1.1 + JSON em loopback**, conforme
**ADR-0008** (gRPC/proto fora da árvore).

## Contexto / evidência

- `pre-release-rust.md` FASE 7 §7.1 planejava "gRPC/REST" sem escolher; a escolha
  foi registrada na `docs/arquitetura/adr/0008-transporte-cold-path-http-json.md`.
- O portal **não tem** backend hoje: `wyd-portal/src/server/grpc.ts` aponta para
  `crates/wyd-admin/proto/admin.proto` (inexistente) e é lazy; 16 serviços
  exportados, todos unários; `admin.ts` já impõe sessão, `moderator_id` da sessão,
  deadline, auditoria e `AdminResult` → HTTP.
- Núcleo Rust entregue: `crates/wyd-admin/src/{auth,audit,api,service}` +
  `tests/unauthorized.rs` (401 sem sessão, 403 sem papel, auditoria registrada).

## Passos

1. Implementar `src/transport/` (árvore §13): `mod.rs` (roteador
   `servico/metodo` → `api/*` + envelope `result`/`trace_id`), `http.rs`
   (HTTP/1.1 mínimo em std, bind `127.0.0.1:7600`, `Content-Length`, limite de
   32 MiB, `Connection: close`), `json.rs` (subconjunto do contrato, com testes).
2. Expor os métodos já mapeados do portal (`grpc.ts`) nas rotas
   `POST /admin/v1/{servico}/{metodo}`, derivando autor de `StaffSession`.
3. Testes (`tests/contract.rs`): rota→método, `401` sem token, `403` player com
   `result: 2`, método desconhecido `404`, corpo grande `413`, `cache-control:
   no-store` + `x-trace-id` ecoado.
4. Adaptar o portal: `admin-client.ts` (fetch) mantendo as exportações de
   `grpc.ts`; `promisify.ts` sem `@grpc/grpc-js`; remover as deps gRPC ao final.
5. `cargo test -p wyd-admin` + `make check` + `npm --prefix wyd-portal run test -- --run`.

## Arquivos

- Criados: `crates/wyd-admin/src/transport/{mod,http,json}.rs`,
  `crates/wyd-admin/tests/contract.rs`, `wyd-portal/src/server/admin-client.ts`.
- Alterados: `crates/wyd-admin/src/lib.rs` (fachada), `src/api/*` (assinatura por
  rota), `wyd-portal/src/server/{grpc.ts→admin-client.ts,promisify.ts}` e os
  imports de `@/server/grpc`; `wyd-portal/package.json` (sai `@grpc/*`).
- Proibido tocar: estado de personagem vivo, segredos, `moderator_id` do corpo,
  `crates/wyd-core`/`wyd-net` (transporte é Cold Path).

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
npm --prefix wyd-portal run test -- --run
```

Saída esperada: `test result: ok` com 401/403/envelope/limites; portal verde com
o cliente JSON; nenhuma referência a `.proto` na árvore.

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
- [ ] Documentar issue (transporte + portal + E2E)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

**Status:** 🚧 (2026-11-07) — contrato decidido (ADR-0008) e backend do contrato
verde; falta implementar `src/transport/` e fiar o portal no cliente JSON.

## Critérios de aceite

- [x] Contrato mínimo decidido e registrado (ADR-0008, rota/params/resposta).
- [ ] Transporte HTTP/JSON implementado com `tests/contract.rs` verde.
- [ ] Portal opera sem Go, usando `admin-client.ts` (nenhum `@grpc/*`).
- [x] Auditoria com autor e timestamp em toda ação.

## Riscos / rollback

- Risco: paridade incompleta entre rota e método do portal → mitigação:
  `tests/contract.rs` cobre cada serviço exportado em `grpc.ts`.
- Risco: paridade Rust↔TS sem codegen → mitigação: contrato deste ADR + teste por
  rota e revisão na troca de cliente.
- Rollback: `git checkout -- crates/wyd-admin wyd-portal`; o portal continua no
  cliente gRPC lazy até a fiação terminar.
