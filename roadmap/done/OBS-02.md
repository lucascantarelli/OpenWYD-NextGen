# OBS-02 — `trace_id` ponta a ponta

**Fase:** revisão · **Depende de:** OBS-01 ✅ · **Esforço:** M · **Estado:** ✅ concluída em 06/11/2026

## Objetivo

Uma requisição do portal é rastreável por um `trace_id` do BFF ao web-api, do tmServer ao
dbServer e ao SQL, com o mesmo formato de log (`ts, level, svc, trace_id, msg, kv`).

## Resultado

- `internal/metrics/trace.go`: `NewTraceID` (16 bytes → 32 hex minúsculo), `ValidTraceID`
  (só aceita o formato), `WithTraceID`/`TraceID`/`EnsureTraceID` no contexto e
  `UnaryClientInterceptor`, que injeta `x-trace-id` em toda RPC de saída (gera um quando o
  contexto não tem).
- `UnaryServerInterceptor` passou a receber o `*slog.Logger`, chama `EnsureTraceID`, devolve o id
  no header da resposta e registra **uma linha por RPC** (`msg="grpc request" svc=… trace_id=…
  method=… code=… dur_ms=…`). Nenhum id entra como label: `grpc_requests_total` continua só com
  `service/method/code`.
- `world.Session` ganhou `TraceID`, gerado no `connectEvent` (aceite CPSock) e presente no log
  `connection accepted`. `handler/login.go` usa
  `metrics.WithTraceID(context.Background(), s.TraceID)` no `World.Go` do login, e as linhas
  `relaying to dbServer` / `OK` / `backend error` levam `trace_id`.
- `tmserver/cmd/tmserver/main.go` instala o interceptor cliente nos dois `grpc.NewClient`
  (dbServer e binServer); `dbserver`, `webserver` e `binserver` passaram o logger ao interceptor de
  servidor, então o id recebido aparece no log deles.
- Portal: `server/trace.ts` (`newTraceId`, `validTraceId`, `traceMetadata`, header
  `x-trace-id`), `promisify` aceita o id como segundo argumento e `adminCall` gera um por ação
  (inclusive para auditar a negação de permissão), gravando `traceId` no `AuditEntry`.

## Evidência

```bash
$ go test ./internal/metrics/ -run 'Trace|Interceptor|Concurrent' -v
TestNewTraceIDFormat PASS · TestValidTraceID PASS · TestServerInterceptorUsesIncomingTraceID PASS
TestServerInterceptorGeneratesWhenMissing PASS · TestServerInterceptorRejectsForgedTraceID PASS
TestConcurrentCallsDoNotMixTraceIDs PASS · TestClientInterceptorGeneratesForOutgoingCall PASS
TestTraceIDIsNeverAMetricLabel PASS

$ go test ./tmserver/internal/world/ -run 'TestConnectAssignsTraceID|TestTickSetsTimestampGauge' -v
--- PASS: TestConnectAssignsTraceID (0.07s)
--- PASS: TestTickSetsTimestampGauge (0.04s)

$ cd portal && npx vitest run src/server/trace.test.ts
 ✓ src/server/trace.test.ts (6 tests) 21ms
```

Prova de vermelho (propagação do cliente desligada, teste restaurado depois):

```text
--- FAIL: TestServerInterceptorUsesIncomingTraceID (0.03s)
    trace_test.go:141: trace_id no header = "97b51499dc3fb03644ad99b5aca0896c", quer "0123456789abcdef0123456789abcdef"
FAIL
ok  openwyd-nextgen/internal/metrics   # restaurado
```

Login e2e real (`golang:1.26-alpine`, rede do compose, containers reconstruídos):

```text
--- PASS: TestE2ESmokeLogin (0.12s)   # LOGIN OK — CNFAccountLogin with full character-selection body
```

Mesmo id nos dois processos (tmserver ↔ dbserver):

```text
tmserver  msg="connection accepted" conn=1 ip=172.28.0.1:46854 trace_id=2408bac88930eced661a7b9e604433e4
tmserver  msg="account login: relaying to dbServer" conn=1 trace_id=2408bac88930eced661a7b9e604433e4 account=test
tmserver  msg="account login: OK" conn=1 trace_id=2408bac88930eced661a7b9e604433e4 account=test id=1 role=admin chars=1
dbserver  msg="grpc request" svc=dbserver trace_id=2408bac88930eced661a7b9e604433e4 method=/db.v1.AccountService/AccountLogin code=OK dur_ms=112
dbserver  msg="grpc request" ... method=/db.v1.AccountService/ListCharacters code=OK dur_ms=8
dbserver  msg="grpc request" ... method=/db.v1.AccountService/LoadCargo code=OK dur_ms=2
dbserver  msg="grpc request" ... method=/db.v1.AccountService/ListPendingDeliveries code=OK dur_ms=0
```

O BFF envia o mesmo header: `portal/src/server/trace.test.ts` prova que `promisify(fn)(req, id)`
passa `x-trace-id: <id>` no metadata (e que sem id um válido é gerado).

## Critérios de aceite

- [x] `trace_id` no aceite da conexão e no BFF, propagado por metadata.
- [x] Login e2e gera a mesma correlação em tmServer e dbServer (acima).
- [x] Nenhum `trace_id` como label de métrica (`TestTraceIDIsNeverAMetricLabel`).
- [x] Teste concorrente prova que ids não se misturam (40 chamadas, cada id em exatamente uma linha).

## Riscos/rollback

Metadata ausente não quebra cliente interno: o servidor gera um id. Rollback remove só o
enriquecimento do log e o interceptor cliente; o header é opcional nas duas pontas.
