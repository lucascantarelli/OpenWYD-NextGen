---
name: wyd-server-ops
description: Sobe, inspeciona e derruba a stack Docker do WYD em Rust (Postgres, gateway, dois canais, portal e monitoramento), valida portas e saúde, e diagnostica boot. Use sempre que a tarefa envolver rodar a stack, ver logs, resetar banco ou investigar por que um serviço não sobe.
---

# Operação da stack WYD (Rust)

Norma: `docs/rust/05-stack-docker.md` §2 (serviços/portas), §5 (banco/migrations),
§6 (canais) e §7 (proibido); runbooks em `docs/rust/06-observabilidade-runbooks.md`.
Não replique a tabela de containers nem o mapa de portas.

## Subir e confirmar

```bash
make stack-config                 # valida base+prod, sem subir
make stack-up                     # db + gateway + 2 canais + portal + monitoramento
docker compose ps                 # Up != pronto
curl -s localhost:7600/health     # gateway + Postgres
curl -s localhost:3000/api/health # 200 com catálogo
```

Se o portal der 502, confira o gateway e os logs antes de suspeitar do portal. Portal em
hot reload: `make portal-dev`. Diagnóstico de container/porta/banco: doc §5–§6 + runbooks.

## Limites

- `docker compose down -v`, `DROP`/`TRUNCATE` ou reset de volume só com pedido explícito;
  backup antes. Não publique 5432 fora do loopback.
- Não reintroduza nomes, portas, variáveis ou topologia da stack Go antiga.
- Suba gateway/canais pelo compose/binário da issue; não invente processo “na mão”.
- Não reinicie a stack durante teste humano em andamento.

## Prova

`make stack-config`; `make stack-up` + os três health checks; logs do serviço que falhou.
