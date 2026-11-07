# Issue 14-1 — Notas de pesquisa (ECS, io_uring, serialização) + PoC

**Fase:** 14 · **Depende de:** — (contínua) · **Bloqueia:** — · **Esforço:** S

## Objetivo

Manter notas datadas por candidato (manter/descartar + motivo + números de PoC),
sem adoção por preferência.

## Contexto / evidência

`pre-release-rust.md` FASE 14: acompanhar ECS, I/O no Linux e serialização de
alta performance. Candidatos: `docs/rust/01-dependencias.md`.

## Passos

1. Para cada candidato (`hecs`/`bevy_ecs`, `tokio-uring`, `zerocopy`/`rkyv`,
   `wasmtime`): versão, changelog relevante, risco ao Hot Path.
2. PoC mínima fora das crates com números antes/depois.
3. Registrar nota (manter/descartar + motivo); se adotar, abrir ADR (13-1).
4. `make check` (workspace inalterado por pesquisa).

## Arquivos

- Criados: notas em `docs/` ou atualização de `docs/rust/01-dependencias.md`.
- Alterados: esta issue.
- Proibido trocar: dependência do workspace sem ADR + PoC.

## Testes / verificação

```bash
make check && git status --short -- crates/ Cargo.toml Cargo.lock
```

Saída esperada: gates verdes; nenhum diff em `crates/`, `Cargo.toml`, `Cargo.lock`.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - lockfile, PoC isolada.
- .agents/skills/software-architecture - avaliação de impacto.
- .agents/skills/writing-plans - notas.

## Documentação de referencia

- pre-release-rust.md (FASE 14).
- docs/rust/01-dependencias.md - catálogo.
- docs/arquitetura/adr/ - ADRs de adoções.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Notas datadas por candidato com números; nenhuma troca sem ADR + PoC.

## Riscos / rollback

- Risco: PoC contaminando o workspace → mitigação: PoC fora de `crates/`,
  descartável.
- Rollback: `git checkout -- docs/rust/01-dependencias.md docs/`.
