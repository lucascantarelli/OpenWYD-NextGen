# ADR 0006 — Arquivar o legado C++ em `_legacy`

**Data:** 2026-11 · **Estado:** aceita

## Contexto

`Source/Code/` tem o servidor original, que é a **evidência** de wire, formato e regra. Ele é grande,
não compila neste ambiente e mistura código vivo de referência com ferramentas que já foram
substituídas (geradores de CSV, patch de cliente). Manter tudo com o mesmo status faz o leitor tratar
arquivo morto como fonte de verdade.

## Decisão

O que já foi substituído vai para `Source/Code/_legacy/`, com um README dizendo **o que substituiu o
quê** e onde a regra vive hoje. Nada é apagado: o legado é a única testemunha de comportamento que
ainda não foi revertido.

## Consequências

- O leitor sabe de imediato o que é referência viva e o que é peça de museu.
- Consulta a evidência continua possível (`arquivo:linha` citado nos docs).
- A regra "arquivar, nunca apagar" também vale para doc: superado vai para `roadmap/doc_legacy/`.
