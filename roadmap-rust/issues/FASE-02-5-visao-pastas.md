# Issue 02-5 — Visão de pastas e mapa do cliente

**Fase:** 2 · **Depende de:** 02-4 · **Bloqueia:** nenhuma (gate transversal) · **Esforço:** S · **Status:** ✅

## Objetivo

Fechar o gate de arquitetura exigido pela missão-mãe: inventário real do repositório,
tabela-alvo pasta→responsabilidade→padrão→SOLID→nomenclatura, regras anti-sprawl da raiz e
o mapa do `client/` como usuário do `wyd-launcher` — tudo antes de qualquer nova implementação.

## Contexto / evidência

A missão-mãe proibiu codar sem planejamento de pastas/arquivos/padrões/SOLID/nomenclatura.
`estrutura-e-padroes.md` + ADR-0002 já normatizam as crates; faltava o mapa do repo inteiro e
a amarração do cliente (build 7662 fechado, ~344 MB fora do git) como consumidor do launcher.

## Resultados (2026-11-07)

| Entrega | Evidência |
|---|---|
| Visão de pastas | `docs/arquitetura/visao-pastas.md`: inventário real (raiz, `crates/`, `tools/`, `docs/`, `fixtures/`, `client/`, `wyd-portal/`) + tabela-alvo + anti-sprawl |
| Mapa do cliente | `docs/cliente/mapa.md`: documentado×lacuna, contrato client↔launcher, roadmap 08-4…08-7, plano de desassemble consolidado |
| Amarrações | índice do dossiê, `README.md`, `AGENTS.md`, skill `wyd-client-reverso`, `engenharia-reversa.md`, index e progresso |

Lacunas registradas sem disfarce: `docs/manual-usuario/` vazia, `tools/dev/` só com `.gitkeep`.

## Arquivos

- Criados: `docs/arquitetura/visao-pastas.md`, `docs/cliente/mapa.md`, esta issue.
- Alterados: `docs/cliente/README.md`, `engenharia-reversa.md`, `README.md`, `AGENTS.md`,
  skill `wyd-client-reverso`, `roadmap-rust/index.md`, `docs/rust/03-progresso.md`.
- Proibido: qualquer `.rs`/`.mjs`/`.ts`/`.py` novo e qualquer arquivo novo na raiz.

## Testes / verificação

```bash
test -f docs/arquitetura/visao-pastas.md -a -f docs/cliente/mapa.md
# nenhum arquivo NOVO de implementação (.rs/.mjs/.ts/.py) nem novo na raiz:
git status --porcelain | grep '^??' | grep -E '\.(rs|mjs|ts|py)$' && echo FALHA || echo SEM-IMPLEMENTACAO-NOVA
git status --porcelain | grep '^??' | grep -v '/' && echo FALHA-RAIZ || echo SEM-ARQUIVO-NOVO-NA-RAIZ
cargo fmt --all -- --check && cargo clippy --workspace -- -D warnings && cargo test --workspace
```

Saída esperada: docs existem, `SEM-IMPLEMENTACAO-NOVA`, `SEM-ARQUIVO-NOVO-NA-RAIZ`,
gate verde, contadores ✅/🚧 iguais no index e no progresso.

## Skills Relacionadas do freebuff

- .agents/skills/software-architecture - responsabilidade única por pasta.
- .agents/skills/solid-design-patterns - SRP/OCP/DIP por camada.
- .agents/skills/wyd-client-reverso - mapa do cliente.

## Documentação de referencia

- docs/arquitetura/estrutura-e-padroes.md - norma das crates (ADR-0002).
- docs/arquitetura/adr/0002-estrutura-modulos.md - árvore e proibições.
- docs/cliente/ - dossiê completo.

## Subtasks

- [x] Codificar issue (documentos)
- [x] Testar issue (links, contadores, gate, ausência de implementação nova)
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Inventário real de `client/`/`crates/`+`tools/`+`docs/`+`fixtures/`+raiz publicado.
- [x] Tabela pasta→responsabilidade→padrão→SOLID→nomenclatura publicada.
- [x] `mapa.md` com documentado×lacuna, relação client↔launcher e plano de desassemble.
- [x] Amarrações atualizadas com links válidos e contadores coerentes.
- [x] Zero arquivo novo de implementação e zero arquivo novo na raiz.

## Riscos / rollback

- Risco: deriva entre `mapa.md` e `engenharia-reversa.md` → mitigação: mapa aponta, não duplica evidência.
- Rollback: `git checkout -- docs/arquitetura/visao-pastas.md docs/cliente/mapa.md` + reverter amarrações.
