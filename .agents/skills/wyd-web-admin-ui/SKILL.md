---
name: wyd-web-admin-ui
description: Constrói e mantém o portal Next.js do WYD (BFF puro sobre `wyd-admin`, sessão httpOnly, overlays auditados). Use ao criar tela, Route Handler, cliente de backend, sessão ou overlay de conteúdo.
---

# Portal BFF sobre `wyd-admin` (Next.js)

Norma: `wyd-portal/README.md` (adminCall, sessão, `toResponse`),
`docs/arquitetura/adr/0008-transporte-cold-path-http-json.md` (HTTP/JSON, sem gRPC/proto)
e `docs/arquitetura/modelo-banco.md` (`delivery_queue`/auditoria). O contrato completo está
lá; não duplique rotas nem envelope na skill.

## Decisões específicas

- Page/action nunca chama backend direto: use `adminCall`, que deriva `moderator_id` da
  sessão, aplica deadline, traduz `AdminResult` e audita.
- Browser não vê banco/DSN/stack. `toResponse` devolve só `{ error }` com
  `cache-control: no-store`; o cliente admin é server-only.
- Escrita de jogo continua na `delivery_queue`; config nova entra com migration + flag OFF
  + applier no boot. A web nunca escreve conteúdo vivo.
- Papel na sessão é UX; `wyd-admin` revalida `account.role` em toda ação. Sem `.proto`,
  `protoc` ou gRPC: o transporte é HTTP/1.1 + JSON em loopback (ADR-0008).
- UI em português, sentence case; nome antes de ID, erro com ação de recuperação e a11y
  vigente. Reuse os componentes/tokens do portal.

## Prova

```bash
npm --prefix wyd-portal run typecheck && npm --prefix wyd-portal run test -- --run
cargo test -p wyd-admin
make stack-config
```
