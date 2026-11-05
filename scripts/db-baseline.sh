#!/usr/bin/env bash
# Baseline do esquema (issue 58, fase 4).
#
#   ./scripts/db-baseline.sh            # gera em internal/migrations/_baseline/ e valida num banco novo
#   ./scripts/db-baseline.sh --apply    # promove para internal/migrations/, arquiva as antigas e adota
#
# O que ele FAZ:
#   1. backup completo (scripts/db-backup.sh) — antes de qualquer coisa
#   2. pg_dump --schema-only → 0001_schema.up.sql (sem schema_migrations, sem meta-comando de psql)
#   3. junta os 12 seeds determinísticos (internal/migrations/seed_*.up.sql)
#   4. valida: cria um banco vazio, aplica tudo com ON_ERROR_STOP e compara as contagens
#   5. com --apply: move as migrations antigas para _archived/, copia o baseline para a raiz do
#      pacote embedded e registra 0001_schema no banco atual (adoção)
#
# POR QUE o filtro de meta-comando: pg_dump 17.6 emite `\restrict`/`\unrestrict`, que só o psql
# entende. O runner Go executa o arquivo como SQL puro e falharia no primeiro `\`.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CONTAINER="${W2PP_DB_CONTAINER:-openwyd-db-1}"
DB_USER="${W2PP_DB_USER:-postgres}"
DB_NAME="${W2PP_DB_NAME:-postgres}"
CHECK_DB="${W2PP_BASELINE_CHECK_DB:-baselinecheck}"
STAGE="internal/migrations/_baseline"
APPLY=0
[[ "${1:-}" == "--apply" ]] && APPLY=1

say() { printf '\n\033[1m→ %s\033[0m\n' "$*"; }

say "1/5 backup"
./scripts/db-backup.sh

