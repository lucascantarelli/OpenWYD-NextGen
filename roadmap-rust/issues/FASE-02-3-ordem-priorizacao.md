# Issue 02-3 — Ordem de priorização por dependências

**Fase:** 2 · **Depende de:** 02-1 · **Bloqueia:** FASE-03 a FASE-15 · **Esforço:** S

## Objetivo

Registrar a ordem vinculante de execução (contratos→engine→persistência→APIs)
com o grafo de dependências das fases.

## Contexto / evidência

`pre-release-rust.md` FASE 2 item 3 + §16: Contratos de Dados/Rede → Engine Core
→ Persistência → APIs/Portal. `roadmap-rust/index.md` já reflete; falta o grafo
explícito fase→fase (o que bloqueia o quê).

## Passos

1. Extrair dependências declaradas de cada `FASE-NN.md` (Depende/Bloqueia).
2. Publicar o grafo ordenado (lista topológica) em `roadmap-rust/index.md`.
3. Marcar o caminho crítico (08→03→04→07 e 03→10→15).
4. `make check`.

## Arquivos

- Criados: nenhum (atualização de índice).
- Alterados: `roadmap-rust/index.md` (grafo + caminho crítico).
- Proibido tocar: `crates/*/src/`.

## Testes / verificação

```bash
make check && grep -c "Bloqueia" roadmap-rust/issues/FASE-*.md
```

Saída esperada: gates verdes; toda fase com dependências declaradas.

## Skills Relacionadas do freebuff

- .agents/skills/writing-plans - grafo e caminho crítico.
- .agents/skills/software-architecture - ordem Hot→Cold.

## Documentação de referencia

- pre-release-rust.md (FASE 2, §16).
- roadmap-rust/index.md - ordem a detalhar.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Grafo publicado; caminho crítico marcado; sem fase órfã.

## Riscos / rollback

- Risco: executar fora de ordem e retrabalhar → mitigação: aceite das fases
  exige dependências fechadas.
- Rollback: `git checkout -- roadmap-rust/index.md`.
