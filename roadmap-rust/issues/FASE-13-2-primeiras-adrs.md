# Issue 13-2 — Primeiras ADRs (monolito, ECS, binário, async)

**Fase:** 13 · **Depende de:** 13-1 · **Bloqueia:** 03-2, 04-2, 05-1, 08-2 · **Esforço:** S

## Objetivo

Promover a `0001` e registrar as ADRs de ECS, formato binário e runtime async,
travando as escolhas que as fases de execução consomem.

## Contexto / evidência

`start-rust.md` §§1–5 e `pre-release-rust.md` §0.2: monolito Hot Path, ECS,
zero-copy, `io_uring`/tokio. Candidatos em `docs/rust/01-dependencias.md`.

## Passos

1. Promover `0001-hot-path-monolitico.md` a aceita (ou registrar pendências).
2. Registrar ADR do ECS (hecs vs bevy_ecs) — consumida por 03-2.
3. Registrar ADR de serialização (`zerocopy`/`nom`/`rkyv`) — consumida por 08-2.
4. Registrar ADR de runtime async (`tokio`, `io_uring`) — consumida por 04-3/06-1.
5. Referenciar cada ADR nas issues consumidoras.

## Arquivos

- Criados: `docs/arquitetura/adr/0002-*.md`, `0003-*.md`, `0004-*.md`.
- Alterados: `0001` (status), issues consumidoras (link da ADR).
- Proibido tocar: `crates/*/src/`.

## Testes / verificação

```bash
make check && ls docs/arquitetura/adr/
```

Saída esperada: gates verdes;ADRs numeradas sem lacuna e com status explícito.

## Skills Relacionadas do freebuff

- .agents/skills/software-architecture - decisões e consequências.
- .agents/skills/writing-plans - redação.

## Documentação de referencia

- start-rust.md (§§1–5) - princípios que motivam as decisões.
- pre-release-rust.md (§0.2, FASE 13).
- docs/rust/01-dependencias.md - candidatos.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] 4 ADRs com status; issues consumidoras vinculadas.

## Riscos / rollback

- Risco: decisão travando alternativa melhor → mitigação: seção de
  consequências + estado "substituída" previsto no rito.
- Rollback: `git checkout -- docs/arquitetura/adr/`.
