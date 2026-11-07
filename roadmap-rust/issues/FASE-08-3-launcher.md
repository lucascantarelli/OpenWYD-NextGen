# Issue 08-3 — Launcher (hash, patch, bootstrap)

**Fase:** 8 · **Depende de:** 01-2 · **Bloqueia:** FASE-07 (distribuição) · **Esforço:** M

## Objetivo

Entregar o `wyd-launcher` que verifica integridade (hash), aplica patch e inicia
o cliente.

## Contexto / evidência

`pre-release-rust.md` FASE 8 §8.2: analisar `ClientPatch_v7662`, launcher
Rust/Tauri ou CLI. Base: `crates/wyd-launcher/src/main.rs`
(`bootstrap_message()`).

## Passos

1. Documentar o mecanismo legado de patch (leitura: `ClientPatch_v7662`, fluxo
   atual de atualização).
2. Implementar verificação de hash por arquivo + manifesto de versão.
3. Implementar aplicação de patch + bootstrap do cliente.
4. Teste: arquivo corrompido detectado; patch aplicado; `make check`.

## Arquivos

- Criados: verificador, patcher e testes em `crates/wyd-launcher/src/`.
- Alterados: `crates/wyd-launcher/src/main.rs`, `Cargo.toml` (deps via ADR).
- Proibido tocar: `client/` (binários), `.deprecated-go/` (leitura apenas).

## Testes / verificação

```bash
cargo test -p wyd-launcher
make check
```

Saída esperada: `test result: ok` incluindo detecção de corrupção.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - build/distribuição do binário.
- .agents/skills/wyd-client-connect - serverlist e diagnóstico.
- .agents/skills/rust-code-quality - testes.

## Documentação de referencia

- pre-release-rust.md (FASE 8 §8.2).
- .deprecated-go/docs/migration/config-ops.md - distribuição legada.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Corrupção detectada com relatório por arquivo; patch + bootstrap funcionam.
- [ ] Manifesto de versão publicado e verificado.

## Riscos / rollback

- Risco: patch parcial corrompendo cliente → mitigação: aplicação atômica
  (tmp + rename) + verificação pós-patch.
- Rollback: `git checkout -- crates/wyd-launcher/`.
