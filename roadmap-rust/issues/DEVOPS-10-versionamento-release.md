# DEVOPS-10 — Versionamento de release e notas rastreáveis

> **ID canônico:** `DEVOPS-10`
> **ID histórico:** `DEVOPS-10`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.
> **Run real:** [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714) provou o
> gate de versão no CI (`versão única ok: 0.1.0`); `release.yml` continua não exercitado (nenhuma tag criada).

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

## Run real (2026-11-13)

- Run [43354688714](https://github.com/lucascantarelli/wyd/actions/runs/43354688714): o passo "Versão
  única (Cargo workspace × portal)" do job `gate` passou com `versão única ok: 0.1.0 (workspace, 10
  membros e portal)` — a fonte única agora é cobrada no CI, não só localmente.
- **Disparo por tag provado** em 2026-11-13 com a tag descartável `v0.1.0`: run
  [43431337735](https://github.com/lucascantarelli/wyd/actions/runs/43431337735) criado e atribuído ao
  workflow `Release` (`event=push`, `branch=v0.1.0`), cancelado de propósito antes de publicar; tag
  apagada e nenhum release/draft ficou no repositório.
- Execução **completa** segue não provada (binários, `SHA256SUMS`, SBOM e notas): publicar release é
  decisão do dono. Comando: `git tag -a v0.1.0 -m "Release 0.1.0" && git push origin v0.1.0` em
  `main`. Rollback por tag+digest e a divergência de tamanho de imagem continuam pendentes.

## Progresso (2026-11-13)

- Fonte única: `[workspace.package] version = "0.1.0"` no `Cargo.toml` raiz, `version.workspace =
  true` nos dez membros e espelho no portal; `tools/dev/check-versao.mjs` roda no `make check` e no
  job `gate` — evidência: `versão única ok: 0.1.0 (workspace, 10 membros e portal)`.
- `.github/workflows/release.yml` (tag `v*.*.*`): `versao` falha se a tag não for a versão do
  workspace, `binarios` publica binários + `SHA256SUMS`, `imagens` constrói as quatro com
  `WYD_VERSION=<tag>` e SBOM, e `notas` monta a release com changelog + `git log` desde a tag anterior.
- Ainda falta: criar a primeira tag e ver a release sair (nunca rodou), o procedimento de rollback
  por tag+digest escrito e a divergência de tamanho de imagem resolvida com medição registrada
  (portal ~320 MB contra ~51 MB do gateway/server).

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

- [x] Fonte única de versão, verificada por script no gate local e no CI.
- [ ] Tag `vX.Y.Z` produz release com notas rastreáveis a issues e imagens rotuladas.
- [x] Procedimento de release e rollback escrito com tag + digest.
- [x] Divergência de tamanho de imagem resolvida com uma medição registrada.

## Riscos / rollback

- Risco: versão herdada quebrar crate que publica (nenhuma publica: `publish = false`) →
  baixo; validar com `cargo build --workspace`.
- Rollback: reverter para versão por crate; a tag já publicada permanece como histórico.

## Fechamento parcial (2026-11-13)

- **Procedimento escrito** em `docs/rust/10-devops-ci.md` §8.1 (tag → run → conferência de
  artefatos) e **§8.2** (rollback por tag anterior, por digest `sha256:` e a ressalva de que
  migration não tem rollback automático — restaurar dump antes de voltar a imagem).
- **Tamanho de imagem medido uma vez** (§8.3): portal **311 MB**, server **51,5 MB**, gateway
  **51,4 MB** no commit `3824cd3`. A diferença de ~6× é o runtime Node/Next do portal, não
  regressão do Rust; os números antigos (223.232 / 296.448 bytes) eram do binário do motor e ficam
  obsoletos.
- **Resta**: publicar a primeira tag real (`v0.1.0`) em `main` — é ação do dono, porque cria
  release pública e imagens `v*`; o disparo por tag já foi provado com tag descartável.
