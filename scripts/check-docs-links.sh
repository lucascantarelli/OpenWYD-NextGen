#!/usr/bin/env bash
# Gate de link da documentação, sem depender de mkdocs instalado (o host é Windows
# e não tem Python). Mede duas coisas em todo `docs/**/*.md`:
#
#   quebrado    → alvo relativo que não existe (o que `mkdocs build --strict` reprova)
#   fora-do-site → alvo que resolve para fora de `docs/` (roadmap/, portal/, …):
#                  existe no repositório, mas num site com `docs_dir: docs` é link morto
#
# Uso: bash scripts/check-docs-links.sh [--list]
set -euo pipefail
cd "$(dirname "$0")/.."

list=0
[ "${1:-}" = "--list" ] && list=1

broken=0
outside=0
checked=0

while IFS= read -r file; do
  dir=$(dirname "$file")
  # ](alvo) — ignora URL absoluta, âncora pura e mailto.
  while IFS= read -r target; do
    case "$target" in
      http*|mailto:*|"#"*|"")
        continue
        ;;
    esac
    target="${target%%#*}"
    [ -z "$target" ] && continue
    checked=$((checked + 1))
    resolved=$(cd "$dir" && realpath -m "$target" 2>/dev/null || printf '%s' "$dir/$target")
    case "$resolved" in
      *"/docs/"*)
        if [ ! -e "$resolved" ]; then
          broken=$((broken + 1))
          [ "$list" = 1 ] && echo "quebrado: $file -> $target"
        fi
        ;;
      *)
        outside=$((outside + 1))
        [ "$list" = 1 ] && echo "fora-do-site: $file -> $target"
        ;;
    esac
  done < <(grep -oE '\]\([^)]+\)' "$file" | sed -E 's/^\]\(//; s/\)$//')
done < <(find docs -name '*.md')

echo "check-docs-links: $checked link(s) relativos, $broken quebrado(s), $outside fora de docs/"
if [ "$broken" -gt 0 ]; then
  echo "check-docs-links: link quebrado dentro de docs/ reprova (rode com --list)" >&2
  exit 1
fi
