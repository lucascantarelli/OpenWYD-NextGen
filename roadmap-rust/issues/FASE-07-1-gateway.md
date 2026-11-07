# Issue 07-1 — Gateway Next.js ↔ `wyd-admin` (contrato)

**Fase:** 7 · **Depende de:** 04-3 · **Bloqueia:** 07-2, 07-3, 07-4 · **Esforço:** M

## Objetivo

Definir e implementar o contrato mínimo do `wyd-admin` (gRPC/REST de controle)
que o portal consome, com auth de staff e auditoria.

## Contexto / evidência

`pre-release-rust.md` FASE 7 §7.1. Portal hoje fala com `webserver` Go via
`portal/src/server/admin.ts` (ver `.deprecated-go/webserver/`). Regra: BFF puro,
`moderator_id` da sessão, nunca do corpo (`AGENTS.md`). Base: `ADMIN_SCOPE` em
`crates/wyd-admin/src/lib.rs`.

## Passos

1. Mapear as RPCs que o portal consome hoje (`admin.ts`, `toResponse`, sessão/papéis).
2. Implementar os endpoints mínimos em `crates/wyd-admin/src/` com auth
   (sem sessão → 401) e auditoria (quem/o quê/quando).
3. Teste: ação admin sem sessão retorna 401; com sessão registra auditoria.
4. `cargo test -p wyd-admin` + `make check`.

## Arquivos

- Criados: handlers + testes em `crates/wyd-admin/src/`.
- Alterados: `crates/wyd-admin/Cargo.toml` (deps via ADR), `portal/src/**`
  (somente adaptação ao novo backend).
- Proibido tocar: estado de personagem vivo, segredos, `moderator_id` do corpo.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok` incluindo 401 sem sessão e auditoria registrada.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - BFF, sessão, papéis.
- .agents/skills/wyd-rust-db - leitura/auditoria.
- .agents/skills/software-architecture - fronteira portal↔admin.

## Documentação de referencia

- portal/README.md - BFF e sessão.
- pre-release-rust.md (FASE 7 §7.1).
- .deprecated-go/webserver/ - web-api Go como referência de contrato.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

**Status:** ✅ (2026-11-07, backend do contrato: `wyd-admin` com auth/auditoria/grants + `tests/unauthorized.rs` verde; fiação do portal aguarda transporte).

## Critérios de aceite

- [x] Contrato mínimo implementado no backend (`wyd-admin`).
- [ ] Portal opera sem Go (fiação aguarda transporte async — portal intocado de propósito).
- [x] Auditoria com autor e timestamp em toda ação.

## Riscos / rollback

- Risco: paridade incompleta com RPCs legadas → mitigação: tabela legado→novo
  antes de implementar.
- Rollback: contrato versionado; portal pinado à versão anterior.
