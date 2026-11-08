# Battle Royale — arena por rounds (modelo B)

**Status: especificado, não implementado.** Este documento é o contrato: regras, estados,
recompensa e onde o código entra. Nada aqui está no jogo ainda.

O modelo A (instância último-de-pé) foi descartado por exigir snapshot de inventário e
teleporte de sala — dois pontos de estado que o Hot Path não precisa assumir agora. O modelo B
reaproveita a mesma base da Arena Real: duas equipes, tempo fixo, pontuação por kill.

---

## 1. Formato

| Item | Valor (configurável em `game_setting`) |
|---|---|
| Equipes | **2**, até **16** jogadores cada (`br_team_size`) |
| Rounds | **3** (`br_rounds`), com **round 3 de desempate** |
| Duração do round | **5 min** (`br_round_ms` = 300000) |
| Reentrada no round | **1** respawn por jogador (`br_respawns`), com 10 s de invulnerabilidade |
| Intervalo entre rounds | **60 s** (`br_break_ms`) |
| Nível mínimo | **200** (`br_min_level`) |
| Entrada | teleporte do NPC da arena, **sem custo** |
| Itens | inventário **real**, drop **desligado** na arena (`br_no_drop = true`) |

**Vitória do round:** a equipe com mais pontos quando o tempo acaba; se uma equipe ficar sem
jogadores vivos, a outra vence o round imediatamente.

**Pontuação:** `+3` por kill, `+1` por assistência (dano ≥ 15% do HP alvo), `0` por morte.
**Desempate do round:** mortes totais → dano total → primeiro kill do round.

**Desempate da série (round 3):** se após 2 rounds cada equipe venceu um, o terceiro round é
jogado com **morte súbita**: sem respawn, primeiro kill ou última equipe viva vence.

---

## 2. Estados (máquina explícita)

```
Idle ──open──▶ Lobby ──full|timeout──▶ Countdown(10s) ──▶ Round(1..3) ──▶ Settling(3s) ──▶ Ended
   ▲                                                                          │
   └─────────────────────────── reset / próximo horário ◀─────────────────────┘
```

Regras de fronteira:

- **Lobby** aceita entrada e saída; ao fechar, quem não estiver na arena é teleportado para
  Armia e recebe aviso.
- **Countdown** congela dano e movimento (`br_frozen`), mas mantém o buff de entrada.
- **Round** é o único estado que pontua; kill fora dele é descartado.
- **Settling** mostra o placar por 3 s e aplica recompensa uma única vez (`br_reward_paid`).
- **Ended** libera os jogadores para a cidade de origem; o estado é registrado em auditoria.

O relógio vem de `ports::Clock` (nunca `SystemTime` dentro do tick), como no resto do core.

---

## 3. Recompensas

Recompensa por **participação + colocação**, paga no `Settling` (uma vez por jogador):

| Colocação | Recompensa |
|---|---|
| Vencedor da série | `br_reward_win` (default: 1 caixa de arena + 300 de fama de guilda) |
| Vice | `br_reward_lose` (default: 1 caixa de arena + 100 de fama) |
| Melhor kill/death da série | item cosmético de temporada (sem efeito de combate) |

A caixa usa **roll por peso**, a mesma semântica já confirmada do `DropItem` — nenhum item de
combate exclusivo: a arena não pode ser a única fonte de poder.

**Proibições explícitas:** nada de item de donate, nada de drop de item equipado, nada de
perda de item na morte (a arena é o único lugar do jogo onde morrer não custa nada).

---

## 4. Pontos de integração (onde o código entra)

| Ponto | Onde | O que faz |
|---|---|---|
| Estado da arena | `wyd-core` (novo componente SoA, 1 por arena) | guarda round, placar, timer — sem `Vec` por jogador |
| Entrada/saída | `wyd-core` + porta de teleporte | valida nível, equipe e lotação; rejeita com erro tipado |
| Dano/assistência | `wyd-core` (combate) | acumula dano por (atacante, alvo) dentro do round |
| Kill/assist | `wyd-core` → evento | emite `ArenaKill`/`ArenaAssist`; **nunca** escreve no banco no tick |
| Recompensa | `wyd-db` (`delivery_queue`) | entrega idempotente, drenada fora do tick |
| Auditoria | `wyd-admin` | registra série, placar e recompensa por personagem |
| Portal | `wyd-portal` | lê placar e agenda; **não** escreve estado vivo (regra do BFF) |

**Tick:** a arena cabe inteira no caminho quente — só comparações e inteiros. Se a
implementação precisar alocar por kill, ela está errada.

**Autoridade:** cliente nunca informa kill, dano ou placar. Toda pontuação nasce do combate
validado pelo servidor, como no resto do jogo.

---

## 5. Configuração (`game_setting`)

| Chave | Default | Observação |
|---|---|---|
| `br_enabled` | `false` | liga a arena |
| `br_rounds` | `3` | 2 + desempate |
| `br_round_ms` | `300000` | 5 min |
| `br_team_size` | `16` | por equipe |
| `br_respawns` | `1` | por jogador no round |
| `br_min_level` | `200` | entrada |
| `br_no_drop` | `true` | drop desligado dentro da arena |
| `br_reward_win` / `br_reward_lose` | caixa + fama | ids do `item_catalog` |

---

## 6. O que ainda não está decidido

- **Agenda**: qual dia/hora (o calendário é nosso, como no resto dos eventos — ver
  `decisoes-servidor.md` §4).
- **Temporada**: se haverá ranking sazonal e reset.
- **Espectador**: fora do escopo; exigiria modo de câmera que o 7662 não tem.

Sem essas três respostas a arena funciona, mas entra no calendário como evento manual.
