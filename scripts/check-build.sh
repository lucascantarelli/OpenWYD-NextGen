#!/usr/bin/env bash
# Gate do BUILD-01 — o rename do modulo nao pode voltar e os artefatos proto tem
# que estar em dia com os .proto versionados.
#
#   bash scripts/check-build.sh              # varredura + idempotencia do proto
#   bash scripts/check-build.sh --path-only  # so a varredura do path antigo
#   bash scripts/check-build.sh --proto-only # so o `make proto` (usado no container)
#
# Sem `protoc` no host, o modo proto roda dentro de golang:1.26-alpine — a mesma
# imagem que gerou os .pb.go versionados (protoc v6.31.1, protoc-gen-go v1.36.11,
# protoc-gen-go-grpc v1.6.2). Assim o gate local e o do CI usam o mesmo ambiente.
set -uo pipefail

PROTO_IMAGE=golang:1.26-alpine
PROTOC_GEN_GO=v1.36.11
PROTOC_GEN_GO_GRPC=v1.6.2

mode=all
case "${1:-}" in
--path-only) mode=path ;;
--proto-only) mode=proto ;;
esac

fail=0

# --- 1. path antigo do modulo -------------------------------------------------
# As strings sao montadas em pedacos de proposito: se o texto procurado existisse
# literalmente aqui, a varredura reprovaria a si mesma.
check_path() {
	local p_full="github.com/jeanluca/w2pp-""openwyd"
	local p_mod="w2pp-""openwyd"
	local p_glitch="openwyd-nex""genal"
	local p_short="nex""genal"
	local pat="$p_full|$p_mod|$p_glitch|$p_short"
	local hits rc

	hits=$(git grep -nI -E "$pat" -- . 2>&1)
	rc=$?
	if [ "$rc" -eq 0 ]; then
		echo "FAIL: residuo do rename em arquivo versionado:"
		printf '%s\n' "$hits" | head -20
		echo "  padrao: $pat"
		return 1
	fi
	if [ "$rc" -ne 1 ]; then
		echo "FAIL: git grep nao pode varrer a arvore (rc=$rc):"
		printf '%s\n' "$hits" | head -5
		return 1
	fi
	echo "OK: nenhum residuo do path antigo em $(git ls-files | wc -l) arquivos versionados"
	return 0
}

# --- 2. idempotencia do `make proto` -----------------------------------------
snapshot() {
	find api -type f -name '*.go' | sort | xargs sha256sum
}

check_proto() {
	local before after out

	if ! command -v protoc >/dev/null 2>&1; then
		echo "FAIL: protoc ausente no PATH (instale ou rode via $PROTO_IMAGE)"
		return 1
	fi
	if ! command -v make >/dev/null 2>&1; then
		echo "FAIL: make ausente no PATH"
		return 1
	fi

	before=$(snapshot)
	if ! out=$(make proto 2>&1); then
		echo "FAIL: make proto falhou:"
		printf '%s\n' "$out" | tail -10
		return 1
	fi
	after=$(snapshot)

	if [ "$before" = "$after" ]; then
		echo "OK: make proto idempotente ($(printf '%s\n' "$after" | wc -l) arquivos em api/, nenhum byte mudou)"
		return 0
	fi

	echo "FAIL: artefatos proto desatualizados — make proto alterou:"
	printf '%s\n' "$before" >/tmp/proto-before.txt
	printf '%s\n' "$after" >/tmp/proto-after.txt
	# awk em vez de diff: a imagem alpine não traz diffutils.
	awk 'NR == FNR { sum[$2] = $1; next }
	     { if (!($2 in sum)) print "  novo:     " $2
	       else if (sum[$2] != $1) print "  alterado: " $2
	       seen[$2] = 1 }
	     END { for (f in sum) if (!(f in seen)) print "  removido: " f }' \
		/tmp/proto-before.txt /tmp/proto-after.txt
	if command -v git >/dev/null 2>&1 && git rev-parse --git-dir >/dev/null 2>&1; then
		if git -c safe.directory='*' checkout -- api 2>/dev/null; then
			echo "(artefatos restaurados; rode 'make proto' e commite o resultado)"
		fi
	fi
	return 1
}

run_proto_in_container() {
	local src
	src=$(pwd)
	if pwd -W >/dev/null 2>&1; then
		src=$(pwd -W)
	fi
	case "$(uname -s)" in
	MINGW* | MSYS*) export MSYS_NO_PATHCONV=1 ;;
	esac
	docker run --rm \
		-v "$src:/src" \
		-w /src \
		-e GOTOOLCHAIN=local \
		-e PATH="/usr/local/go/bin:/go/bin:/root/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
		"$PROTO_IMAGE" \
		sh -c "apk add -q protobuf make bash git && go install google.golang.org/protobuf/cmd/protoc-gen-go@$PROTOC_GEN_GO && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$PROTOC_GEN_GO_GRPC && bash scripts/check-build.sh --proto-only"
}

if [ "$mode" = path ] || [ "$mode" = all ]; then
	check_path || fail=1
fi

if [ "$mode" = proto ] || [ "$mode" = all ]; then
	if command -v protoc >/dev/null 2>&1; then
		check_proto || fail=1
	elif [ "$mode" = proto ] || command -v docker >/dev/null 2>&1; then
		echo "protoc ausente no host: rodando o gate de proto em $PROTO_IMAGE"
		run_proto_in_container || fail=1
	else
		echo "FAIL: sem protoc no PATH e sem docker para checar os protos"
		fail=1
	fi
fi

if [ "$fail" -ne 0 ]; then
	echo "GATE: FALHOU"
	exit 1
fi
echo "GATE: OK"
