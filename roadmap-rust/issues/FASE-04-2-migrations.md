# Issue 04-2 — Migrations versionadas e seeds reproduzíveis

**Fase:** 4 · **Depende de:** 04-1 · **Bloqueia:** 04-3 · **Esforço:** M

## Objetivo

Subir o PostgreSQL do zero via migrations versionadas (up/down testado) com
seeds dev/prod reproduzíveis.

## Contexto / evidência

`pre-release-rust.md` FASE 4 §4.2: `sqlx-cli`/`refinery` no pipeline de build.
Base: `crates/wyd-db/src/lib.rs` (`validate_scheme()`, `DEFAULT_SCHEME`).

## Passos

1. Escolher a ferramenta via ADR (`sqlx-cli` vs `refinery`) e registrar.
2. Criar `crates/wyd-db/migrations/` a partir do modelo 04-1 (uma migration por
   domínio, reversível).
3. Criar seeds (dev mínimo + prod base) com checksums.
4. Validar contra Postgres descartável: `migrate` do zero + `revert` + `migrate`
   (log anexado).

## Arquivos

- Criados: `crates/wyd-db/migrations/*.sql`, seeds, testes de integração.
- Alterados: `crates/wyd-db/Cargo.toml` (ferramenta via ADR).
- Proibido tocar: `crates/wyd-core/src/`, DSN real.

## Testes / verificação

```bash
cargo test -p wyd-db
make check
```

Saída esperada: `test result: ok`; log mostrando zero→última→revert→última sem erro.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-db - migrations e seeds.
- .agents/skills/rust-code-quality - erros tipados.

## Documentação de referencia

- pre-release-rust.md (FASE 4 §4.2).
- docs/rust/01-dependencias.md - sqlx/refinery.
- roadmap-rust/issues/FASE-04-1-modelagem.md - modelo fonte.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

**Status:** ✅ (2026-11-07, entregue dentro da FASE-04: 6 migrations + `seeds/dev.sql` + `tests/constraints.rs`; prova ao vivo zero→última→revert→última em PG17 descartável. Ferramenta runner (sqlx/refinery) pendente — ver ADR-0006; aplicação por `psql -f` até lá).

## Critérios de aceite

- [x] Migrations sobem do zero e revertem (evidência anexada); FK violada falha
  em teste.
- [x] Seeds reproduzíveis dev/prod.

## Riscos / rollback

- Risco: migration irreversível em produção futura → mitigação: toda migration
  com down testado nesta issue.
- Rollback: `git checkout -- crates/wyd-db/`.
