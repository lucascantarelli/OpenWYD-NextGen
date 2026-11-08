# Guerras — cidade, castelo, reinos, torres e PK

WYD tem **quatro** camadas de PvP organizado e uma de PvP aberto. Elas se encaixam: a guilda
precisa de **cidade** para disputar o **castelo**, e de **castelo** para receber **imposto**.
Este documento segue essa cadeia.

**Fonte principal:** central Kersef ("Siege Warfare", "Guia: Guild", "Guia: Guerra de
Reinos"), WYD Global (guia de Guild and War Systems), WYD Brazuca (Guerra de Noatun, Guerra
de Cidades), Wikipédia (visão geral).

---

## 1. Visão de conjunto

| Guerra | Quando | Quem entra | Prêmio |
|---|---|---|---|
| **Guerra de Cidades** | domingo, 20:00–20:30 | 26 × 26 (top level) | a cidade + zona de guilda |
| **Guerra de Noatun** | domingo, 21:00–22:00 | guilda campeã × invasoras | castelo + **Coroa de Athila** + 40% do imposto |
| **Guerra de Reinos (RvR)** | sábado, 22:00–22:50 | 39 × 39 | fama de guilda + imposto |
| **Guerra de Torres** | horário de `GTorreHour` (22h) | guilda | fama por torre |
| **PK / Chaos** | sempre | qualquer um em zona PK | pontos de caos (penalidade) |
| **Arena Real / Battle Royale** | horário de `BRHour` (19h) | individuais/equipes | ranking |

Os horários acima vêm das fontes **e** de `gameconfig.txt` (`GTorreHour = 22`,
`RVRHour = 23`, `BRHour = 19`) — os três existem em `game_setting`, provando que o
agendamento é **dado**, não código.

---

## 2. Guerra de Cidades (Armia, Azran, Erion)

### Passo 1 — apostar (sábado)

| Item | Regra |
|---|---|
| Janela | **sábado, 00:00–23:59** (fontes antigas citam sexta — divergência registrada) |
| Requisito de guilda | **≥ 100 de fama** e **≥ 1.000 de influência** |
| Quem aposta | **Líder ou vice-líder**, no NPC da cidade |
| Custo | ouro (vai para o imposto) |

**NPCs (coordenadas oficiais):**

| Cidade | NPC | Coordenada |
|---|---|---|
| Armia | **Balmus** | (2110, 2145) |
| Azran | **Kara** | (2537, 1716) |
| Erion | **Empis** | — |

**CONFIRMADO** (WYD Global, "Guild and War Systems Guide"; central Kersef "(MAPA) ARMIA").

### Passo 2 — a batalha (domingo, 20:00–20:30)

- O sistema seleciona automaticamente **os 26 personagens de maior nível** de cada guilda
  presente na cidade → duelo **26 × 26**.
- **Erion é exceção:** só entram os 26 **Mortais** mais fortes, com **nível máximo 150**.
  Isso mantém Erion como arena de low level — e é a razão de existir a restrição.
- Vence a guilda com **maior pontuação** ao fim do horário; em empate técnico, a que tiver
  mais membros vivos.
- **CONFIRMADO** (WYD Global; WYD Brazuca, "Guerra de Cidades").

### Passo 3 — o que o vencedor ganha

| Prêmio | Efeito |
|---|---|
| **Cidade** | guilda campeã; zona de guilda exclusiva na cidade |
| **Imposto** | percentual das transações da cidade |
| **King of Noatun** | se também vencer Noatun, o líder é coroado |
| **Guild Zone** | sala com teleporte que volta ao castelo/cidade |

**Regra de cascata (importante):** se a guilda possui **apenas uma** cidade além de Noatun e
**perde essa cidade**, **perde também Noatun**. **CONFIRMADO** (WYD Global).

> Ou seja: castelo sem cidade é castelo em risco. O servidor precisa validar isso **no
> fechamento da guerra**, não na próxima rodada.

---

## 3. Guerra de Noatun (o castelo)

### Requisito de entrada

Só entra guilda que **domine ao menos uma** das três cidades (**Armia, Azran ou Erion**).
**CONFIRMADO**.

### Estrutura do mapa (é o que define a tática)

