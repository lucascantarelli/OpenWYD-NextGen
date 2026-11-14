# Sistema — Inventário e equipamento

> **Paridade: P4 no core e no loot, P2 no fio, P1 no item no chão, P0 em baú/durabilidade.**
> O inventário é autoritativo e persistido; o jogador ainda não **move** nem **equipa** pelo
> protocolo.

## 1. Escopo

Slots, limites, empilhamento, divisão, transferência, equipar/desequipar, remover, descartar,
armazenar (baú), restrições de classe/nível, durabilidade, itens especiais, persistência,
sincronização.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `CItem.cpp`, `_MSG_{GetItem,Deposit,Withdraw,UpdateItem,DropItem,DeleteItem}.cpp` |
| Cliente | `0x0270` GetItem (28 B), `0x0171` CNFGetItem, `0x0272` DropItem, `0x0175`, `0x0374` UpdateItem, `0x0387/0x0388` |
| Rust | `core/systems/inventory.rs`, `gameplay::{add_item,use_item}`, `character/{equipment,snapshot}.rs` |
| Banco | `0003_inventario`, `character_items`, `0012` (gema, ADR-0012) |
| Testes | `integracao/inventario`, `campo_treinamento` (loot/slot), `refino_piedade` |
| Doc | ADR-0012, ADR-0015, `docs/rust/08-checklist-funcionalidades.md` §5 |

## 3. Regras confirmadas

| Regra | Valor |
|---|---|
| Slots de cargo | `16..79`, pilha até **120** |
| Faixa usada no loot | `16..80` |
| Slots de equipamento | **16** neste build (OSS v769: 18) |
| Gema | slot próprio com SancRate (ADR-0012) |
| Loot | grava com `replace_items` na mesma transação do abate |
| Sem slot | loot **não** é inventado (recusa logada) |
| Equipar | valida classe/nível (`ReqLvl`) |
| Persistência | `character_items` por personagem |

## 4. Fluxo de loot

```
morte → rolls → add_item (empilha ou slot livre) → replace_items (tx) → SendItem por slot
      → sem slot: recusa logada, sem duplicar
```

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Cargo/empilhamento/persistência | Implementado e testado | P4 |
| Loot no inventário | Implementado e testado | P4 |
| Mover/equipar no fio | core sem pacote | **P2** |
| Item no chão + pickup | regra pura | **P1** |
| Baú/armazém | ausente | P0 |
| Durabilidade/reparo | ausente | P0 |
| Trade | bloqueado por layout | P1 |
| Divisão de pilha | ausente | P0 |

## 6. Lacunas e ações

1. `0x0374 UpdateItem` (mover/equipar) — maior ganho imediato de jogabilidade.
2. `0x026E`/`0x0270` (chão/pickup) após captura (direção já corrigida em `DIV-02`).
3. Baú: schema `vault_items` + `0x0387/0x0388`.
4. Durabilidade: decidir se o 7662 a usa (`UNK-32`).
5. Trade: só com frame real (`CLIENT-10`).

## 7. Riscos

| Risco | Nota |
|---|---|
| Duplicação por trade/chão | manter `require_ported` e transação única |
| `MAX_CARGO` 120 × 128 | misturar dialetos quebra leitura |
| Persistência parcial | save deve ser tudo-ou-nada por operação |
