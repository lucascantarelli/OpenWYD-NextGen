# ADR-NNNN — Título curto e verificável

- **Estado:** proposta | aceita | substituída por ADR-NNNN
- **Data:** AAAA-MM-DD
- **Decisores:** dono do projeto + agente (registrar quem decidiu)
- **Governa:** issues de `roadmap-rust/issues/` que dependem desta decisão

## Contexto

Qual problema real obriga a decidir agora. Cite caminho de código, doc ou
medição (`crates/...`, `docs/...`, `pre-release-rust.md §...`). Marque o que é
**hipótese** como hipótese.

## Decisão

Uma frase imperativa: "Adotamos X para Y, porque Z". Se precisar de mais de uma
frase, provavelmente são duas decisões — abra duas ADRs.

## Consequências

- **Positivas:** o que fica mais simples/rápido/seguro.
- **Negativas:** custo assumido (complexidade, dependência, lock-in).
- **Invariantes afetados:** o que passa a ser proibido a partir de agora (ex.:
  "nenhum `await` de banco dentro do tick").

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| … | … |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/…`
- Docs atualizadas nesta ADR: `docs/…`
- Evidência (benchmark, prova de conceito): `…`

## Revisão

- Revisar quando: (mudança de escala, novo hardware, nova versão de dependência).
- Sinais de que a decisão envelheceu: (métrica, bug recorrente, dificuldade de
  contribuição).
