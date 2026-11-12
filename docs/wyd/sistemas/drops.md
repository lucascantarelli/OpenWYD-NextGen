# Drop e farm — algoritmo, escalas e o que cai onde

## 1. O algoritmo (é simples — e é por isso que quase todo mundo erra)

**CONFIRMADO** (WYD International, "Drop System"; central Kersef para o efeito de itens):

1. Cada item de um monstro tem um valor de **Drop Rate** próprio.
2. Ao morrer, o servidor sorteia um número de **1 a 1.000.000** para **cada item**
   (o *Drop Roll*).
3. Se `roll ≤ drop_rate`, o item **cai**. Se `roll > drop_rate`, não cai.
4. **Cada item tem o seu próprio roll** — não é um sorteio único.
5. Logo, um kill pode dar **vários itens** ou **nenhum**.

Escala: **1 = 0,0001%** → **1.000 = 0,1%** → **10.000 = 1%**.

> O erro clássico de implementação é fazer **um** roll e escolher **um** item da tabela
> ponderada. Isso dá drop "mais justo" e **está errado**: no original, matar 1.000 monstros
> com item de rate 1.000 não garante 1 item — a distribuição é binomial, e a variância é alta
> de propósito. Foi assim que o jogo ficou famoso por drop seco.

### Variações de servidor (não são o original)

| Variação | O que muda |
|---|---|
| **Guaranteed drop** | contador por monstro: após N kills, entrega o item mesmo sem sorte (comum em privates recentes) |
| **Bônus de canal** | guilda que mata Kefra aumenta drop/EXP do canal |
| **Gema de diamante** | +8% de taxa no item composto |
| **Bônus de evento** | `game_matrix` "Item Drop Bonus Settings" — 16 valores (um por faixa/nível) |

## 2. Configuração no dado do servidor

`gameconfig.txt` → `game_setting` / `game_matrix` (dado **E**, já no banco):

| Chave | Valor | Significado |
|---|---|---|
| `Etc Settings / DropItem` | 0 | multiplicador/toggle global de drop |
| `Drop Item Event / evon` | 0 | evento de drop ligado/desligado |
| `Drop Item Event / evitem` | 0 | item que o evento dropa |
| `Drop Item Event / evrate` | 0 | taxa do evento |
| `Drop Item Event / evstart` | `0 0 0 1` | janela de início (4 valores → `value_text`) |
| `Drop Item Event / evdelete` | 0 | apaga o item do evento ao fim |
| `Drop Item Event / evindex` | 0 | índice do evento |
| `Etc Event Settings / double` | 200 | bônus de "double" (drop/EXP) |
| `Etc Settings / partydif` | 200 | diferença de nível permitida no grupo para dividir drop/EXP |
| `Item Drop Bonus Settings` (matriz) | 16 × `50` | bônus por faixa de nível (4 blocos, 16 valores cada) |

**Leitura importante:** `partydif = 200` significa que grupo com diferença de nível acima de
200 não compartilha recompensa. Isso é regra de jogo, não detalhe.

## 3. O que cai onde — mapa por mapa

Consolidado de **drop lists reais** (droplist.raidhut.com, WYD Brazuca, WYD Winter, WYD WoW) e
de guias de servidor. Nível **B/C/D** — é o "onde farmar", não o "quanto cai".

### Cidades e arredores

| Mapa | Grades | Itens característicos |
|---|---|---|
| **Armia** | A | sets/armas A, repleção A, resto e poeira de Ori/Lac, **ovos**, essências |
| **Azran** | B | sets/armas B, repleção B, poeira, ovos, essências |
| **Erion** | B e C | sets B/C, poeira, **Pergaminho da Água (Normal/Medium/Arcane)**, **Carta de Duelo**, pedra secreta, grades D |

### Masmorras

