# Issue 09 — IA, skills e ferramentas auxiliares

**Fase:** 9 · **Depende de:** FASE-01 · **Bloqueia:** — (contínua) · **Esforço:** S

## Objetivo

Fechar o mapeamento Go→Rust de todas as skills, sem instruções conflitantes, com
responsabilidade única por skill e critérios de qualidade executáveis.

## Contexto / evidência

Estado atual (verificado): 6 skills Rust criadas (`wyd-rust-core`, `wyd-rust-net`,
`wyd-rust-db`, `wyd-rust-sim`, `wyd-rust-obs`, `wyd-rust-toolchain`) + 3 de boas
práticas (`rust-code-quality`, `software-architecture`, `solid-design-patterns`) +
`AGENTS.md` §0 vigente. Pendente: `wyd-testing-bots` → harness Rust, `wyd-server-ops`
→ operação Compose+Rust, `wyd-web-admin-ui` → portal contra `wyd-admin`,
`wyd-game-admin`/`wyd-economy`/`wyd-client-connect`/`wyd-database` → sucessoras ou
atualização. Pedido original: `start-rust.md` ("analisar `.agents/skills/*`").

## Passos

1. Auditar cada skill Go restante: responsabilidade, sobreposição, trechos que
   citam código arquivado — listar em tabela nesta issue.
2. Criar/atualizar sucessoras (mesmo padrão: frontmatter `name`+`description`,
   regras numeradas, "Antes de mudar", referências reais).
3. Garantir: nenhuma skill Rust referencia `tmserver/*.go`, `make run-local` ou
   gate Go como vigente.
4. `make check` (sanidade do workspace; skills são docs, sem build próprio).

## Arquivos

- Criados: `.agents/skills/<nova>/SKILL.md`.
- Alterados: `.agents/skills/<go>/SKILL.md` (somente nota de sucessão + link),
  `AGENTS.md` §0 (se o mapa mudar), `roadmap-rust/issues/FASE-09.md` (este).
- Proibido tocar: `crates/*/src/` (fase de docs), `.deprecated-go/`, `client/`.

## Testes / verificação

```bash
make check
grep -rln "tmserver/.*\.go\|make run-local" .agents/skills/wyd-rust-* .agents/skills/rust-* .agents/skills/software-* .agents/skills/solid-* || echo SKILLS-LIMPAS
```

Saída esperada: gates verdes e `SKILLS-LIMPAS` (nenhuma skill Rust cita fluxo Go
como vigente).

## Skills Relacionadas do freebuff

- .agents/skills/skill-creator - padrão de criação de skills.
- .agents/skills/documentation-writer - redação das skills.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- docs/rust/00-visao-estrutura.md - mapa de skills vigentes.
- pre-release-rust.md (FASE 9) - consolidação de IA e ferramentas.
- start-rust.md - pedido original de análise das skills.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Toda skill Go tem sucessora ou nota de sucessão (tabela nesta issue).
- [ ] Zero conflito Go↔Rust nas skills vigentes (grep acima limpo).
- [ ] Evidência anexada.

## Riscos / rollback

- Risco: agente seguir instrução Go arquivada → mitigação: grep de conferência
  no aceite + `AGENTS.md` §0 como fonte única.
- Rollback: `git checkout -- .agents/ AGENTS.md`.
