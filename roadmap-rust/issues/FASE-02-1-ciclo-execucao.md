# Issue 02-1 — Documento do ciclo de execução unitária

**Fase:** 2 · **Depende de:** FASE-01 · **Bloqueia:** 02-2, 02-3 · **Esforço:** S

## Objetivo

Publicar `docs/rust/02-execucao.md` com o ciclo obrigatório por issue
(Análise→Design→Código→Teste→Docs).

## Contexto / evidência

`pre-release-rust.md` FASE 2 item 1: uma issue por vez, ciclo estrito. Sem
documento, cada agente improvisa o ritmo.

## Passos

1. Criar `docs/rust/02-execucao.md`: fases do ciclo, gate `make check` por
   issue, política de branch `fase/NN-descricao` → PR → `develop` (ver 00-3).
2. Definir o que é "pronto" por tipo de issue (código, docs, processo).
3. Validar lendo o documento em voz fria: um novato executa sem perguntar?
4. `make check`.

## Arquivos

- Criados: `docs/rust/02-execucao.md`.
- Alterados: nenhum outro.
- Proibido tocar: `crates/*/src/`, `.deprecated-go/`.

## Testes / verificação

```bash
test -f docs/rust/02-execucao.md && echo RITMO-OK && make check
```

Saída esperada: `RITMO-OK` + gates verdes.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - desenho do ciclo.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (FASE 2).
- roadmap-rust/issues/FASE-00-3-branch-develop.md - política de branch.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] `02-execucao.md` existe com ciclo, gate e fluxo de branch.
- [ ] Evidência anexada.

## Riscos / rollback

- Risco: processo ignorado na pressa → mitigação: gate exigido no aceite de
  toda issue seguinte.
- Rollback: `git checkout -- docs/rust/02-execucao.md`.
