# Mundo — cidades, teleportes, regiões e dungeons

Este documento é o mapa operacional. As **52 regiões** abaixo vêm do insumo
`Regions.txt`, já no banco (`region`) — é dado **E**, não estimativa.

---

## 1. As cidades

| Cidade | Papel | Particularidade |
|---|---|---|
| **Armia** | cidade inicial; capital de serviços | muralha, cavalaria real, **todos os Skillmasters** |
| **Azran** | cidade de composição | **Aylin** (+10), **Compositor** (Anct), **Agatha** (imbue), dragão |
| **Erion** | cidade PvP de baixo nível | guerra de cidade só para **Mortais ≤ 150** |
| **Noatun** | castelo | Coroa de Athila, 40% do imposto |
| **Kefra (cidade do desconhecido)** | hub de runas e composição avançada | **Odin**, Uxmal, Pedra Lunar/Joia da Escuridão |
| **Niffleheim / Gelo** | campo de endgame | set **E**, God Set |
| **Uxmal** | quests de runa | Nightmare (3 dificuldades) |

### Armia — NPCs por número (mapa oficial)

| # | NPC | Função |
|---|---|---|
| 1 | Zona de Guilda | só membros da guilda campeã de Armia |
| 2 | Zona PvP | combate livre |
| 3 | **Martin** | recuperação, pergaminhos; **agrupa poeira** (Escritura de Ori/Lac) |
| 4 | **Balmos** | inscrição da guerra de guilda de Armia (2110,2145) |
| 5 | **Rainy** | defesa de Huntress |
| 6 | **ForeLearner** | Skillmaster Huntress |
| 7 | **Cap. Cavaleiros** | Skillmaster TransKnight |
| 8 | **Ferreiro** | defesa de TransKnight e Foema |
| 9 | **Arnod** | defesa de BeastMaster |
| 10 | **Ferreiro** | Skillmaster BeastMaster |
| 11 | **Premium Neil** | itens cash |
| 12 | **Foema Anciã** | Skillmaster Foema |
| 13 | **Guarda Carga** | baú da conta (compartilhado entre personagens) |
| 18 | **Mestre de Habilidade** | **30 safiras** para reconfigurar status |
| 19 | Zona de Teleporte | **Castelo de Noatun** |
| — | **Kibita** | cidadania de reino por canal (1 Pó de Oriharukon) |
| — | **9thMaster** (2136,2115) | 9ª skill |
| — | **Jazeg** | 12ª skill |

**CONFIRMADO** (central Kersef, "(MAPA) ARMIA"; wiki de NPCs).

> No seed os quatro Skillmasters existem de verdade: **Foema Anciã**, **Troll_Caçador**,
> **Mestre Archi**, **ForeLearner**, **Capitão Cavaleiros** — e o NPC de recompensa de guarda
> (`guard_spawn`, 5 guardas de Tauron) está em Armia.

---

## 2. Teleportes — os seis caminhos

| Caminho | De → para | Custo |
|---|---|---|
| **Zona de teleporte** (Armia, Azran, Erion) | cidade → **Town Zone** de Noatun | — |
| **Portal de retorno** (guild zone do castelo) | Noatun → cidade de origem | — |
| **Griffon Master / teleporte de mapa** | Armia → dungeons e áreas de quest | — |
| **Pedido de Caça** (Kefra, SubMundo, Deserto, Gelo) | direto para área de caça | item |
| **Pergaminho do Portal + Gema Estelar** | ponto gravado pelo jogador | item |
| **Pergaminho do Retorno** | última cidade visitada | item |
| **Comando `/Kingdom`** | reino do personagem (ou alterna, sem reino) | — |

**CONFIRMADO** para as zonas e para o comando; **B/C** para os itens.

Regra de cerco: durante a guerra de Noatun, o teleporte das cidades leva ao **Town Zone**, e o
portão entre Town e War Zone **não abre**. Ver `guerras.md`.

### Teleporte de Kefra (o mais usado)

```
Lich Crunt (2202, 3866)
   └── entrada do Hall do Kefra (2365, 3892)
         └── NPC Sobrevivente (2371, 3887): entregar Pergaminho Selado
               └── Hall do Kefra
                     └── portal (só após o Kefra morrer no canal)
                           └── Continente Desconhecido: Odin, Uxmal
```

**CONFIRMADO**. Observações que geram suporte:

- O **portal só abre no canal onde o Kefra foi derrotado** (em algumas versões, só no canal
  novato).
- **30 minutos antes** do Kefra nascer, os teleportes para o continente **bloqueiam**.
- Não há **restrição de nível ou classe** para entrar no Continente Desconhecido.

---

## 3. As 52 regiões (dado do servidor)

Extraídas de `Regions.txt` → `region` (nome, x1, y1, x2, y2). Classificação por finalidade é
leitura do nome, não campo do banco.

### Cidades e serviços

| Região | Retângulo |
|---|---|
| `Armia` | 2164,2054 – 2684,2170 (e 2171,2045 – 2720,2322) |
| `Erion` | 2438,1914 – 2680,2168 (e 2435,1916 – 2688,2070) |
| `Azran_Reino` | 1788,1538 – 2185,1785 |
| `Azran_Jardim` | 2180,1533 – 2435,1787 |
| `Azran_Zakun` | 2179,1293 – 2295,1534 |
| `Azran_Torre` | 2444,1748 – 2552,1840 |
| `Campo_de_Treino` | 2055,1927 – 2166,2059 |
| `Entrada_Dungeon` | 2313,2170 – 2428,2296 |
| `Kefra_City` | 3199,1667 – 3321,1785 |

