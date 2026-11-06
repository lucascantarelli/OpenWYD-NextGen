# ADR 0002 — Goroutine única dona do estado do mundo

**Data:** 2026-10 · **Estado:** aceita

## Contexto

O legado mutava estado de jogo a partir de handlers de socket, com locks pontuais e bugs de duplicação
de item que só apareciam sob concorrência. Estado de jogo é um grafo muito interligado: personagem,
chão, spawn, guilda, evento — travar um pedaço e não os vizinhos produz corrida lógica, não só de
memória.

## Decisão

Todo o estado do mundo vive em **uma goroutine** (`world.World.Run`). Handlers rodam dentro dela e
mutam estado direto. Chamada bloqueante (dbserver, billing) sai via `World.Go` e o resultado volta por
canal.

## Consequências

- Não existe lock no estado do mundo; a serialização é a própria fila de eventos.
- Uma operação lenta **congela o mundo**: por isso I/O nunca acontece no laço.
- O teste de corrida fica simples de interpretar, e duplicação de item por concorrência deixa de ser
  possível por construção.
