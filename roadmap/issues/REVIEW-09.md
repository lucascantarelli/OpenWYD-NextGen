# REVIEW-09 — Observabilidade e segurança operacional

**Fase:** revisão · **Depende de:** 55 · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Sair de "só log" para diagnóstico com número: métricas por serviço, `trace_id` correlacionando
cliente → tmServer → dbServer → banco, alertas das filas e um checklist de segurança que o CI cobra.

## Contexto/evidência

- `logpipe/` existe (container + `Dockerfile`) e o compose tem healthchecks; **não há `/metrics`**,
  tracing nem alerta (dívida **D7** em `docs/analysis/qualidade-e-processo.md`).
- Filas que podem crescer sem ninguém ver: `delivery_queue` (grants da web), outbox de auditoria,
  reconexão gRPC do tmServer ao dbServer.
- Segredos: argon2id (`internal/secret`), `SESSION_SECRET` obrigatório fora de dev, mTLS opcional
  por overlay — mas não há verificação automática de que o ambiente de dev não vazou para produção
  (`.env` com senha `dev`).
- Incidentes recentes que teriam sido mais rápidos com métrica: build local derrubado por disco cheio,
  `npc_shop_item` duplicado por reconcile (6.711 × 3.538), cliente congelando (log sem correlação).

## Passos

1. **Métricas**: endpoint `/metrics` (Prometheus) em `tmserver`, `webserver`, `dbserver` e `binserver`
   com o mínimo útil: conexões ativas, tick do mundo (duração), fila de entrega, RPC por método/erro,
   latência p95 do banco.
2. **Correlação**: `trace_id` gerado no aceite da conexão do cliente e propagado por metadata gRPC;
   formato de log único (`ts, level, svc, trace_id, msg, kv`).
3. **Dashboards e alertas**: um dashboard no Grafana (via compose, perfil opcional) e 4 alertas:
   `delivery_queue` crescendo, RPC com erro > 1%, mundo sem tick por 5 s, disco < 10%.
4. **Segurança operacional**: checklist no CI (`gitleaks`), verificação de que `SESSION_SECRET` !=
   default e de que `.env` não é versionado (já coberto por `.gitignore`, mas falta o gate), e
   rotação documentada de segredos.
5. **Backup**: ligar o alerta de falha do `db-backup.sh` e registrar a data do último restore
   validado (REVIEW-05).

## Arquivos

- `tmserver/cmd/tmserver/` e `webserver/` (endpoint de métricas)
- `logpipe/` (formato de log, coleta)
- `docker-compose.yaml` (perfil `monitoring`)
- `docs/migration/config-ops.md` (portas e variáveis novas)

## Testes/verificação

```bash
curl -s http://127.0.0.1:8281/metrics | head
curl -s http://127.0.0.1:7600/metrics | head
docker compose --profile monitoring up -d
```

## Critérios de aceite

- [ ] `/metrics` nos 4 serviços com as métricas listadas.
- [ ] Um log de uma requisição do portal atravessa BFF → web-api → banco com o mesmo `trace_id`.
- [ ] 4 alertas configurados e testados (simular fila crescendo).
- [ ] `gitleaks` no CI e checklist de segredos documentado.
- [ ] Falha de backup gera alerta.

## Riscos/rollback

Risco: instrumentação adicionar latência no laço do mundo — proibido bloquear a goroutine única.
Mitigação: métricas em memória, exportadas fora do loop (mesmo padrão de `World.Go`). Rollback:
endpoint desligado por flag.

## Issues que este review abre

- `OBS-01` `/metrics` por serviço
- `OBS-02` `trace_id` ponta a ponta
- `OBS-03` dashboard + 4 alertas
- `OBS-04` `gitleaks` e checklist de segredos
