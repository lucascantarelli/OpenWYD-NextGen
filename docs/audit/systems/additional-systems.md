# Sistema — Demais sistemas encontrados

> Recorte dos sistemas que não têm documento próprio na matriz. Todos foram encontrados no
> projeto (`docs/wyd/sistemas.md` lista 20 em `game_system`) ou no cliente/legado.

## 1. Progressão e camadas

| Item | Estado | Evidência |
|---|---|---|
| Nível/EXP persistidos | ✅ | `0002`, `selchar.rs` |
| Pontos por nível (5 + 3 skill) | ✅ grava / ❌ distribui | `0032`, `PAR-CHR-05` |
| Curva de EXP | 🚧 | `docs/wyd/progressao.md` (`UNK-25`) |
| Camadas/tier e transformação | 🚧 | doc; “400 por camada” |
| Arch / pós-400 | ⬜ | `docs/wyd/sistemas/quests.md` |
| EXP por flags | ⬜ | `GAME-22` |

## 2. Montarias, fadas e familiares

| Sistema | Estado | Evidência | Observação |
|---|---|---|---|
| Montaria | ⬜ (só `game_system.montaria`) | Global: `MountData.bin` 70.004 B | 7662 **não tem** arquivo próprio (`UNK-16`) |
| Fadas (fairies) | ⬜ | Global: 5 fadas; azul 3 dias/80 NPs | `GAME-14` |
| Familiares (Nyerds/Imp/Griupan/Trajes) | ⬜ | `GAME-13`, `docs/wyd/sistemas/familiares.md` | **não é sistema oficial** com esse nome (`DIV-14`) |
| Slot de familiar | desconhecido | `UNK-16` | depende de UI + struct |

**Regra:** enquanto o slot não for provado, não criar `familiar_slot` no schema.

## 3. PK, ranking e punição

| Item | Estado |
|---|---|
| `pvp_points` / `chaos_points` | ✅ coluna (`0002`, `0048`) |
| Modo PK (`0x0399`) | ❌ handler |
| Penalidade de morte | ❌ |
| Ranking (`0x039F`) | ❌ |
| Cor de nome por chaos | ❌ |
| Guerra como exceção de PK | ❌ |

Números oficiais divergem (−65 × −75, `DIV-13`): decidir e documentar.

## 4. Refino avançado, soul e aglutinação

| Item | Estado | Doc |
|---|---|---|
| Refino +0…+9 | ✅ | `sistemas/refino.md` |
| Anct/+12…+15 | 🚧 | `sistemas/refino-auditoria.md` |
| Repleção/imbue | ⬜ | `sistemas/refino.md` |
| Aglutinação/soul | ⬜ | `sistemas/aglutinacao.md` |
| Craft | ⬜ | `game_system.craft` |

## 5. Mundo: clima, água, dungeons, instâncias

| Item | Estado |
|---|---|
| Clima/ciclo dia-noite | ⬜ |
| Água/nado | ⬜ |
| Dungeon/instância | ⬜ |
| Battle Royale | ⬜ (`battle-royale.md`) |
| Invasões/nightmare | ⬜ |

## 6. Administração, GM e anti-cheat

| Item | Estado | Evidência |
|---|---|---|
| Comandos GM | parcial | `wyd-admin` (grants, moderação), sem console GM no canal |
| Auditoria | ✅ | `0005`, `audit_log` |
| Papéis e bloqueio | ✅ | `0023`, E2E |
| Anti-cheat server-side | ✅ parcial | autoridade + rate-limit + limite por IP |
| Anti-cheat cliente | presença (XTrap/jlock) | eficácia desconhecida (`UNK-33`) |
| Backup/recuperação | 🚧 | stack/Docker; sem teste de restore documentado aqui |
| Comunicação entre serviços | ✅ | gateway HTTP/JSON (ADR-0008) |

## 7. Prioridade

| Ordem | Sistema | Por quê |
|---:|---|---|
| 1 | Progressão/distribuição de pontos | personagem “quebrado” sem isso |
| 2 | Clima (se barato) | imersão, sem impacto em regra |
| 3 | Montaria/fadas | conteúdo esperado pelo jogador |
| 4 | Ranking/PK | depende de PvP |
| 5 | Dungeon/BR/guerra | bloco grande, só após paridade base |

## 8. O que não fazer

- Criar schema para “familiar” sem saber o slot (`UNK-16`).
- Portar `soul`/aglutinação sem taxa citável.
- Implementar ranking antes de PvP.
- Tratar `familiares` como sistema oficial: é nomenclatura nossa (`DIV-14`).
