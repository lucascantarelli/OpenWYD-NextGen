# Fadas — as cinco, o que cada uma faz, quanto dura e como o servidor deve aplicá-las

Fada é **item de apoio que ocupa o slot Familiar** (`Equip[13]`), com **prazo** que só corre
enquanto está equipada. Ela não combate, não tem HP e não é pet: é um **modificador de drop/EXP**
com automação de coleta nas versões prateada e dourada.

Fontes: `docs/wyd/fontes.md` §1 (**A** oficial · **B** oficial de servidor · **C** privado com guia ·
**D** comunidade · **E** dado do repositório). Nada vira regra por estar aqui: entra por
`game_setting` + teste.

---

## 1. As cinco fadas

| Fada | Drop | EXP | Como se obtém | Duração | Nível |
|---|---:|---:|---|---|---|
| **Azul** | **+32%** | — | NPC **Prisang Nell** (Premium Neil), **80 NP** | 3 dias, só equipada | **B** |
| **Verde** | — | **+16%** | Prisang Nell, **120 NP** | 3 dias | **B** |
| **Vermelha** | **+32%** | **+16%** | Prisang Nell, **200 NP** | 3 dias | **B** |
| **Prateada** | +32% | +16% | **Vermelha + Flash Prateado** (NPC **Farche/Parche**, Erion; 2kk–5kk gold conforme servidor) | 3 dias | **B/C** |
| **Dourada** | **+50%** | **+18%** | **Vermelha + Flash Dourado** (NPC **Mai/Martin**, Niflheim/Cidade do Gelo; 12kk–15kk) | 3 dias | **B/C** |

Duas variações de fonte, registradas e não somadas:

- **Flash**: o guia oficial (Kersef) diz Prateado em Farche/Erion por **2kk** e Dourado em
  Mai/Niflheim por **12kk**; a wiki de servidor privado diz **5kk** e **15kk**. Preço é
  `npc_shop_item`, não constante.
- **Fada Verde 3h**: aparece como prêmio de NPC por **50 Moedas de Ouro** (WYD Brazuca) — é a
  mesma fada com **duração menor**, não outra fada. Duração é dado do item, não do tipo.

**Fada Dourada "abrange mais itens agrupáveis"** que a Prateada (UoK/Kersef, **C**). A lista
publicada da Prateada: **Pedras Arch**, **poeiras e restos de Oriharucon e Lactolerium**; a
Dourada soma outros agrupáveis. Ou seja: **quais itens a fada agrupa é lista de servidor**, não
regra do item.

---

## 2. O que o código legado prova (fonte primária)

Do servidor legado (`Source/TMSrv/.../Functions.cpp`):

| Prova | Trecho | O que estabelece |
|---|---|---|
| `Equip[13]` é o slot Familiar | `GetTeamAlive`: `player->Equip[13].Index == 769` conta como "estar de Nyerds" | **slot 13 do `Equip[16]`** — resolve a lacuna que estava UNVERIFIED |
| Fadas são itens com prazo e o timer manda `EQUIP, 13` | `TimeLimitCheck(..., bool Fairy, ...)`: `if (Item->Index >= 3900 && Item->Index <= 3908 && Fairy) { ... SendItem(cId, EQUIP, 13, Item); }` | faixa de índices **3900–3908**; o item é o **mesmo** com contador decrescente |
| Contador é data/hora decrescente | `EFV3--` (com `EFV3 = 59` ao virar), `EFV2--` (com `EFV2 = 24`), `EFV1--` → `AmountMinus` | **três bytes** de tempo no próprio item: horas/minutos/segundos (ou dias/horas/min) |
| Expirou ⇒ item some | `AmountMinus(Item)` + `SendItem(...)` | expiração é **destrutiva**, não desequipa |
| Outras faixas com prazo | `3980–3989`, `4152–4158` | itens temporais usam o mesmo esquema de `EFV1..3` |
| Havia "fada" em outro slot no legado | `TimeLimitCheck` recebe `Fairy` como parâmetro e age em `EQUIP, 13` | a checagem é por **slot**, não por nome |

