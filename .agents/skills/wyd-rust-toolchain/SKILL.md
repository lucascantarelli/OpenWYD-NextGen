---
name: wyd-rust-toolchain
description: Opera a toolchain e a qualidade do workspace Rust do WYD (rustup stable, cargo fmt/clippy/test/bench, audit/deny, lockfile). Use ao instalar/atualizar Rust, adicionar dependências, configurar CI/Docker ou diagnosticar build falhando.
---

# Toolchain Rust (workspace)

Norma: `rust-toolchain.toml`, `.github/workflows/rust.yml`,
`docs/rust/01-dependencias.md` e `docs/rust/02-execucao.md` (gates/DoD). O comando do gate
e a regra 100% Rust estão no `AGENTS.md`; não os repita aqui.

## Decisões específicas

- `stable` é pinado no `rust-toolchain.toml`. Mudança de MSRV ou dependência pesada exige
  ADR + PoC medida (`docs/rust/01-dependencias.md`), não “última versão”.
- Lockfile consistente: adicionar/atualizar crate justifica custo, roda `cargo audit`/
  `cargo deny` quando aplicável e mantém build offline ou rede declarada.
- CI não substitui o gate local; `fmt`, `clippy -D warnings`, build e testes rodam antes.
- Diagnóstico de toolchain é `cargo --version`, `rustc --version` e `rustup show
  active-toolchain`; não reintroduza caminho Go/stack antiga.

## Prova

`make check` após qualquer mudança de toolchain/dependência; registre a saída na issue.
