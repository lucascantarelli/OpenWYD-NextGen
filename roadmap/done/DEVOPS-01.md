# DEVOPS-01 — Padronização de imagens, containers e composes

**Fase:** infraestrutura · **Depende de:** OBS-03 (perfil monitoring) · **Esforço:** M · **Estado:** ✅ feito (06/11/2026)

## Objetivo

Um nome por coisa: serviço, container e imagem seguem o mesmo padrão, os composes são **base +
overrides** (dev, mTLS, monitoring) e a ordem de subida é declarada por dependência real, não por
sorte de `depends_on: started`.

## Resultado (o que ficou valendo)

| Serviço (hostname) = container | Imagem | Papel |
|---|---|---|
| `openwyd-nextgen1-tmsrv` | `openwyd-nextgen1-tmsrv:latest` | canal 1 (ServerIndex 0), CPSock 8281 |
| `openwyd-nextgen2-tmsrv` | `openwyd-nextgen2-tmsrv:latest` | canal 2 (ServerIndex 1), CPSock 8282 |
| `openwyd-dbserver` | `openwyd-dbserver:latest` | persistência gRPC 7514 |
| `openwyd-binserver` | `openwyd-binserver:latest` | billing gRPC 3000 |
| `openwyd-webserver` | `openwyd-webserver:latest` | web-api gRPC 7600 |
| `openwyd-postgres` | `postgres:17-alpine` | banco |
| `openwyd-db-roles` | `postgres:17-alpine` | one-shot das roles (DB-01) |
| `openwyd-portal` | `openwyd-portal:latest` | Next.js 3000 |
| `openwyd-logpipe` | `openwyd-logpipe:latest` | logs por SSE |
| `openwyd-prometheus` | `prom/prometheus:v3.5.0` | métricas (perfil `monitoring`) |
| `openwyd-grafana` | `grafana/grafana:12.2.0` | dashboards (perfil `monitoring`) |
| `openwyd-node-exporter` | `prom/node-exporter:v1.9.1` | disco do host (perfil `monitoring`) |
| `openwyd-mkdocs` | `openwyd-mkdocs:latest` | site de docs (perfil `docs`) |

Imagens de terceiro mantêm o nome upstream: renomear o que não é nosso só criaria divergência com a
documentação do fornecedor.

## Camadas

| Arquivo | Papel |
|---|---|
| `docker-compose.yaml` | base: serviço/imagem/comando, healthchecks, dependências, volumes — **sem porta publicada** |
| `docker-compose.override.yaml` | dev (carregado automático): portas em loopback + os IPs de canal `192.168.100.3/.4` |
| `docker-compose.mtls.yaml` | opcional: certificados de cliente/servidor nos links gRPC |

`docker compose up -d` = base + override. `docker compose -f docker-compose.yaml up -d` sobe a base
pura, sem porta no host (produção). O projeto continua `openwyd`, então a rede segue
`openwyd_default`, que é o que os comandos e2e e os scripts usam.

## Ordem de subida

`depends_on` com `condition:` — nunca `started` em dependência crítica:

- `openwyd-postgres` → `pg_isready`;
- `openwyd-db-roles` → `service_completed_successfully` (antes de dbserver/webserver);
- `openwyd-dbserver` e `openwyd-webserver` → Postgres saudável + roles concluídas;
- `openwyd-nextgen{1,2}-tmsrv` → dbserver **e** binserver saudáveis;
- `openwyd-portal` → webserver e logpipe saudáveis;
- `openwyd-grafana` → Prometheus saudável.

## Healthcheck dos serviços Go

A imagem final é `distroless/static` (sem shell, sem `wget`), então o healthcheck é o próprio
binário: `CMD ["/app","healthcheck"]` consulta `http://127.0.0.1:<porta>/healthz` e sai 0/1 — mesmo
desenho que o `logpipe` já usava. Implementação em `internal/metrics/healthcheck.go`
(`HealthcheckArgs` no topo do `main` de cada serviço; porta de `W2PP_METRICS_ADDR`, default
9101–9104).

**Descartado:** copiar `grpc_health_probe` para a imagem (mais um binário de terceiro para manter) e
trocar a base por `alpine` com `wget` (+5 MB e superfície maior). O binário consultando o seu próprio
`/healthz` resolve sem nenhum dos dois.

## Arquivos removidos na reorganização

- `docker-compose.dev.yaml` — recorte de hot-reload (tmserver com `go run`, sem `-dbserver` e sem
  `-content`) que duplicava metade de um serviço e não era uma topologia jogável. O caminho virou
  `docker compose up -d openwyd-postgres` + `make run`.
- `Dockerfile.dev` — existia só para aquele compose; sem ele ficaria órfão.

## Ajustes que o rename exigiu

- `logpipe/main.go`: `known` passou a listar os nomes do compose e `matchesService` compara o nome
  inteiro (o container tem exatamente o nome do serviço), aceitando o sufixo antigo (`-1`/`_1`).
- `monitoring/prometheus.yml`: alvos `openwyd-nextgen1-tmsrv:9101`, `openwyd-nextgen2-tmsrv:9101`,
  `openwyd-dbserver:9102`, `openwyd-binserver:9103`, `openwyd-webserver:9104`,
  `openwyd-node-exporter:9100`.
- `portal/src/server/docker.ts` e a aba de logs: whitelist com os nomes novos (o logpipe casa por
  nome de container, então os ids são os do compose).
- `scripts/gen-certs.sh`: SANs do certificado de servidor com os nomes novos, para o tmClient
  validar o peer pela authority do dial sem `W2PP_TLS_SERVER_NAME`.
- `scripts/run-local.sh`, `scripts/e2e-worldevents.sh`, `Makefile`, `AGENTS.md`, docs e skills.

## Verificação

```bash
docker compose config --quiet
docker compose build openwyd-dbserver openwyd-binserver openwyd-webserver \
                     openwyd-logpipe openwyd-portal \
                     openwyd-nextgen1-tmsrv openwyd-nextgen2-tmsrv
docker compose down && docker compose up -d
docker compose ps --format '{{.Service}} {{.Name}} {{.Image}} {{.Status}}'
make test-channel    # 127.0.0.1:8282 no loopback pelo override
bash scripts/run-local.sh
make gate && go build ./... && go vet ./... && go test ./...
```
