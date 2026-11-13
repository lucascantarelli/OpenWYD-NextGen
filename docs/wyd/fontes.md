# Fontes de pesquisa — o que cada site serve (e onde engana)

Este documento existe porque **nem toda fonte vale para todo tipo de afirmação**. Taxa de
refino de servidor privado não serve para regra de jogo; coordenada de NPC serve. Sem essa
distinção, o manual vira colagem de números incompatíveis.

## 1. Níveis de confiança

| Nível | O que é | Pode virar regra de código? |
|---|---|---|
| **A — oficial** | central do publicador, site do jogo original | sim, com teste |
| **B — oficial de servidor** | docs de servidor oficial/licenciado (WYD Global/RaidHut) | sim, marcando que é configuração |
| **C — privado com guia técnico** | servidor privado que publica guia detalhado e consistente | mecânica sim; **número não** |
| **D — comunidade/fórum** | tópicos, blogs, tutoriais de jogador | só mecânica qualitativa, sempre marcado |
| **E — dado do repositório** | seed, migração, binário do cliente 7662 | **é a verdade do nosso servidor** |

Regra de ouro: **mecânica é estável, número não é.** "A 2ª sala da Pista de Runas é defender
a Torre de Thor por 15 minutos" vale em qualquer servidor. "A taxa é 55%" muda de servidor
para servidor e até entre patches do mesmo servidor.

## 2. Fontes por site

### Oficiais / licenciadas

| Site | Nível | Cobre bem | Armadilha |
|---|---|---|---|
| `feedback.kersef.com` (WYD Global / RaidHut, PT-BR e EN) | **A/B** | sistemas completos: refino, repleção, god, skills 9–12, Arch, guild, guerra de reinos, Pista de Runas, mapa de Armia com coordenadas | artigos PT e EN do **mesmo** sistema às vezes divergem no custo (ex.: +10 com 30 Lac+5kk vs Pedra do Sábio+50kk) — são versões de servidor diferentes |
| `wydglobal.raidhut.com` (notícias/eventos) | **B** | eventos sazonais, tabelas de recompensa, NPCs com coordenada, missões numeradas | é o servidor **atual**; muita coisa não existe no 7662 |
| `wyd.ongame.com.br` via Wayback Machine (Guia Básico/Intermediário/Avançado, ~2010–2012) | **A** | mecânica da era 7.x: grupo/XP, skills, guilda, cidadania, refino +1~+11, replation, ancient, montaria, chat, C.C. | número é de época; confrontar com o cliente 7662 antes de virar regra — detalhe em `ongame-guia.md` |
| `droplist.raidhut.com` | **B** | **drop list pesquisável por mapa, monstro e item**, com nomes em PT-BR | reflete o balanceamento atual do Global |
| `en.wikipedia.org/wiki/With_Your_Destiny` | **A** (visão geral) | lore, reinos, modos de guerra, 1000 jogadores, 1500 níveis | rasíssimo em mecânica; não cite para número |

### Servidores privados com documentação séria (**C**)

| Site | Cobre bem | Por que confiar | Onde engana |
|---|---|---|---|
| `wyd-international.com` | **tabelas de taxa** (refino, composição, imbue), drop system com escala, prologue, mount system, calendário de eventos, guias de quest | publica taxa por passo e descreve o algoritmo de drop | é **servidor privado** (declara isso): as taxas são dele |
| `wydbrazuca.com.br` (guias + lista de drop) | refino de amuleto com **custos por nível**, Calabouço por andar/camada, Kefra, Anct, lista de drop por mapa | guias operacionais longos, com coordenada e nome de monstro | servidor customizado (tem Arcanjo, nível 800) |
| `wyddol.com` | composição Ancient, mounts por rank, Pista de Runas com limite por IP | consistente com as oficiais | taxas e limites próprios |
| `wydwow.com` | **droplist por área** (Vale Escondido, Lan Mortal/Celestial, Pista +5, quests Kayzen/Hydra/Elfos) | mostra o padrão de "o que cai onde" | servidor muito customizado |
| `wydwinter.wixsite.com` | **droplist completa por mapa e por faixa de quest** (1–40, 41–115, 116–190) | organiza por faixa, igual ao desenho do jogo | projeto antigo (2015) |
| `portaluok.com` / `portalrogue.com.br` | Kefra (drops do kill), Pista de Runas sala por sala, criação de Arch | detalha recompensa por sala | taxa de servidor |
| `wydwar.com` | ranking, domínio de cidade, imposto, horários de guerra | confirma estrutura de guerra | landing comercial |
| `pathoflords.com` (FreeWyd) | **matemática de combate**: velocidade de ataque, golpe duplo/triplo, crítico, evasão, com tabela | o próprio site diz que publica "as regras que mudam em relação ao WYD original" — serve de **hipótese** para a GAME-20, nunca de regra |
| `worldwyd.webnode.com.br`, `wydvaelion.online` | Soul (combinações e percentuais), Celestial/Subcelestial, quests por faixa, comandos | guias de jogador compilados de vários servidores; número é do servidor |

### Fontes técnicas de código (**D/E**, nunca contrato universal)

