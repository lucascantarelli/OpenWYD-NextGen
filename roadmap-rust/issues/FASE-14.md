# Issue 14 — Pesquisa técnica contínua

**Fase:** 14 · **Depende de:** — (contínua) · **Bloqueia:** — · **Esforço:** S

## Objetivo

Manter notas de pesquisa versionadas sobre o ecossistema Rust de jogos (ECS,
I/O async, serialização), com janela de upgrade avaliada por prova de conceito —
sem adoção por preferência.

## Contexto / evidência

Ecossistema ECS, `tokio-uring` e serialização zero-copy evoluem rápido; adotar por
moda quebra determinismo. Regra: `pre-release-rust.md` FASE 14 — acompanhar e
registrar, com PoC antes de qualquer troca. Candidatos já catalogados em
`docs/rust/01-dependencias.md`.

## Passos

1. Para cada candidato (`hecs`/`bevy_ecs`, `tokio-uring`, `zerocopy`/`rkyv`,
   `wasmtime`): versão atual, changelog relevante, risco para o Hot Path.
2. PoC mínima fora das crates (branch/rascunho) com números antes/depois.
3. Registrar nota datada (manter ou descartar + motivo); se adotar, abrir ADR.
4. `make check` (workspace inalterado por pesquisa).

## Arquivos

- Criados: notas em `docs/` (pesquisa) ou atualização de `docs/rust/01-dependencias.md`.
- Alterados: `roadmap-rust/issues/FASE-14.md` (este).
- Proibido trocar: dependência do workspace sem ADR + PoC (ver FASE-13),
  `.deprecated-go/`.

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; workspace inalterado (`git status` sem diff em
`crates/`, `Cargo.toml`, `Cargo.lock`).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - lockfile, PoC isolada.
- .agents/skills/software-architecture - avaliação de impacto.
- .agents/skills/writing-plans - registro das notas.

## Documentação de referencia

- docs/rust/01-dependencias.md - catálogo de candidatos.
- pre-release-rust.md (FASE 14) - pesquisa contínua.
- docs/arquitetura/adr/ - ADRs resultantes de adoções.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Notas datadas por candidato (manter/descartar + motivo + números).
- [ ] Nenhuma troca de dependência sem ADR + PoC.
- [ ] Evidência anexada.

## Riscos / rollback

- Risco: PoC contaminando o workspace → mitigação: PoC fora de `crates/`,
  descartável, nunca importada.
- Rollback: `git checkout -- docs/rust/01-dependencias.md docs/`.
