# REVIEW-18 — Revisão permanente: matemática do jogo e balanceamento

**Fase:** revisão contínua · **Depende de:** — · **Bloqueia:** — · **Esforço:** contínuo · **Estado:** 🔁 viva

## Objetivo

Ponto de controle das fórmulas do WYD2 (dano, velocidade, crítico, drop, EXP, refino, skills) com
origem e nível de confiança de cada número, separando **medido**, **documentado** e **hipótese**.

## Escopo

- Catálogo de atributos: origem, unidade, faixa, limite, arredondamento, onde é aplicado.
- Ordem de aplicação dos modificadores de dano (base, equipamento, skill, crítico, defesa, resistência).
- Velocidade de ataque: relação com hits, intervalo, DPS e o comportamento observado acima de ~200%.
- Crítico: chance, multiplicador, independência por hit.
- Skills e magias: custo, cooldown, alcance, escalonamento, divergência cliente × servidor.
- NPCs/mobs: atributos, IA, recompensa, drop, respawn.
- Classes e builds: comparação em condições equivalentes.
- Bug hunter matemático: overflow, divisão por zero, arredondamento, multiplicador duplicado.

## Regra de evidência

Cada fórmula registrada recebe um nível:

| Nível | Significado |
|---|---|
| **A — medido** | provado por teste/golden ou por observação no servidor com log |
| **B — documentado** | está em `Source/` e foi lido, sem teste que trave o número |
| **C — hipótese** | relato de comunidade, comportamento observado sem causa identificada |

Fórmula de nível C **não** entra em código como se fosse A. É preferível registrar a lacuna.

## O que registrar a cada rodada

| Item | Como medir |
|---|---|
| Fórmulas com teste | `go test ./tmserver/internal/... -run 'Damage\|Exp\|Drop\|Refine'` |
| Golden cases | arquivos `*_test.go` com número esperado |
| Divergência cliente × servidor | comparar `protocol` com o que o cliente desenha |
| Achados novos | severidade, impacto, reprodutibilidade, correção proposta |

## Critérios de aceite (por rodada)

- [ ] Nenhuma fórmula nova sem nível de confiança declarado
- [ ] Toda correção de bug matemático tem golden case que falha antes
- [ ] Balanceamento só é proposto com simulação e amostra declarada
- [ ] Roadmap reflete toda lacuna de conhecimento aberta (MATH-01/02)

## Histórico

| Data | Achados | Destino |
|---|---|---|
| 06/11/2026 | não existe catálogo único de atributos/fórmulas com nível de confiança; regras espalhadas em `docs/migration/game-rules.md` e `docs/sistemas/` | MATH-01 (aberta) |
| 06/11/2026 | comportamento de hits acima de ~200% de velocidade não investigado; sem simulador | MATH-02 (aberta) |
