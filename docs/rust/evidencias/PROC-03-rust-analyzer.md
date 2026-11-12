# Evidência — PROC-03: rust-analyzer no workspace

> **Issue:** `PROC-03` — Reconfiguração do VS Code para Rust (concluída).
> **Registro canônico:** [`roadmap-rust/changelog.md`](../../../roadmap-rust/changelog.md).
> **Data da coleta:** 2026-11-12.
> **Objetivo:** provar que o workspace é analisado pelo rust-analyzer sem erros e que o VS Code está configurado para Rust, sem extensão Go.

## Ambiente

- Workspace: `D:\dolorestec\wyd2`.
- `rust-analyzer 0.3.3113-standalone (e5e7830640 2026-11-08)`.
- Binário usado (extensão oficial do VS Code):
  `~/.vscode/extensions/rust-lang.rust-analyzer-0.3.3113-win32-x64/server/rust-analyzer.exe`.
- Toolchain `stable`, com `rustfmt` e `clippy` em `rust-toolchain.toml`.
- `code --list-extensions` inclui `rust-lang.rust-analyzer` e não lista extensão Go.

## Comandos executados

```bash
RA="$HOME/.vscode/extensions/rust-lang.rust-analyzer-0.3.3113-win32-x64/server/rust-analyzer.exe"
"$RA" --version
"$RA" analysis-stats . --parallel
"$RA" diagnostics .
```

`analysis-stats . --parallel` terminou com **exit 0** e os seguintes números:

```text
Total Statistics:
  crates: 77, mods: 418, decls: 5587, bodies: 5419
Workspace:
  lines of code: 74_384, item trees: 269
Parallel Inference:
  exprs: 246281, ??ty: 55 (0%), ?ty: 6 (0%), !ty: 0
  pats: 37597, ??ty: 0 (0%), ?ty: 0 (0%), !ty: 0
  panics: 0
Data layouts:
  Failed data layouts: 0 (0%)
Const evaluation:
  Failed const evals: 10 (1%)
```

`diagnostics .` terminou com **exit 0**, imprimiu `diagnostic scan complete` e foi resumido por:

```text
Erros (linhas com " Error "): 0
WeakWarnings: 410
  inactive_code: 409  (código atrás de #[cfg(test)] ou target_os = "linux")
  remove-unnecessary-else: 1 (crates/wyd-launcher/src/portal_client.rs:465)
```

Não há erro de compilação, resolução ou tipo. O único aviso de estilo é
`remove-unnecessary-else` em `crates/wyd-launcher/src/portal_client.rs:465`; ele não
bloqueia o critério “analisa o workspace sem erros” e fica registrado para limpeza
posterior.

## Configuração do VS Code

`.vscode/extensions.json`:

```json
{
  "recommendations": [
    "rust-lang.rust-analyzer",
    "tamasfe.even-better-toml",
    "serayuzgur.crates",
    "vadimcn.vscode-lldb",
    "github.vscode-github-actions",
    "ms-playwright.playwright"
  ]
}
```

`.vscode/settings.json`:

```json
{
  "rust-analyzer.check.command": "clippy",
  "rust-analyzer.check.extraArgs": ["--", "-D", "warnings"],
  "editor.formatOnSave": true,
  "[rust]": {
    "editor.defaultFormatter": "rust-lang.rust-analyzer"
  },
  "[toml]": {
    "editor.defaultFormatter": "tamasfe.even-better-toml"
  }
}
```

## Conclusão

O critério de aceite foi cumprido: rust-analyzer analisou o workspace com **0 erros**,
fmt/clippy estão integrados ao VS Code e a lista de extensões recomenda rust-analyzer,
sem extensão Go. A entrada do fechamento fica no changelog canônico e este arquivo
preserva a prova fora de `tmp/`.
