# DEVOPS-01 — Padronização de imagens, containers e composes

**Fase:** infraestrutura · **Depende de:** OBS-03 (perfil monitoring) · **Esforço:** M · **Estado:** ⏳ proposta (aguarda janela sem teste de cliente em andamento)

## Objetivo

Um nome por coisa: serviço, container e imagem seguem o mesmo padrão, os composes são **base +
overrides** (dev, mTLS, monitoring) e a ordem de subida é declarada por dependência real, não por
sorte de `depends_on: started`.

## Por que não foi feito junto da OBS-03

Renomear serviço muda **hostname** dentro da rede do compose (`dbserver:7514`, `webserver:7600`,
`logpipe:7788`, alvos do Prometheus) e o `logpipe` tem **whitelist de nomes de container** para o
`/logs` do portal. Fazer isso enquanto o cliente gráfico está sendo testado derruba o ambiente do
teste no meio. Esta issue é a janela dedicada, com o teste do cliente parado.

## Padrão pedido

| Serviço (hostname) | Container | Imagem | Papel |
|---|---|---|---|
| `openwyd-nextgen1-tmsrv` | `openwyd-nextgen1-tmsrv` | `openwyd-nextgen1-tmsrv:latest` | canal 1 (ServerIndex 0), CPSock 8281 |
| `openwyd-nextgen2-tmsrv` | `openwyd-nextgen2-tmsrv` | `openwyd-nextgen2-tmsrv:latest` | canal 2 (ServerIndex 1) |
| `openwyd-dbserver` | `openwyd-dbserver` | `openwyd-dbserver:latest` | persistência gRPC 7514 |
| `openwyd-binserver` | `openwyd-binserver` | `openwyd-binserver:latest` | billing gRPC 3000 |
| `openwyd-webserver` | `openwyd-webserver` | `openwyd-webserver:latest` | web-api gRPC 7600 |
| `openwyd-postgres` | `openwyd-postgres` | `postgres:17-alpine` (upstream) | banco |
| `openwyd-portal` | `openwyd-portal` | `openwyd-portal:latest` | Next.js 3000 |
| `openwyd-logpipe` | `openwyd-logpipe` | `openwyd-logpipe:latest` | logs por SSE |
| `openwyd-prometheus` | `openwyd-prometheus` | `prom/prometheus:v3.5.0` | métricas |
| `openwyd-grafana` | `openwyd-grafana` | `grafana/grafana:12.2.0` | dashboards |
| `openwyd-node-exporter` | `openwyd-node-exporter` | `prom/node-exporter:v1.9.1` | disco |

Imagens de terceiro mantêm o nome upstream (não se renomeia o que não é nosso); os nossos passam a
ser construídos com `image:` explícito, para o nome não depender do diretório.

## Passos

1. **Composes em camadas**: `docker-compose.yaml` (base, sem portas publicadas), 
   `docker-compose.override.yaml` (dev: publica 8281/80/3000/7600 em loopback e os IPs de canal),
   `docker-compose.monitoring.yaml` (hoje `profiles: monitoring`), `docker-compose.mtls.yaml`
   (existe) e `docker-compose.dev.yaml` (hot reload, hoje separado). Comentar em pt-BR o que cada
   camada acrescenta.
2. **Renomear** serviço/container/imagem conforme a tabela; atualizar em conjunto:
   `logpipe` (whitelist do `/logs`), `portal` (`WEB_API_ADDR`, `LOG_PIPE_URL`),
   `monitoring/prometheus.yml` (alvos), comandos do `tmserver` (`-dbserver`, `-binserver`),
   docs que citam hostnames e `scripts/*.sh`.
3. **Dependências de verdade**:
   - `openwyd-postgres` com `healthcheck: pg_isready` (já tem) e `db-roles` com
     `service_completed_successfully` (já tem);
   - `openwyd-dbserver`/`openwyd-webserver` só depois de `db-roles`;
   - `openwyd-nextgen1-tmsrv` só depois de `dbserver` **healthy**, não `started`;
   - `openwyd-portal` depois de `webserver` e `logpipe` healthy.
4. **Healthcheck dos serviços Go**: hoje não há, porque a imagem é `distroless/static` sem shell.
   Duas opções, decidir na issue: (a) `grpc_health_probe` copiado para a imagem, batendo no
   `/healthz` que já existe em 9101–9104; (b) trocar a base de runtime por `alpine` com `wget`
   (mais simples, +5 MB). Recomendo (a): mantém a superfície mínima.
5. **Fechar** o `Dockerfile` único com `ARG SVC` — ele funciona, mas cada serviço deveria ter nome
   de imagem próprio (passo 2) e o build paralelo desta máquina derruba o daemon; manter o
   `make docker-build` serializado.

## Testes/verificação

```bash
docker compose config --quiet                     # base + overrides válidos
docker compose -f docker-compose.yaml -f docker-compose.override.yaml up -d
docker compose ps --format '{{.Service}} {{.Name}} {{.Image}}'   # nome==container==imagem
curl -s http://127.0.0.1:9101/metrics | head -3   # serviços no ar após o rename
# /logs do portal listando os serviços (whitelist do logpipe atualizada)
```

## Critérios de aceite

- [ ] Serviço, container e imagem seguem a tabela; nenhum `openwyd-*` genérico sobra.
- [ ] Base + overrides: `docker compose config` verde em todas as combinações.
- [ ] Ordem de subida por healthcheck real (sem `started` em dependência crítica).
- [ ] `/logs` do portal continua funcionando após o rename (whitelist conferida).
- [ ] Docs e scripts sem hostname antigo.

## Riscos/rollback

O rename quebra hostname interno e a whitelist do `logpipe` — fazer em um único commit, com a
stack parada. Rollback é `git revert` do commit e `docker compose up -d --build`.
