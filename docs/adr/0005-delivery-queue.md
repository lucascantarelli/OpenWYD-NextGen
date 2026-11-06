# ADR 0005 — `delivery_queue` para grants da web

**Data:** 2026-10 · **Estado:** aceita

## Contexto

O portal precisa entregar item, moeda e EXP a um personagem (donate, recompensa, correção de suporte).
Escrever direto na tabela do personagem enquanto o tmServer tem o estado em memória cria duas
verdades: o jogador não recebe até relogar, e o save seguinte sobrescreve a alteração.

## Decisão

A web **enfileira intenção** em `delivery_queue`; o tmServer drena a fila no laço do mundo e aplica a
entrega no estado vivo. A escrita no save continua sendo do servidor de jogo.

## Consequências

- Entrega idempotente por linha de fila, com estado (`pending`/`delivered`/`failed`) auditável.
- Nenhuma transação da web compete com o save do personagem.
- Se o tmServer estiver fora, a fila acumula — e isso é visível, não silencioso.
