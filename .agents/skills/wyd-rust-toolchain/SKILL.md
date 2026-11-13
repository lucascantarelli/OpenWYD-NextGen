---
name: wyd-rust-toolchain
description: Opera a toolchain e a qualidade do workspace Rust do WYD (rustup stable, cargo fmt/clippy/test/bench, audit/deny, lockfile). Use ao instalar/atualizar Rust, adicionar dependências, configurar CI/Docker ou diagnosticar build falhando.
---

# Toolchain Rust (workspace)

Norma: `rust-toolchain.toml`, `.github/workflows/rust.yml`,
`docs/rust/01-dependencias.md`, `docs/rust/02-execucao.md` (gates/DoD) e
`docs/rust/10-devops-ci.md` (pipeline, imagens, compose e deploy local). O comando do gate
e a regra 100% Rust estão no `AGENTS.md`; não os repita aqui.

## Decisões específicas

- `channel = "stable"` **não é pin**: a versão flutua e `clippy -D warnings` pode quebrar sem
  uma linha alterada. Pin por versão (`1.9x.y`) é a meta (`DEVOPS-05`); até lá, registre a
  versão usada na evidência. Mudança de MSRV ou dependência pesada exige ADR + PoC medida
  (`docs/rust/01-dependencias.md`), não “última versão”.
- Lockfile consistente: adicionar/atualizar crate justifica custo, roda `cargo audit`/
  `cargo deny` quando aplicável e mantém build offline ou rede declarada.
- CI não substitui o gate local; `fmt`, `clippy -D warnings`, build e testes rodam antes.
- Diagnóstico de toolchain é `cargo --version`, `rustc --version` e `rustup show
  active-toolchain`; não reintroduza caminho Go/stack antiga.

## CI, imagem e deploy (resumo acionável)

- **Gate tem que significar a mesma coisa nos dois lados.** `make check` e o workflow precisam
  cobrir as mesmas verificações; hoje o CI roda só `docs-check` e `make ci` não é espelho
  (`DEVOPS-04`). Ao mexer em gate, ajuste os dois na mesma entrega.
- **Cache não é opcional em CI que compila cinco vezes** — `Swatinem/rust-cache` + `cache-to:
  type=gha` nos `docker build` (`DEVOPS-05`).
- **Imagem de CI sem `WYD_VERSION`/`WYD_COMMIT` mente na procedência** e Dockerfile que nunca é
  construído apodrece: build + smoke + scan para as quatro imagens (`DEVOPS-06`).
- **Build não é prova**: um job que constrói imagem roda um comando que só passa com ela
  funcionando (`--healthcheck`, `check`, `--help`).
- **Deploy local** (runner `self-hosted`) nunca em `pull_request`; desenho e travas em
  `docs/rust/10-devops-ci.md` §9 (`DEVOPS-09`).

## Prova

`make check` após qualquer mudança de toolchain/dependência; `make ci` antes de tocar o
workflow, comparando a lista de verificações com a tabela de paridade do doc de DevOps;
registre a saída na issue.
