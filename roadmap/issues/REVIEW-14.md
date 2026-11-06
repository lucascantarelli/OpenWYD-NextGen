# REVIEW-14 — Revisão permanente: DevOps, CI/CD e observabilidade

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle dos pipelines, imagens, compose e observabilidade: previsibilidade, tempo de
execução, permissões, cache, artefatos e cobertura de métrica/alerta.

## Escopo

- Workflows (`.github/workflows/`): reuso via `workflow_call`, path filter, permissões mínimas,
  Actions fixadas, artefatos e proveniência.
- `make gate`/`make ci`: o que roda no dev e no CI precisa ser o mesmo comando.
- Compose em camadas (base + override), healthchecks, ordem por `condition`, nomes `openwyd-*`.
- Observabilidade: `/metrics` dos 5 alvos, `trace_id`, alertas, dashboard, `logpipe`/`/logs`.
- Tempo de pipeline e custo: job que roda sem provar nada é dívida.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Workflows | `ls .github/workflows/` e diff de jobs entre eles |
| Tempo do pipeline | duração por job no GitHub Actions |
| Gate local | `make gate` exit 0 |
| Alvos coletados | `curl -s localhost:9090/api/v1/targets \| jq '[.data.activeTargets[] \| select(.health=="up")] \| length'` |
| Regras de alerta | `promtool check rules monitoring/…` |
| Segredo exposto | `bash scripts/check-secrets.sh` |
| Achados novos | uma linha por achado |

## Critérios de aceite (por rodada)

- [ ] `make ci` verde do zero em clone sem cache
- [ ] Nenhum workflow com permissão de escrita desnecessária
- [ ] Todo serviço novo tem `/metrics`, healthcheck e alerta quando faz sentido
- [ ] Nenhum artefato publicado sem origem rastreável

## Histórico

| Data | Workflows | Achados | Destino |
|---|---|---|---|
| 06/11/2026 | `_go.yml`, `_portal.yml`, `ci.yml`, `db.yml` | boot completo com mapas fora do git ainda não roda no CI | CI-03 (aberta) |
| 06/11/2026 | + alvo `bot-test` no `Makefile` | laboratório de bots fora de `make ci` e sem dashboard | BOT-08 (aberta) |
