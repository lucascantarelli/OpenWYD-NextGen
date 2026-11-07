# Issue 04 — Banco de dados e persistência (`wyd-db`)

**Fase:** 4 · **Depende de:** FASE-01 · **Bloqueia:** FASE-07 · **Esforço:** L

## Objetivo

Subir o PostgreSQL do zero via migrations versionadas e acesso async (`sqlx`) fora
do tick loop, com integridade referencial e seeds reproduzíveis dev/prod.

## Contexto / evidência

O legado persistia `.dat` instáveis em disco (`dbServer`); a fase Go usava
`pgx` + migrations em `internal/migrations/` (arquivadas em
`.deprecated-go/internal/migrations/`). Destino: `crates/wyd-db` com `sqlx`,
`sqlx-cli`/`refinery`, Redis transacional onde fizer sentido
(`pre-release-rust.md` FASE 4). Esqueleto atual: `crates/wyd-db/src/lib.rs`
(validação de scheme) — `cargo test -p wyd-db` verde.

## Passos

1. Modelar contas, personagens, inventários, baús, guildas e auditoria com FKs,
   CHECKs e índices para o caminho quente (documentar em `docs/arquitetura/`).
2. Criar `crates/wyd-db/migrations/` versionadas + seeds (`sqlx-cli`), com
   up/down e rollback testado.
3. Implementar pool async em `crates/wyd-db/src/` (timeouts, transações curtas,
   sem I/O bloqueante no tick).
4. Teste de integração contra Postgres descartável (container), validando
   constraints (FK violada deve falhar).
5. `make check`.

## Arquivos

- Criados (árvore normativa em `estrutura-e-padroes.md` §12):
  `migrations/NNNN_descricao.sql`, `src/error.rs`, `src/model/*.rs`,
  `src/repo/*.rs`, `src/pool/config.rs`, `src/transfer/delivery_queue.rs`,
  `tests/constraints.rs`, doc do modelo em `docs/arquitetura/`.
- Alterados: `crates/wyd-db/Cargo.toml` (sqlx + refinery via ADR).
- Proibido tocar: `crates/wyd-core/src/` (engine não conhece SQL),
  `.deprecated-go/`, `client/`, segredos (nunca versionar DSN real).

## Testes / verificação

```bash
cargo test -p wyd-db
make check
```

Saída esperada: `test result: ok`; migrations aplicam do zero e rollback funciona
(log do `sqlx migrate` anexado).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-db - modelagem, migrations, sqlx.
- .agents/skills/rust-code-quality - erros tipados, sem segredo em log.
- .agents/skills/software-architecture - fronteira engine↔persistência.

## Documentação de referencia

- docs/rust/01-dependencias.md - sqlx/refinery/redis.
- pre-release-rust.md (FASE 4) - modelagem e migrations.
- .deprecated-go/docs/analysis/banco-de-dados.md - análise do banco legado.
- .deprecated-go/internal/migrations/ - migrations Go como referência de schema.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Migrations sobem do zero e revertem sem perda (evidência anexada).
- [x] FKs/CHECKs validados por teste (violação falha como esperado).
- [x] Nenhum acesso a banco dentro do tick; DSN real nunca versionado.

**Status:** ✅ (2026-11-07)

## Resultados

- `docs/arquitetura/modelo-banco.md`: ER, 8 tabelas com FKs/CHECKs/índices,
  6 decisões; lacuna do arquivo registrada (pastas do legado vazias).
- 6 migrations reversíveis (`-- migrate:up/down`): contas, personagens
  (soft-delete), inventário+baú (normalizado), guildas, auditoria, delivery_queue.
- `src/{error.rs (DbError sem segredo), model/, repo/ (traits Accounts/
  Characters/Grants), pool/config.rs, transfer/delivery_queue.rs (mpsc
  limitada + drain)}`; `tests/constraints.rs` (marcadores, reversibilidade,
  FKs, seeds); `SCHEMA_VERSION=6`; `Cargo.toml` intocado (sqlx exige rede+ADR).
- Prova ao vivo em PG17 descartável: 6 ups → 8 tabelas; FK/CHECK/UNIQUE
  violados falham; revert → 0 tabelas; re-apply + `seeds/dev.sql` OK.
  Container e senha efêmera destruídos; DSN nunca versionado.
- Escopo consciente: implementação `pg`/pool async (04-3) aguarda rede + ADR;
  traits e fila prontos para ela.

## Riscos / rollback

- Risco: perda/divergência de dados na migração → mitigação: backup + dry-run +
  rollback testado antes de qualquer ambiente real.
- Rollback: `sqlx migrate revert` + `git checkout -- crates/wyd-db/`.
