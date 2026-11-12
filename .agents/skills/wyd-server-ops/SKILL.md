---
name: wyd-server-ops
description: Sobe, inspeciona e derruba a stack Docker do WYD em Rust (Postgres, gateway, dois canais, portal e monitoramento), valida portas e saúde, e diagnostica boot. Use sempre que a tarefa envolver rodar a stack, ver logs, resetar banco ou investigar por que um serviço não sobe.
---

# Operação da stack WYD (Rust)

Mapa normativo: `docs/rust/05-stack-docker.md` (crates → deployáveis, saúde,
observabilidade e pendências). Gates: skill `wyd-rust-toolchain`.

## 1. O que sobe hoje

`docker compose up` lê `docker-compose.yml` + `docker-compose.override.yml` (dev).
Produção: `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d`.

| Serviço | Porta host (dev) | Papel |
|---|---|---|
| `db` | `127.0.0.1:5432` | PostgreSQL 17; schema aplicado no boot |
| `gateway` | `127.0.0.1:7600` | HTTP/JSON do portal; `/health` faz `SELECT 1` |
| `wyd-canal-1` | `127.0.0.1:8281` (TCP/UDP) | monólito do canal 1: CPSock, status, sessão e seleção |
| `wyd-canal-2` | `127.0.0.2:8281` (TCP/UDP) | canal 2, mesmo binário/Postgres |
| `portal` | `127.0.0.1:3000` | BFF Next.js sobre o gateway; `/api/health` 200 com catálogo |
| `prometheus` / `grafana` | `127.0.0.1:9090` / `:3001` | métricas e dashboards |
| `loki` / `alloy` / `node-exporter` | interno / `:9100` | logs duráveis, coleta e métricas do host |

`wyd-launcher` é executável do cliente e **nunca** entra no compose (imagem só para
release/smoke da QA-03). O servidor é `crates/wyd-server`, um container por canal (ADR-0022).

## 2. Pré-voo (30 segundos)

```bash
# a) o Docker responde e não há stack antiga pendurada?
docker compose ps

# b) as portas estão livres? (5432/7600/3000 no loopback; 8281 nos IPs dos canais)
netstat -ano | grep LISTENING | grep -E ":(5432|7600|3000|8281)\b"

# c) o compose está válido nos dois modos? (não sobe nada)
make stack-config
```

## 3. Subir e confirmar

```bash
make stack-up                  # db + gateway + canais + portal + monitoramento
docker compose ps              # Up != pronto: ver os sinais abaixo
curl -s localhost:3000/login | head -3        # UI do portal no ar
curl -s localhost:3000/api/health             # 200 com contagem do catálogo
curl -s localhost:7600/health                 # gateway + Postgres
docker compose exec db psql -U wyd -c "\dt"   # schema aplicado?
```

Sinais de stack saudável: containers `Up`, portal em `/login`, `/api/health` 200 e banco
com as tabelas do `wyd-db`. Se o health do portal der 502, confira o gateway e os logs
antes de suspeitar do portal.

Portal em modo dev (hot reload, sem container): `make portal-dev`
(`http://127.0.0.1:3000`).

## 4. Banco, migrações e reset controlado

- Schema: `crates/wyd-db/migrations/` (inclui `delivery_queue`). Não rode SQL manual para
  “criar tabela”; migrations rodam no boot do gateway/canal (ADR-0024).
- Backup lógico: `docker compose exec -T db pg_dump -U wyd > backup-$(date +%F).sql`.
- Reset total (APAGA o banco, só com pedido explícito): `docker compose down -v`.
- Conta de teste local: `PORTAL_ADMIN_USER`/`PORTAL_ADMIN_PASSWORD` no `.env`; o gateway
  faz o seed de staff no boot. Não invente seed manual.

## 5. Quando algo não sobe

| Sintoma | Causa provável | Ação |
|---|---|---|
| `config` falha em `SESSION_SECRET`/`POSTGRES_PASSWORD` | sem `.env` (ver `.env.example`) | copiar o exemplo e preencher |
| portal `unhealthy` / `/api/health` 502 | gateway fora ou unhealthy | `docker compose logs gateway`; conferir `:7600/health` |
| canal reiniciando | banco/senha divergente ou porta 8281 ocupada | conferir `.env`, logs do canal e IPs `WYD_CANAL*_IP` |
| `db` reiniciando | senha/usuário divergente do volume `wyd-pgdata` | conferir `.env`; se trocou senha, recriar volume (com pedido) |
| porta em uso no Windows | outro Postgres/Node no host | liberar 5432/3000/7600 ou ajustar `PGPORT`/`PORTAL_PORT` |
| build do portal usa cache velho | layer stale após mexer em `api/` ou deps | `docker compose build --no-cache portal` |

## 6. O que nunca fazer

- `docker compose down -v`, `DROP DATABASE` ou reset de volume sem pedido explícito.
- Publicar 5432 fora do loopback; produção não publica banco.
- Reintroduzir nomes, portas, variáveis ou topologia da stack antiga: a fonte canônica é
  `docs/` + `crates/`.
- Subir `gateway`/canais “na mão” sem o binário e a issue correspondente.
