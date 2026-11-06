#!/usr/bin/env bash
# Gate de comentário (REFACTOR-03): nos diretórios já migrados, nenhum comentário
# escrito em inglês e nenhuma dependência de issue/roadmap/caminho de doc legado
# dentro do código.
#
# A lista é intencionalmente incremental: ligar repo-wide hoje deixaria o job
# vermelho para sempre, e gate permanentemente vermelho é gate desligado. Migre um
# diretório, acrescente-o aqui.
#
# Uso:
#   bash scripts/check-comments.sh                 # confere a lista versionada
#   bash scripts/check-comments.sh internal/store  # confere diretórios avulsos
set -euo pipefail
cd "$(dirname "$0")/.."

CLEAN_DIRS=(
  internal/crossover
  internal/droprate
  internal/textenc
)

if [ "$#" -gt 0 ]; then
  CLEAN_DIRS=("$@")
fi

# Palavras que denunciam inglês num comentário (não ocorrem em português).
# Sem 'real'/'raw'/'só': são palavras (ou quase) em português e gerariam falso
# positivo — "senha real" já derrubou este gate uma vez.
ENGLISH='// .*\b(the|The|is|are|that|this|with|from|each|every|must|never|only|same|so|but|when|which|into|before|after|until|because|instead|while)\b'
# Histórico mora no git e no roadmap/, não no código.
HISTORY='issue #?[0-9]+|roadmap/|docs/|Source/Code|Basedef\.h|CFileDB\.cpp|\.cpp:'

fail=0

for dir in "${CLEAN_DIRS[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "check-comments: diretório listado não existe: $dir" >&2
    fail=1
    continue
  fi

  english=$(grep -rn --include='*.go' -E "$ENGLISH" "$dir" || true)
  if [ -n "$english" ]; then
    echo "check-comments: comentário em inglês em $dir:" >&2
    echo "$english" >&2
    fail=1
  fi

  history=$(grep -rn --include='*.go' -E "$HISTORY" "$dir" || true)
  if [ -n "$history" ]; then
    echo "check-comments: referência histórica em $dir:" >&2
    echo "$history" >&2
    fail=1
  fi
done

[ "$fail" = 0 ] && echo "check-comments: ${#CLEAN_DIRS[@]} diretório(s) limpos"
exit "$fail"