| Mapa | Entrada | Grades | Destaques |
|---|---|---|---|
| **Calabouço 1º andar (N)** | **Mortais** | iniciais | armas/armaduras, jóias de Anct, ovos e âmagos, **ouro** |
| **Calabouço 2º andar (M)** | **Arch e Celestiais** | M | idem, mais ouro |
| **Calabouço 3º andar (A)** | **Celestiais e "Arcanjos"** (custom) | A | idem, ouro maior |
| **Dark Dungeon** | — | D | poeira, sets/armas D, ovos, essências |
| **Kefra** | ver `quests.md` | D | **arma D**, ovos, essências, **Refinação Abençoada**, **Emblema da Proteção**, **Pedra Lunar**, **Joia da Escuridão**, pedra secreta, incubadora |

**Regra dos andares do Calabouço:** o andar é liberado por **camada** (Mortal → Arch/Celestial
→ Celestial/Arcanjo). Não é nível: é estado de transformação. **C**.

Monstros especiais (Kalintz, Berserker, Valquíria) **só renascem** quando todos os outros do
andar morrem — é isso que torna o andar "farmável em ciclo".

### Campos e zonas

| Mapa | Grades | Destaques |
|---|---|---|
| **Niffleheim / Snowfield (Gelo)** | E | **God Set**, sets/armas/repleção E, fadas, acessórios, livros, pedra secreta |
| **Noatun / Deserto** | D | **jóias**, **Pedra do Sábio**, Pedra Reluzente, **Sefirot (TK/FM/BM/HT)**, vínculos de pedra eterna |
| **Lan N** | D | poeira, armas D, sets C/D, poção 50, **Garnet**, âmagos de cavalo |
| **Lan M** | E | Ressurreição, Vinha, armas E |
| **Lan A / Celestial** | E | Concentração, Espiritual, set mortal E, **adds tops**, poeira |
| **Uxmal (runas)** | — | poeira, ovos, essências, **Nightmare (Normal/Solo, Medium/Party, Arcane/Party)** |
| **Vale Escondido** | — | 5kk, PO/PL, itens de evento |

### Faixas de quest (o "farm guiado" do early game)

| Faixa | Local | Recompensa de drop |
|---|---|---|
| **1–40** | Campo de Treino (stages) | chave de stage, set A |
| **41–115** | Spirit Guardian | Caixa de Sabedoria (100k EXP/100k gold), orbes, colar, restos/poeira |
| **116–190** | Jardim de Deus | Lágrimas de Anjo (200k EXP/200k gold), olhos, poeira |
| **Mortal Quest 1–351** | cadeia principal | progressão de equipamento |

**D** (WYD Winter), mas bate com a estrutura oficial de quests por faixa.

## 4. Onde farmar por faixa de nível

Síntese prática (o "passo a passo" que o jogador pede):

| Nível | Onde | Por quê |
|---|---|---|
| 1–40 | Armia, cemitério (X:2374 Y:2097), Campo de Treino | primeiro set, primeiro ouro |
| 41–115 | Calabouço 1º andar (N) | armas/armaduras + jóias de Anct + ouro |
| 116–190 | Jardim de Deus + Calabouço 1º | Caixa/Lágrima = EXP + ouro, set da faixa |
| 191–250 | Deserto/Noatun | **jóias e Pedra do Sábio** (o gargalo do +10) |
| 251–320 | Kefra, Lan N | arma D, poeira, Garnet, âmagos |
| 321–400 | Lan A/Celestial, Gelo | set E, adds tops, God Set |
| Endgame | Uxmal (Nightmare), Dragon Field | Nightmare, essências, itens de God2 |

> O gargalo econômico do servidor é **Pedra do Sábio** (50kk + 2 itens +9 por tentativa de
> +10) e **jóias**. Quem controla o Deserto controla o mercado de +10. Isso é design, não
> acidente: a guerra de Noatun acontece no mapa adjacente ao Deserto por esse motivo.

## 5. Números concretos de drop (amostra)

Do `droplist.raidhut.com`, mapa **Kefra** (nível B, nomes PT-BR):

