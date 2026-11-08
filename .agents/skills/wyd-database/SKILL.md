---
name: wyd-database
description: Opera o PostgreSQL do WYD em Rust (schema, migrations, seeds, índices, backup). Use ao modelar tabela, escrever migration, investigar consulta lenta ou revisar normalização e segurança do banco.
---

# Banco de dados Rust (PostgreSQL + `wyd-db`)

> Sucessão 09-2: mantido o nome; ferramenta e caminhos migrados para Rust. Fonte canônica: `docs/arquitetura/modelo-banco.md`, `crates/wyd-db/`.

Sucessora conceitual integra `wyd-rust-db`. Referências: `roadmap-rust/issues/FASE-04.md`, `docs/rust/05-stack-docker.md`.

## Regras

1. **Modelo canônico em `docs/arquitetura/modelo-banco.md`.** 8 tabelas com FKs/CHECKs/índices; `delivery_queue` é a única ponte web→jogo; `audit_log` sem FK.
2. **Migrations versionadas e reversíveis.** `crates/wyd-db/migrations/NNNN_descricao.sql` com `-- migrate:up/down`; seeds em `crates/wyd-db/seeds/dev.sql` (valores `DEV-ONLY`, nunca hash real); DSN real nunca versionado.
3. **A web nunca escreve estado vivo.** Grants via `delivery_queue` (lido por `crates/wyd-db/src/transfer/`); migration/seed é a única escrita fora do jogo.
4. **Async fora do tick.** Pool dimensionado, transações curtas, timeouts (`crates/wyd-db/src/pool/`); `wyd-core` consome via traits, nunca SQL direto.
5. **Integridade e comentários.** `COMMENT ON TABLE/COLUMN` quando semântica não óbvia; índice só com leitor identificado; `TIMESTAMPTZ`, `BIGINT`, `TEXT+CHECK`.

## Comandos

```bash
docker compose exec db psql -U wyd -c "\dt"   # schema aplicado?
docker compose exec -T db pg_dump -U wyd > backup-$(date +%F).sql
cargo test -p wyd-db                          # constraints (FK/CHECK falham como esperado)
```

## Não faça

- `DROP`/`TRUNCATE` sem squash autorizado + backup; dump com dado de jogador no repo.
- Seed com `pass_hash`/segredo; `UPDATE` em massa em tabela viva durante teste humano.
- SQL manual para "criar tabela" fora de migration.
