# DEVOPS-10 — Versionamento de release e notas rastreáveis

> **ID canônico:** `DEVOPS-10`
> **ID histórico:** `DEVOPS-10`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** S · **Depende de:** DEVOPS-06 · **Bloqueia:** QA-10

## Objetivo

Ter **uma** fonte de versão, tag `vX.Y.Z` que dispara release, notas de release geradas com
rastreabilidade para issues e imagens identificáveis por tag imutável — hoje o único sinal de
versão é o build-arg default `0.1.0`.

## Contexto / evidência

- `Cargo.toml` raiz não tem `[workspace.package] version`: cada crate carrega a sua, e o portal
  tem `0.1.0` no `package.json`. Não existe fonte única.
- Nenhuma tag/release no fluxo documentado; `roadmap-rust/changelog.md` é histórico de execução
  (com evidência de issue), **não** nota de release.
- `docs/rust/02-execucao.md` §6 registra tamanho de imagem e commit de build, mas não um
  procedimento de release com tag + digest para rollback.
- Comentário do `[profile.release]` cita `323.072 → 223.232 bytes`; o §6 do mesmo doc registra
  `296.448` — duas fontes para o mesmo número.
- Análise completa em `docs/rust/10-devops-ci.md` §8 (achados C17, C18).

## Passos

1. `[workspace.package] version = "0.1.0"` + `version.workspace = true` nas crates (incluindo
   `apps/launcher-ui`); portal com a mesma versão no `package.json`. Um script de checagem
   (`tools/dev/check-versao.mjs`) falha se divergirem — entra em `make check` e no CI.
2. Workflow de release: tag `v*.*.*` em `main` → build `--release`, imagens com
   `WYD_VERSION=${tag}` e `WYD_COMMIT=${{ github.sha }}`, SBOM e notas geradas.
3. Notas de release: seção por issue fechada desde a tag anterior (`changelog.md` filtrado por
   data + PRs), com link para a issue e para a evidência. Template versionado.
4. `docs/rust/02-execucao.md` §6: procedimento de release (tag, digest das imagens, smoke) e
   procedimento de rollback (tag/digest anterior). Corrigir a divergência 223.232 × 296.448 com
   uma medida só, registrada.
5. `docs/rust/10-devops-ci.md` §8 passa a ser o contrato; `AGENTS.md` §4 ganha a linha de
   "release/versão" apontando para ele.

## Arquivos

- Criados: `.github/workflows/release.yml`, `tools/dev/check-versao.mjs`,
  `.github/release-template.md`.
- Alterados: `Cargo.toml`, `crates/*/Cargo.toml` (versão herdada), `wyd-portal/package.json`,
  `Makefile` (`versao-check`), `docs/rust/02-execucao.md` §6, `docs/rust/10-devops-ci.md` §8,
  `AGENTS.md` §4.
- Proibido: reescrever tag publicada; `latest` móvel em produção.

## Testes / verificação

```bash
node tools/dev/check-versao.mjs        # falha se Cargo × portal divergirem
cargo build --release --workspace
docker image inspect --format '{{json .Config.Labels}}' wyd/gateway:<tag>
node tools/dev/check-docs-consistency.mjs
```

Saída esperada: versão única nos três lugares; tag gera run de release com notas e SBOM; labels
de imagem iguais à tag; `check-versao` no `make check` e no CI.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain — release, imagem e gate.
- .agents/skills/software-architecture — rastreabilidade e ADR quando a versão mudar de política.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §8.
- `docs/rust/02-execucao.md` §6.
- `roadmap-rust/changelog.md`.

## Critérios de aceite

- [ ] Fonte única de versão, verificada por script no gate local e no CI.
- [ ] Tag `vX.Y.Z` produz release com notas rastreáveis a issues e imagens rotuladas.
- [ ] Procedimento de release e rollback escrito com tag + digest.
- [ ] Divergência de tamanho de imagem resolvida com uma medição registrada.

## Riscos / rollback

- Risco: versão herdada quebrar crate que publica (nenhuma publica: `publish = false`) →
  baixo; validar com `cargo build --workspace`.
- Rollback: reverter para versão por crate; a tag já publicada permanece como histórico.
