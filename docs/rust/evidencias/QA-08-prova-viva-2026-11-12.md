# Evidência — QA-08: dashboards com série real do motor

> **Issue:** `QA-08` — Dashboards Grafana + alertas (Rust, banco, produção), concluída.
> **Registro canônico:** [`roadmap-rust/changelog.md`](../../../roadmap-rust/changelog.md).
> **Data da coleta:** 2026-11-12.
> **Objetivo:** fechar o resíduo “painéis sem série do motor” com os dois canais reais.

## Sintoma e causa

O Prometheus não tinha o job `wyd-server`: `/api/v1/targets` listava apenas Prometheus,
node-exporter e gateway. O log do container repetia:

```text
Error adding file watch ... discovery=file config=wyd-server
path=/etc/prometheus/targets/ err="no such file or directory"
```

O container em execução ainda montava o diretório **antigo** `monitoring/prometheus`, removido
quando o monitoramento foi movido para `docker/monitoring/prometheus`. O `prometheus.yml` e o
`targets/wyd-server.json` atuais estavam corretos; o processo é que lia o mount velho.

## Correção aplicada

```bash
docker compose up -d --no-deps prometheus
```

Nenhum código ou configuração versionada foi alterado nesta correção: apenas o container foi
recriado com o bind mount atual.

## Prova

Alvos ativos:

```text
canal 1: up
canal 2: up
```

Séries reais consultadas no Prometheus:

```text
wyd_ticks_total       canal 1 = 927210   canal 2 = 927319
wyd_entities_active   canal 1 = 6110     canal 2 = 6110
```

Smoke de monitoramento (`node tools/dev/monitoring-smoke.mjs`, exit `0`):

```text
OK alvo wyd-server canal 1 up (http://server-canal-1:9898/metrics)
OK alvo wyd-server canal 2 up (http://server-canal-2:9898/metrics)
OK regras carregadas: 10 (gateway + servidor + motor)
OK dashboard provisionado: WYD — saúde do motor (Rust) (12 painéis)
smoke de monitoramento: tudo verde
```

O dashboard tem 12 painéis e o Prometheus tem 10 regras; os dois canais publicam tick,
orçamento de tick, entidades, pacotes, recusas, handshakes e sessões. O runbook
`docs/rust/06-observabilidade-runbooks.md` foi corrigido para não afirmar mais que o arquivo de
alvos está vazio.

## Gate

```text
make check -> 0
issue-status: 36/36 issues ativas, cobertura incompleta 0
docs: 36 issues, 9 crates, 10 membros, 48 migrations, 181 docs, 264 links
tools: 7/7
```

## Limites declarados

- O alerta de **falha de migration** continua sem métrica na QA-04; criar essa métrica é mudança
  de contrato e fica registrado como lacuna, não como cobertura inventada.
- A prova é de métricas/alvos/dashboard provisionado; não foi feita inspeção visual de cada
  painel no Grafana.
