# OBS-02 — `trace_id` ponta a ponta

**Fase:** revisão · **Depende de:** OBS-01 ✅ · **Esforço:** M · **Estado:** ⏳ proposta

## Objetivo

Uma requisição do portal é rastreável por um `trace_id` do BFF ao web-api, do tmServer ao
dbServer e ao SQL, com o mesmo formato de log (`ts, level, svc, trace_id, msg, kv`).

## Contexto

A `OBS-01` entregou `/metrics` e o interceptor gRPC, mas ainda não há correlação entre linhas de
log de processos diferentes. O `logpipe/` só transporta texto; uma falha de login hoje exige
procurar por conta/horário em três serviços. O `trace_id` **não** vira label de métrica
(cardinalidade): ele pertence ao contexto e ao log.

## Passos

1. Gerar `trace_id` no aceite da conexão CPSock e no BFF; aceitar um `X-Trace-Id` de entrada
   somente se vier no formato esperado.
2. Propagar por metadata gRPC (`x-trace-id`) no cliente e no interceptor de servidor; o
   interceptor devolve o id no contexto e o logger o inclui em todas as linhas.
3. Cobrir o caminho do portal: BFF → web-api → store, e tmServer → dbServer → query.
4. Teste de regressão: um RPC de teste com metadata conhecida produz a mesma linha nos dois
   servidores; sem metadata, o id é gerado e não vaza entre requisições concorrentes.

## Arquivos

- `internal/metrics` (interceptor e helpers de contexto)
- `tmserver/cmd/tmserver/`, `dbserver/cmd/dbserver/`, `webserver/cmd/webserver/`,
  `binserver/cmd/binserver/`
- `portal/src/server/admin.ts` (BFF)

## Testes/verificação

```bash
go test ./internal/metrics/... ./tmserver/internal/world/...
# log de login com o mesmo trace_id em tmserver e dbserver
docker compose logs --since=2m tmserver dbserver | grep trace_id
```

## Critérios de aceite

- [ ] `trace_id` no aceite da conexão e no BFF, propagado por metadata.
- [ ] Login e2e gera a mesma correlção em tmServer e dbServer.
- [ ] Nenhum `trace_id` como label de métrica.
- [ ] Teste concorrente prova que ids não se misturam.

## Riscos/rollback

Risco: metadata ausente quebrar clientes internos. Mitigação: id é opcional na entrada e
gerado quando falta; rollback remove só o enriquecimento do log.
