# Ferramentas de desenvolvimento

Scripts Node sem dependências, chamados pelo `make check` ou por alvos específicos.

| Script | Alvo | O que valida |
|---|---|---|
| `check-docs-consistency.mjs` | `make docs-check` | membros do `Cargo.toml` × `crates/*`, contagem/nomes de crates em `README.md`/`AGENTS.md` e links relativos de `README.md`, `AGENTS.md`, `docs/` e `roadmap-rust/` |
| `bench-gate.mjs` | `make bench-gate` | regressão de benchmarks contra baseline por cenário |
| `monitoring-smoke.mjs` | `make monitoring-check` | stack de monitoramento viva |
| `clean-tmp.mjs` | `make clean-tmp` | limpeza de artefatos conhecidos de `tmp/` |

`check-docs-consistency.mjs` não corrige nada: divergência é erro com lista de caminhos.
