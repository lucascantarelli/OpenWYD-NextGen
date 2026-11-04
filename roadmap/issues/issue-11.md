# Issue 11 — Combate PvE, IA de mob, drop e EXP

**Fase:** 3 · **Depende de:** 10 · **Bloqueia:** 13, 16 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Fechar o laço de progressão: mob agride/persegue/ataca, o jogador causa dano, o mob morre, cai loot e
gold, a EXP sobe, o personagem sobe de nível — com números medidos no Campo de Treino.

## Contexto / evidência

- Dano é server-authoritative (`handler/combat.go` + `combat/`), com acerto/parry/reflect e resist.
- **`HEADER.ID = ESCENE_FIELD` (30000)** é obrigatório no broadcast do ataque; sem isso a barra de EXP
  não anda e o personagem "não upa" mesmo com o servidor contando certo.
- IA em `handler/mobai.go`/`world/tick.go`: aggro por proximidade (Chebyshev 4) e retaliação,
  perseguição 1 tile/tick, ataque na cadência, recuo de ranged, hostilidade por clan
  (`world/clan.go`), pathfinding real (`route.Next` sobre HeightMap+AttributeMap).
- Drop em `loot/` + `ItemDropList.txt` + matriz de bônus do `gameconfig.txt`, com paridade de LCG.
- EXP: `level.ExpApply` + curva `g_pNextLevel`; Campo de Treino (células `0x80` do
  `AttributeMap.dat`, ~X 2064–2159 / Y 1932–2051) aplica `-training-field-exp` (padrão **6**) para
  mortais < 35 e **ignora** eventos globais (Kefra ÷2, newbie, double).
- Meta calibrada (issue #383): **nível 35 em ~15 minutos** caçando no Campo de Treino.

## Passos

1. Confirmar boot com mapas: `walkability grid baked dim=4096` e
   `training-grounds generators marked generators>0`.
2. Matar 10 mobs fracos fora da cidade e conferir: dano, HP do mob, morte, `MSG_RemoveMob`, loot.
3. Medir EXP/min no Campo de Treino por 10 minutos com personagem mortal nível 1 → registrar
   progressão e extrapolar o tempo até 35.
4. Sofrer aggro: entrar em área de mob hostil (clan 1/5) e conferir perseguição, ataque, dano no
   jogador, morte e `/restart` (revive com HP=2 e recall à cidade).
5. Matar mob ranged (ex.: `Ciclope_Arqueiro`, `EF_RANGE`) e conferir que ele não cola no corpo a corpo.
6. Verificar que cidade é **safe zone**: mob não agride dentro do retângulo de cidade e larga o alvo
   quando o jogador entra.
7. Conferir level-up: `MSG_UpdateScore` + `MSG_Motion(14,3)`, HP/MP máximos e pontos de atributo.

## Arquivos

- Consulta: `handler/combat.go`, `handler/mobai.go`, `handler/character.go` (restart),
  `world/tick.go`, `world/generator.go`, `loot/`, `level/`, `combat/`, `route/`.
- Dados: `Release/TMsrv/run/npc/*` (Exp por mob), `ItemDropList.txt`, `gameconfig.txt`.
- Testes existentes a preservar: `combat_regression_test.go`, goldens de `loot`/`refine`.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
# 1. suíte de combate/IA
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -race ./tmserver/internal/combat/ ./tmserver/internal/handler/ ./tmserver/internal/loot/
# 2. tick com 10k mobs (custo do loop)
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -run XXX -bench BenchmarkTickIdle10k -benchmem ./tmserver/internal/handler/
# 3. ataques chegando com ID de cena (log de envio)
docker compose logs --since 5m tmserver | grep -E "0x0367" | head
```

Saída esperada: testes verdes; benchmark na ordem de décimos de ms/tick; log de ataque presente.

## Critérios de aceite

- [ ] Mob morre e desaparece para todos (`MSG_RemoveMob`), sem mob fantasma
- [ ] Loot e gold caem e são coletáveis
- [ ] EXP sobe por kill e o level-up dispara com score/HP/MP corretos
- [ ] Barra de EXP do cliente anda (prova de `ESCENE_FIELD`)
- [ ] Perseguição/ataque/recuo funcionam; cidade é safe zone; regen funciona
- [ ] Morte → `/restart` revive com HP=2 e recall
- [ ] Medição de EXP/min no Campo de Treino registrada, com estimativa até o nível 35

## Riscos / rollback

- **Risco:** "não upa" ser bug de pacote e não de regra. **Mitigação:** comparar EXP no servidor ×
  barra no cliente antes de mudar fórmula.
- **Risco:** ajustar EXP/drop sem atualizar golden quebra a suíte. **Mitigação:** atualizar golden com
  o novo valor justificado — nunca remover asserção.
- **Risco:** mexer na cadência de tick afeta todo o servidor. **Mitigação:** medir antes/depois com o
  benchmark e citar números.
- **Rollback:** reverter flags (`-training-field-exp`, `-double-exp`) antes de reverter código.
