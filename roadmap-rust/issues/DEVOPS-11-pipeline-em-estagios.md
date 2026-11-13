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

## Comprovação por workflow (2026-11-13)

Estado observado no GitHub, não lido do YAML — a mesma tabela está em `10-devops-ci.md` §1.1.

| Arquivo | Estado | Prova |
|---|---|---|
| `rust.yml` | verde, 25 jobs, só `notify` skipped | [43429981097](https://github.com/lucascantarelli/wyd/actions/runs/43429981097) |
| `ci-content.yml` / `ci-rust.yml` / `ci-portal.yml` / `ci-infra.yml` | chamados pelo orquestrador, todos os jobs verdes | jobs `conteúdo|rust|portal|infra / …` no run acima |
| `images.yml` | verde nas quatro imagens | [43422723816](https://github.com/lucascantarelli/wyd/actions/runs/43422723816) |
| `bench.yml` | verde por `workflow_dispatch` (5m10) | [43427765855](https://github.com/lucascantarelli/wyd/actions/runs/43427765855) |
| `deploy-local.yml` | verde no `wyd2-desktop` | [43430395464](https://github.com/lucascantarelli/wyd/actions/runs/43430395464) |
| `supply-chain.yml` | verde nos 6 jobs | [43432022839](https://github.com/lucascantarelli/wyd/actions/runs/43432022839) |
| `release.yml` | disparo por tag provado; execução completa depende de tag do dono | [43431337735](https://github.com/lucascantarelli/wyd/actions/runs/43431337735) (cancelado de propósito, tag apagada) |

## Critérios de aceite

- [x] Grupos reutilizáveis (`ci-content`, `ci-rust`, `ci-portal`, `ci-infra`) chamados pelo orquestrador.
- [x] Matrizes: conteúdo 6, sistema 2, portal 3, infra 9.
- [x] `gate` agregador com `needs` + `always()` — único check obrigatório de branch protection.
- [x] `bench` fora do PR; cache com `shared-key`.
- [ ] Notificação de falha chegando no canal (depende do segredo `SLACK_CI_WEBHOOK`).
- [x] Tempo de parede medido antes × depois, registrado na issue.

## Medição (2026-11-13)

| Configuração | Tempo de parede | Caminho crítico |
|---|---|---|
| Gate monolítico (7 jobs, 10 passos em série no `gate`) | **3 min 17 s** (run 43417374963) | `build+test` |
| Grupos em paralelo, caches frios (`shared-key` novo) | 5 min 39 s (run 43422724003) | `bench` 5 min |
| Grupos em paralelo, cache quente | 4 min 29 s (run 43425395680) | `bench` 4,3 min / `build+test` 2,6 min / `clippy` 2,3 min / `sistema` 2,2 min |
| Grupos em paralelo **sem** `bench` (estado atual) | **2 min 32 s** (run 43426393075, 25 jobs em 4 grupos) | `build+test` ~2,6 min |

Leitura honesta: fatiar sem medir **piorou** o tempo de parede, porque `bench` compila o workspace
inteiro no perfil de bench e passou a ditar todos os pushes. O ganho real vem de tirá-lo do caminho
crítico (`bench.yml`, gatilho por caminho + agenda) e de manter em matriz só o que é barato e
independente (conteúdo, portal, infra). Fica registrado que paralelizar `clippy` e `build+test`
duplica compilação: se o objetivo for minutos e não isolamento de falha, eles devem voltar a um job.

## Riscos / rollback

- Risco: matriz multiplica runners e, com workspace grande, cada job compila o grafo inteiro.
  Mitigação: `shared-key` no cache e `bench` fora do PR; medir antes de fatiar mais.
- Rollback: `rust.yml` é o único ponto de entrada — voltar o arquivo restaura o gate monolítico
  sem tocar nos grupos (que ficam inertes, pois só rodam por `workflow_call`).

> **Consolidação (2026-11-13):** os oito workflows de CI viraram um só — `.github/workflows/ci.yml`, com estágios encadeados por `needs` (qualidade → build/teste → imagens/cobertura → gate/deploy) e notificação sempre ao fim. `bench.yml` e `release.yml` seguem separados. Nomes de arquivo citados acima são **históricos** desta seção.

> **Celeridade e notificações (2026-11-13, medido):** deploy local reconstruía as imagens a
> cada push alterando `crates/` (21 min 20 s no run 43456241374) porque `imagens` usa `load: true`
> e não publica em registry; a matriz de build já existe, falta `push` + `pull`. Cache do
> `Swatinem/rust-cache` é por branch e salvo no fim do job: compartilha em sequência, **não** entre
> jobs paralelos. `build + testes` fica no mesmo job de propósito (§14.2 do doc de CI).
> Entregas desta rodada: anotação de deploy no Grafana, anúncio em `#wyd-releases` (segredo
> `SLACK_RELEASES_WEBHOOK`), notificação de CI com tempo por job/parede/paralelismo e corte da
> redundância do alerta de deploy bem-sucedido no canal de operação. Aberto: registry, cobertura
> fora do caminho crítico, `cargo nextest` + matriz de teste.
