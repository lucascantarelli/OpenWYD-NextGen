# WYD (Rust) — automações executadas pelo agente e pelo CI.
# Mapa skills → targets:
#   rust-code-quality      → fmt, clippy
#   wyd-rust-toolchain     → build, test, audit, check
#   software-architecture  → check (gates antes de qualquer merge)
#   github-actions-docs    → ci (espelho local do workflow)
# Uso: `make check` antes de considerar qualquer issue pronta.

CARGO ?= cargo

.PHONY: help fmt clippy build test test-all check audit deny bench bench-build bench-gate release docker docker-smoke gateway-image gateway-smoke server-image server-smoke telemetry-smoke clean ci migrations-check migrations-apply db-migrate db-reset banco-catalogo banco-verify stack-config stack-up stack-down stack-logs local-rebuild prod-env-smoke monitoring-rules monitoring-up monitoring-down monitoring-check portal-install portal-dev portal-test portal-build tools-re tools-binarios tools-strings tools-pacotes

help:
	@echo "Targets: fmt clippy build test check audit bench clean ci"
	@echo "         stack-config stack-up stack-down stack-logs local-rebuild db-migrate db-reset prod-env-smoke"
	@echo "         gateway-image gateway-smoke monitoring-check"
	@echo "         portal-install portal-dev portal-test portal-build"

fmt: ## formatação (skill: rust-code-quality)
	$(CARGO) fmt --check

clippy: ## lints como erro, incluindo testes e benches (skill: rust-code-quality)
	$(CARGO) clippy --workspace --all-targets -- -D warnings

build: ## compila o workspace (skill: wyd-rust-toolchain)
	$(CARGO) build --workspace

test: ## testes do workspace (skill: wyd-rust-toolchain)
	$(CARGO) test --workspace

test-all: ## unitários + integração + doc-tests + alvos de bench (espelho do CI)
	$(CARGO) test --workspace --all-targets

check: fmt clippy build test migrations-check issue-status-check tools-test ## gate completo — exigido por toda issue

migrations-check: ## ordem/FK/down das migrations, sem Postgres (tools/content/check-migrations.mjs)
	node tools/content/check-migrations.mjs

issue-status-check: ## status de issue igual no índice e no painel (tools/content/check-issue-status.mjs)
	node tools/content/check-issue-status.mjs

migrations-apply: ## aplica a série num Postgres efêmero: up → down → up (exige Docker)
	node tools/content/migrate-cycle.mjs

banco-catalogo: ## regenera docs/arquitetura/banco-catalogo.md a partir das migrations
	node tools/content/gen-catalogo-banco.mjs

banco-verify: ## banco vazio → migrations → compara com o catálogo via information_schema (exige Docker)
	node tools/content/verify-schema.mjs

audit: ## supply-chain (skill: wyd-rust-toolchain; exige cargo-audit instalado)
	$(CARGO) audit

deny: ## licenças, duplicidade, fontes e wildcards (exige cargo-deny instalado)
	$(CARGO) deny check

bench: ## benchmarks / simulações pesadas (skill: wyd-rust-sim)
	$(CARGO) bench --workspace

bench-build: ## compila os benches sem rodar (gate de compilação do CI)
	$(CARGO) bench --workspace --no-run

bench-gate: ## mede e falha em regressão de baseline por cenário (tools/dev/bench-gate.mjs)
	node tools/dev/bench-gate.mjs

telemetry-smoke: ## telemetria em execução: log estruturado + GET /metrics real (FASE-06-1)
	$(CARGO) test -p wyd-core --test sistema_telemetry -- --nocapture
	$(CARGO) test -p wyd-net --features metrics --test integracao_metrics -- --nocapture

release: ## release otimizado (LTO thin, codegen-units=1, strip — ADR-0017)
	$(CARGO) build --release --workspace

docker: ## imagem mínima do launcher (distroless, non-root — FASE-05-2)
	docker build -f docker/Dockerfile.launcher -t wyd/launcher:$${TAG:-dev} \
		--build-arg WYD_VERSION=$${WYD_VERSION:-0.1.0} \
		--build-arg WYD_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) .

docker-smoke: docker ## tamanho da imagem + `check` do launcher rodando no container
	docker image inspect --format 'wyd/launcher size={{.Size}} bytes' wyd/launcher:$${TAG:-dev}
	docker run --rm -v "$$(pwd)/target/smoke:/jogo:ro" wyd/launcher:$${TAG:-dev} check /jogo /jogo/manifest.toml

clean:
	$(CARGO) clean

clean-target: ## cache de build (target/, recreado pelo próximo cargo)
	$(CARGO) clean

clean-tmp: ## artefatos conhecidos de tmp/ (logs, drivers, E2E, instalação)
	node tools/dev/clean-tmp.mjs --yes

tools-test: ## testes do app apps/tools (node:test, sem dependências)
	node --test apps/tools/test/

tools-inventario: ## regenera docs/cliente/inventario-gerado.md a partir de client/
	node apps/tools/bin/wyd-tools.mjs inventario client --out docs/cliente/inventario-gerado.md

tools-binarios: ## docs/cliente/re-binarios.md (seções/imports/exports/strings/funções)
	node apps/tools/bin/wyd-tools.mjs binarios client

tools-strings: ## docs/cliente/re-strings.md (tabela de strings do WYD.exe)
	node apps/tools/bin/wyd-tools.mjs strings client --bin WYD.exe

