#!/usr/bin/env bash
# Preflight do ambiente local do WYD2 (issue 02).
#
# Roda ANTES de `make run-local` e falha de forma legível quando algo essencial
# falta — cada checagem já custou tempo de diagnóstico pelo menos uma vez:
#
#   1. checkout: é a raiz do repo (go.mod + docker-compose.yaml) e não um sparse
#      checkout sem pasta irmã;
#   2. conteúdo: `Release/` completo (sem ItemList.csv o servidor sobe "vazio");
#   3. Docker: daemon respondendo + compose v2/v1 (o build usa um dos dois);
#   4. portas: 80/8281 (cliente) e 7600/3000 (web-api/portal) — ocupadas pela
#      própria stack contam como OK, ocupadas por OUTRO processo são suspeita;
#   5. Go: no host (>= 1.26) ou, sem ele, o caminho oficial em container.
#
# Uso:
#   ./scripts/preflight.sh            # tudo, sai 1 se houver [FAIL]
#   ./scripts/preflight.sh --quiet    # só as linhas [FAIL]/[WARN]
#
# Nada aqui escreve no repositório: só lê arquivos, roda `docker version`/`ps`
# e consulta portas com ss/lsof/netstat.
set -uo pipefail

cd "$(dirname "$0")/.."

QUIET=0
[ "${1:-}" = "--quiet" ] && QUIET=1

FAILS=0
WARNS=0

section() { [ "$QUIET" = 1 ] || printf '\n== %s ==\n' "$1"; }
ok() { [ "$QUIET" = 1 ] || printf '  [ OK ] %s\n' "$1"; }
warn() { printf '  [WARN] %s\n' "$1"; WARNS=$((WARNS + 1)); }
fail() { printf '  [FAIL] %s\n' "$1"; FAILS=$((FAILS + 1)); }

need_file() {
	if [ -f "$1" ]; then
		ok "$1"
	else
		fail "$1 ausente"
	fi
}

need_dir() {
	if [ -d "$1" ]; then
		ok "$1/ ($(find "$1" -maxdepth 1 -type f | wc -l | tr -d ' ') arquivos)"
	else
		fail "$1/ ausente"
	fi
}

# Porta em LISTEN? 0 = ocupada, 1 = livre, 2 = não deu para checar.
port_in_use() {
	local port=$1
	if command -v ss >/dev/null 2>&1; then
		ss -ltn 2>/dev/null | awk '{print $4}' | grep -qE "[:.]${port}$" && return 0
		return 1
	fi
	if command -v lsof >/dev/null 2>&1; then
		lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1 && return 0
		return 1
	fi
	if command -v netstat >/dev/null 2>&1; then
		netstat -an 2>/dev/null | grep -E "[.:]${port}[[:space:]]" | grep -qi 'LISTEN' && return 0
		return 1
	fi
	return 2
}

# Compose disponível? Preenche COMPOSE=(...) e devolve 0/1.
COMPOSE=()
detect_compose() {
	if docker compose version >/dev/null 2>&1; then
		COMPOSE=(docker compose)
		return 0
	fi
	if command -v docker-compose >/dev/null 2>&1; then
		COMPOSE=(docker-compose)
		return 0
	fi
	return 1
}

