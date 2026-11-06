# OBS-01 — `/metrics` por serviço

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ✅ concluída (06/11/2026)

> A `REVIEW-09` separa esta issue em quatro: `OBS-01` (endpoint e métricas), `OBS-02`
> (`trace_id`), `OBS-03` (dashboard + alertas) e `OBS-04` (`gitleaks`/segredos). O arquivo
> antigo duplicava os quatro escopos; esta entrega fecha **o endpoint e as métricas**, e as
> demais linhas de aceite continuam nas issues filhas.

## Objetivo

Cada processo Go expõe `/metrics` no formato texto do Prometheus, com conexões CPSock, duração
do loop do mundo, fila de entrega, pool/RPC e um `up` por serviço — sem bloquear `World.Run`.

## O que foi entregue

- `internal/metrics`: registry próprio (sem dependência Prometheus), contador/gauge/histograma,
  labels escapados, `GaugeFunc` avaliado só no scrape, `Handler`/`Serve` e interceptor gRPC.
- Endpoints em porta separada, publicados em loopback no compose:
  `tmserver :9101`, `dbserver :9102`, `binserver :9103`, `webserver :9104` e o segundo canal
  `tmserver2 :9111`. `-metrics-addr ""` desliga; `/healthz` e `/readyz` acompanham.
- `tmserver`: `cpsock_connections_active`, `cpsock_connections_total`,
  `world_event_duration_seconds{kind=frame|tick|callback|other}` e `world_events_total`.
  A instrumentação do loop só toca em memória (mutex curto), fora de qualquer I/O.
- `dbserver`/`webserver`: `db_pool_{acquired,idle,total,max}_connections`,
  `delivery_queue_rows` (consulta com timeout no scrape) e `grpc_requests_total` /
  `grpc_request_duration_seconds` por método e código.
- `binserver`: `up` e o mesmo interceptor gRPC.
- `docs/migration/config-ops.md`: portas de observabilidade documentadas.

## Evidência (06/11/2026, stack reconstruída)

```text
:9101 /healthz -> ok
up{service="tmserver"} 1
cpsock_connections_active{service="tmserver"} 0
cpsock_connections_total{service="tmserver"} 8
world_event_duration_seconds_bucket{kind="callback",le="0.001",service="tmserver"} 10
world_event_duration_seconds_count{kind="callback",service="tmserver"} 10

:9102 /healthz -> ok
up{service="dbserver"} 1
grpc_requests_total{code="OK",method="/db.v1.ContentCatalogService/FetchItemCatalog",service="dbserver"} 2
db_pool_total_connections{service="dbserver"} 2
delivery_queue_rows{service="dbserver"} 0

:9103 /healthz -> ok        up{service="binserver"} 1
:9104 /healthz -> ok        up{service="webserver"} 1
db_pool_total_connections{service="webserver"} 1
delivery_queue_rows{service="webserver"} 0
:9111 /healthz -> ok        up{service="tmserver"} 1   (tmserver2)
```

Os histogramas são a fonte do p95 (`histogram_quantile`), que será ligado no dashboard da
`OBS-03`; não há coletor Prometheus nesta entrega. Latência de banco é observável pelo
histograma por método do `dbserver`/`webserver` e pelo estado do pool.

Testes e regressão:

```text
go test ./internal/metrics ./tmserver/internal/world ./tmserver/internal/handler   exit 0
TestE2ESmokeLogin               PASS (3.03s)
TestE2EChannelChange            PASS (0.38s)  canal 1 -> /srv -> canal 2
TestE2EServerChangeSameChannel  PASS (3.22s)
docker compose logs --since=6m db | grep -c 42P01  -> 0
```

## Critérios de aceite

- [x] `/metrics` nos 4 serviços, em porta própria, com `up` por processo.
- [x] Conexões CPSock, tick do mundo, fila de entrega, pool e RPC por método/erro.
- [x] Nenhum bloqueio novo dentro de `World.Run` (observação só em memória).
- [ ] Dashboard/alertas e p95 consultável: [OBS-03](../issues/OBS-03.md).
- [ ] `trace_id` ponta a ponta: [OBS-02](../issues/OBS-02.md).
- [ ] `gitleaks` e checklist de segredos: [OBS-04](../issues/OBS-04.md).

## Riscos/rollback

`-metrics-addr ""` (ou `W2PP_METRICS_ADDR=""`) desliga o endpoint sem tocar no jogo. A porta
publicada é só loopback; expor fora da máquina exige decisão explícita de rede.
