# Itens — grades, uso e de onde vêm

Este é o documento de **uso de item**: o que cada coisa faz, quem consome, e o que muda entre
as grades. Números de arma vêm de guias de jogador (nível D) e devem ser conferidos contra
`item_catalog`; nomes e funções vêm da central oficial (nível A/B).

---

## 1. Grades — o eixo que organiza tudo

| Grade | Nome | O que muda |
|---|---|---|
| **N** | Normal | base |
| **M** | Místico | atributos maiores |
| **A** | Arcano | atributos maiores ainda |
| **Le** | Lendário | topo da linha "mortal" |
| **Anct** | Ancient | arma com efeito de joia embutido, obtida por composição |
| **E** | Especial | sets/armas de classe E e Arch: **não exigem pontos de atributo** |

Duas regras medidas na comunidade e consistentes entre si (**D**, mas repetidas em vários
guias):

- Cada grade acima de N soma cerca de **8% de atributo** em relação à anterior.
- Para usar sets **E** e sets **Arch** não é preciso ter pontos de atributo; para
  Templário/Corvo/Legionário/Mortal é preciso **nível 285+**.

**Consequência para o banco:** `grade` é numérico no catálogo e **não** deve ser traduzido
para letra por palpite — a correspondência exata N→0/M→1… não está provada para os 3.220
itens. O que existe é a **ordem**, não o mapeamento.

### Passar adicional para o set Arch (Agatha, Azran)

| Insumo | Quantidade |
|---|---|
| peça **D +9** com o adicional desejado | 1 |
| peça **Arch +9** | 1 |
| **Pedra da Luz** | 4 |

Chance por grade da peça Arch: **N 12% · M 18% · A 22% · Le 29%**. Em falha, perdem-se as
pedras **e** o set Arch. **D** (guia de 2009, consistente com a lógica oficial de imbue).

> Repare que a chance **cresce com a grade**, ao contrário do refino. É o oposto do instinto:
> aqui, item melhor tem **mais** chance de receber adicional.

---

## 2. Uso de consumível — a tabela que falta em todo guia

Lista montada a partir do catálogo do NPC **Premium Neil** (nível A/B) e de guias de servidor
(nível C). A coluna "onde usa" é o que evita o jogador perder item.

| Item | O que faz | Onde usa |
|---|---|---|
| **Poeira de Oriharukon** | refina item comum, passo baixo | no item, pelo inventário |
| **Poeira de Laktolerium** | refina passo alto (+6 a +9) e alimenta composições | idem |
| **Escritura de Oriharukon / de Lactolerium** | **agrupa** 10 poeiras em pack | NPC **Martin** (ex.: Cidade do Gelo 3643,3106) |
| **Pack de Lac** | 10 poeiras num item; usado em +12 a +15 | NPC **Odin**; para desfazer, mover o pack ao último slot e refinar algo na capa |
| **Pedra do Sábio (Wise Man)** | insumo obrigatório do +10 | NPC **Aylin**, Azran |
| **Diamante** | +8% drop na composição | Aylin / Compositor |
| **Esmeralda** | +40 perfuração | idem |
| **Coral** | +2% EXP | idem |
| **Garnet** | +40 absorção de dano | idem |
| **Pedra secreta** (Sol/Água/Terra/Vento) | +1% por unidade em +12..+15; também entra na composição celestial | Odin |
| **Refinação Abençoada (+N)** | bônus de 2% a 15% conforme refino | Odin |
| **Emblema da Proteção** | falha em +12..+15 **rebaixa 1 nível** em vez de destruir | Odin |
| **Essência dos Deuses (+N)** | sobe item celestial/God2; 4× +9 = 40% | InfWeapon/InfArmor e Odin |
| **Pedra Lunar** | composição God de **armadura** | Odin |
| **Joia da Escuridão** | composição God de **arma** | Odin |
| **Selo Mágico** | destrave de nível do Arch | NPC Odin / quests |
| **Pergaminho Selado** | **entrada** no Hall do Kefra (1 = 100 entradas em versões antigas, 1 = 1 em outras) | NPC **Sobrevivente** (2371,3887) |
| **Pedido de Caça (Kefra/SubMundo/Deserto/Gelo)** | teleporta direto para a área de caça | NPC de teleporte / Destiny |
| **Pergaminho do Retorno** | volta à última cidade | inventário |
| **Pergaminho do Portal + Gema Estelar** | grava um ponto e teleporta para ele | inventário |
| **Poção Divina (7 dias)** | buff de longa duração | inventário |
| **Pote de HP (120 un.)** | recupera 500 de HP | inventário |
| **Ração / Mount Recovery Potion** | alimenta e cura montaria | inventário |
| **Âmago (essência) da montaria** | sobe nível da montaria | montaria equipada |
| **Ovo** | gera cria (precisa de incubação) | incubadora / reino |
| **Coração do Kefra** | **não é consumido**, não vende a NPC, é negociável | prova de kill do canal |
| **Chifre do Kefra** | entregue ao NPC **Borruk** no Hall | recompensa de Kefra |
| **Runa** (Ansuz, Uraz, Thurisaz, Fehu, Raidho, Kenaz, Gebo, …) | insumo de pedra secreta / quest | Odin |
| **Safira** | entrada de reino (nível 220) e **respec de 30** | Rei / Mestre de Habilidade |
| **Medalha (verde/roxa/dourada)** | moeda de evento | NPCs de evento |
| **Wyden Coin** | moeda premium de evento (200 NP) | NPCs de evento |
| **Cupom / Ticket** | troca em NPC de evento | NPC específico |
| **Barra de prata (1kk / 100kk / 1bi)** | ouro em forma de item | vender / trocar |

