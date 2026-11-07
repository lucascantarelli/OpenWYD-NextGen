---
name: wyd-rust-sim
description: Opera o laboratório matemático do WYD em Rust (crates/wyd-sim): simulações determinísticas de combate, builds, velocidade/crítico, detecção de overflow e comparação de fórmulas. Use ao validar balanceamento, reproduzir anomalias de DPS/hits ou comparar versões de fórmulas antes de tocar a engine.
---

# Laboratório matemático Rust (wyd-sim)

Sucessora da parte investigativa de `wyd-testing-bots`/`wyd-economy` para Rust.
Referências: `pre-release-rust.md` FASE 10, `roadmap-rust/issues/FASE-10.md`.

## Regras

1. **Determinismo total.** Seed explícita, sem relógio/aleatoriedade implícita;
   mesma entrada, mesma saída, sempre.
2. **Servidor é autoridade.** O simulador valida regras; quem decide dano,
   velocidade, cooldown e inventário em produção é `wyd-core`, nunca o cliente
   nem o simulador.
3. **Ordem dos modificadores é contrato.** Base → atributos → equipamentos →
   skill → buffs → crítico → defesa/resistências → resultado; arredondamento e
   limites documentados por fórmula.
4. **Bug hunt classificado.** Overflow/underflow, divisão por zero, percentuais
   duplicados e divergência cliente×servidor entram com severidade, impacto,
   reprodutibilidade e correção proposta.

## Antes de mudar

Árvore arquivo-a-arquivo: `docs/arquitetura/estrutura-e-padroes.md` §14.
Diferencie fato de hipótese (código confirmado × observado × relato × hipótese).
Cenários reproduzíveis primeiro (`cargo test -p wyd-sim`); integração com o jogo
real depois, nunca como substituto.
