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