| Monstro | Drops |
|---|---|
| **FunerSeamer** | resto de Lac, resto de Ori, **Botas/Elmo/Armadura/Calça/Manopla Anã (Le)**, Pergaminho Selado, poeira de Lac |
| **FunerMomenter** | resto/poeira de Lac, resto de Ori, **Chapéu/Túnica/Calça/Luvas/Botas Conjuradoras (Le)**, **Gungnir** |
| **GrubSwarm** | resto de Ori, **Presas de Behemoth**, poeira de Lac, peças Mytril (M) |
| **Ape** | resto/poeira, **Pedra de Adamantita**, **Arco Divino**, **Espada Vorpal**, **Garra Draconiana**, poeira de Ori, **Âmago de Unicórnio** |
| **ApeBleg** | poeira, **Elmo Embutido (A)**, **Chapéu Mytril (A)**, **Peitoral de Teia (A)** |

Repare no padrão: **o mesmo monstro entrega peça (M) e (A) e (Le)** — a grade é sorteada
junto do item. Por isso `drop` precisa de **uma linha por (monstro, item, grade)**, não uma
linha por monstro.

## 6. Modelo de dado

```
drop_entry (id, monster_id, item_index, grade, rate, min_level, quest_only)
drop_bonus (block, line_index, values smallint[])   -- já existe: game_matrix
drop_event (index, on, item, rate, start_text, delete)
drop_claim (character_id, monster_id, item_index, killed_at)  -- se usar guaranteed
```

### Regras

1. **Uma linha por item+grade.** Sem isso, não se reproduz o drop real.
2. **`rate` em escala 1..1.000.000**, igual ao original. Não usar float de 0 a 1 sem
   converter — a diferença aparece no rate 1.
3. **Roll por item**, nunca ponderado.
4. **Bônus de canal** aplicado como multiplicador **depois** do roll, com teto — senão um
   evento de Kefra transforma rate 1 em 100.
5. **Log por kill** (`drop_claim`) só se o servidor usar guaranteed; no original não existe.

## 7. Lacunas

| Ponto | Status |
|---|---|
| rate de cada item no build 7662 | **vem do servidor**, não do cliente — ver `docs/cliente/tabelas-bin.md` e `docs/wyd/decisoes-servidor.md` §1 |
| tabela completa monstro→item | parcial (drop lists de servidores) |
| se `DropItem` é multiplicador ou toggle | **é PESO**: 2º argumento comparado ao roll; 4º é flag. `DropItem(item, 14500000, 0, 1)` em servidor prova que não é probabilidade absoluta |
| efeito exato de `double = 200` | **percentual de EXP: 200 = 200% (2×)** — `/gm +set double` liga/desliga EXP dobrada |

## 8. Drop direto no inventário, caixa no chão e aglutinação

Regra **do nosso servidor** (decisão de projeto do mantenedor, 2026-11-10 — o original deixa tudo
no chão; a versão moderna entrega no inventário). O sorteio do §1 não muda: um roll por item,
escala 1..1.000.000.

1. **O drop vai direto para o inventário de quem matou.**
2. **Só o que não coube vira caixa no chão.** "Não coube" = nenhuma pilha do mesmo item abaixo do
   teto **e** nenhum slot livre.
3. **A caixa nasce reservada ao matador por 30 s.** Outro jogador não consegue pegar nessa janela.
4. **Depois de 30 s ela abre para qualquer jogador** — inclusive para quem matou.
5. **Aos 3 min ela some**, com ou sem dono. Não há timer por caixa: a janela é calculada na
   leitura, a partir do instante da queda (`dropped_at_ms`, relógio do processo — nunca do
   cliente).
6. Dono que morre ou sai não estende a reserva: a janela corre igual.

| Regra | Onde | Prova |
|---|---|---|
| depósito: pilha → slot livre → sobra | `crates/wyd-core/src/systems/inventory.rs::deposit` | 5 testes: completa pilha, derrama em slot livre, mochila cheia devolve sobra, item sem aglutinação ocupa 1 slot por unidade, item vazio não entra |
| janela da caixa: dono 30 s → aberta → some em 3 min | `crates/wyd-sim/src/formula/drop.rs::GroundLoot` / `LootAccess` | 3 testes: 29 999 ms exclusivo, 31 000 ms aberto, 181 000 ms `Gone`, relógio para trás não expira |

