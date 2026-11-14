# Sistema — Eventos, calendário e arena

> **Paridade: P1.** Existe schema e documentação; **nenhum agendador** roda no servidor.

## 1. Escopo

Eventos automáticos, agendados, manuais, sazonais, de mapa, de combate, de NPC, globais;
ativação/desativação, participação, condições, recompensas, sincronização, persistência,
mensagens, estados, temporizadores.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | Go `worldevent.go`, `weather.go`, `water.go`, `nightmare*.go`, `training_field.go` |
| Cliente | `0x0101` (aviso), `ESCENE_FIELD` (spawn/remoção), `0x0E0E` (guerra) |
| Rust | ausente (só `config` e seeds) |
| Banco | `0014_eventos`, `0015_arena`, `game_setting` (`ev*`) |
| Testes | nenhum dedicado a evento |
| Doc | `docs/wyd/sistemas/{eventos.md,battle-royale.md}` |

## 3. O que existe hoje

| Item | Estado |
|---|---|
| Tabela `eventos` | ✅ (dado) |
| Tabela `arena` | ✅ (dado) |
| Configuração `ev*` em `game_setting` | ✅ (dado) |
| Agendador/timer | ❌ |
| Ativação por horário | ❌ |
| Participação/inscrição | ❌ |
| Recompensa de evento | ❌ |
| Aviso ao jogador | parcial (`send_message`) |
| Spawn/despawn de evento | ❌ |

## 4. Calendário oficial (referência, não regra)

| Evento | Janela | Fonte |
|---|---|---|
| Noatun War | a cada 2 semanas, domingo 21h–22h (GMT-3) | WYD Global |
| Boss Kefra | sábado 17h, Hall do Kefra (2366,3932) | WYD Global |
| Guerra de Reinos | portal de Noatun; invencível por 8 min | KB oficial |

**Não copiar horário** para o nosso servidor sem decisão: é configuração de outro servidor.

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Eventos automáticos | ausente | P1 |
| Eventos sazonais | ausente | P0 |
| Arena/Battle Royale | ausente | P0 |
| Avisos | parcial | P2 |
| Persistência de participação | ausente | P0 |

## 6. Lacunas e ações

1. Agendador determinístico no tick (com clock injetável para teste).
2. Modelo `event_definition` (janela, mapa, condições, recompensa) + `event_participation`.
3. Ativação/desativação com log e `/health`.
4. Aviso via `0x0101` e spawn via cena.
5. Arena só depois de PvP (`PAR-CBT-06`).

## 7. Riscos

| Risco | Nota |
|---|---|
| Timer no Hot Path com I/O | viola invariante 1 |
| Evento sem idempotência | recompensa dupla |
| Horário de outro servidor | expectativa errada do jogador |
