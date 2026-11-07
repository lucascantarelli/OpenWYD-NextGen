---
name: wyd-rust-db
description: Desenvolve a persistência async do WYD em Rust (crates/wyd-db): PostgreSQL via sqlx, migrations versionadas, integridade referencial, sem I/O bloqueante no tick loop. Use ao modelar tabelas, escrever migrations, seeds ou código de acesso a dados.
---

# Persistência Rust (wyd-db)

Sucessora da `wyd-database` para Rust. Referências: `pre-release-rust.md` FASE 4,
`roadmap-rust/issues/FASE-04.md`.

## Regras

1. **Migrations versionadas e reversíveis.** Nada de DDL manual; seeds
   versionados; `sqlx-cli`/`refinery` no pipeline (ver FASE-05).
2. **Integridade no banco, não só na aplicação.** FKs, CHECKs e índices para o
   caminho quente de leitura.
3. **Async fora do tick.** Pool dimensionado, transações curtas, timeouts;
   segredos nunca em log ou no repo (ver `.gitleaks.toml`).
4. **Rastreabilidade.** Migration ↔ modelo de domínio ↔ teste de integração.

## Antes de mudar

Árvore arquivo-a-arquivo: `docs/arquitetura/estrutura-e-padroes.md` §12.
Confira `docs/analysis/banco-de-dados.md` e rode o fluxo de validação de
migrations da FASE-05 antes de marcar a issue como pronta.
