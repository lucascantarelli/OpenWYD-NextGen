# Issue 05-1 — Pipeline CI (fmt, clippy, test, bench, audit, deny)

**Fase:** 5 · **Depende de:** FASE-01 · **Bloqueia:** 05-2 · **Esforço:** M

## Objetivo

Pipeline verde e espelhável localmente (`make ci`) com fmt, clippy, testes,
benches, `cargo audit` e `cargo deny`.

## Contexto / evidência

`pre-release-rust.md` FASE 5 §5.1. Base: `.github/workflows/rust.yml`,
`Makefile` (`check`, `ci`, `audit`, `bench`). Falta: `deny.toml`, benches reais,
falha comprovada ante vulnerabilidade.

## Passos

1. Adicionar `deny.toml` e integrar `audit`/`deny` ao workflow e ao `Makefile`.
2. Criar `benches/` por crate crítica (`wyd-core`, `wyd-net`) e validar
   `make bench`.
3. Subir `cargo test --all-targets` no workflow (unitários, integração, doc-tests).
4. Provar que o pipeline falha: teste com vulnerabilidade conhecida/advisory
   (evidência anexada) e reverter.
5. `make ci` verde local.

## Arquivos

- Criados: `deny.toml`, `crates/*/benches/*.rs`.
- Alterados: `.github/workflows/rust.yml`, `Makefile` (se necessário).
- Proibido tocar: segredos, `.github/workflows/rust.yml` (referência).

## Testes / verificação

```bash
make ci
make bench 2>&1 | tail -n 5
```

Saída esperada: `make ci` verde; benches executam e publicam números.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - gates e supply-chain.
- .agents/skills/github-actions-docs - sintaxe do workflow.
- .agents/skills/rust-code-quality - benches sem flakiness.

## Documentação de referencia

- pre-release-rust.md (FASE 5 §5.1).
- docs/rust/01-dependencias.md - audit/deny.
- .github/workflows/rust.yml - workflow vigente.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Workflow verde em PR que toque `crates/**`; falha comprovada e revertida.
- [ ] `deny` + `audit` passando (evidência anexada).

> Nota (2026-11-07): `audit`/`deny`/`criterion` exigem rede (instalação de binários/crates) — ver ADR-0006. Pipeline executável offline hoje: `make check` (fmt+clippy+build+test), já verde em toda entrega.

## Riscos / rollback

- Risco: gate flaky bloqueando PRs → mitigação: warmup e tolerância nos
  benches; nunca `#[ignore]` para esconder falha.
- Rollback: `git checkout -- .github/workflows/rust.yml Makefile deny.toml`.
