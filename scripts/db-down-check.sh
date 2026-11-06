#!/usr/bin/env bash
# DB-02: prova que o `down` do baseline funciona, num banco descartável.
#
#   ./scripts/db-down-check.sh
#
# Fluxo: cria `downcheck`, aplica 0001_schema.up.sql, confere por consulta que o
# schema tem objetos, aplica 0001_schema.down.sql e confere que voltou ao estado
# vazio (0 tabelas, 0 índices, 0 sequências, 0 tipos compostos, nenhum
# to_regclass). A verificação é por catálogo do Postgres, não por leitura de
# arquivo. O banco é derrubado no fim.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CONTAINER="${W2PP_DB_CONTAINER:-openwyd-db-1}"
DB_USER="${W2PP_DB_USER:-postgres}"
CHECK_DB="${W2PP_DOWNCHECK_DB:-downcheck}"

q() { docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$CHECK_DB" -At -v ON_ERROR_STOP=1 "$@"; }
admin() { docker exec -i "$CONTAINER" psql -U "$DB_USER" -d postgres -At -v ON_ERROR_STOP=1 "$@"; }

echo "→ banco descartável: $CHECK_DB"
admin -c "DROP DATABASE IF EXISTS $CHECK_DB" -c "CREATE DATABASE $CHECK_DB"

echo "→ up (0001_schema.up.sql)"
docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$CHECK_DB" -q -v ON_ERROR_STOP=1 \
    -f - <internal/migrations/0001_schema.up.sql

up_tables=$(q -c "select count(*) from pg_class c join pg_namespace n on n.oid=c.relnamespace where n.nspname='public' and c.relkind='r'")
up_indexes=$(q -c "select count(*) from pg_indexes where schemaname='public'")
up_seqs=$(q -c "select count(*) from pg_class c join pg_namespace n on n.oid=c.relnamespace where n.nspname='public' and c.relkind='S'")
up_types=$(q -c "select count(*) from pg_type t join pg_namespace n on n.oid=t.typnamespace where n.nspname='public' and t.typtype='c'")
echo "   up: tables=$up_tables indexes=$up_indexes sequences=$up_seqs composite_types=$up_types"
if [[ "$up_tables" -lt 40 || "$up_indexes" -lt 40 ]]; then
    echo "erro: o up não criou o baseline esperado" >&2
    exit 1
fi

echo "→ down (0001_schema.down.sql)"
docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$CHECK_DB" -q -v ON_ERROR_STOP=1 \
    -f - <internal/migrations/0001_schema.down.sql

down_tables=$(q -c "select count(*) from pg_class c join pg_namespace n on n.oid=c.relnamespace where n.nspname='public' and c.relkind='r'")
down_indexes=$(q -c "select count(*) from pg_indexes where schemaname='public'")
down_seqs=$(q -c "select count(*) from pg_class c join pg_namespace n on n.oid=c.relnamespace where n.nspname='public' and c.relkind='S'")
down_types=$(q -c "select count(*) from pg_type t join pg_namespace n on n.oid=t.typnamespace where n.nspname='public' and t.typtype='c'")
down_rel=$(q -c "select coalesce(to_regclass('public.characters')::text, 'NULL')")
echo "   down: tables=$down_tables indexes=$down_indexes sequences=$down_seqs composite_types=$down_types characters=$down_rel"

fail=0
[[ "$down_tables" == 0 ]] || { echo "erro: sobraram $down_tables tabelas" >&2; fail=1; }
[[ "$down_indexes" == 0 ]] || { echo "erro: sobraram $down_indexes índices" >&2; fail=1; }
[[ "$down_seqs" == 0 ]] || { echo "erro: sobraram $down_seqs sequências" >&2; fail=1; }
[[ "$down_types" == 0 ]] || { echo "erro: sobraram $down_types tipos compostos" >&2; fail=1; }
[[ "$down_rel" == "NULL" ]] || { echo "erro: public.characters ainda resolve para $down_rel" >&2; fail=1; }

admin -c "DROP DATABASE IF EXISTS $CHECK_DB" >/dev/null
if [[ "$fail" != 0 ]]; then
    exit 1
fi
echo "ok: up+down do baseline verificado (schema vazio, sem objeto órfão)"
