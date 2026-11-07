# Issue 04-1 — Modelagem relacional e integridade

**Fase:** 4 · **Depende de:** 01-2 · **Bloqueia:** 04-2 · **Esforço:** M

## Objetivo

Modelar contas, personagens, inventários, baús, guildas e auditoria com FKs,
CHECKs e índices para o caminho quente, documentado em `docs/arquitetura/`.

## Contexto / evidência

`pre-release-rust.md` FASE 4 §4.1. Referência de schema legado em
`.deprecated-go/internal/migrations/` e análise em
`.deprecated-go/docs/analysis/banco-de-dados.md`.

## Passos

1. Desenhar o ER (entidades, chaves, constraints, índices quentes) e publicar em
   `docs/arquitetura/`.
2. Definir tipos (IDs, moedas, timestamps) e regras (ex.: inventário limitado,
   unicidades).
3. Revisar contra o schema legado: o que manter, fundir ou descartar (tabela).
4. `make check` (sem código ainda; modelo é documento).

## Arquivos

- Criados: doc do modelo em `docs/arquitetura/`.
- Alterados: nenhum código.
- Proibido tocar: `crates/*/src/`, `.deprecated-go/`, DSN real.

## Testes / verificação

```bash
make check && ls docs/arquitetura/ | grep -i -E "model|banco|db|er"
```

Saída esperada: gates verdes; documento do modelo presente e revisado.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-db - modelagem e integridade.
- .agents/skills/software-architecture - fronteira engine↔persistência.

## Documentação de referencia

- pre-release-rust.md (FASE 4 §4.1).
- .deprecated-go/docs/analysis/banco-de-dados.md.
- .deprecated-go/internal/migrations/ - schema legado.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

**Status:** ✅ (2026-11-07, entregue dentro da FASE-04: `docs/arquitetura/modelo-banco.md` com ER, decisões e tabela legado→novo — arquivo legado vazio, registrado).

## Critérios de aceite

- [x] ER com FKs/CHECKs/índices documentado; tabela legado→novo publicada.

## Riscos / rollback

- Risco: índice faltando no caminho quente → mitigação: listar queries quentes
  antes de fechar o modelo.
- Rollback: `git checkout -- docs/arquitetura/`.
