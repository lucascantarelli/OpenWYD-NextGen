# Eventos — configuração, invasões, calendário e sazonais

## 1. O que é "evento" em WYD

Existem **três coisas diferentes** chamadas de evento, e confundi-las é a origem de metade dos
bugs de agenda:

| Tipo | Quem dispara | Exemplo |
|---|---|---|
| **Evento de configuração** | o próprio servidor, por chave em `gameconfig` | Drop Item Event (`evon`, `evrate`) |
| **Evento automático** | agendador do servidor, sem GM | Boss Fight no Deserto, Scramble Words |
| **Evento sazonal** | GM/equipe, com data de início e fim | Halloween, Aniversário |

**CONFIRMADO** (WYD International, "Monthly Event Calendar"; WYD Global, "Schedule of
activities").

---

## 2. Evento de drop — 100% dado, 0% código

`gameconfig.txt` → `game_setting` (já no banco):

| Chave | Valor no insumo | Papel |
|---|---|---|
| `evindex` | 0 | qual evento (índice) |
| `evon` | 0 | ligado/desligado |
| `evitem` | 0 | **qual item** o evento adiciona ao drop |
| `evrate` | 0 | **taxa** do item de evento |
| `evstart` | `0 0 0 1` | janela de início (4 valores → `value_text`) |
| `evdelete` | 0 | apaga o item do evento ao terminar |

Ou seja: **ligar um evento de drop é UPDATE em duas linhas**, não deploy. Isso é o desenho
original e deve ser preservado — o servidor de produção não pode depender de reinício para
abrir evento.

> `evstart` com **4 valores** é o motivo de a coluna ser `value_text`: o legado não cabe em
> `value_int`. Ver `modelo-banco.md` §6.7.

---

## 3. Kefra (evento semanal com consequência de canal)

| Item | Regra |
|---|---|
| Frequência | **1× por semana** por canal |
| Horário | varia por servidor (fim de semana 23h, terça 19:20, etc.) |
| Local | Hall do Kefra (2365,3892) |
| Pré-requisito | Pergaminho Selado no NPC Sobrevivente |
| Apoio | lichs + mini-bosses em número alto |
| Prêmio de guilda | **bônus de drop e EXP no canal** |
| Bloqueio | teleportes para o continente **fecham 30 min antes** |

`kefrastatus = 156` em `game_setting` é o estado do Kefra no momento do snapshot — ou seja, o
servidor **persiste o estado do boss** entre reinícios. **E**.

Mecânica de combate e drops em `quests.md` §3.

---

## 4. Invasão de monstros

| Item | Regra |
|---|---|
| Mapa | **BigCubo** (`region.Big_Cubo`, 1272,1427–1403,1532) |
| Objetivo | enfrentar os monstros invocados |
| Prêmio | **por mob abatido**: poeira de Lac/Ori, barras de 1kk, Wyden Coin |
| Restrição | algumas rodadas limitam a **Mortais ≤ 40**, outras a **≤ 400** |
| Anúncio | mensagem global + canal de evento |

**CONFIRMADO** (WYD Global, "Cronograma de Atividades").

Variação comum: **"Mob Premiado"** — um mob específico é anunciado e quem o mata recebe
Wyden Coin. **B**.

---

## 5. Calendário automático (o que roda sozinho)

Exemplo real de calendário de servidor (WYD International), que reflete o padrão da indústria:

| Evento | Frequência | O que é |
|---|---|---|
| **Boss Fight** | várias por dia | boss enorme no **Deserto** |
| **[PvP] Ice Lich** | várias por dia | ativação de PvP em área |
| **[PvP] Dark Matter** | várias por dia | idem |
| **[PvP] Valley Demon** | várias por dia | idem |
| **Scramble Words** | várias por dia | corrida de palavras (comando `/scramble`) |
| **Treasure Box Hunt** | período limitado | caixas que sobem de nível |
| **Golden Invasion** | sazonal | invasão dourada |
| **Raid** | sazonal | chefe de guilda/grupo |

**CONFIRMADO** (WYD International; nota: tudo no servidor 1, exceto Treasure Box).

### Horários que vêm do nosso dado

| Chave | Valor | Evento |
|---|---|---|
| `GTorreHour` | **22** | guerra de torres |
| `RVRHour` | **23** | guerra de reinos |
| `BRHour` | **19** | battle royale |
| `maxNightmare` | **3** | limite de dificuldade de pesadelo |

**E** (`game_setting`). Qualquer agenda hardcoded contradiz o dado do servidor.

---

## 6. Treasure Box Hunt — o evento com matemática própria

| Regra | Valor (WYD International) |
|---|---|
| Mob mínimo | **124** (uma versão) / **300+** (outra) |
| Drop rate base da caixa | **150–200** (escala 1 = 0,0001%) |
| Upgrade de caixa | **20%** por caixa |
| Troca | permitida |

**Regra de soma:** se a soma das taxas de recompensa de uma caixa for **menor que 100%**, o
restante **rola para a caixa seguinte**. Isso é o que impede "caixa vazia".

Recompensas por faixa de caixa (resumo):

| Caixa | Conteúdo |
|---|---|
| 1 | poeira de Ori/Lac, barra de 1kk, âmagos, repleção D/E |
| 2 | âmagos melhores (Unisus), poeira |
| 3 | poeira de Lac 15, Caixa de EXP, barra 5kk, frango, âmagos de grifo |
| 4 | Mandragora, Caixa de EXP, barra 10kk, pedra espiritual, ovo/âmago raros |
| 5 | barra de 100kk/50kk, Lac 100, Fragmento de Alma, **Refinação Abençoada**, **Essência dos Deuses** |

**B**. Modelar como `event_box (event_id, box_index, item_index, qty, rate_pct)` com a regra
de rollover explícita — não como cadeia de `if`.

---

## 7. Evento de teleporte (o mais lucrativo para o jogador novo)

Cadeia de NPCs numerados (I a VI). O jogador recebe um **Teleport Token** do NPC **Adventurer**
(Armia, X:2122 Y:2113), encontra o NPC I (que **anda** pela cidade) e vai avançando.

| NPC | Condição | Recompensa |
|---|---|---|
| I | ter o token | teleporte aleatório / para o mapa do II |
| II | token | poeira, Caixa de EXP, **safira**, barra 1kk, medalha, **Refinação Abençoada +9** |
| III | token | poeira 100, barra, pedra brilhante, **Pedra do Sábio**, buff 50% EXP |
| IV | token | barra 10kk, **fragmento de água**, buff 50% drop, **Nightmare**, medalha, **Refinação Abençoada**, **Emblema da Proteção** |
| V | token | barra 10kk ×2, **Emblema da Proteção** |
| VI | — | prêmio final garantido |

**B** (WYD International "Special Event Guide").

> Padrão de design: o **prêmio final é garantido**, os intermediários são aleatórios. Isso
> mantém o jogador andando até o fim. Se o servidor sortear tudo, o evento morre na metade.

---

## 8. Eventos sazonais (o que muda de verdade)

### Halloween (exemplo completo)

| Elemento | Regra |
|---|---|
| Duração | 1 semana de drops + 2 dias de eventos intensos |
| Drop | **Abóboras Malditas** em Dungeon, Underworld, Desert, Snowfield, **Kefra** |
| Boss | Whisper Lord / Cursed Soldiers, Archers, Horses |
| Troca | NPC **Hollow Jack** na zona do evento |
| Bênçãos "anormais" | **Odin** (sem redução de refino até +14, +15% de sucesso), **Freya** (sem redução de nível de montaria, 65%), **Mythril** (+10%) |

**CONFIRMADO** (WYD International, Halloween).

O detalhe importante: o evento **mexe nas taxas de refino**. Ou seja, taxa é **dado
temporário**, não constante — outro argumento para `refine_rate` com vigência
(`valid_from`, `valid_to`).

### Aniversário / Starcall / Balder

| Evento | Mecânica central |
|---|---|
| **Aniversário** | moeda comemorativa + arma comemorativa (dropa Bolsa Verde) + NPCs trocadores |
| **Starcall** | pacotes, traje com +450 status, desafios de 9 dias |
| **Balder's Lost Island** | moeda → 40% de "Gift Event" → desbloqueia 7 caixas |
| **Teleport Royale** | cupom de mercenário, grupo de 6+, NPC Vingren |

**B**. Todos têm em comum: **moeda de evento + NPC trocador + tabela de prêmio**. Isso é um
único sistema no banco, não sete.

---

## 9. Modelo de dado

```
game_event (id, code, kind, enabled, starts_at, ends_at, channel, config jsonb)
event_spawn (event_id, monster_id, region_id, x, y, count, respawn_seconds)
event_reward (event_id, source_kind, source_id, item_index, qty, rate_pct, box_index)
event_currency (code, item_index, npc_id)
event_schedule (event_id, cron, duration_seconds, announce_lead_seconds)
```

### Regras que evitam incidente

1. **Agenda vem do banco** (`event_schedule`), não de `if hour == 22`.
2. **Anúncio com antecedência** configurável (6 min para guerra, 3 min para minigame).
3. **Evento não sobrevive a restart sem estado**: `game_event.enabled` + `starts_at` no banco.
4. **Antiabuso:** limite por IP em eventos com recompensa (runas: 3/IP; loja: 2 contas/IP).
5. **Rollover de caixa** implementado como soma, com teste de "soma < 100%".
6. **Evento sazonal não deleta item do jogador** ao terminar — exceto os marcados
   `evdelete`/temporários, e isso precisa de aviso.

## 10. Lacunas

| Ponto | Status |
|---|---|
| calendário do build 7662 | **UNVERIFIED** — o insumo só traz o estado dos eventos, não a agenda |
| horário do Kefra por servidor | varia; `game_setting` não guarda hora |
| texto de anúncio | não publicado |
| recompensa de Boss Fight no Deserto | **UNVERIFIED** (varia por servidor) |
