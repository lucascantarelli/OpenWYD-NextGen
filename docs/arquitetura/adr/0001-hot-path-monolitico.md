# ADR-0001 — Hot Path monolítico em Rust (processo único por nó)

- **Estado:** aceita
- **Data:** 2026-11-07
- **Governa:** FASE-03 (engine), FASE-04 (persistência), FASE-07 (portal)

## Contexto

Microsserviços no tick loop introduzem hops de rede, serialização e jitter,
intoleráveis num game loop de 20–50 Hz (`start-rust.md` §1;
`pre-release-rust.md` §0.1). A tentativa anterior (Go) usou goroutine única com
travas globais, o que serializa o mundo e impede thread-per-core.

## Decisão

Adotamos motor monolítico por nó de mundo, em um único espaço de memória, com o
Cold Path (auth, billing, portal, admin) desacoplado de forma assíncrona —
porque qualquer salto de rede no tick compra jitter que não se recupera.

## Consequências

- **Positivas:** latência previsível, sem serialização de pacote no quente;
  estado do mundo em memória contígua (combina com ECS/DOD).
- **Negativas:** escala vertical por nó (escala horizontal = mais mundos, não
  mais serviços por mundo); um crash derruba a região inteira.
- **Invariantes afetados:** proibido gRPC/HTTP/`await` de banco dentro do tick;
  proibido mutex global no estado do mundo; Cold Path conversa por canal/fila.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Microsserviços por sistema (mapa, inventário, chat) | hops de rede no tick, jitter e P99 imprevisível |
| Monolito com travas globais | serializa o mundo; mata thread-per-core e SIMD em lote |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/changelog.md`,
  `GAME-02-ecs.md`, `PROC-16-persistencia-async.md`.
- Docs: `docs/rust/00-visao-estrutura.md`, `docs/rust/02-execucao.md`.

## Revisão

- Revisar quando: um único nó não comportar o mapa (medir tick P99 com carga real).
- Sinal de envelhecimento: P99 crescendo com número de jogadores na mesma região.
