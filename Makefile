.PHONY: build binaries test lint fmt vet run run-local vuln tidy proto certs certs-clean exp-data content-check content-sync item-icons item-icons-publish gate check-build portal-install portal-dev portal-test portal-build portal-image guias guias-check gate check-skips check-comments ci client-serverlist seedgen up down ps logs db-psql db-down-check test-db test-e2e

build:
	go build ./...

# Build each service binary into bin/.
binaries:
	go build -o bin/tmserver ./tmserver/cmd/tmserver
	go build -o bin/dbserver ./dbserver/cmd/dbserver
	go build -o bin/binserver ./binserver/cmd/binserver
	go build -o bin/webserver ./webserver/cmd/webserver
	go build -o bin/exptool ./tmserver/cmd/exptool

# Restamp the monster templates' Exp field with the balanced curve (issue #43).
# Os templates vivem em npc_template (Postgres, issue 58); este alvo reprocessa
# uma árvore legada antes de reimportar. Use -dry-run primeiro.
exp-data:
	go run ./tmserver/cmd/exptool -npc tmserver/content/TMsrv/run/npc -quiet

# Conteúdo do servidor (LEGACY-02). `check` confere o manifesto (config
# versionada + mapas por bootstrap); `sync` copia de uma árvore legada o que não
# é versionado e revalida. W2PP_CONTENT_SOURCE aponta para a árvore legada do Win32.
# Regenera os seeds de referência a partir do banco (issue 58). Idempotente: rodar
# duas vezes seguidas não muda um byte — é o que prova que o seed reflete o banco.
# Usa o DSN do migrator (DB-01) quando definido; seedgen precisa de DML no catálogo.
seedgen:
	@test -n "$${W2PP_DB_MIGRATE_DSN:-$${W2PP_DB_DSN}}" || { echo "defina W2PP_DB_MIGRATE_DSN ou W2PP_DB_DSN"; exit 1; }
	go run ./tmserver/cmd/seedgen -dsn "$${W2PP_DB_MIGRATE_DSN:-$${W2PP_DB_DSN}}"

content-check:
	bash scripts/sync-content.sh --check

content-sync:
	W2PP_CONTENT_SOURCE="$(W2PP_CONTENT_SOURCE)" bash scripts/sync-content.sh --sync

# Extract proprietary client icons into ignored deployment artifacts. Usage:
#   make item-icons CLIENT_DIR=/path/to/WYD-7662
item-icons:
	test -n "$(CLIENT_DIR)" || (echo "CLIENT_DIR is required" >&2; exit 1)
	go run ./webserver/cmd/itemiconpack -client "$(CLIENT_DIR)" -content tmserver/content -out "$${OUTPUT_DIR:-dist/item-icons}"

# Publish through Jean1dev/storage-manager-server. The command is resumable and
# writes every returned URL to dist/item-icons/item-icons-upload-<version>.json
# (ao lado do pack, fora de docs/: auditoria de execução não é conteúdo de site).
item-icons-publish:
	sh ./scripts/upload-item-icons-storage-manager.sh

# Gate do BUILD-01: resíduo do rename (qualquer arquivo versionado) e artefatos proto
# em dia. Sem `protoc` no host o script cai no container golang:1.26-alpine, então o
# comando é o mesmo na máquina do dev e no CI.
gate:
	bash scripts/check-build.sh

# Nome explícito do mesmo gate (BUILD-01): `make check-build`.
check-build: gate

# Higiene de teste: skip sem motivo declarado e comentário fora do padrão nos
# diretórios já migrados. São baratos e rodam antes do build no CI.
check-skips:
	bash scripts/check-skips.sh

check-comments:
	bash scripts/check-comments.sh

# Conteúdo de /guias: gerado a partir de docs/ e versionado, porque a imagem do
# portal copia só portal/ e não vê docs/. Rodar depois de editar docs/guias,
# docs/sistemas ou docs/lore.
guias:
	cd portal && npm run guias

# Reprova se o JSON versionado divergir de docs/ (é o mesmo passo do CI).
guias-check:
	cd portal && npm run guias && git diff --exit-code -- src/content/guias.json

# O que o CI roda, na mesma ordem: os gates baratos falham rápido, depois o build,
# o teste e o portal. `ci.yml` chama exatamente estes passos (via _go.yml e
# _portal.yml), então rodar `make ci` local é a mesma coisa que rodar o CI.
ci: gate check-skips check-comments guias-check vet build test portal-test

test:
	go test -race -cover ./...

vet:
	go vet ./...

