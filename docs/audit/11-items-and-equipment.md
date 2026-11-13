# 11 — Itens, equipamentos, armas e sets

> Base: `docs/wyd/itens.md`, `docs/wyd/sistemas/refino.md`, migrations
> `0017/0018/0022/0025/0026/0037/0038/0043/0050`, `crates/wyd-core/src/systems/inventory.rs`,
> `docs/cliente/formatos.md` e medição de `client/ItemList.bin` (2026-11-13).

## 1. Anatomia de um item — onde cada campo vive

| Campo | Onde é definido | Onde é usado | Estado |
|---|---|---|---|
| Índice (`ItemID`) | `item_catalog` (Postgres) **e** `ItemList.bin` (cliente) | protocolo, drop, inventário | ✅ (dado) |
| Nome | `item_catalog.name` + `Itemname.bin` | UI do cliente | ✅ / divergência possível |
| Tipo/categoria/subcategoria | `0017_seed_taxonomia.sql` | equipar, refinar, vender | ✅ |
| Atributos (AC, dano, hit…) | `item_catalog` + `effect` | `wyd-sim`, combate | 🚧 (fórmula não portada) |
| Requisitos (classe, nível) | catálogo (`ReqLvl`) | `can_equip` | ✅ |
| Preço | catálogo / `ItemPrice.bin` (Global) | loja | 🚧 |
| Durabilidade | **não modelada** | — | ⬜ |
| Sanc/refino | `character_items.sanc` | refino, bônus | ✅ |
| Efeitos (`stEffect[3]`) | `STRUCT_ITEM` (8 B: `sIndex` + 3 efeitos de 2 B) | protocolo, refino, mix | ✅ (protocolo) |
| Ícone | `itemicon.bin` (6.500 × 4 B) | cliente | ✅ (tamanho) |
| Sprite/mesh/textura | `Mesh/` + `.wys` | cliente | Desconhecido (layout) |
| Som | `AniSound4.txt` | cliente | ✅ (texto) |
| Comportamento no servidor | `Gameplay::{use_item,refine,loot}` | jogo | 🚧 |
| Persistência | `0003_inventario.sql`, `character_items` | save | ✅ |

**Duas verdades:** catálogo no Postgres (regra) e `ItemList.bin` no cliente (apresentação).
Nunca assumir que estão sincronizados (`DIV-07`).

## 2. `STRUCT_ITEM` no fio — o único item com layout provado

| Campo | Bytes | Evidência |
|---|---:|---|
| `sIndex` | 2 | `0x486c6e add ecx,0x12` (lido em `+0x12`) |
| `stEffect[0..2]` | 6 | dois dwords consecutivos `+0x12`/`+0x16` |
| **Total** | **8** | sem folga → **packed** |

No pacote de item no chão (`0x026E`), o `STRUCT_ITEM` fica em `+0x12`, e o frame tem 30 B
(`06-protocol-catalog.md`). Isso é raro: a maioria dos itens compostos no wire segue `Desconhecido`.

## 3. Itens funcionais — como cada tipo deveria funcionar

| Tipo | Fluxo esperado | Estado no Rust |
|---|---|---|
| Consumível (poção) | `UseItem` → valida → aplica efeito → decrementa → `SendItem` | 🚧 (só refino implementado) |
| Item de refino (pedra/pó) | `UseItem` no alvo → roll → grava sanc + piedade | ✅ |
| Item entregue a NPC | diálogo → valida posse → remove → recompensa | ⬜ |
| Item que teleporta | `UseItem` → valida mapa → move | ⬜ |
| Item que inicia quest | `UseItem`/`Quest` → cria `character_quest` | ⬜ |
| Item de combinação (mix) | `CombineItem*` → valida receita → consome → gera | 🚧 (core sem fio) |
| Item com efeito temporário | affect no personagem | ⬜ (affects ausentes) |
| Item com restrição | valida classe/nível/mapa no uso | parcial |
| Item de evento/cash | entrega por `delivery_queue` | 🚧 |
| Item vinculado (bind) | marca no `STRUCT_ITEM`/catálogo | ⬜ (campo não modelado) |

**Regra de ouro:** o servidor é autoridade — o item que o cliente diz ter não existe até o banco
confirmar (`ADR-0015`).