| Fonte | Serve para | Armadilha |
|---|---|---|
| [WYD2EncDec](https://github.com/raphaelts3/wyd2encdec) | header de 12 B, ofuscação, `0x0FDE`/`0x0FDF` (PIN) | amostras de uma versão; não cobre todos os opcodes |
| [W2PP `Basedef.h`](https://github.com/ErickAlcan/W2PP/blob/master/Code/Basedef.h) | `ESCENE_FIELD = 30000`, `STRUCT_SELCHAR`, `STRUCT_MOB` | layout C depende de `#pragma pack`/compilador/versão |
| [w2pp-OpenWYD](https://github.com/Jean1dev/w2pp-OpenWYD) | reescrita Go do servidor, alvo 7662 | engenharia reversa; estado variável |
| [WYD-Server-CPP](https://github.com/Amatsukan/WYD-Server-CPP) | source C++ de estudo (DBSrv/TMSrv) | o próprio README diz que é para estudo |
| [WYD2Bot](https://github.com/kevinkouketsu/WYD2Bot) | lê `ItemList.csv`/`SkillData.csv` | é cliente/bot, **não** servidor nem regra |
| [RaidHut droplist](https://global-droplist.raidhut.com/) | “o que cai onde” no WYD Global | balanceamento do Global, não do 7662 |

> Pesquisa consolidada, com URLs e perguntas abertas: `referencia-externa-2026-11.md`.
> `INITCODE = 0x1F11F311` **não** tem fonte pública: é medido no cliente do repositório (dado E).

### Fóruns e blogs (**D**)

| Site | Cobre bem | Como usar |
|---|---|---|
| `tapatalk.com/groups/wydbrazuca` | 7.7k tópicos: guerra, comércio, bugs, servidor 6.13/6.56/6.70 | procurar **discussão de mecânica** (ex.: "imu não funciona na 6.70"), nunca taxa |
| `tutorialwyd.blogspot.com` | stats de arma por nível (luna 140, Vorpal 157, Dianus 254…), sets por classe, composição celestial | melhor fonte **D** para número de arma; conferir contra o catálogo |
| `natashas2wyd.wordpress.com` | sets/armas com add, passagem de adicional para set Arch (N 12% M 18% A 22% LE 29%) | útil para entender o sistema de adicional |
| `blogwydtutoriais.blogspot.com` | grades N/M/A com **+8% de atributo por grade**, adicionais "tops", repleção A–E | bom para vocabulário de adicional |
| `pointdosxd.blogspot.com` | drop list 7.58/7.59 (Asgard, Origem, Global) por cidade | comparação entre servidores da mesma época |
| `tourwyd.wordpress.com` | lista de servidores privados 7.55/7.59 e o que cada um tinha | contexto histórico, não mecânica |
| `wydtutoriais.wordpress.com` | Pista de Runas, Arch, mapa | tutorial de jogador |

## 3. Fontes que **não** existem ou não servem

- **Não há** API ou base oficial de dados do build 7662. Nem item, nem drop, nem EXP.
- **Não há** publicação da curva de EXP por nível. Quem diz que tem, copiou de servidor.
- Documentação coreana original (WYD2) é fechada; o pouco que circula está em fóruns
  arquivados e é sobre o cliente, não sobre regra.

## 4. Como este manual usa cada nível

| Documento | Base principal |
|---|---|
| `lore.md` | A (Wikipédia) + A/B (Kersef "História", prólogo oficial) |
| `classes.md`, `skills.md` | A/B (Kersef 9ª–12ª) + C (WYD International 9–12) |
| `progressao.md` | **E** (`level_item`, `game_setting`) + B (tabela de pontos do Arch) |
| `sistemas/refino.md` | B (Kersef) + C (WYD International, taxas por passo) |
| `sistemas/drops.md` | B (drop system, escala 1..1.000.000) + **E** (`game_matrix`, `game_setting`) + C/D (o que cai onde) |
| `sistemas/guerras.md` | B (Siege Warfare, Guerra de Reinos) + C (WYD Global war guide) |
| `sistemas/mundo.md` | **E** (52 regiões do seed) + B (mapa de Armia com coordenadas) |
| `sistemas/quests.md` | A/B (Pista de Runas oficial) + C (sala por sala) |
| `sistemas/eventos.md` | B (calendário e configuração) + **E** (`game_setting` ev*) |
| `itens.md` | **E** (3.220 itens) + B (Premium Neil, uso de consumível) + D (stats de arma) |

## 5. Armadilhas que já causaram erro neste projeto

1. **Taxa de servidor privado tratada como regra.** WYD International dá +5→+6 em 55%; outro
   servidor dá 40%. Nenhum é "o certo": ambos vão para `refine_rate` com `source`.
2. **Nível máximo variável.** Servidores anunciam nível 800/1500. O build 7662 tem **400** por
   camada, com transformação — não confundir "nível 800" (servidor custom) com o original.
3. **Item com nome igual e função diferente.** Servidores renomeiam e reusam índices. Sempre
   cruzar `item_index` com o nosso `item_catalog`, nunca só o nome.
4. **Coordenada de NPC muda entre versões.** Odin aparece em (3238,1962) e Uxmal em
   (3301,1692) ou (3293,1694) conforme o servidor. A coordenada é **dado de mundo**, não
   constante de código.
5. **Guia de servidor descreve mecânica que não existe no original** (Arcanjo, nível 800,
   Refinador de Amuleto como item). Serve para entender a **família** do sistema, não para
   afirmar que o WYD2 tem aquilo.
6. **Matemática de servidor privado lida como "a fórmula do jogo".** O FreeWyd publica uma
tabela de velocidade/duplo/triplo que explica numericamente a "anomalia de 200%" — mas ele
próprio declara que lista **o que muda em relação ao original**. Ela entra como hipótese
testável (GAME-20), nunca como constante de `wyd-core`.
