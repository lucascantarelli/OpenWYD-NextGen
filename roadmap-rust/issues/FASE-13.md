# Issue 13 — ADRs (decisões de arquitetura)

**Fase:** 13 · **Depende de:** FASE-01 · **Bloqueia:** — (contínua) · **Esforço:** S

## Objetivo

Instituir o rito ADR como pré-requisito de mudança estrutural, com template fixo
e as primeiras decisões registradas (monolito Hot Path, ECS, formato binário,
runtime async).

## Contexto / evidência

Decisão: ADRs vivem em `docs/arquitetura/adr/` (canônico; `docs/rust/adr/` foi
movido para lá). Existe `0001-hot-path-monolitico.md` (proposto). Regra:
`pre-release-rust.md` FASE 13 — nenhuma refatoração grande sem ADR prévia.
Sem rito, decisões se perdem em chat e o código diverge do motivo.

## Passos

1. Criar `docs/arquitetura/adr/0000-template.md` (contexto → decisão →
   consequências → alternativas).
2. Promover/registrar: `0001-hot-path-monolitico` (aceita), ECS (hecs vs
   bevy_ecs), formato binário (`zerocopy`/`rkyv`), runtime (`tokio`/`io_uring`).
3. Referenciar a ADR em cada issue que ela governa (ex.: FASE-03 ← ECS).
4. `make check` + conferir links (`docs/arquitetura/adr/` listado no README).

## Arquivos

- Criados: `docs/arquitetura/adr/0000-template.md`, `0002-*.md`, `0003-*.md`…
- Alterados: `docs/arquitetura/adr/0001-hot-path-monolitico.md` (status),
  `roadmap-rust/issues/FASE-13.md` (este).
- Proibido tocar: `crates/*/src/` (ADR precede código, não o contém),
  .

## Testes / verificação

```bash
make check
ls docs/arquitetura/adr/
```

Saída esperada: gates verdes; diretório contém `0000-template.md` + ADRs
numeradas sem lacuna.

## Skills Relacionadas do freebuff

- .agents/skills/software-architecture - C4, fronteiras, consequências.
- .agents/skills/writing-plans - redação das decisões.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- docs/arquitetura/adr/0001-hot-path-monolitico.md - primeira decisão.
- pre-release-rust.md (FASE 13) - rito de ADRs.
- start-rust.md (§§1–5) - princípios que motivam as decisões.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Template fixo publicado e seguido por todas as ADRs.
- [ ] Nenhuma lacuna na numeração; status de cada ADR explícito.
- [ ] Issues estruturais citam sua ADR governante.

## Riscos / rollback

- Risco: ADR escrita depois do código (fachada) → mitigação: issue estrutural
  só fecha com ADR datada de antes da implementação.
- Rollback: `git checkout -- docs/arquitetura/adr/`.
