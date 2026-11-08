# Issue 00-2 — Reconfiguração do VS Code para Rust

**Fase:** 0 · **Depende de:** 00-1 · **Bloqueia:** FASE-01 a FASE-15 · **Esforço:** S

## Objetivo

VS Code com rust-analyzer, formatação e lint integrados ao workspace, sem
extensões Go residuais.

## Contexto / evidência

`start-rust.md` exige reconfiguração completa do VS Code para as novas
tecnologias. Existe `.vscode/` (`extensions.json`, `settings.json`) — conferir o
conteúdo e completar o que faltar.

## Passos

1. Ler `.vscode/extensions.json` e `.vscode/settings.json` atuais.
2. Garantir extensões recomendadas: `rust-analyzer`, `EvenBetterTOML`,
   `CodeLLDB` (debug), `ErrorLens` (opcional).
3. Garantir settings: format on save via rust-analyzer, `cargo clippy` como
   check command, `rust-toolchain.toml` respeitado.
4. Validar abrindo `crates/wyd-core/src/lib.rs` sem erros de análise.

## Arquivos

- Criados: nenhum novo (completar `.vscode/` se faltar algo).
- Alterados: `.vscode/extensions.json`, `.vscode/settings.json`.
- Proibido tocar: `crates/*/src/`.

## Testes / verificação

```bash
cat .vscode/extensions.json
```

Saída esperada: lista contém `rust-analyzer` (publisher `rust-lang`) e nenhuma
extensão Go; rust-analyzer sem erros no workspace.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - toolchain e rust-analyzer.

## Documentação de referencia

- start-rust.md - reconfiguração do VS Code.
- docs/rust/00-visao-estrutura.md - estrutura do workspace.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] rust-analyzer analisa o workspace sem erros; fmt/clippy integrados.
- [ ] Zero extensão Go instalada.

## Riscos / rollback

- Risco: settings de usuário sobrescrevendo as do workspace → mitigação:
  preferir settings do workspace e documentar.
- Rollback: `git checkout -- .vscode/`.
