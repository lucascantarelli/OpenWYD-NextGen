# Atlas do mundo — zonas, coordenadas e portais (snapshot)

> **O que é isto:** a lista que faltava — todas as áreas que existem no jogo, com coordenadas, e o mapa
> de rotas de teleporte. Snapshot tirado do código e do conteúdo em 2026-11-03, para consulta imediata.
> A versão viva (gerada, versionada por hash e com UI) é a **issue 41**; a edição das rotas é a **42**.
>
> Provas de origem:
> - cidades: `tmserver/internal/world/city.go:18` (tabela fixa, `Basedef.cpp:54`)
> - regiões: `tmserver/content/TMsrv/run/Regions.txt` (retângulos nomeados)
> - teleportes: `tmserver/internal/world/teleport.go:22` (`GetTeleportPosition`)
> - itens fixos/portões: `tmserver/content/TMsrv/run/InitItem.csv` (48 linhas)
> - spawns: `tmserver/content/TMsrv/run/NPCGener.txt` (6.099 geradores, ~11.930 mobs)

## 1. Cidades (5) — limite, spawn e arena de guerra

Coordenada `x,y` no grid único do mundo (4096²). `limit` é o retângulo que a `Village()` usa; `war` é a
arena de guilda (`Arena()`); `spawn` é onde o personagem nasce.

| # | Cidade | Spawn | Limite (x1,y1 → x2,y2) | Arena de guerra |
|---|---|---|---|---|
| 0 | Armia | 2086,2093 | 2052,2052 → 2171,2163 | 197,213 → 238,230 |
| 1 | Azran | 2494,1707 | 2432,1672 → 2675,1767 | 197,149 → 238,166 |
| 2 | Erion | 2453,2000 | 2448,1966 → 2476,2024 | 141,213 → 182,230 |
| 3 | Nippleheim | 3652,3122 | 3605,3090 → 3690,3260 | 141,149 → 182,166 |
| 4 | Noatum | 1050,1706 | 1036,1700 → 1072,1760 | 4000,4000 → 4010,4010 |

## 2. Rotas de teleporte (23)

Origem é arredondada para múltiplo de 4 (`x &^ 3`); o cliente manda `_MSG_ReqTeleport` ao pisar no tile.
Destino ganha spread aleatório de 0..2. Noatum é o hub: entrar custa 700, sair é de graça.

| # | Origem (x,y) | Destino (x,y) | Custo | Requisito |
|---|---|---|---|---|
| 1 | 2116,2100 | 1044,1724 | 700 | — (Armia → Noatum) |
| 2 | 2480,1716 | 1044,1716 | 700 | — (Azran → Noatum) |
| 3 | 2456,2016 | 1044,1708 | 700 | — (Erion → Noatum) |
| 4 | 1044,1724 | 2116,2100 | 0 | — |
| 5 | 1044,1716 | 2480,1716 | 0 | — |
| 6 | 1044,1708 | 2456,2016 | 0 | — |
| 7 | 2548,1740 | 2281,3688 | 0 | Fada do Vale (item 3916 equipado) |
| 8 | 2364,3924 | 3250,1703 | 0 | Kefra morto |
| 9 | 1052,1708 | 3650,3110 | 0 | — (Noatum → Nippleheim) |
| 10 | 3648,3108 | 1054,1710 | 0 | — |
| 11 | 2140,2068 | 2588,2096 | 0 | — (Armia → campo) |
| 12 | 2468,1716 | 2248,1556 | 0 | — (Azran → campo) |
| 13 | 2364,2284 | 144,3788 | 0 | — (campo → Dungeon 1) |
| 14 | 144,3788 | 2364,2284 | 0 | — |
| 15 | 2668,2156 | 148,3774 | 0 | — (entrada alternativa) |
| 16 | 144,3772 | 2668,2156 | 0 | — |
| 17 | 1824,1772 | 1172,4080 | 0 | — (campo Azran → Submundo) |
| 18 | 1172,4080 | 1824,1772 | 0 | — |
| 19 | 144,3780 | 1004,4028 | 0 | — (Dungeon 1 → andar 2) |
| 20 | 148,3780 | 1004,4028 | 0 | — (mesmo destino, bloco vizinho) |
| 21 | 1004,4028 | 148,3780 | 0 | — (volta cai no bloco leste) |
| 22 | 408,4072 | 1004,4064 | 0 | — (atalho interno) |
| 23 | 1004,4064 | 408,4072 | 0 | — |
## 3. Regiões nomeadas (52, de `Regions.txt`)

