# Issue 55 — Qualidade: CI completo, banco efêmero e observabilidade

**Fase:** 8 · **Depende de:** 02 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Fechar as dívidas D4–D7 de [`docs/analysis/qualidade-e-processo.md`](../../docs/analysis/qualidade-e-processo.md):
teste de integração que não destrói dados, CI cobrindo tudo com `-race`, cobertura por pacote crítico
e métricas/tracing para diagnosticar produção.

## Contexto

- `internal/store/*_integration_test.go` usa `resetTestSchema`, que **apaga o schema** — por isso
  ninguém roda; a rede de segurança do store não existe na prática.
- CI atual (`.github/workflows/ci.yml`) roda gofmt, build, testes e portal, mas sem Postgres, sem
  `-race` global, sem cobertura e sem imagem.
- Operação hoje é só log (mesmo com `logpipe`/`/logs`); não há métrica nem correlação.

## Passos

1. **Harness de banco efêmero**: `testPool(t)` cria `schema test_<random>`, aplica `Migrate`,
   `SET search_path`, derruba no `t.Cleanup`. Proibir `resetTestSchema` no schema `public`.
2. **CI matriz**: Postgres service, `go test -race ./...`, `-tags=integration`, cobertura publicada com
   piso por pacote (`world`, `handler`, `store`, `protocol` ≥ 70%), `govulncheck`.
3. **Migrações**: teste que aplica todas em base vazia e roda cada `.down.sql` (ordem reversa).
4. **Build reprodutível**: imagens das 5 services + portal no CI, com SBOM e tag por commit.
5. **Observabilidade**: `/metrics` (Prometheus) em `tmserver`/`webserver` (conexões, tick duration,
   fila `delivery_queue`, RPC por código), `trace_id` nos logs (slog handler), health por serviço.
6. **Refatorar wiring**: extrair de `cmd/*/main.go` para `internal/bootstrap` (testável), sem mudar
   comportamento.
7. **Validação única**: gerar as regras de validação compartilhadas (conta, motivo, item) de um schema
   único (ou duplicar com teste de contrato que compare mensagens Go × TS).

## Arquivos

- `internal/store/testdb_test.go`, `internal/store/*_integration_test.go`, `internal/migrations/*_test.go`,
  `.github/workflows/ci.yml`, `tmserver/internal/observability/`, `webserver/internal/observability/`,
  `internal/bootstrap/`, `portal/src/lib/contracts.test.ts`.

## Verificação

```bash
go test -race ./... 2>&1 | tail -20
go test -tags=integration ./internal/store/... 2>&1 | tail -10
curl -s localhost:8281/metrics | head   # endpoint novo (debug port separado)
cd portal && npx tsc --noEmit && npx vitest run
```

## Critérios de aceite

- [ ] Nenhum teste apaga schema `public`; suíte de integração roda em CI com Postgres
- [ ] `go test -race ./...` verde em CI
- [ ] Cobertura publicada com piso por pacote crítico
- [ ] Migrações testadas para cima e para baixo
- [ ] `/metrics` e `trace_id` correlacionando log ↔ RPC
- [ ] `cmd/*/main.go` sem regra de negócio (só wiring)

## Riscos

- **Piso de cobertura virar meta** e gerar teste vazio: piso só nos 4 pacotes críticos, revisão humana.
- **Métrica com cardinalidade alta** (por player): usar buckets, nunca id como label.
