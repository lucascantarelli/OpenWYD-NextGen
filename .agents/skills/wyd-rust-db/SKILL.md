---
name: wyd-rust-db
description: Desenvolve e opera a persistência do WYD em Rust (crates/wyd-db): PostgreSQL via sqlx, migrations versionadas, seeds, integridade referencial, backup e diagnóstico de consultas, sem I/O bloqueante no tick loop. Use ao modelar tabelas, escrever migrations, revisar schema ou acessar dados.
---

# Persistência Rust (wyd-db)

Referências: `docs/arquitetura/modelo-banco.md`, `crates/wyd-db/`,
`docs/rust/05-stack-docker.md`, `pre-release-rust.md` FASE 4.

## Regras

1. **Modelo canônico em `docs/arquitetura/modelo-banco.md`.** Mantenha FKs, CHECKs e índices;
   `delivery_queue` é a única ponte web→jogo; `audit_log` não tem FK.
2. **Migrations versionadas e reversíveis.** `crates/wyd-db/migrations/NNNN_descricao.sql` com
   `-- migrate:up/down`; nada de DDL manual. Seeds em `crates/wyd-db/seeds/dev.sql` com valores
   `DEV-ONLY`; nunca hash/segredo real. DSN real nunca versionado.
3. **Integridade no banco, não só na aplicação.** FKs, CHECKs, `TIMESTAMPTZ`, `BIGINT`,
   `TEXT+CHECK`; índice só para leitor identificado.
4. **Async fora do tick.** Pool dimensionado, transações curtas, timeouts; `wyd-core` consome
   via traits, nunca SQL direto. Segredos fora de log e do repo (ver `.gitleaks.toml`).
5. **Rastreabilidade.** Migration ↔ modelo de domínio ↔ teste de integração; mudança de schema
   atualiza `docs/arquitetura/modelo-banco.md` na mesma entrega.

## Comandos

```bash
docker compose exec db psql -U wyd -c "\dt"   # schema aplicado?
docker compose exec -T db pg_dump -U wyd > backup-$(date +%F).sql
cargo test -p wyd-db                          # constraints e repositórios
make migrations-check                         # ordem, FKs e down, sem Postgres
```

## Não faça

- `DROP`/`TRUNCATE`/`down -v` sem squash autorizado + backup; dump com dado de jogador no repo.
- Seed com `pass_hash`/segredo; `UPDATE` em massa em tabela viva durante teste humano.
- SQL manual para “criar tabela” fora de migration.

## Antes de mudar

Árvore arquivo-a-arquivo: `docs/arquitetura/estrutura-e-padroes.md` §12. Confira o modelo e
rode o fluxo de validação de migrations da FASE-05 antes de marcar a issue como pronta.
