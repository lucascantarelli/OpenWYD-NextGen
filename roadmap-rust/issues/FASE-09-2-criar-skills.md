# Issue 09-2 — Criar/atualizar skills sucessoras Rust

**Fase:** 9 · **Depende de:** 09-1 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Publicar as skills sucessoras (padrão frontmatter + regras numeradas) e anotar
as Go com sucessão, sem instruções conflitantes.

## Contexto / evidência

`pre-release-rust.md` FASE 9: padrões ECS, erros idiomáticos e concorrência para
assistentes com máxima precisão. Plano fonte: relatório 09-1.

## Passos

1. Criar/atualizar sucessoras conforme o plano (mesmo padrão das 9 existentes).
2. Anotar cada skill Go com nota de sucessão + link (sem apagar conhecimento).
3. Garantir responsabilidade única e critérios de qualidade executáveis por skill.
4. Grep de conferência + `make check`.

## Arquivos

- Criados: `.agents/skills/<nova>/SKILL.md`.
- Alterados: skills Go (somente nota de sucessão), `AGENTS.md` §0 se o mapa mudar.
- Proibido tocar: `crates/*/src/`, `.deprecated-go/`.

## Testes / verificação

```bash
make check
grep -rln 'tmserver/.*\.go\|make run-local' .agents/skills/wyd-rust-* .agents/skills/rust-* .agents/skills/software-* .agents/skills/solid-* || echo SKILLS-LIMPAS
```

Saída esperada: gates verdes + `SKILLS-LIMPAS`.

## Skills Relacionadas do freebuff

- .agents/skills/skill-creator - criação conforme padrão.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- roadmap-rust/issues/FASE-09-1-auditoria-skills.md - plano.
- pre-release-rust.md (FASE 9).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Plano 09-1 100% executado; grep limpo; evidência anexada.

## Riscos / rollback

- Risco: agente seguir instrução Go arquivada → mitigação: grep no aceite +
  `AGENTS.md` §0 como fonte única.
- Rollback: `git checkout -- .agents/ AGENTS.md`.