say "2/5 dump do esquema"
rm -rf "$STAGE"
mkdir -p "$STAGE"
# --exclude-table=schema_migrations: o runner é o dono dessa tabela.
#
# Dois filtros no dump, ambos por incompatibilidade com o runner (que executa o
# arquivo como SQL puro numa conexão reaproveitada):
#   /^\\/                       meta-comandos do psql (\restrict, \unrestrict)
#   set_config('search_path'     SET de SESSÃO: sobrevive à transação e faz o
#                                `INSERT INTO schema_migrations` do runner falhar
#                                com 42P01 (relation does not exist)
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" \
    --schema-only --no-owner --no-privileges --exclude-table=schema_migrations \
  | sed -e '/^\\/d' -e "/set_config('search_path'/d" >"$STAGE/0001_schema.up.sql"
printf '   %s linhas, %s\n' "$(wc -l <"$STAGE/0001_schema.up.sql")" \
    "$(du -h "$STAGE/0001_schema.up.sql" | cut -f1)"

# O down do baseline é destrutivo por definição: só existe para deixar explícito
# o que "desfazer um esquema inteiro" significa (e nunca roda no boot).
cat >"$STAGE/0001_schema.down.sql" <<'SQL'
-- 0001_schema (down) — DESTRUTIVO. O baseline é o esquema inteiro: desfazê-lo
-- apaga todos os dados, inclusive conta e personagem. Só use num banco
-- descartável; em produção a recuperação é restaurar o dump, não este arquivo.
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
SQL

# Os seeds moram na RAIZ do pacote embedded (`//go:embed *.sql` não enxerga
# subpasta). O `seeds/` da fase 3 deixou de existir na promoção: um diretório
# paralelo com o mesmo SQL divergiria do que o runner aplica.
say "3/5 seeds"
cp internal/migrations/seed_*.up.sql "$STAGE/"
ls "$STAGE" | sed 's/^/   /'

say "4/5 validação em banco vazio ($CHECK_DB)"
docker exec "$CONTAINER" psql -U "$DB_USER" -q \
    -c "DROP DATABASE IF EXISTS $CHECK_DB" -c "CREATE DATABASE $CHECK_DB" >/dev/null
for f in "$STAGE"/*.up.sql; do
  if ! docker exec -i "$CONTAINER" psql -q -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$CHECK_DB" <"$f" >/dev/null; then
    echo "FALHOU ao aplicar $f" >&2
    exit 1
  fi
done

Q="select 'guild_zone',count(*) from guild_zone
   union all select 'castle_quest_state',count(*) from castle_quest_state
   union all select 'kefra_state',count(*) from kefra_state
   union all select 'world_event_config',count(*) from world_event_config
   union all select 'world_event_meta',count(*) from world_event_meta
   union all select 'npc_config_meta',count(*) from npc_config_meta
   union all select 'npc_definition',count(*) from npc_definition
   union all select 'npc_shop_item',count(*) from npc_shop_item
   union all select 'item_catalog',count(*) from item_catalog
   union all select 'skill_catalog',count(*) from skill_catalog
   union all select 'npc_template',count(*) from npc_template
   union all select 'world_item_seed',count(*) from world_item_seed
   order by 1"

docker exec "$CONTAINER" psql -U "$DB_USER" -At -d "$DB_NAME" -c "$Q" >"${TMPDIR:-/tmp}/baseline-live.txt"
docker exec "$CONTAINER" psql -U "$DB_USER" -At -d "$CHECK_DB" -c "$Q" >"${TMPDIR:-/tmp}/baseline-new.txt"
if ! diff "${TMPDIR:-/tmp}/baseline-live.txt" "${TMPDIR:-/tmp}/baseline-new.txt"; then
  echo "contagens divergem — baseline NÃO promovido" >&2
  exit 1
fi
echo "   contagens idênticas ao banco de origem"

# Tabelas criadas: o banco novo tem de ter as mesmas que o esquema declara.
# schema_migrations fica de fora: ela é criada pelo runner (store.Migrate), não
# pelo dump, então a origem sempre teria uma a mais.
TABLES="select count(*) from information_schema.tables where table_schema='public' and table_type='BASE TABLE' and table_name <> 'schema_migrations'"
ORIG_TABLES="$(docker exec "$CONTAINER" psql -U "$DB_USER" -At -d "$DB_NAME" -c "$TABLES")"
NEW_TABLES="$(docker exec "$CONTAINER" psql -U "$DB_USER" -At -d "$CHECK_DB" -c "$TABLES")"
echo "   tabelas: origem $ORIG_TABLES / novo $NEW_TABLES"
if [[ "$ORIG_TABLES" != "$NEW_TABLES" ]]; then
  echo "contagem de tabelas diverge — baseline NÃO promovido" >&2
  exit 1
fi
docker exec "$CONTAINER" psql -U "$DB_USER" -q -c "DROP DATABASE IF EXISTS $CHECK_DB" >/dev/null

if [[ "$APPLY" == "0" ]]; then
  say "dry-run: nada promovido. Revise $STAGE e rode com --apply"
  exit 0
fi

say "5/5 promoção + adoção"
mkdir -p internal/migrations/_archived
# As antigas saem do pacote embedded; _archived/ fica fora do //go:embed (*.sql) e é
# gitignored — quem versiona o histórico pré-squash é o git, não esta pasta.
# .down.sql também sai: sem o up correspondente ele só polui o embed (*.sql).
find internal/migrations -maxdepth 1 -name '0*.sql' ! -name '0001_schema.*' \
  -exec mv {} internal/migrations/_archived/ \;
cp "$STAGE"/*.up.sql internal/migrations/
# Os seeds já estão na raiz (vieram do staging no cp acima); o diretório
# `seeds/` da fase 3 não existe mais e não deve voltar.

docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -q -c \
  "INSERT INTO schema_migrations(version) VALUES ('0001_schema') ON CONFLICT DO NOTHING"
echo "   adoção registrada em schema_migrations"

say "pronto"
echo "Revise: git status internal/migrations | head"
echo "Gate: docker compose build --no-cache dbserver tmserver, depois docker compose up -d e os logs de boot."
