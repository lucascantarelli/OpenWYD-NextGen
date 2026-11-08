# Issue 02-4 — Estrutura de módulos, nomenclatura e padrões (gate de código)

**Fase:** 2 · **Depende de:** 02-1 · **Bloqueia:** todas as issues com código (FASE-03 a FASE-15) · **Esforço:** S

## Objetivo

Publicar a árvore normativa de diretórios/arquivos por crate, as regras de
nomenclatura, os padrões adotados e a aplicação de SOLID, como pré-requisito de
qualquer arquivo novo nas `crates/`.

## Contexto / evidência

A execução anterior da FASE-08-1 criou `crates/wyd-protocol/src/{header,payload,
obfuscation,opcodes,keyword}.rs` sem plano: `payload.rs` mistura login, combate,
comércio e UI, `opcodes.rs` tem 1.256 linhas geradas na raiz, e não havia onde
declarar testes de integração. Resultado: **protótipo congelado** (ver
`docs/rust/03-progresso.md`), não arquitetura.

## Passos

1. Publicar `docs/arquitetura/estrutura-e-padroes.md` com: DAG entre crates,
   árvore canônica de crate, árvore alvo de `wyd-protocol`, tabela de
   nomenclatura, padrões (§5), SOLID (§6), onde moram os testes (§7).
2. Registrar a decisão em `docs/arquitetura/adr/0002-estrutura-modulos.md`.
3. Auditar as crates atuais contra a árvore e registrar o que está fora
   (`crates/wyd-protocol/src/*.rs` na raiz).
4. Atualizar skills (`software-architecture`, `wyd-rust-net`,
   `rust-code-quality`) e `AGENTS.md` §0 com a regra "estrutura antes de código".
5. `make check` (sem código novo nesta issue).

## Arquivos

- Criados: `docs/arquitetura/estrutura-e-padroes.md`,
  `docs/arquitetura/adr/0002-estrutura-modulos.md`.
- Alterados: `.agents/skills/{software-architecture,wyd-rust-net,rust-code-quality}/SKILL.md`,
  `AGENTS.md`, `roadmap-rust/index.md`, `docs/rust/03-progresso.md`, `README.md`.
- Proibido tocar: `crates/*/src/` (esta issue planeja, não implementa),
  `client/`.

## Testes / verificação

```bash
make check
for p in $(grep -o '](\([^)]*\.md\))' README.md | sed 's/](//; s/)$//' | sort -u); do test -e "$p" || echo "QUEBRADO $p"; done; echo LINKS-OK
```

Saída esperada: gates verdes (nenhum código alterado) e `LINKS-OK`.

## Skills Relacionadas do freebuff

- .agents/skills/software-architecture - fronteiras, padrões, SOLID.
- .agents/skills/solid-design-patterns - catálogo idiomático em Rust.
- .agents/skills/rust-code-quality - regras de código e testes.
- .agents/skills/documentation-writer - redação normativa.

## Documentação de referencia

- docs/arquitetura/estrutura-e-padroes.md - o plano em si.
- docs/arquitetura/adr/0001-hot-path-monolitico.md - decisão de monolito.
- pre-release-rust.md §1.1 - análise antes da implementação.

## Subtasks

- [x] Codificar issue (documento + ADR)
- [x] Testar issue (links + gates)
- [ ] Revisar issue (revisão fria da árvore por outro agente/humano)
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Árvore normativa publicada por crate, com nomenclatura e padrões.
- [x] ADR-0002 aceita.
- [x] Skills e `AGENTS.md` citam a regra e o documento.
- [ ] Revisão fria aprovada (subtarefa aberta).

## Riscos / rollback

- Risco: documento virar ficção (código divergir) → mitigação: revisão
  consolidada (02-2) confere a árvore real contra o plano.
- Rollback: `git checkout -- docs/arquitetura/ AGENTS.md .agents/skills/`.
