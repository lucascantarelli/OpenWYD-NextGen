---
name: wyd-rust-db
description: Desenvolve e opera a persistência do WYD em Rust (crates/wyd-db): PostgreSQL via sqlx, migrations versionadas, seeds, integridade referencial, backup e diagnóstico de consultas, sem I/O bloqueante no tick loop. Use ao modelar tabelas, escrever migrations, revisar schema ou acessar dados.
---

# Persistência Rust (wyd-db)

Norma: `docs/arquitetura/modelo-banco.md` (schema, tabelas, migrations, seeds),
`docs/arquitetura/banco-catalogo.md` (catálogo gerado) e `docs/arquitetura/estrutura-e-padroes.md`
§12; aplicação no boot em ADR-0024. Não replique o ER nem o catálogo aqui.

## Decisões específicas

- Migration `NNNN_descricao.sql` com `up`/`down`; schema, modelo de domínio e teste de
  integração mudam na mesma entrega. DDL manual não substitui migration.
- `delivery_queue` é a única ponte web→jogo; `audit_log` não ganha FK. O catálogo é gerado
  e validado contra a série, não editado à mão.
- `wyd-core` não fala SQL: use as portas/traits existentes. Pool, transação curta e timeout
  ficam fora do tick; DSN/segredo não entram em log, seed ou repo.
- Seed de desenvolvimento é `DEV-ONLY`; produção não versiona segredo nem conta real.

## Prova

`make migrations-check` e `cargo test -p wyd-db` (sistema com `WYD_TEST_DSN` quando a issue
exigir banco real); depois `make check`.