O que o legado **não** prova: o valor do bônus. Os percentuais (32/16/50/18) vêm de guia oficial de
servidor (**B**), não de constante lida no binário — o legado guarda bônus em tabela de servidor.

---

## 3. Como o bônus entra no cálculo (ordem que importa)

O texto oficial do WYD Global para a loja cash (2026) fecha o teto de agrupamento:

> "o sistema de **Agrupamento Inteligente** agrupa automaticamente, no momento da entrega, os itens
> idênticos … com um **limite máximo de 120 unidades por slot** do mesmo item."

Isso é **B** e confirma, no servidor oficial atual, o número que a comunidade de 2009 já dizia:
**120 por slot**. Ver `sistemas/aglutinacao.md` §3.

Ordem correta no abate (servidor autoridade):

1. **roll por item** (`1..1.000.000`, `docs/wyd/sistemas/drops.md` §1) — fada **não** muda o roll;
2. **bônus de taxa** aplicado como multiplicador **depois** do roll, com teto na escala
   (`drop.effective_rate`) — é o mesmo caminho da gema de diamante (+8%). Fada **soma** com gema;
3. **depósito no inventário** (`inventory::deposit_into_bag`), completando pilha até **120**;
4. **fada Prateada/Dourada**: o que não coube **não vira caixa** — item **agrupável** é agrupado
   (ou seja, o depósito é o mesmo, mas a fada amplia a chance de caber por agrupar);
   item **não agrupável** é **vendido automaticamente** (gold), não largado no chão;
5. o que sobrar sem fada, ou item fora da lista dela, segue a regra normal: **caixa no chão**
   (30 s do dono → aberto → 3 min).

Ponto delicado: **auto-venda não pode ser silenciosa**. O legado não tem esse sistema; a fada é
mecânica de servidor moderno (**B**). Para auditoria, cada venda automática precisa de log
(`event=autosell`, item, qty, gold) — sem isso, jogador perde item sem rastro.

---

## 4. Duração — o detalhe que mais erra

- O prazo **só corre equipada** ("Tempo não é gasto a não ser que seja equipada", **C**, coerente
  com o legado: o timer é chamado no laço do jogador em jogo).
- Expirar **destrói** o item (legado: `AmountMinus`), não devolve ao inventário.
- Trocar de fada **não** acumula: cada item tem seu contador.
- Duração é **por item**, não por tipo: existe Fada Verde de 3 dias (NP) e de 3 horas (50 Moedas
  de Ouro). Modelar duração no item (`game_setting`/`item_catalog`), nunca no `kind`.

---

## 5. O que o nosso servidor precisa

| Requisito | Onde | Estado |
|---|---|---|
| slot Familiar = `Equip[13]` | `crates/wyd-core/src/ecs/components.rs` (`EQUIP_SLOTS` 16) | **provado no legado**; constante nomeada falta |
| bônus de drop/EXP da fada | `crates/wyd-server/src/gameplay.rs` (`effective_rate`) | hoje só gema de diamante (+8%) |
| duração com contador no item | `character_items` (flags/efeitos) | **não existe**; precisa de 3 bytes ou data |
| auto-venda de não agrupável | caminho de loot | **não existe** |
| agrupamento garantido (prateada/dourada) | `deposit_into_bag` | regra pronta; falta a lista de itens da fada |
| log `event=autosell` | `cpsock` | **não existe** |

Issue: `roadmap-rust/issues/GAME-14-fadas-bonus-e-auto-coleta.md`.

---

## 6. Lacunas

| Ponto | Status |
|---|---|
| constante de bônus (32/16/50/18) no código legado | **não está no cliente**; é tabela de servidor → `game_setting` |
| faixa exata de índices das fadas (3900–3908 lido; azul/verde/vermelha/dourada mapeiam para quais?) | **UNVERIFIED** item a item |
| lista de itens que cada fada agrupa | **C** (publicada só parcialmente) |
| fada e morte do jogador (prazo corre morto?) | **UNVERIFIED** |
| fada dourada 2026 muda bônus? | **UNVERIFIED** — números acima são de 2009–2011 |
| "fada" de montaria/pet (homônimo) | não existe no WYD clássico; não confundir |