# Serviço da stack está rodando? (só faz sentido com daemon + compose)
compose_running() {
	local service=$1
	[ ${#COMPOSE[@]} -gt 0 ] || return 1
	"${COMPOSE[@]}" ps --status running --services 2>/dev/null | grep -qx "$service"
}

# Porta publicada por um serviço da stack (lendo o compose, sem depender do daemon).
port_published_by() {
	local service=$1 port=$2
	[ ${#COMPOSE[@]} -gt 0 ] || return 1
	"${COMPOSE[@]}" ps --format '{{.Service}} {{.Ports}}' 2>/dev/null |
		awk -v s="$service" -v p="$port" '$1 == s && $0 ~ ("[:]" p "->") { found = 1 } END { exit found ? 0 : 1 }'
}

version_ge() { # version_ge 1.26.0 1.26
	local have=${1#go} want=$2
	[ "$(printf '%s\n%s\n' "$want" "$have" | sort -V | head -n1)" = "$want" ]
}

section "checkout"
if [ -f go.mod ] && [ -f docker-compose.yaml ]; then
	ok "raiz do repositório ($(basename "$PWD"))"
else
	fail "não parece a raiz do repo (go.mod e docker-compose.yaml devem existir aqui)"
fi
if [ -f .git/info/sparse-checkout ]; then
	warn "sparse checkout ativo: confira se Release/ e api/ estão materializados"
fi
need_file Makefile

section "conteúdo (Release/)"
need_file Release/Common/ItemList.csv
need_file Release/Common/SkillData.csv
need_file Release/TMsrv/run/NPCGener.txt
need_dir Release/TMsrv/run/npc
need_file Release/DBsrv/run/BaseMob/TK
need_file Release/DBsrv/run/BaseMob/FM
need_file Release/DBsrv/run/BaseMob/BM
need_file Release/DBsrv/run/BaseMob/HT

# Mapas são opcionais no boot (warning), mas sem eles os mobs andam "cego".
if [ -f Release/TMsrv/run/HeightMap.dat ] && [ -f Release/TMsrv/run/AttributeMap.dat ]; then
	ok "mapas de altura e atributo"
else
	warn "HeightMap.dat/AttributeMap.dat ausentes: mobs sem pathfinding e EXP de treino desligada"
fi

section "Docker"
if ! command -v docker >/dev/null 2>&1; then
	fail "docker não encontrado no PATH"
else
	ok "$(docker --version 2>/dev/null)"
	if docker info >/dev/null 2>&1; then
		ok "daemon respondendo"
		if detect_compose; then
			ok "compose: ${COMPOSE[*]}"
			local_mem=$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)
			if [ "${local_mem:-0}" -gt 0 ] && [ "$local_mem" -lt 6442450944 ]; then
				warn "daemon com menos de 6 GB de memória: build de 4 serviços pode cair com 'rpc error ... EOF'"
			fi
		else
			fail "nem 'docker compose' nem 'docker-compose' disponível"
		fi
	else
		fail "daemon do Docker não responde (Docker Desktop parado?)"
	fi
fi

section "portas"
for pair in "80:tmserver" "8281:tmserver" "7600:webserver" "3000:portal"; do
	port=${pair%%:*}
	service=${pair##*:}
	port_in_use "$port"
	case $? in
	0)
		if compose_running "$service" && port_published_by "$service" "$port"; then
			ok "porta $port ocupada pela stack ($service) — esperado"
		else
			warn "porta $port ocupada por outro processo (a stack $service não está no ar?)"
		fi
		;;
	1)
		ok "porta $port livre"
		;;
	*)
		warn "não deu para checar a porta $port (sem ss/lsof/netstat)"
		;;
	esac
done

section "Go"
if command -v go >/dev/null 2>&1; then
	have=$(go env GOVERSION 2>/dev/null | sed 's/^go//')
	if version_ge "$have" "1.26"; then
		ok "go $have no host"
	else
		warn "go $have no host, mas o projeto usa 1.26 — prefira o container golang:1.26-alpine"
	fi
else
	warn "sem Go no host: use o caminho oficial em container"
	printf '         MSYS_NO_PATHCONV=1 docker run --rm -v "%s:/src" -w /src golang:1.26-alpine go build ./...\n' "$PWD"
fi

printf '\n== resumo ==\n'
printf '  %d falha(s), %d aviso(s)\n' "$FAILS" "$WARNS"
if [ "$FAILS" -gt 0 ]; then
	printf '  preflight FALHOU — corrija os [FAIL] acima antes de subir a stack\n'
	exit 1
fi
printf '  preflight OK — próximo passo: make run-local\n'
