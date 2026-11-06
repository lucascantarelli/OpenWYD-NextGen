#!/usr/bin/env bash
# Gate de skip: todo `t.Skip`/`t.Skipf` precisa declarar o motivo no próprio texto,
# no formato `motivo: <classe> — <razão>` (classes em TEST-01: conteúdo, ambiente,
# feature, morto). Um skip sem motivo é uma promessa que ninguém cobra.
#
# Uso: bash scripts/check-skips.sh [--list]
set -euo pipefail
cd "$(dirname "$0")/.."

list_only=0
[ "${1:-}" = "--list" ] && list_only=1

hits="$(grep -rn --include='*_test.go' -E '\bt\.Skip(f)?\(' . || true)"
[ -z "$hits" ] && { echo "check-skips: nenhum t.Skip no repositório"; exit 0; }

total=$(printf '%s\n' "$hits" | wc -l | tr -d ' ')
bad=$(printf '%s\n' "$hits" | grep -v 'motivo:' || true)
bad_count=0
[ -n "$bad" ] && bad_count=$(printf '%s\n' "$bad" | wc -l | tr -d ' ')

if [ "$list_only" = 1 ]; then
  printf '%s\n' "$hits"
  exit 0
fi

if [ "$bad_count" -gt 0 ]; then
  echo "check-skips: $bad_count de $total skip(s) sem 'motivo:' — classifique (TEST-01):" >&2
  printf '%s\n' "$bad" >&2
  exit 1
fi

echo "check-skips: $total skip(s), todos com motivo declarado"
