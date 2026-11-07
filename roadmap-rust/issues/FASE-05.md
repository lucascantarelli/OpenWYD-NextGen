# Issue 05 — DevOps e CI/CD (stack Rust)

**Fase:** 5 · **Depende de:** FASE-01 · **Bloqueia:** FASE-15 · **Esforço:** M

## Objetivo

Pipeline previsível para Rust (fmt → clippy → build → test → bench → audit/deny)
com release LTO e imagem mínima multi-stage, espelhável localmente via `make`.

## Contexto / evidência

Base existente: `.github/workflows/rust.yml` (fmt/clippy/build/test),
`Makefile` (`check`, `audit`, `bench`, `ci`), Dockerfiles por serviço em
`docker/` (`Dockerfile.launcher`, `Dockerfile.portal`; mapa em
`docs/rust/05-stack-docker.md`). Faltam: `cargo audit`/`deny` configurados, `cargo bench`
com alvo real e release com LTO (`pre-release-rust.md` FASE 5). O CI Go
arquivado está em `.deprecated-go/.github/workflows/` (referência, não copiar
segredos).

## Passos

1. Adicionar `deny.toml` + `audit` ao workflow e ao `Makefile` (já existe o alvo;
   validar que falha com vulnerabilidade conhecida).
2. Criar ao menos um `benches/` por crate crítica (`wyd-core`, `wyd-net`) e
   validar `make bench`.
3. Configurar perfil release com LTO no workspace e validar
   `cargo build --release --workspace`.
4. Validar `docker build -f docker/Dockerfile.launcher .` (imagem mínima,
   binário `wyd-launcher` como entrypoint) + `docker build -f
   docker/Dockerfile.portal .` (portal standalone).
5. `make ci` verde local.

## Arquivos

- Criados: `deny.toml`, `crates/wyd-core/benches/*.rs`, `crates/wyd-net/benches/*.rs`.
- Alterados: `.github/workflows/rust.yml`, `Makefile`, `Cargo.toml`
  (perfil release), `docker/Dockerfile.*` (se necessário).
- Proibido tocar: `.deprecated-go/.github/` (referência), segredos e
  `client/`.

## Testes / verificação

```bash
make ci
cargo build --release --workspace 2>&1 | tail -n 3
docker build -f docker/Dockerfile.launcher -t wyd/launcher:check .
docker build -f docker/Dockerfile.portal -t wyd/portal:check .
```

Saída esperada: `make ci` verde; release compila; imagens buildam.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - gates, lockfile, supply-chain.
- .agents/skills/github-actions-docs - sintaxe e pinning de actions.
- .agents/skills/rust-code-quality - benches sem flakiness.

## Documentação de referencia

- docs/rust/01-dependencias.md - audit/deny/tarpaulin/miri.
- pre-release-rust.md (FASE 5) - gates e proveniência.
- .github/workflows/rust.yml - workflow vigente.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Workflow verde em PR que toque `crates/**`.
- [ ] `cargo deny check` e `cargo audit` passando (evidência anexada).
- [ ] Imagem mínima builda e roda `wyd-launcher`.

## Riscos / rollback

- Risco: gate flaky bloqueando PRs → mitigação: benches com warmup e tolerância;
  nunca `#[ignore]` para esconder falha real.
- Rollback: `git checkout -- .github/workflows/rust.yml Makefile`.
