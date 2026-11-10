# Aglutinação, `EF_AMOUNT` e quantidade — o que é teto, o que é conteúdo e o que é receita

Este documento existe porque a mesma pergunta ("quanto cabe?") tem **quatro respostas diferentes**
no WYD, e confundi-las estraga o drop, a receita de composição e o catálogo. Ele separa:

1. **Teto de aglutinação** — quanto cabe numa pilha de um mesmo item;
2. **Conteúdo de pack** — quantas unidades um item *embalagem* carrega dentro de si;
3. **Custo de composição** — quantas unidades uma receita **consome** (10, 20, 30, 100…);
4. **Quantidade dropada** — quantas unidades um monstro entrega de uma vez (`Poeira x10`).

As quatro são independentes. Uma receita de +10 que pede **30 Poeiras de Lac** não prova que o
teto da poeira seja 30, e um pack de **10** poeiras não prova que o teto seja 10.

Fontes: nível de confiança em `docs/wyd/fontes.md` §1 (**A** oficial · **B** oficial de servidor ·
**C** privado com guia · **D** comunidade · **E** dado do repositório). Nada aqui vira regra por
estar escrito aqui: regra entra por `game_setting`/catálogo + teste.

---

## 1. O que o binário e o código legado provam

`STRUCT_ITEM`/`st_Item` tem **8 bytes**: `short sIndex` + `Effect[3]`, cada efeito
`{u8 Index; u8 Value}`. O efeito `EF_AMOUNT` é o **índice 61** (`0x3D`), e o `Value` é **1 byte**.

| Prova | Onde | O que estabelece |
|---|---|---|
| `Effect[i].Index == 61` decrementa `Value` | servidor legado, `Functions.cpp::AmountMinus` | `EF_AMOUNT = 61`; consumir uma unidade é `Value--` |
| `ReturnItemAmount` devolve o `Value` de `EF_AMOUNT` como `INT8` | idem | quantidade é **1 byte** (0–255, lido com sinal no legado — `>127` estoura) |
| `GetCoinOnItens` multiplica o preço pelo `EF_AMOUNT` | idem, com o comentário *"Caso esteja em pack, multiplica o preço pago pelo número de itens no pack"* | **`EF_AMOUNT` é conteúdo de pack**, não teto de pilha |
| `GetItemPointer`: `EQUIP` 0–15, `INVENTORY` 0–63, `CARGO` 0–127 | idem | mochila = **64** slots; baú/cargo = **128** |
| `CreateItem` do cliente recebe `STRUCT_ITEM` em +0x12 com `stEffect[0] = 3D 78` | `docs/cliente/pacotes.md` §"Caixa de item" | 120 (`0x78`) cabe no mesmo byte de quantidade |
| `MSG_GetItem` do cliente: `memset(buf,0,28)` e `SendPacket(ptr, 0x1c)` | idem | tamanho de fio do pacote, não do item |

**Consequência direta:** o catálogo **não** guarda "quanto cabe na pilha". Ele guarda, no máximo,
quanto um item-embalagem contém. Procurar o teto de 120 no `ItemList`/`item_catalog` é procurar no
lugar errado — e é por isso que a medição abaixo dá valores estranhos.

---

## 2. A medição no nosso catálogo (dado E, 2026-11-10)

`item_catalog.fields` é **array JSON** (`["412","Poeira_de_Oriharucon","61.0",…]`), não objeto.
`fields ? 'EF_AMOUNT'` funciona em array; `fields->>'EF_AMOUNT'` devolve **NULL** em todas as
linhas — armadilha que já custou uma consulta errada neste repositório. A extração correta é por
par posicional (`EF_AMOUNT` seguido do valor).

| Medida | Valor |
|---|---|
| itens com `EF_AMOUNT` | **94** |
| distribuição do valor | **5** (37 itens) · **10** (24) · **20** (12) · **100** (15) · **1** (3) · **120** (3) |
| `stackable = true` | **368** |
| `stackable = false` | **1.320** |
| `stackable` nulo (todas sem `kind`) | **1.532** |
| `stackable = true` **sem** `EF_AMOUNT` | **333** |
| dos 94 com `EF_AMOUNT`: `stackable = true` | **35** — os outros **59** têm `stackable` nulo e só empilham pela fonte `legado:ef_amount` |

Os 94 com `EF_AMOUNT` são, quase todos, **embalagem**: `Kit_de_Cura_(5_M.)`, `Kit_de_Cura_(10_XG.)`,
`Kit_de_Mana_(20_XG.)`, `Pão_5_Unidades`, `Pergaminho_de_Portal(10)`, `Resto_de_Oriharucon`. Ou
seja: o valor é **quantas unidades vêm dentro** — exatamente o que o comentário do `GetCoinOnItens`
descreve. **`EF_AMOUNT` não é teto.**

