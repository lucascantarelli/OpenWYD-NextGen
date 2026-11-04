# Issue 12 — Comércio: loja NPC, banco/cargo, trade e autotrade

**Fase:** 3 · **Depende de:** 09, 10 · **Bloqueia:** 13, 15 · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Toda a economia transacional funcionando sem duplicar nem perder item: comprar/vender em NPC,
depositar/sacar no cargo da conta, trocar entre jogadores e operar loja pessoal.

## Contexto / evidência

- Loja de NPC: `handler/shop.go` (`MsgREQShopList`, `MsgBuy`, `MsgSell`), preço de
  `ItemPrices` (catálogo `ItemList.csv`); itens de preço 0 e item com efeito são casos já tratados.
- Cargo/banco de conta: `handler/cargo.go` — gold por `_MSG_Deposit` (`0x0388`) /
  `_MSG_Withdraw` (`0x0387`); movimento de item usa `UseItem`; NPC `Guarda_Carga` (Merchant=2) abre o
  cofre. Cargo é **por conta**, não por personagem.
- Trade P2P: `MsgTrade`/`MsgTradingItem`/`MsgQuitTrade` em `handler/trade.go`.
- Autotrade/loja pessoal: `MsgSendAutoTrade`/`MsgReqTradeList`/`MsgReqBuy` em
  `handler/autotrade.go`; fechar loja passa por `removeTrade`.
- Ouro tem três lugares: carregado, cargo da conta e chão. Confundir os três é a fonte clássica de
  "sumiu meu gold".

## Passos

1. Abrir a loja de um mercador (ex.: Armia), comprar 1 item barato e conferir gold e slot.
2. Vender o mesmo item e conferir o crédito (regra de venda do `Sell`).
3. Testar compra com gold insuficiente (deve recusar, sem debitar) e compra com inventário cheio.
4. Cargo: depositar gold, conferir no personagem e em **outro personagem da mesma conta**; sacar de
   volta. Repetir com item.
5. Trade: com dois clientes, oferecer item, confirmar dos dois lados, conferir troca e ausência de
   duplicata (inclusive cancelar no meio).
6. Autotrade: montar loja pessoal, comprar de outro cliente, conferir item/gold e o fechamento.
7. Deslogar/relogar após cada operação crítica e reconferir.

## Arquivos

- Consulta: `handler/shop.go`, `handler/cargo.go`, `handler/trade.go`, `handler/autotrade.go`,
  `handler/item.go`, `internal/migrations/` (tabelas de item e cargo).
- Dados: `Release/Common/ItemList.csv` (preços), templates de mercador em `Release/TMsrv/run/npc/`.
- Proibido: "consertar" economia com UPDATE direto no banco (o loop sobrescreve e não há auditoria).

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
docker run --rm -v "$PWD":/src -w /src golang:1.26-alpine \
  go test -race ./tmserver/internal/handler/ -run 'Shop|Cargo|Trade|AutoTrade' -v

# estado após as operações
docker compose exec -T db psql -U postgres -c \
  "select name, coin from character order by name;"
docker compose logs --since 10m tmserver | grep -E "0x0373|0x0387|0x0388|trade" | head
```

Saída esperada: testes verdes; gold inicial + valor esperado exato (sem delta inexplicado); trade sem
linha duplicada na tabela de item.

## Critérios de aceite

- [ ] Comprar e vender com preço correto, incluindo item de preço 0
- [ ] Compra recusada por gold insuficiente/inventário cheio **sem** debitar
- [ ] Cargo compartilha entre personagens da conta e persiste após relogin
- [ ] Trade troca item/gold corretamente e não duplica (testado com cancelamento)
- [ ] Autotrade abre, vende e fecha sem deixar item preso
- [ ] Nenhum item/gold perdido ou criado fora das regras (conferência SQL antes/depois)

## Riscos / rollback

- **Risco:** duplicação em trade por ordem de save. **Mitigação:** testar com relogin imediato após a
  troca; se reproduzir, é bug crítico — abrir issue própria e **não** seguir para 13.
- **Risco:** cargo por conta confundido com por personagem. **Mitigação:** testar com 2 chars da mesma
  conta explicitamente.
- **Risco:** `-npc-editing` ligado sem `import-npcs` → lojas somem. **Mitigação:** manter off por
  padrão; conferir `merchant_blocks_skipped` no boot.
- **Rollback:** operações são persistidas; reverter exige compensação in-game (`/gm`), registrando o
  que foi feito.
