# OBS-01 — `/metrics` por serviço e `trace_id` ponta a ponta

**Fase:** revisão · **Depende de:** — · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Diagnóstico com número: cada serviço expõe métricas mínimas, e uma requisição do portal é rastreável
por `trace_id` do BFF até o banco.

## Contexto/evidência (05/11/2026)

```bash
git grep -l '"/metrics"' -- '*.go' | wc -l    # 0
```

- Não existe endpoint de métricas em nenhum serviço; `logpipe/` só transporta log para o portal.
- Filas invisíveis: `delivery_queue` (grants da web), outbox de auditoria, reconexão gRPC
  tmServer↔dbServer.
- Incidentes desta sessão que teriam sido mais rápidos com métrica: stack parada sem ninguém notar
  (`docker compose ps` vazio), `dbserver` falhando por DNS do `db` e seguindo, `go build` vermelho sem
  gate, build frio de ~10 min sem visibilidade.

## Passos

1. Endpoint `/metrics` (formato Prometheus, sem dependência pesada) em `tmserver`, `webserver`,
   `dbserver` e `binserver`, em porta separada do gRPC quando houver colisão.
2. Métricas mínimas: conexões CPSock ativas, duração do tick do mundo (histograma), tamanho de
   `delivery_queue`, RPC por método/erro, latência p95 do banco, e um `up` por serviço.
3. `trace_id` gerado no aceite da conexão do cliente e propagado por metadata gRPC; formato de log
   único (`ts, level, svc, trace_id, msg, kv`).
4. Perfil `monitoring` no compose (Prometheus + Grafana) e 4 alertas: fila crescendo, erro de RPC >
   1%, mundo sem tick por 5 s, disco < 10%.
5. Instrumentação do mundo **fora** do loop único (contadores em memória exportados pelo coletor) —
   nunca bloquear `World.Run`.

## Arquivos

- `tmserver/cmd/tmserver/`, `webserver/cmd/webserver/`, `dbserver/cmd/dbserver/`,
  `binserver/cmd/binserver/`
- `internal/metrics/` (novo, se compartilhado)
- `docker-compose.yaml` (perfil `monitoring`), `docs/migration/config-ops.md` (portas novas)

## Testes/verificação

```bash
curl -s http://127.0.0.1:8281/metrics | head
curl -s http://127.0.0.1:7600/metrics | head
docker compose --profile monitoring up -d
```

## Critérios de aceite

- [ ] `/metrics` nos 4 serviços com as métricas listadas.
- [ ] Um log de requisição do portal com o mesmo `trace_id` em BFF, web-api e banco.
- [ ] 4 alertas configurados e testados (simular fila crescendo).
- [ ] Nenhum bloqueio novo dentro do loop do mundo.

## Riscos/rollback

Risco: instrumentação adicionar latência no laço do mundo. Mitigação: métricas em memória e coleta
fora do loop; rollback desliga o endpoint por flag.