| Zona | Regra |
|---|---|
| **Town Zone** (canto inferior esquerdo) | acesso por teleporte das cidades; NPCs do castelo |
| **Castle War Zone** (parte alta) | objetivo da guerra |
| **Fora da paliçada** | zona **normal** — PK conta pelo sistema de PK |
| **Portões** | **fecham** quando a guerra começa |
| **Torre de Thor** | aparece fora da cidade e na cidade interna; **ataca todos** que se aproximam, mas pode ser destruída |
| **Portão interno** (entre Town e War Zone) | **não** pode ser destruído nem aberto durante a guerra |

**CONFIRMADO** (central Kersef, "Siege Warfare"; WYD Brazuca).

### Como se ganha

1. **Aviso:** mensagem global "a guerra começa em 6 minutos" + música própria.
2. **Portões fecham.** Invasora escolhe: **derrubar os portões** ou **dar a volta** e destruir
   só o último.
3. **Altar de Thor:** o **líder ou vice** precisa **ficar em cima do altar** o maior tempo
   possível, acumulando pontos.
4. **Placar** é anunciado **a cada 5 minutos**.
5. Se a campeã **defender por 50 minutos**, não há segundo round — ela continua campeã.
6. Caso contrário, decide-se em rounds; quem vencer o **2º round** leva a coroa.

**CONFIRMADO** (WYD Brazuca, "Guerra de Noatun"; central Kersef, "Guia: Guild").

### Prêmios

| Prêmio | Detalhe |
|---|---|
| Castelo + Guild Zone de Noatun | guilda vencedora |
| **Coroa de Athila** | só o **guild master** recebe |
| **Dragão do Castelo** | **Dragão de Hekalotia** para guilda do reino **Blue**; **Dragão de Akelonia** para guilda **Red** |
| **40% do imposto das 3 cidades** | + imposto das transações dos NPCs do castelo |
| **+200 de fama de guilda** | creditado na segunda-feira |
| Canhão | o master vencedor recebe |

**CONFIRMADO**. **Requisito prático:** o master precisa estar **online** e com **≥ 15 slots
livres** no inventário, senão perde parte da premiação.

---

## 4. Guerra de Reinos (RvR)

### Estrutura

| Item | Regra |
|---|---|
| Início | **sábado 22:00**, mensagem global; entrada pelo portal de Noatun |
| Fim | quando o **Rei do reino rival** morre, ou às **22:50** |
| Entrada | guilda precisa ter **torre** e fama alta |
| Composição | **26** da guilda + **13** do mesmo reino = **39 por lado**, **78 total** |
| Custo | defensor **não paga**; atacante paga **1.000.000 de ouro** (vai para o imposto) |
| Equipamento | entra com o **próprio** equipamento e montaria |

**CONFIRMADO** (central Kersef, "Guia: Guerra de Reinos").

### NPCs e fama por morte

| Reino Akeronia | Fama | Reino Hekalotia | Fama |
|---|---|---|---|
| King **Harbarad** | 300 | King **Glantwat** | 300 |
| Akero Cannon | 10 | Hekalo Cannon | 10 |
| Akero Training Camp | 20 | Hekalo Training Camp | 20 |
| Akero Commander | 10 | Hekalo Commander | 10 |
| Akero ent | 10 | Hekalo ent | 10 |
| Blaze ranger / shaman / sorcerer / templer | 0 | Frozen ranger / shaman / sorcerer / templer | 0 |

**Regra de crédito:** a fama vai para a guilda de quem deu **o último hit**. **CONFIRMADO**.

### Recompensa

- **Defensor:** 40% do imposto de Noatun do canal onde a guerra ocorre.
- **Atacante:** o custo de entrada acumula no imposto.
- Só guildas com **cidade naquele canal** acessam a área final do castelo. **CONFIRMADO**.

---

## 5. Torres, influência e fama de guilda

- **Guerra de Torres:** a guilda que derruba cada torre ganha fama (20 por torre em builds
  custom; o valor é configuração).
- **Influência:** moeda de guilda gasta em aposta/manutenção. Em **Lanhouse A**, monstros
  podem dropar **63 de influência** para a guilda de quem matou. **CONFIRMADO**.