tools-pacotes: ## docs/cliente/re-pacotes.md (opcodes × handlers × estruturas × capturas)
	node apps/tools/bin/wyd-tools.mjs pacotes

tools-re: tools-binarios tools-strings tools-pacotes ## todos os docs de RE do cliente

tools-zip-check: ## confere client.zip contra client/manifest.txt (bytes reais)
	node apps/tools/bin/wyd-tools.mjs zip-check client.zip --manifest client/manifest.txt

client-zip: ## gera client.zip (entradas relativas, sem artefatos mutáveis)
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/client-zip.ps1 -Client client -Out client.zip

ci: check test-all bench-build audit deny ## espelho local do .github/workflows/rust.yml

stack-config: ## valida o compose (base+dev, prod e prod com .env vivo) sem subir nada
	docker compose config --quiet && docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet && docker compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.prod-env.yml config --quiet

db-migrate: ## aplica as migrations no banco da stack (mesmo runner do boot)
	docker compose run --rm gateway --migrate

db-reset: ## recria o schema do banco da stack do zero (DROP SCHEMA + série)
	docker compose run --rm gateway --migrate --reset

gateway-image: ## imagem do gateway HTTP/JSON do portal (ADR-0022, FASE-07-1)
	docker build -f docker/Dockerfile.gateway -t wyd/gateway:$${TAG:-dev} \
		--build-arg WYD_VERSION=$${WYD_VERSION:-0.1.0} \
		--build-arg WYD_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) .

server-image: ## imagem do monolito de jogo (um container POR canal, ADR-0022)
	docker build -f docker/Dockerfile.server -t wyd/server:$${TAG:-dev} \
		--build-arg WYD_VERSION=$${WYD_VERSION:-0.1.0} \
		--build-arg WYD_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo unknown) .

server-smoke: ## handshake + serv00.htm dos DOIS canais (exige stack-up)
	@curl -fsS http://$${WYD_CANAL1_IP:-127.0.0.1}/serv00.htm; echo
	@curl -fsS http://$${WYD_CANAL1_IP:-127.0.0.1}/health; echo
	@cargo test -p wyd-server --test sistema_handshake -- --nocapture

gateway-smoke: ## /health e /metrics do gateway rodando no compose (exige stack-up)
	@curl -fsS http://127.0.0.1:7600/health && echo
	@curl -fsS http://127.0.0.1:7600/metrics | head -20

stack-up: ## sobe a stack de dev inteira (db + gateway + portal + canais + monitoramento)
	docker compose up -d --build

local-rebuild: ## rebuild completo do local na 3000 + migrations + prova de saúde (fechamento de issue)
	docker compose build
	docker compose up -d --wait --wait-timeout 180
	docker compose ps
	@test "$$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/login)" = "200" || { echo "FALHOU: /login != 200"; exit 1; }
	@test "$$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/entregas)" = "307" || { echo "FALHOU: /entregas != 307"; exit 1; }
	@curl -fsS http://127.0.0.1:7600/health; echo
	@curl -fsS http://127.0.0.1:7600/health | grep -q '"schema":32' || { echo "FALHOU: schema != 32"; exit 1; }
	@if netstat -ano 2>/dev/null | grep LISTENING | grep -Eq ':(3100|3110)[[:space:]]'; then echo "FALHOU: porta de teste ocupada (3100/3110)"; exit 1; fi

prod-env-smoke: ## smoke do overlay de produção com .env montado (não derruba o dev)
	@docker compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.prod-env.yml config --quiet
	@docker compose -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.prod-env.yml run --rm --no-deps gateway --seed-staff

stack-down: ## derruba a stack de dev (mantém o volume pgdata)
	docker compose down

stack-logs: ## segue os logs da stack
	docker compose logs -f

# MSYS_NO_PATHCONV=1: sem isso o Git Bash no Windows converte /etc/prometheus em caminho
# de host e o container recebe um workdir inválido. Em Linux a variável é inerte.
monitoring-rules: ## promtool: config + regras + teste de disparo/resolução (FASE-12-1)
	MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd)/docker/monitoring/prometheus:/etc/prometheus:ro" -w /etc/prometheus --entrypoint promtool prom/prometheus:v3.5.0 check config prometheus.yml
	MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd)/docker/monitoring/prometheus:/etc/prometheus:ro" -w /etc/prometheus --entrypoint promtool prom/prometheus:v3.5.0 check rules alerts.yml
	MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd)/docker/monitoring/prometheus:/etc/prometheus:ro" -w /etc/prometheus --entrypoint promtool prom/prometheus:v3.5.0 test rules alerts.test.yml

monitoring-up: ## sobe Prometheus + Grafana + node-exporter (parte do `up` desde 2026-11-08)
	docker compose up -d prometheus grafana node-exporter

monitoring-down: ## para só a observabilidade (mantém os volumes)
	docker compose stop prometheus grafana node-exporter

monitoring-check: monitoring-rules ## smoke da stack de monitoramento (ready, targets, datasource, dashboard)
	node tools/dev/monitoring-smoke.mjs

portal-install: ## npm ci no portal (skill: wyd-web-admin-ui)
	cd wyd-portal && npm ci

portal-dev: ## portal em http://127.0.0.1:3000 (hot reload)
	cd wyd-portal && npm run dev

portal-test: ## tsc --noEmit + vitest do portal
	cd wyd-portal && npx tsc --noEmit && npx vitest run

portal-build: ## next build (valida rotas e o standalone)
	cd wyd-portal && npm run build
