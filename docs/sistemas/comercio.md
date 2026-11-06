# Comércio e loja de NPC

Comprar, vender, trocar entre jogadores e negociar em modo automático. O servidor é dono da bolsa do
NPC e do ouro do personagem; o cliente só pede.

## Loja de NPC

`shop.go` abre a loja e trata a compra/venda; `item.go` move o item no inventário. A bolsa do vendedor
vive no banco (`npc_shop_item`, ligada a `npc_template`) — não em CSV. Item comprado com preço 0 é
válido e existe no conteúdo; não é bug.

## Troca entre jogadores

`trade.go` implementa a janela de troca: as duas partes confirmam, o servidor valida posse e espaço,
e só então move os itens. "Aceitou" não é confirmação — a troca só fecha com as duas confirmações, e
qualquer alteração invalida o aceite anterior.

`autotrade.go` é a negociação automática (modo de venda em pé), `cargo.go`/`carry.go` cuidam de
capacidade e peso antes de qualquer movimento.

## Preço

Preço de item é dado de conteúdo + taxa global, não constante no código. O portal expõe a leitura em
`/precos` e o catálogo em `/itens`; ajuste de preço é ação administrativa auditada, nunca escrita
direta no banco pelo navegador.

## Onde está

| Peça | Arquivo |
|---|---|
| Loja de NPC | `shop.go` |
| Inventário e item | `item.go` |
| Troca entre jogadores | `trade.go` |
| Negociação automática | `autotrade.go` |
| Capacidade e peso | `cargo.go`, `carry.go` |
| Fluxo de trade | `docs/migration/flows.md` §5 |
| Portal | `/itens`, `/precos`, `/npcs` |

## Como verificar

```bash
go test ./tmserver/internal/handler/ -run 'Shop|Trade|Autotrade|Cargo|Carry|Item'
```

No jogo: comprar na loja de um NPC de Armia, conferir o item no inventário e o ouro debitado; depois
trocar com outro personagem e confirmar que a troca só fecha com as duas confirmações.

!!! warning "UNVERIFIED"
    A fórmula de preço final (base × taxa) não está transcrita em `docs/` para todos os casos; a
    referência prática é o valor que o servidor envia na abertura da loja. Divergência entre portal e
    jogo é bug de contrato, não de conteúdo — vale issue, não ajuste manual.

## Armadilha histórica

Mercadores duplicados em Armia nasceram de um bloco importado sombreando o seed curado (B18, 04/11).
Uma fonte de verdade por NPC: se a loja mostra item a mais, procure a segunda origem antes de "limpar"
a tela.
