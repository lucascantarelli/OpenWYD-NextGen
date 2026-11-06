#!/usr/bin/env bash
# Bootstrap e verificação do conteúdo do servidor (LEGACY-02).
#
# A árvore `tmserver/content/` é a única fonte de conteúdo do servidor Go:
# taxas (`Common/Settings`), `serv00.htm`, templates `BaseMob`/`BaseSummon` e os
# dois mapas. A config pequena é versionada junto do código; os mapas (17 MiB)
# ficam fora do git e o resíduo Win32 vive em `tmserver/content/_legacy/`
# (ignorado, nunca lido pelo servidor).
#
# Uso:
#   scripts/sync-content.sh --check     # verifica o manifesto (default)
#   scripts/sync-content.sh --sync      # copia o que falta e verifica
#
# O `--sync` precisa de `W2PP_CONTENT_SOURCE` apontando para uma árvore legada
# (por exemplo a distribuição antiga do servidor Win32). Ele nunca sobrescreve
# arquivo presente: só preenche o que falta, e o manifesto é a autoridade.
set -euo pipefail
cd "$(dirname "$0")/.."

ROOT=tmserver/content
MANIFEST="$ROOT/content.sha256"
MODE=${1:---check}
SOURCE=${W2PP_CONTENT_SOURCE:-}

if [ ! -f "$MANIFEST" ]; then
	echo "erro: manifesto ausente: $MANIFEST" >&2
	exit 1
fi

hash_of() { sha256sum "$1" | cut -d' ' -f1; }

missing=()
changed=()

if [ "$MODE" = "--sync" ]; then
	if [ -z "$SOURCE" ] || [ ! -d "$SOURCE" ]; then
		echo "erro: --sync exige W2PP_CONTENT_SOURCE apontando para uma árvore legada" >&2
		exit 1
	fi
	echo "==> preenchendo arquivos ausentes a partir de $SOURCE"
fi

while read -r sum path; do
	[ -n "${sum:-}" ] || continue
	dest="$ROOT/$path"
	if [ ! -f "$dest" ]; then
		if [ "$MODE" = "--sync" ] && [ -f "$SOURCE/$path" ]; then
			mkdir -p "$(dirname "$dest")"
			cp "$SOURCE/$path" "$dest"
			echo "    copiado: $path"
		else
			missing+=("$path")
			continue
		fi
	fi
	if [ "$(hash_of "$dest")" != "$sum" ]; then
		changed+=("$path")
	fi
done <"$MANIFEST"

status=0
if [ ${#missing[@]} -gt 0 ]; then
	echo "faltando (${#missing[@]}):" >&2
	printf '  - %s\n' "${missing[@]}" >&2
	echo "  rode: W2PP_CONTENT_SOURCE=/caminho/para/Release scripts/sync-content.sh --sync" >&2
	status=1
fi
if [ ${#changed[@]} -gt 0 ]; then
	echo "conteúdo divergente do manifesto (${#changed[@]}):" >&2
	printf '  - %s\n' "${changed[@]}" >&2
	echo "  se a mudança é intencional, regenere: (cd $ROOT && sha256sum ...) > content.sha256" >&2
	status=1
fi

if [ "$status" -eq 0 ]; then
	echo "OK: $(wc -l <"$MANIFEST") arquivo(s) de conteúdo conferidos contra o manifesto"
fi
exit "$status"
