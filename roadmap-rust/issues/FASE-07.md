# Issue 07 — Portal web e ecossistema digital

**Fase:** 7 · **Depende de:** FASE-04 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Integrar o portal Next.js (mantido fora do arquivo) ao control plane Rust
(`wyd-admin`) via APIs de controle, com E2E Playwright dos fluxos críticos e
auditoria de ações administrativas.

## Contexto / evidência

O portal fala hoje com o `webserver` Go via gRPC (`portal/README.md`,
`.deprecated-go/webserver/`); com o Go arquivado, o BFF precisa de um backend
Rust. Regra permanente: portal é BFF puro — lê e envia intenções, nunca escreve
estado de personagem vivo (`AGENTS.md`). Destino: `crates/wyd-admin` expõe
gRPC/REST de controle (`ADMIN_SCOPE` já existe em `crates/wyd-admin/src/lib.rs`).

## Passos

1. Mapear as RPCs que o portal consome hoje (`portal/src/server/admin.ts`,
   `toResponse`, sessão/papéis) e listar o contrato mínimo do `wyd-admin`.
2. Implementar os endpoints de controle em `crates/wyd-admin/src/` com auth
   (sessão de staff, `moderator_id` da sessão — nunca do corpo).
3. Registrar auditoria de toda ação admin (quem, o quê, quando).
4. E2E Playwright: login, área admin, fluxo de loja/donate (rastreável, sem
   duplicidade de entrega).
5. `make check` + `portal-test` (tsc + vitest) no portal.

## Arquivos

- Criados (árvore normativa em `estrutura-e-padroes.md` §13): `src/error.rs`,
  `src/api/*.rs`, `src/auth/*.rs`, `src/audit/log.rs`,
  `src/service/grant_service.rs`, `tests/unauthorized.rs`, specs E2E no portal.
- Alterados: `crates/wyd-admin/Cargo.toml` (deps via ADR), `portal/src/**`
  (somente adaptação ao novo backend, sem mudar regras de negócio).
- Proibido tocar: estado de personagem vivo, `.deprecated-go/`, segredos,
  `moderator_id` vindo do corpo da request.

## Testes / verificação

```bash
cargo test -p wyd-admin
make check
```

Saída esperada: `test result: ok`; E2E crítico verde (log do Playwright anexado);
ação admin sem sessão retorna 401 (teste dedicado).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-web-admin-ui - BFF, sessão, papéis, auditoria.
- .agents/skills/wyd-rust-db - leitura/auditoria no banco.
- .agents/skills/frontend-design - direção visual do portal.
- .agents/skills/ui-ux-practices - revisão de UX.

## Documentação de referencia

- portal/README.md - BFF, telas, overlays, sessão.
- pre-release-rust.md (FASE 7) - gateway Next.js e E2E.
- .deprecated-go/webserver/ - web-api Go como referência de contrato.
- .deprecated-go/docs/migration/game-messages.md - catálogo de avisos (Language.txt).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Portal opera contra `wyd-admin` sem falar com Go.
- [ ] Auditoria registra toda ação admin com autor e timestamp.
- [ ] E2E (auth, admin, loja) verde; evidência anexada.

## Riscos / rollback

- Risco: entrega duplicada de benefício da loja → mitigação: idempotência por
  transação rastreável + teste de duplicidade.
- Rollback: contrato versionado; portal pinado à versão anterior do backend.
