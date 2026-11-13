#!/usr/bin/env bash
# Prova do reaper de character_session no servidor de cena implantado.
#
# Mecânica para exercitar a varredura sem o Drop do lease: SIGKILL no container do
# canal (docker kill -s KILL) deixa a linha de character_session órfã; o canal sobe de
# novo com WYD_SERVER_SESSION_STALE_SECS=20 e o reaper libera a linha sozinho.
#
# Uso: bash scripts/prova_reaper.sh   (evidência em /tmp/prova_reaper.log)
set -u
LOG=/tmp/prova_reaper.log
: > "$LOG"
exec > >(tee -a "$LOG") 2>&1

COMPOSE="docker compose -f docker-compose.yml -f docker-compose.override.yml -f docker-compose.reaper.yml"
CANAL=server-canal-1
CONTA=qavivo01
SENHA=qa12345678
PIN=2468
SLOT=0

echo "=== $(date -Is) build start"
BUILD_INICIO=$(date +%s)
$COMPOSE up -d --build "$CANAL"
BUILD_FIM=$(date +%s)
echo "=== $(date -Is) build end (${BUILD_FIM}s) BUILD_INICIO=$BUILD_INICIO"

for _ in $(seq 1 90); do
  s=$(docker inspect -f '{{.State.Health.Status}}' wyd-${CANAL}-1 2>/dev/null)
  [ "$s" = healthy ] && break
  sleep 2
done
echo "HEALTH=$s"
echo "=== inspect"; docker inspect -f 'StartedAt={{.State.StartedAt}} PID={{.State.Pid}}' wyd-${CANAL}-1
echo "=== log reaper"; docker compose logs --tail=200 "$CANAL" | grep -E "reaper" || true
echo "=== health vazio"; curl -s http://127.0.0.1/health; echo

# Sessão real em cena.
WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=$CONTA WYD_LIVE_PASSWORD=$SENHA \
WYD_LIVE_PIN=$PIN WYD_LIVE_CHARACTER_SLOT=$SLOT WYD_LIVE_HOLD_SECS=40 \
cargo test -p wyd-server --test cena_mobs -- --nocapture >/tmp/cena_reaper.log 2>&1 &
CLIENTE=$!
sleep 8
echo "=== health em cena"; curl -s http://127.0.0.1/health; echo
echo "=== SQL em cena"; docker compose exec -T db psql -U admin -d wyd -At -c \
 "SELECT id, character_id, released_at, renewed_at FROM character_session WHERE released_at IS NULL ORDER BY id DESC LIMIT 2;"

# Derruba o SERVIDOR (não o cliente): sem Drop do lease, a linha fica órfã.
echo "=== $(date -Is) kill -9 no servidor"
docker kill -s KILL wyd-${CANAL}-1
kill $CLIENTE 2>/dev/null
echo "=== SQL após kill"; docker compose exec -T db psql -U admin -d wyd -At -c \
 "SELECT id, character_id, released_at, renewed_at FROM character_session WHERE released_at IS NULL ORDER BY id DESC LIMIT 2;"

# Sobe de novo (mesma imagem, sem rebuild) com TTL=20s.
echo "=== $(date -Is) restart"
$COMPOSE up -d --no-deps "$CANAL"
for i in $(seq 1 120); do
  h=$(curl -s --max-time 2 http://127.0.0.1/health | grep -o '"online":[0-9]*')
  echo "$(date -Is) $h"
  echo "$h" | grep -q '"online":0' && break
  sleep 1
done
echo "=== log reaper após restart"; docker compose logs --tail=200 "$CANAL" | grep -E "reaper" || true
echo "=== SQL final"; docker compose exec -T db psql -U admin -d wyd -At -c \
 "SELECT id, character_id, released_at, renewed_at FROM character_session WHERE character_id = 136 ORDER BY id DESC LIMIT 3;"
echo "=== $(date -Is) fim"
