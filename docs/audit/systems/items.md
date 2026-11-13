# Sistema — Itens, refino, mix e sets

> **Paridade: P4 no refino +0…+9, P2 em mix/sets, P0 em craft/imbue.** Catálogo é dado validado;
> “regra implementada” é outra coluna.

## 1. Escopo

Tipos, categorias, propriedades, efeitos, requisitos, durabilidade, encantamento, aprimoramento,
combinação, consumo, venda, entrega a NPC, ativação de evento, representação visual.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_UseItem.cpp` (6.686 linhas), `_MSG_CombineItem*.cpp` (10+ variantes), `ItemEffect.h`, `CompRate.txt`/`SancRate.txt` |
| Cliente | `ItemList.bin` 6.500×140+4; `SkillData.bin` 248×96+4; `Mixlist.bin`; `itemicon.bin` 6.500×4 |
| Rust | `gameplay::{use_item,refine}`, `core/systems/{inventory,compose}.rs`, `sim/formula/{refine,effect}` |
| Banco | `0018_item_catalog`, `0017_taxonomia`, `0022_seed_refino`, `0025/0026_sanc_rate`, `0037` (portal), `0038` (acessório), `0043` (piedade) |
| Testes | `config_gameplay`, `refino_piedade` 5/5, `integracao/compose`, `contrato/catalogo` |
| Doc | `docs/wyd/{itens.md,sistemas/refino.md}`, ADR-0012/0019 |

## 3. Regras confirmadas

| Regra | Estado |
|---|---|
| Refino lê SancRate por tentativa | ✅ |
| Recusa (teto +6/+9, selado, `EF_NOSANC`, taxa ausente) sem mutar | ✅ |
| Grava item + piedade na **mesma transação** | ✅ |
| Emite `SendItem` (0x0182) | ✅ |
| Taxa editável no portal | ✅ (`config_revision`) |
| Mix no core | ✅ testado |
| Mix no fio | ❌ (sem opcode) |
| Craft | ❌ |
| Imbue/repleção | ❌ |
| Cadeia +12…+15 | ❌ |
| Bônus de set | **desconhecido** (`UNK-14`) |

## 4. Modelo de dados de um item

```
item_catalog: id, name, type, subtype, req_lvl, class_mask, price, primary_system (20 sistemas)
character_items: owner, slot, item_id, sanc, effects[3]
STRUCT_ITEM (wire, 8 B): sIndex + 3 efeitos de 2 B
```

`primary_system = NULL` significa “não classificado” — o portal mostra assim, não inventa.

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Catálogo (3.220) | Implementado e testado | P4 |
| Refino +0…+9 | Implementado e testado | P4 |
| Anct/celestial/+12…15 | parcial | P2 |
| Mix | core sem fio | P2 |
| Craft/imbue | ausente | P0 |
| Sets | desconhecido | P2 |
| Item funcional (consumível, teleporte, quest) | só refino ligado | P1 |
| Representação visual (ícone/mesh) | dado fixo no cliente | P3 |

## 6. Lacunas e ações

1. Definir o payload de mix (`0x03A6` ou variante) com captura (`UNK-10`).
2. Investigar se o 7662 tem set (`UNK-14`) **antes** de modelar bônus.
3. Portar tabela B16 (+25AC/+40dmg) e cadeia +12…+15.
4. Implementar consumível genérico (efeito + decremento + `SendItem`).
5. Não crescer `ItemList.bin` sem mudar o leitor do cliente (`11-items-and-equipment.md` §6).

## 7. Riscos

| Risco | Nota |
|---|---|
| Índice divergente banco × cliente | item errado na mochila |
| Refino sem transação | item duplicado/perdido |
| Mix adivinhado | consumo de item sem resultado |
| Taxa de servidor privado | balanceamento não-original |
