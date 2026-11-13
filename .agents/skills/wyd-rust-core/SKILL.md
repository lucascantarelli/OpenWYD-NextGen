---
name: wyd-rust-core
description: Desenvolve a engine Hot Path do WYD em Rust (crates/wyd-core): tick determinístico, ECS/DOD, spatial grid, sem GC e sem mutex global no estado de jogo. Use ao alterar game loop, componentes ECS, particionamento espacial, autoridade de cena ou fixed-point.
---

# Engine Rust (wyd-core)

Norma: `docs/arquitetura/estrutura-e-padroes.md` §10 (árvore) e ADR-0001;
`docs/arquitetura/matematica-sim.md` §3.3 (ordem do tick); `docs/arquitetura/protocolo.md` §5
e ADR-0025 (`ESCENE_FIELD`). Não repita essas tabelas aqui.

## Decisões específicas

- Alteração de regra entra em `wyd-core`; I/O, persistência e portal ficam no Cold Path.
  Não mova regra de jogo para `wyd-server`/`wyd-net` só para “facilitar” a chamada.
- A árvore módulo a módulo é o §10. Arquivo novo fora dela exige ADR; não crie `helpers.rs`
  genérico nem estado global para evitar passar contexto.
- Tick fixo, ordem de RNG/eventos e determinismo são contrato. Mudança nesses pontos exige
  golden/determinism atualizado com motivo, não ajuste silencioso.
- `ESCENE_FIELD` é o pacote autoritativo de cena; o layout está no protocolo/ADR-0025.
  Cliente não dita posição, dano ou inventário.

## Prova

`cargo test -p wyd-core` e o teste de determinismo; depois `make check` (gate do `AGENTS.md`).
