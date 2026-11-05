#!/usr/bin/env bash
# Falha se o repositório perder a normalização de fim de linha.
#
# Motivo (NORMALIZE-01): o repositório não tinha `.gitattributes`, a máquina de
# desenvolvimento usa core.autocrlf=true e 718 arquivos de texto ficavam com LF no
# índice e CRLF no worktree, além de um arquivo com EOL misto. Toda escrita de
# texto (inclusive a varredura de um rename) passa a depender do humor do checkout.
set -uo pipefail

fail=0

if [ ! -f .gitattributes ]; then
	echo "FAIL: .gitattributes ausente (nenhuma regra de normalizacao)"
	fail=1
elif ! grep -q '^\* text=auto' .gitattributes; then
	echo "FAIL: .gitattributes sem a regra '* text=auto'"
	fail=1
fi

icrlf=$(git ls-files --eol | awk '$1 == "i/crlf"' | wc -l)
if [ "$icrlf" -ne 0 ]; then
	echo "FAIL: $icrlf arquivo(s) com CRLF gravado no indice"
	git ls-files --eol | awk '$1 == "i/crlf"' | head -10
	fail=1
fi

mixed=$(git ls-files --eol | awk '$2 == "w/mixed"' | wc -l)
if [ "$mixed" -ne 0 ]; then
	echo "FAIL: $mixed arquivo(s) com fim de linha misto no worktree"
	git ls-files --eol | awk '$2 == "w/mixed"' | head -10
	fail=1
fi

if [ "$fail" -eq 0 ]; then
	echo "OK: normalizacao de fim de linha em dia ($(git ls-files --eol | wc -l) arquivos conferidos)"
fi
exit "$fail"
