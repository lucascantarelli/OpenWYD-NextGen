#!/usr/bin/env bash
# DB-01: provisiona as roles de banco no Postgres local (idempotente).
#
#   ./scripts/db-roles.sh
#
# Roda como superusuário (é o único momento que precisa dele): cria/atualiza
# openwyd_migrator, openwyd_app e openwyd_ro, transfere a posse dos objetos e
# aplica os grants. O compose executa o mesmo SQL num serviço one-shot (`db-roles`)
# antes de dbserver/webserver subirem.
set -euo pipefail

CONTAINER="${W2PP_DB_CONTAINER:-openwyd-db-1}"
DB_NAME="${W2PP_DB_NAME:-postgres}"

export OPENWYD_MIGRATOR_PASSWORD="${OPENWYD_MIGRATOR_PASSWORD:-dev-migrator}"
export OPENWYD_APP_PASSWORD="${OPENWYD_APP_PASSWORD:-dev-app}"
export OPENWYD_RO_PASSWORD="${OPENWYD_RO_PASSWORD:-dev-ro}"

docker exec -i \
    -e OPENWYD_MIGRATOR_PASSWORD -e OPENWYD_APP_PASSWORD -e OPENWYD_RO_PASSWORD \
    "$CONTAINER" psql -U postgres -d "$DB_NAME" -v ON_ERROR_STOP=1 -f - \
    <"$(dirname "${BASH_SOURCE[0]}")/db-roles.sql"

echo "ok: roles openwyd_migrator/openwyd_app/openwyd_ro prontas em $DB_NAME"