### Regras de uso que quebram jogador novo

1. **Pack de poeira** só desfaz no **último slot** do inventário, refinando algo.
2. **Pergaminho Selado** é consumido na **entrada**, não no boss.
3. **Coração do Kefra** não pode ser vendido a NPC — se o jogador tentar, o item "trava" a
   venda; o guia precisa avisar.
4. **Emblema da Proteção** precisa estar **na composição**, não no inventário "por perto".
5. **Gema de efeito** é escolhida **na hora** da composição: não dá para trocar depois sem
   recompor.

---

## 3. Armas — números reais por faixa

Valores de guia de jogador (**D**), para `+0`. Servem para calibrar o catálogo, não para
sobrescrever o dado.

| Arma | Tipo | Nível | FOR | DES/CON | Add dano | Extra |
|---|---|---|---|---|---|---|
| **Luna** (`2h_sword05`) | espada 2 mãos | 140 | 456 | 135 DES | 255 | — |
| **Vorpal** (`1h_sword10`) | espada 1 mão | 157 | 395 | 268 DES | 228 | espada dupla |
| **Presas de Behemoth** | garra | 146 | 196 | 177 CON | 235 | defesa 30 |
| **Garra Draconiana** | garra | 159 | 219 | 192 CON | 272 | defesa 34 |
| **Dianus** | garra | 254 | 0 | 0 | 300 | defesa 45, classe E |
| **Arco Divino** | arco | 157 | — | 240 DES | 258 | vel. ataque 10% |
| **Arco Guardião** | arco | 260 | — | — | 290 | vel. ataque 10% |
| **Cajado duplo** | cajado | 255 | — | — | 164 | ataque mágico 38% |
| **Caliburn** | Arch | — | — | — | 255 | classe E |
| **Thrasytes** | Arch | — | — | — | — | classe E |

Observações que valem como regra:

- **Armas Arch são dropadas**, não compostas. **D**.
- Arma de classe **E** e **Arch** não pede atributo — por isso o endgame "abre" para qualquer
  build.
- `level_item` entrega armas em **154** (Gram, Gungnir, Arco Élfico, Olho do Carbunkle, Lança
  Relâmpago, Presas de Behemoth) e **254** (Foice Platinada, Karikas, Dianus, Arco Guardião,
  Cajado Caótico, Asa Draconiana). **E** — é o nosso dado, prioridade sobre o guia.

---

## 4. Sets por classe e por camada

| Classe | Linha "D" | Linha "E" mortal | Uso |
|---|---|---|---|
| TransKnight | **Embutido** | **Flamejante** | tanque/dano |
| Foema | **Mytril** | **Templário**, **Guardião** | magia |
| BeastMaster | **Elemental** | **Corvo**, **Destruição** | híbrido |
| Huntress | **Teia** | **Legionário**, **Hack** | crítico/arco |