## 4. Equipamento e slots

| Item | Valor | Fonte |
|---|---|---|
| Slots de equipamento | **16** neste build | `docs/cliente/superficie-conhecida.md` §6 (OSS v769 tem 18) |
| Cargo (inventário) | 120 por pilha; slots `16..79` | checklist 5.3 / GAME-11 |
| Faixa de slot usada pelo loot | `16..80` | `gameplay::add_item` |
| Gema | slot próprio, ADR-0012 | ADR |
| Familiar/montaria | slot **não confirmado** | GAME-13 (`UNK-16`) |

> `MAX_EQUIP` 16 × 18 é **dialeto de build**, não bug. Misturar structs de forks quebra o fio
> (`docs/cliente/superficie-conhecida.md` §6).

## 5. Armas e sets — o que é dado e o que é regra

| Peça | Dado | Regra | Lacuna |
|---|---|---|---|
| Arma | `item_catalog` (tipo, dano, req) | `can_equip`, combate | fórmula de dano não portada (`PAR-CBT-02`) |
| Armadura | idem + AC | `can_equip`, dano recebido | idem |
| Set (conjunto) | **não há tabela de bônus de conjunto** | — | 7662 embute? `UNK-14` |
| Adicional/grade (N/M/A) | parcial no catálogo | refino | semântica de +8% por grade é de fonte D |
| Sanc (+1…+9) | `character_items.sanc` | `Gameplay::refine` | ✅ |
| Anct/celestial/God2 | flags | parcial | cadeia +12…+15 ausente |
| Amuleto | refino de acessório como dado (`0038`) | ✅ | estatística do B16 em aberto |

**Sets:** o cliente do Global tem `ItemSet.bin` (928.004 B) como tabela própria. O 7662 **não
tem** esse arquivo. Ou os sets estão embutidos em outro `.bin`, ou o build não tem bônus de
conjunto. **Não é possível decidir sem RE** — está em `18-unknown-features.md` (`UNK-14`).

## 6. Manual de expansão de conteúdo (passo a passo)

Para adicionar **um item** sem quebrar cliente nem servidor:

1. **Escolher índice livre** — verificar `item_catalog` **e** `ItemList.bin` (6.500 posições).
   Reusar índice de item morto é mais seguro que crescer a tabela.
2. **Semear no Postgres** — migration nova (`0051_...sql`), com `primary_system` entre os 20 de
   `docs/wyd/sistemas.md`; `NULL` = “não classificado”.
3. **Definir efeito** — `stEffect[3]` (3 pares tipo/valor) e requisitos (`ReqLvl`, classe).
4. **Ícone** — `itemicon.bin` aponta para índice de ícone; sem ícone, o cliente desenha vazio.
5. **Nome** — `item_catalog.name` **e** `Itemname.bin`; texto longo demais quebra a coluna.
6. **Preço** — catálogo; se houver loja, `ReqBuy` usa preço do servidor.
7. **Teste** — criar item via GM/portal, equipar, refinar, dropar, vender.
8. **Evidência** — registrar no checklist e na matriz de paridade.

### 6.1 Armas

Além do item: mesh (`Mesh/`), textura (`.wys`), animação de ataque (`AniSound4.txt` mapeia som) e
 compatibilidade com o esqueleto do personagem. **Sem conversor de malha provado**, criar arma
nova é projeto de RE (`09-client-customization.md` §5.2), não de seed.

### 6.2 Sets

Bônus de conjunto exige: (a) saber se o 7662 tem o conceito (`UNK-14`), (b) modelar tabela
`item_set`/`item_set_bonus`, (c) aplicar no cálculo de atributos, (d) refletir no cliente (que
mostra apenas soma de status). **Não implementar antes de (a).**

## 7. O que não fazer

1. Não crescer `ItemList.bin` além de 6.500 sem mudar o leitor do cliente.
2. Não renomear item só no banco: o cliente mostra `Itemname.bin`.
3. Não criar item com efeito que o `STRUCT_ITEM` não expressa (3 pares).
4. Não tratar preço de servidor privado como “preço do WYD2” (`docs/wyd/fontes.md` §5).
5. Não assumir que bônus de set existe porque o Global tem `ItemSet.bin`.
