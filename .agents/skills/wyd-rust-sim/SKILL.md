---
name: wyd-rust-sim
description: Opera o laboratório matemático do WYD em Rust (crates/wyd-sim): simulações determinísticas de combate, builds, velocidade/crítico, detecção de overflow e comparação de fórmulas. Use ao validar balanceamento, reproduzir anomalias de DPS/hits ou comparar versões de fórmulas antes de tocar a engine.
---

# Laboratório matemático Rust (wyd-sim)

Norma: `docs/arquitetura/matematica-sim.md` (sistemas, unidades, ordem do golpe/tick,
arredondamento, hipóteses e provas) e `docs/arquitetura/estrutura-e-padroes.md` §14.
A ordem dos modificadores e a tabela de fórmulas vivem lá; não as duplique.

## Decisões específicas

- Fórmula pura, com seed e relógio explícitos; mesma entrada → mesma saída. Simulador não
  é autoridade de produção: `wyd-core` aplica a regra validada.
- Reproduza primeiro no cenário/golden (`cargo test -p wyd-sim`); só depois compare com o
  jogo real. Não use observação não capturada para “corrigir” um golden.
- Classifique cada achado como código confirmado, observado, relato ou hipótese; hipótese
  não vira constante nem regra de balanceamento.
- Overflow/underflow, divisão por zero e divergência cliente×servidor entram na issue com
  severidade, impacto e reprodutibilidade.

## Prova

`cargo test -p wyd-sim`; `make check` quando a mudança tocar código compartilhado.
