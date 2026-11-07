# WYD (Rust) — automações executadas pelo agente e pelo CI.
# Mapa skills → targets:
#   rust-code-quality      → fmt, clippy
#   wyd-rust-toolchain     → build, test, audit, check
#   software-architecture  → check (gates antes de qualquer merge)
#   github-actions-docs    → ci (espelho local do workflow)
# Uso: `make check` antes de considerar qualquer issue pronta.

CARGO ?= cargo

.PHONY: help fmt clippy build test check audit bench clean ci stack-config stack-up stack-down stack-logs portal-install portal-dev portal-test portal-build

help:
	@echo "Targets: fmt clippy build test check audit bench clean ci"
	@echo "         stack-config stack-up stack-down stack-logs"
	@echo "         portal-install portal-dev portal-test portal-build"

fmt: ## formatação (skill: rust-code-quality)
	$(CARGO) fmt --check

clippy: ## lints como erro (skill: rust-code-quality)
	$(CARGO) clippy --workspace -- -D warnings

build: ## compila o workspace (skill: wyd-rust-toolchain)
	$(CARGO) build --workspace

test: ## testes do workspace (skill: wyd-rust-toolchain)
	$(CARGO) test --workspace

check: fmt clippy build test ## gate completo — exigido por toda issue

audit: ## supply-chain (skill: wyd-rust-toolchain; exige cargo-audit instalado)
	$(CARGO) audit

bench: ## benchmarks / simulações pesadas (skill: wyd-rust-sim)
	$(CARGO) bench --workspace

clean:
	$(CARGO) clean

ci: check ## espelho local do .github/workflows/rust.yml

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
