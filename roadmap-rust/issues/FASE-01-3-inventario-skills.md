# Issue 01-3 — Inventário Go→Rust das skills e instruções de agentes

**Fase:** 1 · **Depende de:** 01-1 · **Bloqueia:** FASE-09 · **Esforço:** S

## Objetivo

Publicar a tabela de sucessão de todas as skills Go (manter, suceder ou
aposentar) como entrada da FASE-09.

## Contexto / evidência

`pre-release-rust.md` FASE 1 §1.3 + `start-rust.md` (analisar
`.agents/skills/*`, criar novas para as tecnologias adotadas). Já existem 9
skills novas; falta o inventário formal das restantes.

## Passos

1. Listar `.agents/skills/*/SKILL.md` e classificar: `wyd-*` Go × novas Rust ×
   boas práticas.
2. Para cada skill Go, decidir: sucessora Rust, atualização ou aposentadoria,
   com motivo de uma linha.
3. Detectar instruções conflitantes (ex.: skill Go citando fluxo arquivado como
   vigente) e listar.
4. Entregar a tabela como insumo da FASE-09 (09-1 executa a auditoria profunda).

## Arquivos

- Criados: tabela de sucessão (nesta issue).
- Alterados: nenhum (inventário não edita skills).
- Proibido tocar: `.agents/skills/*/SKILL.md` (edição é da FASE-09),
  .

## Testes / verificação

```bash
ls .agents/skills/ | wc -l
```

Saída esperada: contagem publicada; 100% das skills classificadas na tabela.

## Skills Relacionadas do freebuff

- .agents/skills/skill-creator - critérios de qualidade de skills.
- .agents/skills/writing-plans - tabela de sucessão.

## Documentação de referencia

- start-rust.md - pedido de análise das skills.
- pre-release-rust.md (FASE 1 §1.3, FASE 9).
- docs/rust/00-visao-estrutura.md - mapa de skills vigentes.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Tabela completa (skill → destino → motivo); conflitos listados.

## Riscos / rollback

- Risco: aposentar skill ainda necessária → mitigação: destino padrão é
  "suceder", nunca "aposentar" sem evidência de desuso.
- Rollback: tabela é documentação nesta issue (sem alteração em disco).
