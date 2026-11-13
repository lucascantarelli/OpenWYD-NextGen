# DEVOPS-12 — Dependências fora de range e pin de actions por SHA

> **ID canônico:** `DEVOPS-12`
> **ID histórico:** `DEVOPS-12`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** M · **Depende de:** DEVOPS-05 · **Bloqueia:** —

## Objetivo

Fechar as duas lacunas de supply-chain que sobraram depois da atualização de 2026-11-13: crates
com **major** fora do range declarado (o dependabot as ignora por política) e **pin de actions por
SHA** em vez de major tag móvel.

## Contexto / evidência

- `cargo update` em 2026-11-13 subiu 5 crates em patch (`itoa 1.0.19`, `rustls-pki-types 1.16.0`,
  `rustls-webpki 0.103.17`, `syn 3.0.10`, `webpki-roots 1.0.12`) e o próprio cargo informou
  **8 dependências atrás da última versão** (`cargo update --verbose` lista quais).
- `.github/dependabot.yml` ignora `semver-major` de propósito (major exige ADR) — portanto esses 8
  **não** viram PR e envelhecem em silêncio.
- Actions hoje por major tag (`actions/checkout@v7`, `upload-artifact@v7`,
  `docker/build-push-action@v7`, `aquasecurity/trivy-action@v0.36.0`, `slackapi/slack-github-action@v4.0.0`).
  Major tag é móvel: o conteúdo muda sem mudar o YAML. O achado C7 de `10-devops-ci.md` continua aberto.
- `cargo-outdated` não está instalado na máquina de dev, então não há como listar major pendente
  sem `cargo update --verbose` manual.

## Progresso (2026-11-13)

- Patch/minor atualizados (cargo + npm) e `make check` verde depois.
- Actions subiram de major (checkout 5→7, artifacts 4→7, buildx 3→4, build-push 6→7, setup-node 4→7).
- `actionlint` 1.7.12 e `promtool` (Prometheus 3.16.0) alinhados entre compose e CI.

## Critérios de aceite

- [ ] Listar os 8 crates atrás da última versão com `cargo update --verbose` e decidir crate a crate
      (subir / justificar espera em `docs/rust/01-dependencias.md`).
- [ ] Instalar `cargo-outdated` na toolchain de dev (ou um alvo `make deps-outdated`) para a lista
      sair sem comando manual.
- [ ] Pinar as actions por SHA com comentário da tag (`uses: actions/checkout@<sha> # v7.0.1`) e um
      script no gate que confere SHA × comentário.
- [ ] Dependabot passa a abrir PR **também** para os majors aceitos (lista explícita em `ignore`
      em vez de `*`), para o que é decisão de arquitetura continuar humano e o resto não envelhecer.

## Riscos / rollback

- Risco: subir major de crate do Hot Path muda semântica de performance/alocação. Mitigação: um
  crate por PR, com `bench.yml` disparado pelo caminho quando tocar `wyd-core`/`wyd-sim`/`wyd-net`.
- Rollback: cada bump é um PR isolado; reverter é reverter o PR.
