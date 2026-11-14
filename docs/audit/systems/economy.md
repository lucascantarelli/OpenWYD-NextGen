# Sistema — Economia, comércio e moeda

> **Paridade: P4 no ouro persistido, P3 na loja (doc divergente), P1 em drenos, P0 em mercado.**
> O ouro existe e é entregue; a economia ainda não **fecha** (não há dreno real).

## 1. Escopo

Lojas, compra/venda, preços, taxas, moedas, transações, trocas, armazenamento, banco, leilão,
validação de saldo, sincronização, concorrência, prevenção de duplicação, reversão, histórico.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_{REQShopList,ReqBuy,Buy,Sell,Trade}.cpp`, `_MSG_SendAutoTrade.cpp`, `UNKGOLD` |
| Cliente | `0x027B`, `0x0379`, `0x037A`, `0x0398`; ouro no `STRUCT_SELCHAR` |
| Rust | `gameplay` (loja viva), `admin/api/{economy,donate,rates}`, `db/repo/economy.rs` |
| Banco | `0010_economia`, `0042_credito_donate`, `0045_donate_debito`, `characters.gold` |
| Testes | `sistema/{gold_inicial,loja_viva}`, E2E do portal (financeiro) |
| Doc | `docs/wyd/economia.md`, `docs/rust/08-checklist-funcionalidades.md` §7 |

## 3. Regras confirmadas

| Regra | Estado |
|---|---|
| Ouro persistido no personagem | ✅ |
| Ouro inicial entregue | ✅ (`gold_inicial.rs`) |
| Loja viva (NPC) | ✅ em teste de sistema (`DIV-09` no checklist) |
| Preço/estoque no servidor | parcial |
| Entrega de ouro por `delivery_queue` | ✅ com idempotência |
| Donate/cash | parcial (`0042/0045`, portal) |
| Imposto/taxa | só seed |
| Mercado/leilão | **não aplicável** ao 7662 |
| Ouro por abate | citado no legado (`rand()%UNKGOLD+1`, teto 2.000) — não portado |

## 4. Fluxo de compra (esperado)

```
REQShopList → servidor envia lista (preço do servidor)
ReqBuy → valida saldo, slot, estoque → debita + entrega (tx) → SendItem/UpdateScore
Sell → valida posse → remove + credita (tx)
```

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Ouro | Implementado e testado | P4 |
| Loja NPC | parcial, doc divergente | P3 |
| Compra/venda transacional | parcial | P2 |
| Imposto/dreno | ausente | P1 |
| Donate/cash no jogo | parcial | P2 |
| Mercado/leilão | N/A | — |
| Histórico de transação | parcial (auditoria) | P2 |
| Prevenção de duplicação | ✅ por `require_ported` + tx | P4 |

## 6. Lacunas e ações

1. Reconciliar `DIV-09` e cobrir **preço, saldo e estoque** com teste de sistema.
2. Fechar o loop de donate (crédito → item/ouro no jogo).
3. Definir drenos (imposto, taxa de loja, reparo) — sem isso a moeda infla.
4. Portar ouro por abate com teto.
5. Nunca implementar trade sem layout (`CLIENT-10`).

## 7. Riscos

| Risco | Nota |
|---|---|
| Loja sem transação | item grátis |
| Preço no cliente | cliente “manda” no preço |
| Donate sem débito | cash infinito |
| Sem dreno | economia quebrada em semanas |
