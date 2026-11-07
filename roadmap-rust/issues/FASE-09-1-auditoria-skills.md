# Issue 09-1 — Auditoria profunda das skills Go restantes

**Fase:** 9 · **Depende de:** 01-3 · **Bloqueia:** 09-2 · **Esforço:** S

## Objetivo

Auditar cada skill Go restante (responsabilidade, sobreposição, trechos que
citam código arquivado) e decidir o destino com motivo.

## Contexto / evidência

`pre-release-rust.md` FASE 9: consolidar para a pilha Rust. Insumo: tabela 01-3.
Alvo: `wyd-testing-bots`, `wyd-server-ops`, `wyd-web-admin-ui`,
`wyd-game-admin`, `wyd-economy`, `wyd-client-connect`, `wyd-database`.

## Passos

1. Ler cada SKILL.md restante e extrair responsabilidades e referências de código.
2. Marcar trechos obsoletos (Go arquivado citado como vigente) linha a linha.
3. Decidir por skill: suceder (nova Rust), atualizar (manter nome) ou aposentar.
4. Publicar o plano de execução da 09-2.

## Arquivos

- Criados: relatório de auditoria (nesta issue).
- Alterados: nenhum (auditoria não edita).
- Proibido tocar: `.agents/skills/*/SKILL.md` (edição é da 09-2).

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; relatório com 100% das skills destinadas.

## Skills Relacionadas do freebuff

- .agents/skills/skill-creator - padrão de qualidade.
- .agents/skills/documentation-writer - relatório.

## Documentação de referencia

- pre-release-rust.md (FASE 9).
- roadmap-rust/issues/FASE-01-3-inventario-skills.md - inventário base.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Cada skill com destino + motivo + trechos obsoletos listados.

## Riscos / rollback

- Risco: perder conhecimento embutido nas skills Go → mitigação: extrair regras
  atemporais para a sucessora antes de aposentar.
- Rollback: relatório é documentação (sem alteração em disco).
