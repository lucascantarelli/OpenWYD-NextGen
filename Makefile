# WYD (Rust) — automações executadas pelo agente e pelo CI.
# Mapa skills → targets:
#   rust-code-quality      → fmt, clippy
#   wyd-rust-toolchain     → build, test, audit, check
#   software-architecture  → check (gates antes de qualquer merge)
#   github-actions-docs    → ci (espelho local do workflow)
# Uso: `make check` antes de considerar qualquer issue pronta.

CARGO ?= cargo

.PHONY: help fmt clippy build test test-all check audit deny bench bench-build bench-gate release docker docker-smoke telemetry-smoke clean ci migrations-check migrations-apply banco-catalogo banco-verify stack-config stack-up stack-down stack-logs portal-install portal-dev portal-test portal-build

help:
	@echo "Targets: fmt clippy build test check audit bench clean ci"
	@echo "         stack-config stack-up stack-down stack-logs"
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

check: fmt clippy build test migrations-check ## gate completo — exigido por toda issue

migrations-check: ## ordem/FK/down das migrations, sem Postgres (tools/content/check-migrations.mjs)
	node tools/content/check-migrations.mjs

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
	$(CARGO) test -p wyd-core --test telemetry -- --nocapture
	$(CARGO) test -p wyd-net --features metrics --test metrics -- --nocapture

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

ci: check test-all bench-build audit deny ## espelho local do .github/workflows/rust.yml

stack-config: ## valida o compose (base+dev e prod) sem subir nada
	docker compose config --quiet && docker compose -f docker-compose.yml -f docker-compose.prod.yml config --quiet

stack-up: ## sobe a stack de dev (db + portal; ver `docs/rust/05-stack-docker.md`)
	docker compose up -d --build

stack-down: ## derruba a stack de dev (mantém o volume pgdata)
	docker compose down

stack-logs: ## segue os logs da stack
	docker compose logs -f

portal-install: ## npm ci no portal (skill: wyd-web-admin-ui)
	cd wyd-portal && npm ci

portal-dev: ## portal em http://127.0.0.1:3000 (hot reload)
	cd wyd-portal && npm run dev

portal-test: ## tsc --noEmit + vitest do portal
	cd wyd-portal && npx tsc --noEmit && npx vitest run

portal-build: ## next build (valida rotas e o standalone)
	cd wyd-portal && npm run build
