# OBS-03 — Dashboard e quatro alertas

**Fase:** revisão · **Depende de:** OBS-01 ✅ · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Um dashboard no Grafana (perfil opcional do compose) e quatro alertas testados: fila de entrega
crescendo, RPC com erro > 1%, mundo sem tick por 5 s e disco < 10%.

## Contexto

Os endpoints já existem e foram validados em 06/11: `:9101` (tmserver), `:9102` (dbserver),
`:9103` (binserver), `:9104` (webserver) e `:9111` (tmserver2). Falta o coletor e as regras.
O `up`, o histograma do loop e `delivery_queue_rows` já são suficientes para três dos quatro
alertas; o de disco vem do exporter do host.

## Passos

1. Perfil `monitoring` no compose com Prometheus + Grafana e scrape dos cinco alvos.
2. Dashboard com CPSock ativo, tick p95, RPC por código, pool e `delivery_queue_rows`.
3. Quatro regras versionadas em `monitoring/alerts.yml`, com `for` curto no tick (5 s) e
   limites explícitos nos demais.
4. Testar cada alerta simulando a condição (fila com linha presa, RPC forçado a erro, ticker
   parado, disco cheio em ambiente controlado) e registrar a evidência.

## Arquivos

- `docker-compose.yaml` (perfil `monitoring`)
- `monitoring/` (Prometheus, regras, provisionamento do Grafana)
- `docs/migration/config-ops.md`

## Testes/verificação

```bash
docker compose --profile monitoring up -d
curl -s http://127.0.0.1:9090/-/ready
curl -s 'http://127.0.0.1:9090/api/v1/alerts' | jq '.data.alerts | length'
```

## Critérios de aceite

- [ ] Prometheus coleta os cinco alvos sem target down.
- [ ] Dashboard mostra tick p95 e fila.
- [ ] Os quatro alertas disparam na simulação e voltam a OK ao remover a causa.

## Riscos/rollback

Perfil é opcional: sem `--profile monitoring` a stack de jogo não muda. Rollback remove o perfil.
