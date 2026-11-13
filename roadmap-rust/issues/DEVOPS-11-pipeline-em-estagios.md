# DEVOPS-11 — Pipeline em estágios: grupos reutilizáveis, matrizes e notificação

> **ID canônico:** `DEVOPS-11`
> **ID histórico:** `DEVOPS-11`
> **Estado 2026-11-13:** 🚧 · ver `roadmap-rust/index.md`.

**Tipo:** DevOps · **Esforço:** M · **Depende de:** DEVOPS-04, DEVOPS-05 · **Bloqueia:** —

## Objetivo

Transformar o gate de **um job com muitos passos** em um pipeline de **grupos paralelos com
matriz**, onde o tempo de parede é o do job mais lento e não a soma dos passos, sem perder
nenhuma verificação e com um único check agregador para branch protection.

## Contexto / evidência

- O `gate` original rodava 10 passos em série (fmt, clippy, build, test, 5 scripts de conteúdo,
  tools-test): fmt e os scripts de conteúdo somavam minutos de fila para segundos de trabalho.
- `portal` rodava `tsc` → `vitest` → `next build` em sequência, embora sejam independentes.
- `sistema` rodava `wyd-server` e depois `wyd-admin` no mesmo job, com um Postgres cada.
- O job `bench` (caro) rodava também em PR, onde regressão de performance não é o critério.
- Cinco jobs compilavam o workspace sem cache compartilhado (`shared-key`), então cada job
  restaurava um cache diferente e nenhum reaproveitava o `target/` do outro.
- Não havia notificação de falha: o vermelho só era visto por quem abria a aba Actions.

## Progresso (2026-11-13)

- Grupo **conteúdo** (`.github/workflows/ci-content.yml`): matriz de 6 jobs, um por script.
- Grupo **rust** (`ci-rust.yml`): `fmt`, `clippy`, `build`+`test` em jobs separados; `sistema` em
  matriz (`wyd-server`, `wyd-admin`), cada um com Postgres 17 efêmero, `--migrate` e
  `WYD_TEST_REQUIRED=1`; `bench` com `if: inputs.bench` (só push em `main`/`develop`).
- Grupo **portal** (`ci-portal.yml`): matriz `typecheck` / `vitest` / `next build`.
- Grupo **infra** (`ci-infra.yml`): matriz com hardening do compose, `config` nos três modos,
  três `promtool`, render do Alertmanager/Slack e `actionlint`.
- Orquestrador `rust.yml`: chama os quatro grupos (`uses: ./.github/workflows/ci-*.yml`) e o job
  `gate` agrega com `needs` + `always()`, falhando se qualquer grupo não for `success`.
- `rust-setup` ganhou `shared-key` para o cache de `target/` ser compartilhado entre jobs.
- Job `notify` (falha do gate → Slack) pronto, condicionado ao segredo `SLACK_CI_WEBHOOK`
  (docs `10-devops-ci.md` §11).

## Critérios de aceite

- [x] Grupos reutilizáveis (`ci-content`, `ci-rust`, `ci-portal`, `ci-infra`) chamados pelo orquestrador.
- [x] Matrizes: conteúdo 6, sistema 2, portal 3, infra 9.
- [x] `gate` agregador com `needs` + `always()` — único check obrigatório de branch protection.
- [x] `bench` fora do PR; cache com `shared-key`.
- [ ] Notificação de falha chegando no canal (depende do segredo `SLACK_CI_WEBHOOK`).
- [ ] Tempo de parede medido antes × depois, registrado na issue.

## Riscos / rollback

- Risco: matriz multiplica runners e, com workspace grande, cada job compila o grafo inteiro.
  Mitigação: `shared-key` no cache e `bench` fora do PR; medir antes de fatiar mais.
- Rollback: `rust.yml` é o único ponto de entrada — voltar o arquivo restaura o gate monolítico
  sem tocar nos grupos (que ficam inertes, pois só rodam por `workflow_call`).
