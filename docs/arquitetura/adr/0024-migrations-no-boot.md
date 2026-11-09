# ADR-0024 — Migrations aplicadas no boot do gateway e dos canais

- **Estado:** aceita
- **Data:** 2026-11-09
- **Decisores:** dono do projeto + agente
- **Governa:** stack local (FASE-07-1/07-4), operação (`05-stack-docker`), CI (`seed-boot`)

## Contexto

O `db` do compose sobe da imagem oficial, sem schema. Até aqui as migrations eram aplicadas por
um passo manual no host (`node tools/dev/db-migrate.mjs` / `make db-migrate`), por um job one-shot
no compose e, no E2E/CI, por cópias próprias do parsing `-- migrate:up/down`. Três consequências:

1. ambiente limpo (`docker compose up -d`) subia o Postgres vazio: o gateway falhava no catálogo
   e o portal nunca ficava saudável na 3000;
2. o job one-shot deixava um container `exited (0)` no `docker compose ps` — o dono pediu
   explicitamente "sem deixar container parado depois de subir a stack";
3. três implementações do mesmo parsing divergiam com o tempo.

A imagem oficial do Postgres **não** resolve: `docker-entrypoint-initdb.d` e o servidor temporário
do entrypoint só rodam quando `PGDATA` está vazio (`if [ -z "$DATABASE_ALREADY_EXISTS" ]` no
`docker-entrypoint.sh` oficial, confirmado na fonte e na documentação). Num volume existente com
migration nova, um Dockerfile de banco não aplicaria nada.

## Decisão

`wyd-gateway` e `wyd-server` aplicam a série **no próprio boot**, antes de abrir o pool, via
`wyd_db::migrate` (`crates/wyd-db/src/migrate.rs`). Não existe job nem serviço de migration no
compose.

Regras do runner:

- diretório `MIGRATIONS_DIR` (default `/migrations`; as imagens distroless copiam a série);
- ledger `schema_migrations(version, checksum, applied_at)`;
- `pg_advisory_lock` de **sessão** na conexão dedicada: gateway e canal subindo juntos serializam;
- **uma transação por arquivo**: falhou, nada daquele arquivo fica;
- **checksum `sha2`** do bloco `up`: migration aplicada não pode ser reescrita;
- **baseline**: volume anterior ao ledger com `accounts` é marcado como aplicado;
- ao fim exige `item_catalog` — health sem catálogo mentiria;
- `--migrate` no binário do gateway (`--reset` opcional) para operação/CI, sem container extra.

O seed de staff roda **depois** do schema, no mesmo boot (idempotente), e `--seed-staff` migra
antes de semear.

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| Job one-shot (`service_completed_successfully`) | Deixa container `exited (0)` no `ps`; é exatamente o que o dono pediu para não ter |
| `pre_start` (Compose v5) | O passo é pulado em `up` seguinte quando a definição não muda — migration nova não rodaria; e dois serviços com `pre_start` migrariam em paralelo |
| Dockerfile do banco + `initdb.d` | Só executa com volume vazio; o resto exigiria wrapper com servidor temporário e acoplaria schema à imagem de dados |
| `sqlx::migrate!` / `refinery` | A série usa `-- migrate:up/down` (fonte canônica congelada); converter 31 arquivos e o tooling de catálogo não se justifica |

## Consequências

- **Positivas:** ambiente limpo sobe sozinho; restart é no-op; zero container extra; um único
  runner para compose, `make`, E2E e CI; app e schema na mesma versão por construção.
- **Negativas:** DDL no boot de dois processos (mitigado pelo advisory lock e por transação por
  arquivo); gateway/canal carregam a série na imagem (~1 MB); falha de migration impede o serviço
  de subir — deliberado, para não servir com schema incompleto.
- **Operação:** `make db-migrate` = `docker compose run --rm gateway --migrate`;
  `make db-reset` acrescenta `--reset`.
