# DEVOPS-08 — Supply-chain contínua e cobertura medida

> **ID canônico:** `DEVOPS-08`
> **ID histórico:** `DEVOPS-08`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** M · **Depende de:** DEVOPS-07 · **Bloqueia:** QA-09

## Objetivo

Transformar supply-chain de "verificação por commit" em verificação **contínua** (agendada e com
atualização de dependências proposta automaticamente) e publicar cobertura real de Rust e portal,
com baseline registrada — sem transformar cobertura em meta decorativa.

## Contexto / evidência

- `supply-chain` (`cargo audit` + `cargo deny check`) roda **só** em push/PR. Advisory publicado
  depois do último commit nunca aparece. Não existe `.github/dependabot.yml` nem Renovate
  (`.github/` tem somente `workflows/`).
- Cobertura não é medida em lugar nenhum: nenhum `cargo tarpaulin`/`llvm-cov`;
  `@vitest/coverage-v8` está em `devDependencies` do portal e não há script de cobertura.
- `deny.toml` está bem configurado (licenças fechadas, `wildcards = deny`, fontes só crates.io,
  `private.ignore`) — a política existe, a cadência não.
- QA-09 (tarpaulin/miri) depende desta base para ter onde publicar número.
- Análise completa em `docs/rust/10-devops-ci.md` §3 (achados B7, B8).

## Progresso (2026-11-13)

- `.github/workflows/supply-chain.yml`: `audit` (`cargo audit` + `cargo deny check`) como **gate**,
  `images` (scan das quatro imagens, `CRITICAL` bloqueante) e `cobertura` (`cargo llvm-cov` +
  `npm run test:cobertura`) publicada como **métrica** com artefato — tudo em agenda (`0 6 * * 1`),
  `workflow_dispatch` e push em `Cargo.lock`/`deny.toml`.
- `.github/dependabot.yml`: cargo, npm (portal), GitHub Actions e docker, com agrupamento por
  ecossistema, limite semanal e `ignore` de majors.
- Ainda falta: baseline de cobertura comparável entre runs (queda vira aviso) e o primeiro run da
  agenda, que nunca aconteceu (nada foi executado no GitHub ainda).

## Passos

1. Workflow `supply-chain.yml` (ou job com `schedule`) semanal + `workflow_dispatch`: `cargo
   audit`, `cargo deny check`, Trivy nas imagens de `DEVOPS-06`. Falha gera issue/notificação,
   não commit.
2. `.github/dependabot.yml`: `cargo` (semanal, agrupado por tipo), `npm` em `/wyd-portal`,
   `github-actions` e `docker` (bases dos Dockerfiles). PR automático **só** de patch/minor;
   major exige ADR (`docs/rust/01-dependencias.md`).
3. Rust: `cargo llvm-cov` (ou tarpaulin) no job de testes, cobertura publicada como artefato;
   baseline registrada em `docs/rust/02-execucao.md` §4 e comparada (queda > 2 p.p. vira aviso,
   não gate, até QA-09 definir o gate).
4. Portal: script `test:cobertura` (`vitest run --coverage`) e publicação do resumo.
5. Registrar em `docs/rust/10-devops-ci.md` §3/§8 o que é gate (advisory, licença, fonte) e o
   que é métrica (cobertura, tamanho de imagem) — misturar os dois gera gate instável.
6. Aviso explícito: cobertura não substitui teste de comportamento; a baseline serve para
   detectar queda, não para perseguir porcentagem.

## Arquivos

- Criados: `.github/workflows/supply-chain.yml`, `.github/dependabot.yml`.
- Alterados: `.github/workflows/rust.yml` (cobertura), `wyd-portal/package.json` (script),
  `Makefile` (`coverage`), `docs/rust/02-execucao.md` §4, `docs/rust/10-devops-ci.md`.
- Proibido: `cargo deny` com `allow` novo sem ADR; ignorar advisory sem prazo.

## Testes / verificação

```bash
cargo audit && cargo deny check
cargo llvm-cov --workspace --all-targets --summary-only   # ou tarpaulin
cd wyd-portal && npm run test:cobertura
node tools/dev/check-docs-consistency.mjs
```

Saída esperada: dois audits verdes (Rust e imagens); cobertura publicada com baseline;
Dependabot abrindo PR agrupado na primeira semana.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain — audit/deny, lockfile e dependências.
- .agents/skills/rust-test-quality — o que conta como teste e como medir.

## Documentação de referencia

- `docs/rust/10-devops-ci.md` §3 e §8.
- `docs/rust/01-dependencias.md` — política de dependência/MSRV.
- `docs/rust/02-execucao.md` §4 — baseline e revisão.

## Critérios de aceite

- [x] Verificação de advisory roda por tempo (agenda) além de por commit.
- [x] Dependabot ativo em cargo, npm, actions e docker, com agrupamento e política de major.
- [ ] Cobertura de Rust e portal publicada com baseline comparável.
- [ ] Gate × métrica separados e escritos no doc de DevOps.

## Riscos / rollback

- Risco: volume de PRs do Dependabot parar o time → mitigação: agrupamento, limite semanal e
  `ignore` para crates presas por ADR.
- Rollback: desligar o agendamento; `cargo audit`/`deny` por commit continua.