> `Armia` e `Erion` têm **dois retângulos cada** — é a duplicidade legítima documentada em
> `modelo-banco.md` §6.6.1. `id` é sintético; o nome não é chave.

### Reinos e PvP

| Região | Retângulo |
|---|---|
| `Reino_Red` | 1669,1544 – 1785,1639 |
| `Reino_Blue` | 1785,1815 – 1814,1913 *(vem invertido no insumo; canonizado)* |
| `Zona_Neutra` | 1670,1640 – 1785,1813 |
| `RvR` | 1031,1927 – 1271,2171 |
| `Guerra_de_Cidades` | 129,137 – 253,243 |
| `Nova_Guerra_Noatun` | 895,1409 – 1146,1534 |
| `Coliseu` | 2589,1671 – 2681,1785 |

### Deserto (farm de jóia)

`Deserto_Pilar` 1137,1669–1282,1786 · `Deserto_Manticora` 1282,1664–1396,1785 ·
`Deserto_Lugefer` 1283,1788–1397,1910 · `Deserto_Baixo` 1397,1671–1521,1785 ·
`Deserto_Reino` 1522,1675–1669,1787

### Dungeons

`Dungeon_1_Andar_Kaizen` 381,3841–511,4086 · `Dungeon_2_Andar` 632,3847–1022,4091 ·
`Dungeon_3_Andar` 898,3712–1143,3830 · `Dugeon_1_Andar_Hidra` 127,3710–767,**6841** ·
`Submundo_1` 1283,3714–1538,3838 · `Submundo_2` 1153,3966–1533,4089 ·
`Castelo_Zakun` 2187,1155–2298,1297 · `Karden` 3391,2649–4027,3255

`Dugeon_1_Andar_Hidra` tem Y2 = **6841**, acima do espaço do cliente (4095). Preservado:
o teto do cliente é regra do loader, não do banco. Ver `modelo-banco.md` §6.6.

### Kefra

`Kefra_Esquerda` 2179,3850–2303,4091 · `Kefra_Meio` 2304,3850–2435,4091 ·
`Kefra_Direita` 2436,3850–2551,4093

### Eventos e minigames

`Cubo_N` 1656,3968–1797,4092 · `Cubo_M` 1794,3845–1910,3963 · `Cubo_A` 1924,3973–2045,4091 ·
`Big_Cubo` 1272,1427–1403,1532 · `Lan_N` 3581,3583–3705,3705 · `Lan_M` 3717,3463–3833,3577 ·
`Lan_A` 3851,3595–3977,3707 · `Pistas` 3330,1025–3602,1659 · `Cassino` 3973,3973–4096,4096 ·
`Monster_City` 261,261–380,380 · `Vale_Escondido` 2173,3583–2307,3711

### Pesadelo (Nightmare)

`Pesadelo_N` 1285,268–1332,367 · `Pesadelo_M` 1026,260–1158,382 · `Pesadelo_A` 1155,130–1290,225

Três dificuldades, coerentes com `maxNightmare = 3` em `game_setting`.

### Outros

`Agua_N` 1029,3465–1141,3569 · `Agua_M` 1161,3595–1267,3695 · `Agua_A` 1289,3467–1397,3569 ·
`Portao_Infernal` 1651,3577–1927,3723 · `Guerra_de_Cidades` (acima)

---

## 4. Zonas e regras de espaço

| Tipo | Efeito |
|---|---|
| **Zona PvP** | combate livre, sem penalidade |
| **Zona de guilda** | só membros da guilda campeã |
| **Zona neutra** | entre reinos |
| **Zona de quest** | instância por grupo, com limite de grupos |
| **Fora da paliçada (Noatun)** | **PK conta** mesmo durante a guerra |

---

## 5. O que o servidor precisa

1. **`region`** já carregada (52 linhas) — serve para bônus de zona, spawn e auditoria.
2. **`teleport_route (from_region, to_region, cost_item, cost_gold, requires_flag)`** — hoje
   inexistente; é o que falta para o jogador não ficar preso.
3. **Colisão** vem do `AttributeMap.dat` do cliente (`docs/cliente/`), não do banco.
4. **Spawn de monstro** por região, com `min_group`/`max_group` de `npc_definition`.
5. **Gate de war zone**: durante cerco, negar passagem pelo portão interno.

### Bug clássico de teleporte

Se o destino é uma região com **dois retângulos** (Armia, Erion), usar o primeiro retângulo
como ponto de chegada coloca o jogador no mapa errado. O ponto de spawn precisa ser
**coordenada explícita**, não "centro da região".

## 6. Lacunas

| Ponto | Status |
|---|---|
| coordenada exata de cada NPC do 7662 | parcial (Kersef lista ~19 de Armia; o resto não) |
| mapa de cada `npc_definition` | existe `map_id`, sem nome de região associado |
| lista de teleportes por NPC | **UNVERIFIED** — muda muito entre versões |
| `Karden`, `Castelo_Zakun` | sem documentação pública de propósito |
