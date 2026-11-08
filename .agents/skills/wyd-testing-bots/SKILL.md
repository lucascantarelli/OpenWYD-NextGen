---
name: wyd-testing-bots
description: Compat de transição para harness e bots do WYD (aponta para a sucessora Rust). Use `wyd-rust-testing` para trabalho novo.
---

# Harness e bots (transição → `wyd-rust-testing`)

> Sucessão 09-2: **sucedida por `wyd-rust-testing`**. Esta skill é um alias histórico; não adicione conteúdo Go aqui.

Todo trabalho novo usa `wyd-rust-testing`:

- Codec real: `crates/wyd-protocol` + `Strict`; goldens em `fixtures/cliente/*.json` (`captura.mjs --check`).
- Gates: `cargo test --workspace`, `cargo test -p wyd-protocol -p wyd-net`, `cargo bench --workspace`.
- Carga/determinismo: `wyd-core/tests/determinism.rs`, `wyd-sim` em lote.
- Higiene: conta de bot dedicada, fixtures sanitizadas, sem `.pcap`/dumps no repo, sem `#[ignore]`.

Conhecimento atemporal preservado: prove no headless primeiro; asserte o pacote S→C; bot descobre, teste reproduz; não reinicie stack alheia.
