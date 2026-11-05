#!/usr/bin/env bash
# Backup do Postgres local (issue 58, fase 4): dump completo em formato custom
# (restaurável com pg_restore) + esquema em texto (diff legível do baseline).
#
#   ./scripts/db-backup.sh [diretório]
#
# O destino padrão fica FORA do repositório: um dump carrega conta, personagem e
# item — dado de jogador nunca entra no git (AGENTS.md §8).
set -euo pipefail

OUT="${1:-${TEMP:-/tmp}/openwyd-backups}"
STAMP="$(date +%Y%m%d-%H%M%S)"
CONTAINER="${W2PP_DB_CONTAINER:-openwyd-db-1}"
DB_USER="${W2PP_DB_USER:-postgres}"
DB_NAME="${W2PP_DB_NAME:-postgres}"

mkdir -p "$OUT"

echo "→ dump completo ($DB_NAME) em $OUT/openwyd-$STAMP.dump"
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" \
    --format=custom --no-owner --no-privileges >"$OUT/openwyd-$STAMP.dump"

echo "→ esquema (texto) em $OUT/openwyd-$STAMP-schema.sql"
docker exec "$CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" \
    --schema-only --no-owner --no-privileges >"$OUT/openwyd-$STAMP-schema.sql"

# Restaurar num banco novo (nunca por cima do que está rodando):
#   docker exec -i openwyd-db-1 psql -U postgres -q -c "CREATE DATABASE restore"
#   docker exec -i openwyd-db-1 pg_restore -U postgres -d restore --no-owner < openwyd-….dump
echo "ok: $(du -h "$OUT"/openwyd-"$STAMP".dump | cut -f1)"
