# Sistema — Combate, morte, EXP e drop

> **Paridade: P3.** O ciclo ataque→dano→morte→drop→EXP funciona e é autoritativo. O que falta é
> **fórmula fiel**, cadência provada, regen/respawn, PvP e affects.

## 1. Escopo

Ataque físico/mágico, dano, defesa, resistência, acerto, esquiva, crítico, skills, cooldown,
alcance, área, buffs/debuffs, morte, ressurreição, EXP, drops, PvP/PvE, mensagens.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_Attack.cpp`, `MobKilled.cpp`, `GetFunc.cpp`, `Basedef.cpp:1265/1486` |
| Cliente | `0x0367`, `0x039D`, `0x039E`; `RemoveMob`, `UpdateScore`, `SendItem` |
| Rust | `core/systems/{combat,skill}.rs`, `gameplay::{attack_at,loot_at_death,exp_reward}`, `sim/formula/*` |
| Banco | `0032_character_scores`, `0043_refine_pity`, `0048_chaos_points`, `0046/0049_campo` |
| Testes | `integracao/{autoridade,determinism}`, `config_gameplay`, `campo_drops`, `campo_treinamento`, `refino_piedade` |
| Doc | `docs/wyd/sistemas/{drops,ameacas}.md`, `docs/rust/06-matematica.md` |

## 3. Regras confirmadas (no Rust)

| Regra | Estado |
|---|---|
| Dano autoritativo | ✅ pacote forjado recusado |
| Cooldown/MP/alcance de skill | ✅ no core (sem fio) |
| Morte remove da cena | ✅ `RemoveMob RemoveType=1` |
| Drop por roll na morte | ✅ (um roll por item da tabela) |
| EXP com multiplicador | ✅ |
| Recusa sem mutação | ✅ |
| Cadência ≥ 800 ms | **não provada** |
| Regen | **ausente** |
| Respawn | **ausente** (só recarga) |
| PvP/chaos | **ausente** |
| Affects | **ausente** |
| Fórmula do legado | **não portada** (`⛔ FASE-10`) |

### 3.1 Fórmulas citadas do C++ (Fortemente indicado, não validadas aqui)

```
melee:  tdam = dam - ac/2
        combat = min(combat/2, 7)
        rnd = rand()%delta + combat + 99
        piso: <-50 → 0 ; 0..50 → 5*tdam/4 + 7 ; mínimo 1
skill:  combat = min(combat,15) ; rnd = rand()%(21-combat) + combat + 90
PvP:    Ac *= 3 ; crítico parcial ×1.3–1.4 (player) / ×1.5–1.6 (mob) ; total ×2
esquiva: rand()%1000+1 < parryretn → dam = -3
```

## 4. Fluxo

```
Attack → rate-limit → alvo válido → Engine::attack_at
  ├─ vivo: HP/alvo atualizados
  └─ fatal: RemoveMob → rolls de drop → Carry (transação) → SendItem → EXP/UpdateScore
```

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Ataque físico | Implementado e testado | P3 |
| Fórmulas | bloqueado por decisão | P1 |
| Skills | core, sem fio | P2 |
| Morte | Implementado e testado | P4 |
| Regen | ausente | P1 |
| PvP/PK/chaos | ausente | P1 |
| Buffs/affects | ausente | P0 |
| Boss | parcial (Campo) | P1 |
| Cadência | não provada | P3 |

## 6. Lacunas e ações

1. **Cadência**: trace de tempo por ataque (`GAME-20`) — impacta dano por segundo.
2. **Fórmula**: portar `Basedef.cpp:1265` para `wyd-sim` com golden e testes de borda.
3. **Regen**: ligar `formula/regen.rs` no tick.
4. **Respawn**: timer por template com teto por área.
5. **Affects**: depende de layout S→C (`UNK-05`).
6. **PvP**: definir número de chaos (−65 × −75) e penalidade.

## 7. Riscos

| Risco | Nota |
|---|---|
| Fórmula inventada | “balanceamento” que não é o original |
| Cadência errada | hits duplicados / DPS 2× |
| Drop sem transação | duplicação de item |
| Affects adivinhados | buff visual sem efeito real |
