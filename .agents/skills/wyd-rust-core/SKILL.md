---
name: wyd-rust-core
description: Desenvolve a engine Hot Path do WYD em Rust (crates/wyd-core): tick determinístico, ECS/DOD, spatial grid, sem GC e sem mutex global no estado de jogo. Use ao alterar game loop, componentes ECS, particionamento espacial, autoridade de cena ou fixed-point.
---

# Engine Rust (wyd-core)

Sucessora da `wyd-server-dev` para o Hot Path. Referências: `pre-release-rust.md`
FASE 3, `roadmap-rust/issues/FASE-03.md`, `docs/rust/00-visao-estrutura.md`.

## Regras que quebram o servidor se ignoradas

1. **Hot Path no mesmo espaço de memória.** Nada de gRPC/HTTP no tick loop;
   Cold Path (auth, billing, portal) fica em `wyd-admin`/`wyd-db`, async e isolado.
2. **DOD/ECS, nunca OOP espalhada no heap.** Dados contíguos (`Vec<Position>`,
   `Vec<Stats>`); varreduras em lote, SIMD-friendly.
3. **Sem `unsafe` fora de interop comprovada.** Erros tipados (`thiserror`/`anyhow`),
   `Result`/`Option`; `cargo clippy -- -D warnings` limpo.
4. **Determinismo.** Tick em frequência fixa; ordem de RNG/eventos é contrato;
   `ESCENE_FIELD` (30000) continua sendo o pacote autoritativo de cena.

## Antes de mudar

Árvore arquivo-a-arquivo: `docs/arquitetura/estrutura-e-padroes.md` §10.
Leia o contrato em `crates/wyd-protocol/src/lib.rs` e `docs/migration/` correspondente.
Todo comportamento novo nasce com teste (`cargo test -p wyd-core`).
