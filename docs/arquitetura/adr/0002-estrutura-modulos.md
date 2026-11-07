# ADR-0002 — Estrutura de módulos, nomenclatura e padrões por crate

- **Estado:** aceita
- **Data:** 2026-11-07
- **Governa:** todas as fases com código (FASE-03 em diante) e a reorganização da FASE-08-1

## Contexto

A primeira execução da FASE-08-1 criou cinco módulos soltos na raiz de
`crates/wyd-protocol/src/` (`header.rs`, `payload.rs`, `obfuscation.rs`,
`opcodes.rs`, `keyword.rs`), com um arquivo de 1.256 linhas gerado e sem plano de
nomenclatura. Funciona como protótipo, mas viola SRP, mistura domínios
(`payload.rs` com login, combate, comércio e UI), não define onde vivem testes de
integração e não tem critério para nomes de arquivos. Escrever as demais crates
nesse formato multiplicaria o problema.

## Decisão

Adotamos a árvore normativa de `docs/arquitetura/estrutura-e-padroes.md`:
diretório por **domínio** (`wire/`, `codec/`, `payload/`), `mod.rs` apenas
declarativo, tabelas geradas isoladas em `generated/`, testes de integração em
`tests/`, nomes de arquivo em `snake_case` por domínio, e os padrões listados na
§5 (Facade, Newtype, enum+match, Typestate, Strategy, Repository, State machine,
Builder, Adapter). Nenhum arquivo novo fora dessa árvore sem ADR.

## Consequências

- **Positivas:** fronteiras claras, arquivo pequeno por responsabilidade, testes
  previsíveis, geração separada do código escrito à mão, onboarding direto.
- **Negativas:** mais arquivos pequenos e um `mod.rs` por diretório; reorganização
  imediata do protótipo de `wyd-protocol` (custo assumido na FASE-08-1).
- **Invariantes afetados:** proibido `utils.rs`/`helpers.rs`/`misc.rs`; proibido
  lógica em `mod.rs`; proibido arquivo gerado fora de `generated/`; dependência
  só na direção do DAG (ADR-0001).

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Manter módulos planos na raiz da crate | mistura domínios, cresce sem limite, dificulta revisão |
| Um arquivo por opcode (198 arquivos) | fragmentação sem ganho; domínio é a unidade de leitura |
| `build.rs` gerando tabelas | adiciona etapa de build e falha silenciosa; tabela versionada é auditável |

## Rastreabilidade

- Doc normativo: `docs/arquitetura/estrutura-e-padroes.md`.
- Issues: `roadmap-rust/issues/FASE-02-4-estrutura-e-padroes.md`,
  `FASE-08-1-tabela-protocolo.md`, `FASE-08-2-parsers.md`.
- Skills: `software-architecture`, `wyd-rust-net`, `rust-code-quality`.

## Revisão

- Revisar quando: uma crate precisar de estrutura diferente (ex.: `wyd-core` com
  ECS e arena) — a diferença vira seção da §9, não exceção informal.
