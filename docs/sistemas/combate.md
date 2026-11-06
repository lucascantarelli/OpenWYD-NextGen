# Combate e dano

O sistema com mais defeito histórico registrado (`docs/migration/ingame-bugs.md`) e o mais sensível a
ordem de chamadas de RNG. Dano, crítico, resistência e morte passam por `combat.go`; o que acontece
**depois** da morte (EXP perdida, drop, respawn) é de `mobkilled.go` e `death_exp.go`.

## Fluxo de um ataque

1. Cliente envia a ação de ataque; `combat.go` valida alcance, alvo vivo e estado do atacante
   (afect impede golpe, `affect_score.go`).
2. Calcula dano conforme `docs/migration/game-rules.md` §4 — **fórmulas verificadas** contra o
   `Source/Code/` legado, com a ordem de `rand()` preservada (`tmserver/internal/rng`, LCG do MSVC).
3. Aplica no alvo, envia stats atualizados (`world/sendstats.go`) e, se zerou o HP, dispara a morte.
4. **Morte**: `mobkilled.go` credita EXP (painel de EXP, não chat), rola drop (§2) e agenda respawn
   (`world/respawn.go`).

## Regras que não podem ser quebradas

- **Ordem de RNG**: qualquer chamada nova de `rand()` antes de um cálculo existente muda drop, crítico
  e refino. É paridade, não preferência.
- **Estado dentro do loop**: handler roda dentro de `world.World.Run`; nada de lock.
- **EXP por morte de jogador**: `death_exp.go` aplica a perda de §0 — morrer para monstro custa EXP.

## Onde está

| Peça | Arquivo |
|---|---|
| Dano e crítico | `combat.go` (+ `combat_test.go`, `critical_test.go`) |
| Morte, EXP e drop | `mobkilled.go`, `death_exp.go` |
| PvP e duelo | `pvpkilled.go`, `duel.go`, `pkmode.go` |
| Fórmulas | `docs/migration/game-rules.md` §4 (dano) e §0 (perda de EXP) |
| RNG | `tmserver/internal/rng` |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Combat|Critical|DeathExp|MobKilled|Duel|PvP'
```

No jogo: bater em um mob com e sem arma, conferir a barra de HP do alvo descendo e o dano mínimo
respeitando a fórmula (§4).

!!! warning "UNVERIFIED"
    Resistência por elemento (`resist_test.go` cobre o cálculo) depende de atributos do item cujo
    mapeamento completo ainda não foi extraído para tabela. O que está provado é o cálculo, não a
    lista de itens que o exercita.

## Armadilha conhecida

Dano "às vezes" errado costuma ser **duas** chamadas de RNG em ordem trocada, não fórmula errada. Ao
investigar, compare a sequência com `Source/Code/.../Combat.cpp`, não só o resultado final.
