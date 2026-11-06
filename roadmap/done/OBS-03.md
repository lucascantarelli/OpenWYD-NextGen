# OBS-03 — Dashboard e quatro alertas

**Fase:** revisão · **Depende de:** OBS-01 ✅ · **Esforço:** M · **Estado:** ✅ concluída em 06/11/2026

## Objetivo

Um dashboard no Grafana (perfil opcional do compose) e quatro alertas testados: fila de entrega
crescendo, RPC com erro > 1%, mundo sem tick por 5 s e disco < 10%.

## Resultado

- Perfil `monitoring` no `docker-compose.yaml` com `prometheus` (`127.0.0.1:9090`), `grafana`
  (`127.0.0.1:3001`, porque `:3000` é do portal) e `node-exporter` (`127.0.0.1:9100`). Sem o
  perfil, a stack de jogo não muda.
- `monitoring/prometheus.yml` coleta os cinco alvos Go (`tmserver:9101`, `tmserver2:9101`,
  `dbserver:9102`, `binserver:9103`, `webserver:9104`), o próprio Prometheus e o exportador do host.
- `monitoring/alerts.yml` com as quatro regras: `DeliveryQueueBacklog` (> 50 linhas por 2 min),
  `GRPCErrorRate` (erro/total > 1% por 2 min), `WorldTickStalled`
  (`time() - world_tick_timestamp_seconds > 5`, `for: 5s`) e `DiskSpaceLow` (< 10% livre por 2 min).
- O alerta de tick precisava de uma fonte explícita: `world.World` passou a gravar
  `world_tick_timestamp_seconds` no `tickEvent.apply` **antes** do hook, então um tick que trava
  também congela o gauge (rate de `world_events_total{kind="tick"}` não distingue "parou" de
  "nunca rodou").
- `monitoring/grafana/`: provisionamento do datasource e do dashboard `OpenWYD — visão geral`
  (serviços no ar, conexões CPSock, fila, idade do último tick, tick p50/p95/p99, RPC por código,
  pool, eventos por tipo, disco e fila ao longo do tempo).
- `monitoring/alerts_test.yml`: seis casos de regra (quatro disparando, dois saudáveis), rodados
  pelo `promtool test rules` — o mesmo motor de regras do Prometheus, sem depender de escrever na
  fila de entrega do banco de desenvolvimento.
- `node-exporter` monta `/:/host:ro` com `--path.rootfs=/host`; o `rslave` do receituário padrão é
  recusado pelo Docker Desktop e foi documentado no compose.
- `docs/migration/config-ops.md` documenta as portas, o perfil, as regras, o comando de validação e
  a limitação do disco em Docker Desktop/Windows (mede a VM do Docker, não o volume `C:`).

## Evidência

```bash
$ docker compose --profile monitoring up -d prometheus grafana node-exporter
openwyd-prometheus-1 Up · openwyd-grafana-1 Up · openwyd-node-exporter-1 Up

$ curl -s http://127.0.0.1:9090/-/ready            # 200
$ curl -s http://127.0.0.1:3001/api/health         # 200

# /api/v1/targets?state=active
binserver  http://binserver:9103/metrics    up
dbserver   http://dbserver:9102/metrics     up
node       http://node-exporter:9100/metrics up
prometheus http://localhost:9090/metrics    up
tmserver   http://tmserver:9101/metrics     up
tmserver2  http://tmserver2:9101/metrics    up
webserver  http://webserver:9104/metrics    up

# /api/v1/rules?type=alert
DeliveryQueueBacklog  inactive  health=ok
GRPCErrorRate         inactive  health=ok
WorldTickStalled      inactive  health=ok
DiskSpaceLow          inactive  health=ok
# /api/v1/alerts → 0 (nenhuma condição real no momento)

# Grafana: provisioning log "finished to provision dashboards"
# /api/search?query=OpenWYD → "OpenWYD — visão geral"

# métrica nova exposta pelo tmserver reconstruído
world_tick_timestamp_seconds{service="tmserver"} 1.793978497e+09

# regras: validação e simulação
$ promtool check rules /m/alerts.yml
  SUCCESS: 4 rules found                       # exit 0
$ promtool test rules alerts_test.yml
  SUCCESS                                      # exit 0
```

Casos simulados pelo `alerts_test.yml`:

| Alerta | Caso que dispara | Caso saudável |
|---|---|---|
| `DeliveryQueueBacklog` | fila 0 → 80 linhas, dispara em 6m | 0 linhas em 2m |
| `GRPCErrorRate` | 2 erros em 101 RPCs (1,98%), dispara em 6m | só `code="OK"` |
| `WorldTickStalled` | timestamp congelado, `time() - ts = 300`, dispara em 5m | timestamp acompanha o relógio |
| `DiskSpaceLow` | 5/100 livres (5%), dispara em 4m | 50/100 (50%) |

## Critérios de aceite

- [x] Prometheus coleta os cinco alvos sem target down (7 alvos `up`, incluindo Prometheus e node).
- [x] Dashboard mostra tick p95 e fila (provisionado e consultável no Grafana).
- [x] Os quatro alertas disparam na simulação e voltam a OK ao remover a causa (promtool).
- [x] Perfil opcional: sem `--profile monitoring` a stack de jogo não muda.

## Riscos/rollback

Rollback remove o perfil e `monitoring/`; nada no código de jogo depende dele além do gauge
`world_tick_timestamp_seconds` (que fica sem uso). Em Docker Desktop o alerta de disco cobre a VM
do Docker — não o disco do Windows; num host Linux ele mede o sistema de arquivos real.