As poções (`400`–`409`) **não têm** `EF_AMOUNT`; o efeito delas é `EF_HP`/`EF_MP`, gravado no
array como par `"EF_HP", "50"` — não como objeto `{"eff":…}`.
O `120` das poções aparece no **`BaseMob`** (`Carry[401]`/`Carry[406]` com `EF_AMOUNT 120`), isto é,
o personagem **nasce** com uma pilha de 120 — prova de que 120 unidades cabem num slot, não que o
teto esteja no catálogo.

---

## 3. Então qual é o teto?

**120 unidades por pilha, para item aglutinável** — é a regra observada do jogo, não um valor do
catálogo:

| Evidência | Fonte | Nível |
|---|---|---|
| "A quantia máxima de agrupamento é de **120 unidades**"; juntar arrastando 1 item sobre o mesmo slot; separar com **SHIFT + clique** escolhendo a quantidade | WYDGalahan, *Path List*, 2009 | **D** |
| Poções vendidas como "Poção Poderosa **c/120**"; pacote de ração "c/100" | World WYD, listas de evento | **C/D** |
| "Agrupamento Inteligente … **limite máximo de 120 unidades por slot** do mesmo item", no momento da entrega | WYD Global, nova loja cash, 30/04/2026 | **B** |
| Moeda de Ouro de evento acumula até **100** por slot | guia WYD Brazuca | **C** |
| Pack de poeira contém **10** unidades (loja: "Lactolerium 10x"; "Máximo de 10 Poeiras por Pack") | WYD Brazuca / WYD Br Team | **C** |
| Fada prateada "agrupa automaticamente os **itens agrupáveis**" e vende os **não agrupáveis** | Kersef (central oficial) | **B** |

O que **não** está provado: onde o número 120 vive no código. O campo de quantidade é 1 byte
(255 possível), o que significa que **120 é limite de projeto**, não de estrutura — o cliente decide
onde parar. A confirmação mais recente é **de servidor oficial** (WYD Global, loja cash de
30/04/2026: "limite máximo de 120 unidades por slot do mesmo item"), o que promove o número de
"observação de comunidade" para **regra vigente de servidor oficial (B)**. Ainda assim, a
constante do cliente/servidor continua sem leitura: entra como regra nossa (`STACK_CAP = 120`) e a
lacuna fica em FASE-03-12.

### Regra por família (decisão, com o porquê)

| Família | Teto | Por quê |
|---|---|---|
| Poção, pergaminho, comida, kit, poeira, joia, gema, moeda de evento, ração, isca | **120** | aglutinável: não tem refino/Anct/gema que distinga duas unidades |
| Moeda de Ouro de evento | **100** | valor oficial de servidor (**C**); `game_setting` pode baixar para 120 se a operação preferir um número só |
| Pack de poeira (item-embalagem) | **120** como pilha, **10 dentro** | o pack empilha como qualquer item; o conteúdo é `EF_AMOUNT` |
| Arma, escudo, elmo, armadura, calça, luva, bota, capa, acessório | **1** | refino, Ancient, gema e ADD tornam duas peças "iguais" **instâncias diferentes** |
| Item sem classificação (`stackable` nulo) | **1** | não sabemos se agrupa; empilhar por chute duplicaria item |

O caso "sem classificação → 1" é conservador de propósito: erra para o lado de **não** agrupar, que
custa um slot, em vez de agrupar, que pode **perder refino/Anct/gema**.

---

## 4. Custo de composição ≠ teto ≠ pack

Números que circulam e que **não** são teto de pilha:

| Número | O que é | Fonte |
|---|---|---|
| 10 restos → 1 poeira | receita do Alquimista (Azran, 1kk) | WYD Br Team / Point dos XD (**C/D**) |
| 30 Poeiras de Lac | **custo** do +10 (junto de 2 itens +9, 4 joias, Pedra do Sábio, gold) | Kersef (**B**) |
| 20 Poeiras de Lac | **custo** de uma troca de evento/PromoQuest (`[20] Poeira de Lactolerium`) | WYD Global, evento (**B**) |
| 2 packs de 10 = 20 poeiras | **custo** do +12..+15 | wiki Fandom (**D**) |
| 100 âmagos + 100 estabilizadores | **custo** da composição de Âmago agrupado | WYD Global, patch 15/10/2020 (**B**) |
| 100 de cada barra | recompensa/entrega de evento | WYD Global (**B**) |
| `Poeira de Lac x10` | **quantidade dropada** de uma vez | WYD Global, patch 15/10/2020 (**B**) |

Regra de implementação: receita consome `qty` **de uma pilha**; se a pilha não tem o total, a receita
falha — o servidor pode procurar em várias pilhas do mesmo item (comportamento do legado em
`FindOcurrences`, que varre o inventário coletando `Index` iguais), mas **nunca** soma além do teto
para "criar" uma pilha maior.

