#!/usr/bin/env bash
# DB-01/DB-02: prova que o último dump restaura de verdade.
#
#   ./scripts/db-restore.sh                 # usa o dump mais recente de db-backup.sh
#   ./scripts/db-restore.sh <arquivo.dump>  # dump específico
#   ./scripts/db-restore.sh --check         # o mesmo, e falha se alguma contagem divergir
#
# Fluxo: cria um banco descartável `restorecheck`, restaura o dump com pg_restore
# (--no-owner, porque as roles do dump não existem no banco novo), compara a
# contagem de TODAS as tabelas com o banco de origem e derruba o banco no fim.
# Um pg_dump que "roda sem erro" não prova nada: o que prova é a contagem bater.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CONTAINER="${W2PP_DB_CONTAINER:-openwyd-db-1}"
DB_USER="${W2PP_DB_USER:-postgres}"
DB_NAME="${W2PP_DB_NAME:-postgres}"
CHECK_DB="${W2PP_RESTORE_DB:-restorecheck}"
BACKUP_DIR="${W2PP_BACKUP_DIR:-${TEMP:-/tmp}/openwyd-backups}"

DUMP=""
CHECK=0
for arg in "$@"; do
    case "$arg" in
        --check) CHECK=1 ;;
        *) DUMP="$arg" ;;
    esac
done

if [[ -z "$DUMP" ]]; then
    DUMP="$(ls -t "$BACKUP_DIR"/openwyd-*.dump 2>/dev/null | head -1 || true)"
fi
if [[ -z "$DUMP" || ! -f "$DUMP" ]]; then
    echo "erro: nenhum dump encontrado; rode ./scripts/db-backup.sh primeiro" >&2
    exit 1
fi

# Contagem por tabela, em ordem estável: é o que compara origem × restaurado.
counts() {
    docker exec -i "$CONTAINER" psql -U "$DB_USER" -d "$1" -At -F' ' -c "
        SELECT c.relname, (xpath('/row/n/text()', query_to_xml(
            format('SELECT count(*) AS n FROM public.%I', c.relname), false, true, '')))[1]::text
        FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public' AND c.relkind = 'r'
        ORDER BY c.relname"
}

echo "→ banco descartável: $CHECK_DB"
docker exec "$CONTAINER" psql -U "$DB_USER" -d postgres -q -v ON_ERROR_STOP=1 \
    -c "DROP DATABASE IF EXISTS $CHECK_DB" -c "CREATE DATABASE $CHECK_DB"

echo "→ restore de $DUMP"
docker exec -i "$CONTAINER" pg_restore -U "$DB_USER" -d "$CHECK_DB" --no-owner --exit-on-error <"$DUMP"

echo "→ comparando contagens ($DB_NAME × $CHECK_DB)"
counts "$DB_NAME" >/tmp/openwyd-restore-src.txt
counts "$CHECK_DB" >/tmp/openwyd-restore-dst.txt

if diff -u /tmp/openwyd-restore-src.txt /tmp/openwyd-restore-dst.txt >/tmp/openwyd-restore.diff; then
    echo "ok: $(wc -l </tmp/openwyd-restore-src.txt) tabelas com contagem idêntica"
else
    echo "erro: contagem divergente" >&2
    cat /tmp/openwyd-restore.diff >&2
    docker exec "$CONTAINER" psql -U "$DB_USER" -d postgres -q -c "DROP DATABASE IF EXISTS $CHECK_DB" || true
    [[ "$CHECK" == 1 ]] && exit 1
fi

docker exec "$CONTAINER" psql -U "$DB_USER" -d postgres -q -c "DROP DATABASE IF EXISTS $CHECK_DB"
echo "ok: banco descartável removido"
