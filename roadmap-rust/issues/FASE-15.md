# Issue 15 — Auditoria final e preparação para release

**Fase:** 15 · **Depende de:** FASE-03 a FASE-08, FASE-10 · **Bloqueia:** release · **Esforço:** M

## Objetivo

Executar a validação fim a fim e liberar somente com tudo verde: suite Rust,
concorrência, carga de rede, integridade do banco e build de produção.

## Contexto / evidência

Critério global (`pre-release-rust.md` §16): estável, sem warnings de clippy,
livre de data races, fórmulas críticas cobertas e alta performance determinística.
Portas de entrada: gates da FASE-05, métricas da FASE-06, dashboards da FASE-12,
fórmulas da FASE-10. Nada é "release" sem evidência anexada.

## Passos

1. `make check` completo + `cargo test --all-targets` (unitários, integração,
   doc-tests).
2. Concorrência: `cargo test` sob carga (threads), `miri`/`tarpaulin` nos
   módulos críticos (protocolo, tick, persistência).
3. Carga de rede: harness contra `wyd-net` medindo pacotes/s, P99 e rejeições
   (comparar com baseline da FASE-06).
4. Banco: migrations do zero + integridade (FASE-04) + seeds conferidos.
5. Build de produção: `cargo build --release --workspace` (LTO) + imagens
   por serviço (`docker/Dockerfile.launcher`, `docker/Dockerfile.portal`;
   mapa em `docs/rust/05-stack-docker.md`) + smoke dos binários.

## Arquivos

- Criados: relatório de release em `docs/` (números, versões, evidências).
- Alterados: ajustes mínimos exigidos pela auditoria (cada um com teste).
- Proibido tocar: `client/`, segredos; nenhum `#[ignore]`
  para esconder falha, nenhum `unwrap` novo no quente.

## Testes / verificação

```bash
make check
cargo test --all-targets 2>&1 | tail -n 5
cargo build --release --workspace 2>&1 | tail -n 3
```

Saída esperada: tudo `ok`/`Finished` sem warnings; relatório com P99, pacotes/s e
cobertura das fórmulas críticas anexado.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - gates, release, lockfile.
- .agents/skills/wyd-rust-obs - P99, carga, regressões.
- .agents/skills/wyd-rust-sim - cobertura das fórmulas.
- .agents/skills/rust-code-quality - revisão final sem concessões.
- .agents/skills/software-architecture - conformidade Hot/Cold.

## Documentação de referencia

- pre-release-rust.md (§16) - ordem e critério global.
- roadmap-rust/index.md - fases bloqueadoras.
- docs/rust/01-dependencias.md - tarpaulin/miri.
- roadmap-rust/issues/FASE-05.md - gates; FASE-06.md - métricas; FASE-10.md - fórmulas.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] `fmt`, `clippy -D warnings`, `test --all-targets`, carga e integridade verdes.
- [ ] Relatório de release com números (P99, pacotes/s, cobertura) anexado.
- [ ] Imagem de produção builda e o binário faz smoke.

## Riscos / rollback

- Risco: liberar com regressão silenciosa → mitigação: baseline da FASE-06 como
  comparador obrigatório; regressão de P99 veta o release.
- Rollback: tag anterior + imagem anterior; relatório registra o procedimento.
