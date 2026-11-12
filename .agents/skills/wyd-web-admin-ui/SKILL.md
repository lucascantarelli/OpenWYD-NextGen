---
name: wyd-web-admin-ui
description: Constrói e mantém o portal Next.js do WYD (BFF puro sobre `wyd-admin`, sessão httpOnly, overlays auditados). Use ao criar tela, Route Handler, cliente de backend, sessão ou overlay de conteúdo.
---

# Portal BFF sobre `wyd-admin` (Next.js)

> Sucessão PROC-18: mantido o nome; backend migrado de web-api Go para `wyd-admin` (WEB-01). Fonte: `wyd-portal/`, `crates/wyd-admin/`, `docs/rust/05-stack-docker.md`.

## Regras

1. **Toda ação admin passa por `adminCall`.** Sessão de staff, `moderator_id` da sessão, deadline, `AdminResult` → HTTP, auditoria; page/action nunca chama backend direto.
2. **Browser nunca vê banco/DSN/stack.** `toResponse` devolve só `{ error }` com `cache-control: no-store`; o cliente admin (`admin-client.ts` com `fetch`) é server-only e não usa `@grpc/*`.
3. **A web nunca escreve estado vivo.** Grants via `delivery_queue`; config nova tem migration + flag OFF + applier no boot; nunca escrever em conteúdo do jogo pela web.
4. **Autorização dupla.** Papel na sessão é UX; `wyd-admin` revalida `account.role` em toda ação e audita.
5. **Contrato `wyd-admin` = HTTP/1.1 + JSON em loopback (ADR-0008).** `POST /admin/v1/{servico}/{metodo}`, envelope `result` 1..5 + `trace_id`; `crates/wyd-admin/src/{transport,api,auth,audit,service}`. Sem `.proto`, sem `protoc`: gRPC/proto não entram na árvore.

## Verificação

```bash
npm --prefix wyd-portal run typecheck && npm --prefix wyd-portal run test -- --run
cargo test -p wyd-admin    # 401/403 + envelope do contrato (tests/contract.rs)
make stack-config          # compose base+prod válidos sem subir
```

## UI

- Português, sentence case; nome antes de ID (índice no `title`); tokens de espaço e a11y vigentes; erro sempre com ação de recuperação.