Formato do arquivo: `x1, y1, x2, y2 = Nome`. Serve para PvP, dano de área e eventos.

| Região | Retângulo |
|---|---|
| Guerra_de_Cidades | 129,137 → 253,243 |
| Monster_City | 261,261 → 380,380 |
| RvR | 1031,1927 → 1271,2171 |
| Pesadelo_N / M / A | 1285,268→1332,367 · 1026,260→1158,382 · 1155,130→1290,225 |
| Deserto_Pilar | 1137,1669 → 1282,1786 |
| Deserto_Manticora | 1282,1664 → 1396,1785 |
| Deserto_Lugefer | 1283,1788 → 1397,1910 |
| Deserto_Baixo | 1397,1671 → 1521,1785 |
| Deserto_Reino | 1522,1675 → 1669,1787 |
| Reino_Red | 1669,1544 → 1785,1639 |
| Zona_Neutra | 1670,1640 → 1785,1813 |
| Reino_Blue | 1814,1815 → 1785,1913 |
| Azran_Reino | 1788,1538 → 2185,1785 |
| Azran_Jardim | 2180,1533 → 2435,1787 |
| Castelo_Zakun | 2187,1155 → 2298,1297 |
| Azran_Zakun | 2179,1293 → 2295,1534 |
| Coliseu | 2589,1671 → 2681,1785 |
| Azran_Torre | 2444,1748 → 2552,1840 |
| Erion | 2438,1914→2680,2168 e 2435,1916→2688,2070 |
| Campo_de_Treino | 2055,1927 → 2166,2059 |
| Armia | 2164,2054→2684,2170 e 2171,2045→2720,2322 |
| Entrada_Dungeon | 2313,2170 → 2428,2296 |
| Karden | 3391,2649 → 4027,3255 |
| Agua_N / M / A | 1029,3465→1141,3569 · 1161,3595→1267,3695 · 1289,3467→1397,3569 |
| Portao_Infernal | 1651,3577 → 1927,3723 |
| Vale_Escondido | 2173,3583 → 2307,3711 |
| Dugeon_1_Andar_Hidra | 127,3710 → 767,6841 |
| Dungeon_1_Andar_Kaizen | 381,3841 → 511,4086 |
| Dungeon_2_Andar | 632,3847 → 1022,4091 |
| Dungeon_3_Andar | 898,3712 → 1143,3830 |
| Submundo_1 / 2 | 1283,3714→1538,3838 · 1153,3966→1533,4089 |
| Cubo_N / M / A | 1656,3968→1797,4092 · 1794,3845→1910,3963 · 1924,3973→2045,4091 |
| Kefra_Esquerda / Meio / Direita | 2179,3850→2303,4091 · 2304,3850→2435,4091 · 2436,3850→2551,4093 |
| Lan_N / M / A | 3581,3583→3705,3705 · 3717,3463→3833,3577 · 3851,3595→3977,3707 |
| Cassino | 3973,3973 → 4096,4096 |
| Pistas | 3330,1025 → 3602,1659 |
| Kefra_City | 3199,1667 → 3321,1785 |
| Big_Cubo | 1272,1427 → 1403,1532 |
| Nova_Guerra_Noatun | 895,1409 → 1146,1534 |

## 4. Portões e itens fixos (49, de `InitItem.csv`)

`InitItem.csv` cria os itens estáticos do mundo (portões de castelo/dungeon, baús). Entram como
`GroundItem.Static` e são abertos por `_MSG_UpdateItem` (chave `EF_KEYID` quando trancados). O catálogo
completo entra na issue 41; a **escrita** desses itens ainda não tem caminho (lacuna §5.2).

## 5. O que precisa ser gerado (issue 41)

O snapshot acima é manual. A issue 41 substitui por um artefato gerado
(`webserver/cmd/worldatlas` → `world-atlas.json`, versionado por hash) contendo: cidades, regiões,
rotas de teleporte, itens estáticos, NPCs (568 definições), spawns de mob (6.099 geradores), bosses e
as caixas de dano — mais a tela `/atlas` com mapa 1024² (downsample do `AttributeMap.dat`) e camadas.
