---
name: wyd-rust-toolchain
description: Opera a toolchain e a qualidade do workspace Rust do OpenWYD (rustup stable, cargo fmt/clippy/test/bench, audit/deny, lockfile). Use ao instalar/atualizar Rust, adicionar dependências, configurar CI/Docker ou diagnosticar build falhando.
---

# Toolchain Rust (workspace)

Referências: `rust-toolchain.toml`, `.github/workflows/rust.yml`,
`roadmap-rust/issues/FASE-05.md`, `docs/rust/01-dependencias.md`.

## Regras

1. **Stable pinado.** `rust-toolchain.toml` (`stable` + `rustfmt`/`clippy`); MSRV
   só muda via ADR (FASE-13).
2. **Gates locais antes do push.** `cargo fmt --check`, `cargo clippy --workspace
   -- -D warnings`, `cargo build --workspace`, `cargo test --workspace`.
3. **Dependências com lockfile consistente.** Adicionar dep exige justificar
   (performance/memória/manutenibilidade), atualizar `Cargo.lock` e manter build
   verificável offline ou com rede declarada; preferir `cargo deny`/`audit`.
4. **Go intacto.** Nada de `deprecate-go`, desinstalação ou push neste fluxo;
   Go continua buildável (`go build ./...`).

## Diagnóstico rápido

```bash
export PATH="/c/Users/Canta/.cargo/bin:$PATH"
cargo --version; rustc --version; rustup show active-toolchain
cargo build --workspace && cargo test --workspace
```
