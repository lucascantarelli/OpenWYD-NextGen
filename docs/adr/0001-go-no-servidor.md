# ADR 0001 — Go como linguagem do servidor

**Data:** 2026-10 · **Estado:** aceita

## Contexto

O servidor original é C++ fechado (MSVC x86), sem fonte do cliente, e cada mudança exigia engenharia
reversa de binário. O servidor precisa ser auditável, testável e subir em container — e a maior parte
do trabalho não é performance, é **fidelidade de comportamento** (layout binário, RNG, ordem de
mensagens).

## Decisão

O servidor é escrito em Go, mantendo o wire e os formatos do legado por offset explícito. C++
permanece como **referência** de comportamento, arquivado e consultável.

## Consequências

- Ganha-se teste automatizado, `go vet`, race detector e build reprodutível — coisas que o binário
  fechado não permitia.
- Paga-se com tradução cuidadosa: alinhamento de struct não é automático e todo save/wire precisa de
  teste de paridade.
- Nunca se assume que "em Go é igual": `struct` com `pack(1)` e offsets explícitos são a regra.
