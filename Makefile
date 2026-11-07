# OpenWYD-NextGen (Rust) — automações executadas pelo agente e pelo CI.
# Mapa skills → targets:
#   rust-code-quality      → fmt, clippy
#   wyd-rust-toolchain     → build, test, audit, check
#   software-architecture  → check (gates antes de qualquer merge)
#   github-actions-docs    → ci (espelho local do workflow)
# Uso: `make check` antes de considerar qualquer issue pronta.

CARGO ?= cargo

.PHONY: help fmt clippy build test check audit bench clean ci

help:
	@echo "Targets: fmt clippy build test check audit bench clean ci"

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
