# Issue 07 — Portal web e ecossistema digital

**Fase:** 7 · **Depende de:** FASE-04 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Integrar o portal Next.js (mantido fora do arquivo) ao control plane Rust
(`wyd-admin`) via APIs de controle, com E2E Playwright dos fluxos críticos e
auditoria de ações administrativas.

## Contexto / evidência

O portal consome hoje um contrato de serviços admin (48 módulos via
`wyd-portal/src/server/admin-client.ts`, desde 2026-11-08) e **não há backend Rust
atrás dele além de 3 rotas**: o Go foi
removido e o cliente ficou lazy (só falha no primeiro uso). Regra permanente:
portal é BFF puro — lê e envia intenções, nunca escreve estado de personagem
vivo (`AGENTS.md`). Transporte e contrato decididos na **ADR-0008**: HTTP/1.1 +
JSON em loopback, `POST /admin/v1/{servico}/{metodo}`, envelope `AdminResult`
(`ADMIN_SCOPE` já existe em `crates/wyd-admin/src/lib.rs`).

## Passos

1. Mapear as RPCs que o portal consome hoje (`wyd-portal/src/server/admin.ts`,
   `toResponse`, sessão/papéis) e listar o contrato mínimo do `wyd-admin`.
2. Implementar o transporte e os endpoints de controle em `crates/wyd-admin/src/`
   (árvore §13 + ADR-0008) com auth (sessão de staff, `moderator_id` da sessão —
   nunca do corpo).
3. Registrar auditoria de toda ação admin (quem, o quê, quando).
4. E2E Playwright: login, área admin, fluxo de loja/donate (rastreável, sem
   duplicidade de entrega).
5. `make check` + `portal-test` (tsc + vitest) no portal.

## Arquivos

- Criados (árvore normativa em `estrutura-e-padroes.md` §13): `src/error.rs`,
  `src/api/*.rs`, `src/transport/{mod,http,json}.rs`, `src/auth/*.rs`,
  `src/audit/log.rs`, `src/service/grant_service.rs`, `tests/unauthorized.rs`,
  `tests/contract.rs`, specs E2E no portal.
- Alterados: `crates/wyd-admin/Cargo.toml` (deps via ADR), `wyd-portal/src/**`
  (somente adaptação ao novo backend, sem mudar regras de negócio).
- Proibido tocar: estado de personagem vivo, segredos,
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
- .agents/skills/frontend-design - direção visual e revisão de UI/UX do portal.

## Documentação de referencia

- wyd-portal/README.md - BFF, telas, overlays, sessão.
- pre-release-rust.md (FASE 7) - gateway Next.js e E2E.
- crates/wyd-admin/src/ - web-api Go como referência de contrato.
- docs/cliente/ + crates/wyd-admin/src/api/ - catálogo de avisos (Language.txt).

## Subtasks

- [x] Codificar issue (núcleo Rust: contrato + auditoria + grants)
- [x] Testar issue (núcleo Rust)
- [x] Revisar issue
- [ ] Documentar issue (portal + E2E pendentes de transporte)
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Portal opera contra `wyd-admin` sem falar com Go.
- [x] Auditoria registra toda ação admin com autor e timestamp.
- [ ] E2E (auth, admin, loja) verde; evidência anexada.

**Status:** 🚧 (2026-11-07) — núcleo Rust entregue e verde; contrato/transporte
decididos na ADR-0008 (HTTP/1.1 + JSON em loopback). Portal+E2E aguardam a
implementação do transporte (runtime async segue na ADR-0006).

## Resultados (núcleo Rust)

- `src/error.rs` (`AdminError`→HTTP 401/403/404/422/409, espelha `AdminResult`;
  `From<DbError>` sem vazar detalhe; infra vira mensagem genérica).
- `src/auth/` (`Session`→`StaffSession`: player é 403; `moderator_id` só da
  sessão; `may_target`/`may_change_role` — moderador só mira jogador).
- `src/audit/log.rs` (`AuditEntry` quem/o quê/quando; `AuditSink` porta +
  `VecSink`; negada autenticada vira `auth.denied`, 401 sem autor não audita).
- `src/api/` (`find/create_account`, `request_grant`; `Option<Session>` para o
  401 ser testável) + `src/service/grant_service.rs` (idempotência por chave:
  replay devolve o recibo original sem re-enfileirar).
- `tests/unauthorized.rs`: 401 sem tocar repo/auditoria; 403 + trilha;
  fluxo staff ponta a ponta (`find`→`grant`→`replay`, 3 auditorias do autor).
- Gate: `fmt` + `clippy -D warnings` + `cargo test --workspace` verdes
  (`wyd-admin`: 11 lib + 4 integração); zero `unsafe`; wyd-portal/ intocado de
  propósito (apontá-lo para backend sem transporte quebraria o portal Go vivo).
- Pendências honestas: troca de papel/bloqueio (repo `pg` + coluna, PROC-16),
  dedup durável da chave (UNIQUE no `pg`), transporte HTTP/JSON (ADR-0008),
  telas WEB-02/WEB-03 e E2E WEB-04.

## Riscos / rollback

- Risco: entrega duplicada de benefício da loja → mitigação: idempotência por
  transação rastreável + teste de duplicidade.
- Rollback: contrato versionado; portal pinado à versão anterior do backend.