---

## 5. Inventário e expansão (bags)

| Fato | Evidência |
|---|---|
| Mochila = `Carry[64]` no `STRUCT_MOB`; baú/cargo = **128** | servidor legado `GetItemPointer` (**fonte primária**) |
| Servidores modernos expandem para **120 slots** apresentados como **40 slots × 3 cargos**, com itens "travados" (cadeado) liberados quando há espaço | WYDGalahan, 2009 (**D**) |
| "Bolsa do Andarilho" é item de conveniência de servidor privado; há servidor que se anuncia "inventário LIVRE" (sem bolsa) | WYD Brazuca (**C**) |
| WYD Global 2026 resolve cosméticos com **inventário virtual** (30 slots web), não com mais `Carry` | WYD Global, roadmap 2º semestre 2026 (**B**) |

Leitura correta: **o teto de 64 slots coexiste com a expansão**. A expansão é convenção de servidor
sobre uma estrutura maior (o `STRUCT_MOB` do 7662 tem 64) — nenhum campo novo do cliente foi
provado. Nosso `BAG_SLOTS` do core é 64 e **não** deve crescer sem captura que mostre onde o cliente
lê o slot extra. UNVERIFIED.

---

## 6. Os 1.532 sem `kind` separados por evidência (2026-11-10)

Medido no banco real pela view `item_bucket` (migration `0039`) — não é estimativa:

| Balde | Linhas | Como foi decidido |
|---|---:|---|
| **real** | **1.414** | nome que não é de NPC/região/classe/monstro **ou** índice usado em loja, prêmio, seed, doação, drop, refino de gema, equipamento de monstro ou teleporte |
| **ruído** | **96** | nome de NPC (72) ou classe (24) **e** preço 0 **e** `unique_id` 0 **e** nenhum uso |
| **placeholder** | **22** | `noname` (12), `not used` (7), `n a` (3) |

As 310 linhas **sem uso conhecido** ficam em `real` de propósito: rebaixar a ruído sem prova é o
único erro irreversível aqui (o item some do jogo). O balde `ruido` só existe quando há **nome
idêntico** a uma entidade não-item no banco.

Consequência para o teto: dos nulos, **55** empilham 120 (53 com `EF_AMOUNT` + 2 restos) e **1.477**
ficam em 1. Sem essa separação, `416 Antídoto` e `419 Resto_de_Oriharucon` caíam em teto 1 e a
receita que consome 10 restos não conseguiria formar a pilha.

Consulta reexecutável e a lista por família/item estão em `docs/wyd/itens.md` §5.1–§5.3.

---

## 7. O que o servidor faz com isso (estado do código)

| Peça | Onde | Estado |
|---|---|---|
| teto por item, com regra 120/1 e conteúdo de pack separado | `crates/wyd-core/src/systems/inventory.rs::StackCaps` | implementado + testes |
| depósito pilha → slot livre → sobra | `inventory::deposit_into_bag` | implementado + testes |
| janela da caixa (30 s dono / 3 min decay) | `crates/wyd-sim/src/formula/drop.rs::GroundLoot` | implementado + testes |
| teto vindo do banco em produção | `wyd_db::PgBackend::item_stack_caps` + view `item_stack_cap` (migration `0039`) | implementado; teste `crates/wyd-db/tests/stack_caps.rs` (gated por `WYD_TEST_DSN`) |
| pacotes `CreateItem`/`GetItem`/`DecayItem` | `crates/wyd-protocol/src/payload/inventory.rs` | **pendente** |
| `CNFGetItem` (0x0171) | — | **UNVERIFIED**: `Basedef.h` declara corpo de 16 B e põe `STRUCT_ITEM` em +20 — se contradiz |

---

## 8. Lacunas (o que ainda não sabemos)

| Ponto | Status |
|---|---|
| constante do teto 120 no binário/servidor | **UNVERIFIED** — campo é 1 byte; 120 é projeto, confirmado em **B** pelo WYD Global (2026), sem leitura de constante |
| teto da Moeda de Ouro (100 oficial × 120 geral) | **C**, decidir por `game_setting` |
| semântica do 4º campo do `ItemList` (`class_mask` no catálogo) | **UNVERIFIED** |
| onde o cliente lê slots além de 63 | **UNVERIFIED** |
| layout de `CNFGetItem` | **UNVERIFIED** (fonte se contradiz) |
| quais dos 1.532 nulos são item real | **resolvido**: 1.414 real · 96 ruído · 22 placeholder (view `item_bucket`) |
| que configuração do catálogo está errada | registrado em FASE-03-15 (`400`/`406` mesmo nome, `419` sem `EF_*`, placeholder com preço) |
| `EF_AMOUNT` como teto em algum item específico | improvável: 94 casos são conteúdo de pack |