Requisitos de nível da linha D (guia **D**): Embutido 142–148 (FOR 219–228, CON 147–153);
Mytril 137–143 (FOR 113–118, INT 198–206); Teia 134–143 (FOR 114–119, CON 186–175).

O nosso `level_item` confirma a progressão real do 7662 por faixa — Malha 29–49, Dourada
84–104, Anã 119–139, Embutida 159–199, e as linhas (M) citadas. **Use o dado, não o guia.**

### Adicionais "tops" que o jogador procura

| Peça | Adicionais valorizados |
|---|---|
| Elmo | 50 HP ou 30 defesa; 12–18% ataque mágico; 15–22% velocidade de ataque |
| Armadura | 30 defesa ou 30 dano; 50 defesa; 7.0 crítico; 10% ataque mágico |
| Botas | 12–16 aprendizagem de skill; 30 dano; 30 defesa |
| Luvas | idem botas |
| Calça | 30 defesa ou 30 dano; 7.0 crítico; 10% ataque mágico |

**D**, mas é vocabulário consistente com `EF_*` do catálogo. Serve para o portal exibir
"item bom" sem inventar score.

---

## 5. Categorias no banco (`item_kind`)

| Código | Slot | Refina? | Repleciona? |
|---|---|---|---|
| `weapon` | mão | sim | não |
| `shield` | mão secundária | sim | não |
| `armor` | corpo | sim | **sim** |
| `helmet` | cabeça | sim | não |
| `boots` / `gloves` | pés / mãos | sim | não |
| `cloak` | costas | sim | não |
| `accessory` | dedo/pescoço/pulso | **não** | não |
| `consumable` | mochila | não | não |
| `material` | mochila | não | não |
| `quest` | mochila | não | não |
| `cash` | mochila | não | não |
| `ammo` | mão/mochila | não | não |
| `container` | mochila | não | não |
| `cosmetic` | — | não | não |

Fonte: central Kersef (refino e repleção) + regra de classificação do gerador.

> **Acessório não refina** é a exceção que mais confunde — e é justamente a que tem sistema
> próprio (benção de refinação de amuleto, com custo por nível: +10 = 500kk, +11 = 700kk,
> +12 = 900kk, +13 = 1,2bi, +14 = 1,6bi, Supremo = 1,8bi, **C**).

---

## 6. Onde cada item nasce (visão macro)

| Origem | O que entrega | Detalhe |
|---|---|---|
| **Drop de monstro** | tudo | ver `sistemas/drops.md` |
| **Prêmio de nível** (`level_item`) | sets e armas por faixa | 161 linhas; idempotente |
| **Quest** | sets, mantos, runas, selos | `sistemas/quests.md` |
| **Refino/composição** | +N, Anct, celestial | `sistemas/refino.md` |
| **NPC de evento** | medalhas, cupons, Wyden Coin | `sistemas/eventos.md` |
| **Loja de NPC** (`npc_shop_item`) | poção, pergaminho, equipamento básico | 6.706 slots no seed |
| **Doação / NP** | conveniência e cosmético | fora do escopo do servidor clássico |

---

## 7. O que o servidor precisa garantir

1. **`item_instance` por item possuído** (refino, ancient, celestial, gema, repleção, imbue,
   `additional_a/b`, `bound_to`). Item de catálogo é **molde**; o que o jogador tem é instância.
2. **Consumo atômico.** Poção/pergaminho: debitar e aplicar no mesmo commit.
3. **Item de quest não vendável.** `Coração do Kefra` (`EF_NOTRADE` ausente, mas
   `sell_to_npc = false`) é caso explícito.
4. **Pack/unpack de poeira** como receita (`pack_recipe`), não como hack de inventário.
5. **Empilhamento por `kind`** e teto por item (`max_stack`), nunca global.

## 8. Lacunas

| Ponto | Status |
|---|---|
| mapeamento exato `grade` numérico → N/M/A/Le | **UNVERIFIED** |
| bônus exato por grade (8% é afirmação de guia) | **UNVERIFIED** |
| stats de todas as 3.220 linhas | dado existe; semântica de coluna não |
| quais itens são "impossível de refinar" | lista não publicada; vem do cliente |
