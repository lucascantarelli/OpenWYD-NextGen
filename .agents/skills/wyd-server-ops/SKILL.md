---
name: wyd-server-ops
description: Sobe, inspeciona e derruba a stack Docker do WYD em Rust (Postgres + portal hoje; tmserver/gateway quando os binários existirem), valida portas e saúde, e diagnostica boot. Use sempre que a tarefa envolver rodar a stack, ver logs, resetar banco ou investigar por que um serviço não sobe.
---

# Operação da stack WYD (Rust)

Mapa normativo: `docs/rust/05-stack-docker.md` (crates → deployáveis,
observabilidade, pendências por issue). Gates: skill `wyd-rust-toolchain`.

## 1. O que sobe hoje

`docker compose up` lê `docker-compose.yml` + `docker-compose.override.yml`
(dev). Produção: `docker compose -f docker-compose.yml -f
docker-compose.prod.yml up -d`.

| Serviço | Porta host (dev) | Papel |
|---|---|---|
| `db` | `127.0.0.1:5432` | PostgreSQL 17 (schema via `crates/wyd-db/migrations/`) |
| `portal` | `127.0.0.1:3000` | BFF Next.js. Sem backend (gateway FASE-07-1 ausente): a UI sobe e `/api/health` responde 502 — esperado |

`tmserver` (CPSock `:8281` + status `:80`) e `gateway` (`:7600`) entram no
compose quando os binários existirem (FASE-03/06 e FASE-07-1). `wyd-launcher`
é exe do cliente: nunca entra no compose (imagem só para release/smoke,
FASE-05-2).

## 2. Pré-voo (30 segundos)

```bash
# a) o Docker responde e não há stack antiga pendurada?
docker compose ps

# b) as portas estão livres? (5432/3000 no loopback)
netstat -ano | grep LISTENING | grep -E ":(5432|3000)\b"

# c) o compose está válido nos dois modos? (não sobe nada)
make stack-config
```

## 3. Subir e confirmar

```bash
make stack-up                  # db + portal em background
docker compose ps              # Up != pronto: ver os sinais abaixo
curl -s localhost:3000/login | head -3        # UI do portal no ar
curl -s localhost:3000/api/health             # {"ok":false,...502} = sem gateway (esperado hoje)
docker compose exec db psql -U wyd -c "\dt"   # schema aplicado?
```

Três sinais de stack saudável: containers `Up`, portal servindo `/login`,
banco com as tabelas do `wyd-db`. O 502 do `/api/health` hoje **não** é
bug: é o portal sem backend (ver §1).

Portal em modo dev (hot reload, sem container): `make portal-dev`
(`http://127.0.0.1:3000`).

## 4. Banco, migrações e reset controlado

- Schema: `crates/wyd-db/migrations/` (inclui `delivery_queue`). Não rode
  SQL manual para "criar tabela".
- Backup lógico: `docker compose exec -T db pg_dump -U wyd > backup-$(date +%F).sql`.
- Reset total (APAGA o banco, só com pedido explícito): `docker compose down -v`.
- Conta de teste local para login no portal: sem equivalente no stack novo
  (seed sairá da FASE-04/07) — não inventar seed manual.

## 5. Quando algo não sobe

| Sintoma | Causa provável | Ação |
|---|---|---|
| `config` falha em `SESSION_SECRET`/`POSTGRES_PASSWORD` | sem `.env` (ver `.env.example`) | copiar o exemplo e preencher |
| portal `unhealthy` / `/api/health` 502 | gateway Rust ausente | esperado até FASE-07-1; UI funciona sem ele |
| `db` reiniciando | senha/usuário divergente do volume `wyd-pgdata` | conferir `.env`; se trocou senha, recriar volume (com pedido) |
| porta em uso no Windows | outro Postgres/Node no host | liberar 5432/3000 ou ajustar `PGPORT`/`PORTAL_PORT` |
| build do portal usa cache velho | layer stale após mexer em `api/` ou deps | `docker compose build --no-cache portal` |

## 6. O que nunca fazer

- `docker compose down -v`, `DROP DATABASE` ou reset de volume sem pedido explícito.
- Publicar 5432 fora do loopback; exigir menos que isso na prod (prod não publica banco).
- Reintroduzir arquivo Go removido (2026-11): nomes, portas, variáveis e
  topologia legadas não entram no stack novo; fonte canônica é `docs/` + `crates/`.
- Tratar o 502 do `/api/health` como bug do compose: é ausência do gateway,
  com issue dona (FASE-07-1).
- Subir `tmserver`/`gateway` "na mão" sem o binário da issue correspondente.