# Requires golangci-lint (see README / run inside the golang:1.26 container).
lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

# Requires govulncheck (go install golang.org/x/vuln/cmd/govulncheck@latest).
vuln:
	govulncheck ./...

tidy:
	go mod tidy

# Stack local: o compose sobe db, dbserver, tmserver, tmserver2, binserver, webserver,
# portal e logpipe. `up` não reconstrói; use `make binaries`/`make portal-image` ou
# `docker compose build <serviço>` quando o código mudou.
up:
	docker compose up -d

down:
	docker compose down

ps:
	docker compose ps

# Logs de todos os serviços (Ctrl+C sai sem derrubar nada).
logs:
	docker compose logs -f --tail=100

# psql no banco de dev (superusuário do compose).
db-psql:
	docker exec -it openwyd-db-1 psql -U postgres -d postgres

# Verificador do `down` do baseline (DB-02): derruba o schema do banco vivo — só
# rode contra banco descartável. O caminho de CI é `make test-db` (TestBaselineDown).
db-down-check:
	bash scripts/db-down-check.sh

# Testes de integração num Postgres real. O CI usa o job `database`
# (.github/workflows/db.yml) com o mesmo comando e um banco efêmero.
#   W2PP_TEST_DSN=postgres://postgres:dev@127.0.0.1:5432/citest make test-db
test-db:
	@test -n "$${W2PP_TEST_DSN:-}" || { echo "defina W2PP_TEST_DSN (banco descartável)"; exit 1; }
	go test -tags=integration ./internal/store/ -count=1

# Login headless contra a stack no ar (não precisa do cliente Windows).
test-e2e:
	W2PP_E2E_ADDR=$${W2PP_E2E_ADDR:-127.0.0.1:8281} \
	W2PP_E2E_ACCOUNT=$${W2PP_E2E_ACCOUNT:-test} \
	W2PP_E2E_PASSWORD=$${W2PP_E2E_PASSWORD:-test123} \
	W2PP_E2E_VERSION=$${W2PP_E2E_VERSION:-12000} \
	go test -tags=e2e -run TestE2ESmokeLogin ./tmserver/internal/world/ -v

run:
	go run ./tmserver/cmd/tmserver

# Bring up the full stack locally, seed a test account, and print the address to
# point a real client at. Override account with W2PP_LOCAL_ACCOUNT/_PASSWORD.
run-local:
	./scripts/run-local.sh

# Portal de administração (Next.js). A imagem Docker é construída pelo compose
# (contexto da raiz, portal/Dockerfile); estes alvos são para rodar no host.
portal-install:
	cd portal && npm ci

portal-dev:
	cd portal && npm run dev

portal-test:
	cd portal && npm run typecheck && npm test

portal-build:
	cd portal && npm run build

# Rebuilda SÓ a imagem do portal e recria o container — o jogo (tmserver) não
# reinicia. É o alvo para publicar mudanças do portal no ambiente local; para
# subir/atualizar a stack inteira (e semear a conta de teste) use run-local.
portal-image:
	docker compose build portal logpipe
	docker compose up -d portal logpipe

# Generate gRPC code from api/ (requires protoc + protoc-gen-go / protoc-gen-go-grpc
# on PATH; install with `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
# and `.../grpc/cmd/protoc-gen-go-grpc@latest`).
proto:
	protoc --go_out=. --go_opt=module=openwyd-nextgen \
	       --go-grpc_out=. --go-grpc_opt=module=openwyd-nextgen \
	       api/db/v1/db.proto api/db/v1/content.proto api/bin/v1/bin.proto api/web/v1/web.proto

# Generate dev mTLS certs into ./certs (gitignored). Apply with the mTLS overlay:
#   make certs && docker compose -f docker-compose.yaml -f docker-compose.mtls.yaml up --build
certs:
	./scripts/gen-certs.sh

certs-clean:
	rm -rf certs

# Gera o serverlist.bin a partir da entrada canônica (client-tools/serverlist/servers.txt,
# que é o -in padrão da ferramenta) e instala em client/. O client/ não é versionado,
# então este alvo é local: o .bin nunca entra no git. Ver docs/migration/canais.md §8.
# Neste Windows não existe `make`: use `mingw32-make client-serverlist`.
CLIENT_SERVERLIST ?= client/serverlist.bin

client-serverlist:
	go run ./client-tools/serverlist -in client-tools/serverlist/servers.txt -out $(CLIENT_SERVERLIST)
	go run ./client-tools/serverlist -print $(CLIENT_SERVERLIST)