**O que falta para o fio:** `_MSG_CreateItem` (0x026E, 30 B packed), `_MSG_GetItem` (0x0270,
28 B), `_MSG_CNFGetItem` (0x0171) e `_MSG_DecayItem` (0x016F). Layout e direção já provados no
binário — ver `docs/cliente/pacotes.md`. Falta a **faixa de ids de caixa** (UNVERIFIED).

### Aglutinação — teto por item, nunca global

O WYD **não** agrupa item igual por padrão: set, arma e peça têm uma instância por slot (refino,
Anct, gema e ADD distinguem duas peças "iguais"). O que agrupa é **item classificado como
aglutinável**, e o teto é **do item** — **não** é o `EF_AMOUNT` (esse é conteúdo de pack; a
distinção completa está em `sistemas/aglutinacao.md`):

| Item | Teto por slot | Fonte |
|---|---|---|
| Poção (Poderosa, Grande, Divina…) | **120** | regra de agrupamento do jogo + "120x Poção Poderosa"; 120 no `BaseMob` (401/406) prova que cabe |
| Moeda de Ouro (evento) | **100** | guia WYD Brazuca: "cada slot pode acumular até 100 unidades" |
| Pack de poeira (Lac/Ori) | **10** | loja: "Pack … contém 10 unidades" |
| Resto de Ori/Lac, poeira, joia (Coral, Garnet, Esmeralda, Diamante) | agrupa | drop lists + refino; teto item a item a confirmar |
| Set, arma, escudo, elmo, bota, luva, calça, capa, acessório | **1** (não agrupa) | uma instância por slot |

**Estado real do catálogo** (`item_catalog`, medido em 2026-11-10):

| Medida | Valor |
|---|---|
| `stackable = true` | **368** (ammo 12, cash 97, consumable 62, container 38, material 95, quest 64) |
| `stackable = false` | **1.320** (weapon 371, armor 360, boots 194, gloves 179, helmet 117, accessory 46, cloak 25, shield 22, cosmetic 6) |
| `stackable` nulo | **1.532** — e **todas** essas linhas também estão sem `kind` |
| linhas com `EF_AMOUNT` no `fields` | **94** |

As 1.532 sem `kind` **misturam três coisas**: ruído (índices 1/6/7/8/9 com nome de classe,
`TransKnight`/`Foema`), **item real não classificado** (`416 Antídoto`, `419 Resto_de_Oriharucon`,
`4141 Lactolerium_100`) e placeholder (`not_used`). Separar os três é a primeira tarefa da
GAME-12 — sem isso, a regra "sem classificação ⇒ teto 1" transformaria poeira real em item não
empilhável.

Duas consequências para o código, medidas agora:

1. **`ItemRules::max_stack` de produção existe agora** (`inventory::StackCaps`: aglutinável = 120,
   resto = 1, conteúdo de pack em mapa separado), com testes. Falta **alimentá-la do banco**: a
   consulta `item_stack_caps` precisa ler `fields` como **array** (`fields ? 'EF_AMOUNT'` funciona,
   `->>` devolve NULL em todas as linhas) — item 3 da GAME-12.
2. **`EF_AMOUNT` é conteúdo de pack, não teto.** O legado multiplica o preço pelo `EF_AMOUNT`
   com o comentário *"itens no pack"*; no catálogo os 94 valores são 5 (37×), 10 (24×), 20 (12×),
   100 (15×), 1 (3×) e 120 (3×) — kits, pacotes e restos ".x N". Em poção, `effects[0]` é a **cura**
   (50/200/300): derivar pilha daí seria errado por dois motivos independentes.

### Bags / expansão de inventário

`BAG_SLOTS` = 64 fixos (`Carry[64]` do legado): 64 é o **tamanho do array no fio**, não limite de
jogo. Servidores que vendem expansão usam a "Bolsa do Andarilho" (item de conveniência); o WYD
Brazuca declara "inventário LIVRE (não temos bolsa do andarilho)". **UNVERIFIED:** como o 7662
representa slot extra — o `STRUCT_MOB` tem 64 posições de `Carry`, então expansão provavelmente é
convenção de servidor sobre o mesmo array, não campo novo. Nada de código hoje aceita slot > 63.