- **Fama de guilda:** sobe com guerra, torre, Noatun (+200 na segunda) e NPC de RvR; **reseta
  parcialmente** — guildas com **≥ 50 de fama** são resetadas semanalmente.
  **CONFIRMADO** (WYD Global, nota de manutenção de segunda).

### Fama por conquista de cidade

O requisito de fama **varia por cidade** (Armia ≠ Azran ≠ Erion). A central publica a tabela
como "Fama, Requisito para Conquista de Cidades". **UNVERIFIED** os valores exatos por
cidade no build 7662 — o servidor deve semear `city_war_requirement(city, fame, influence)`.

---

## 6. PK e Chaos

| Regra | Detalhe |
|---|---|
| Zona PK | combate livre, **sem penalidade** nas zonas marcadas |
| Chaos Points | todo personagem começa com **0**; cada kill de jogador soma pontos |
| Consequência | pontos altos mudam a penalidade de morte e o acesso a cidades |
| Reino | matar cavaleiro do reino adversário em zona PK **rende recompensa ao reino** |
| Fora da paliçada em Noatun | PK **conta** pelo sistema normal, mesmo durante a guerra |

**CONFIRMADO** (WYD International, "Chaos System (PK System)"; Kersef, "Sistema de Reinos";
"Siege Warfare").

> Detalhe que quebra implementação ingênua: **a guerra não desliga o PK**. Durante o cerco,
> fora da paliçada o jogador ainda acumula caos. Se o servidor tratar "estou em guerra" como
> "PK liberado sem consequência", o jogador perde a conta por um erro de design.

---

## 7. Ranking

Quatro categorias espelham o ranking in-game: **Level**, **PvP**, **Patentes** e
**Arena Real**. Servidores usam "capa" (manto) como recorte visual de reino no ranking.
**CONFIRMADO** (página de ranking de servidor WYD WAR; WYD Global).

---

## 8. O que o servidor precisa implementar

### Tabelas

```
guild (id, name, master_id, fame, influence, city_id NULL, castle_id NULL)
guild_member (guild_id, character_id, rank)
city_war_bet (id, guild_id, city_id, gold, placed_at, closed)
city_war (id, city_id, starts_at, ends_at, winner_guild_id)
siege (id, castle_id, starts_at, ends_at, round, attacking_guild_id, defending_guild_id)
siege_object (id, siege_id, kind, x, y, hp)      -- gates, Tower of Thor
siege_score (siege_id, guild_id, points, updated_at)
kingdom_war (id, channel, starts_at, ends_at, winner_kingdom)
tax_ledger (id, source, city_id, amount, guild_id, at)
```

### Máquina de estado de guerra

1. `IDLE` → `ANNOUNCED` (aviso de 6 min, mensagem + música)
2. `ANNOUNCED` → `GATES_CLOSED` (fecha portões, spawna Torre de Thor)
3. `GATES_CLOSED` → `SCORING` (altar conta pontos a cada tick; placar a cada 5 min)
4. `SCORING` → `SETTLED` (define vencedor; aplica cascata cidade→castelo)
5. `SETTLED` → `REWARDED` (coroa, dragão, 40% de imposto, +200 fama na segunda)

### Regras que evitam bug

- **Portão interno intocável:** dano nele deve ser rejeitado, não absorvido.
- **Altar só conta líder/vice**, e só enquanto vivo e sobre o altar.
- **Cascata** (perder a única cidade → perder o castelo) roda no `SETTLED`, não no próximo
  domingo.
- **Imposto** é ledger imutável; recalcular saldo por soma é o que permite auditoria no
  portal (BFF lê, nunca escreve).
- **Slots do master** verificados **antes** de creditar prêmio — senão o item some.

## 9. Lacunas de fonte

| Ponto | Status |
|---|---|
| valores de fama exigidos por cidade | **UNVERIFIED** |
| horário exato de cada servidor | varia; `game_setting` manda |
| número de rounds de Noatun | "até 2" na fonte; regra de desempate não publicada |
| bônus de imposto por cidade | 40% do total é confirmado; rateio por cidade não |
| Battle Royale / Arena Real | regras de servidor, não do jogo original |
