# Refino e combine

Dois sistemas distintos que o jogador confunde porque ambos "mudam o item":

- **Refino** sobe o nível de aprimoramento de um item (com risco de quebrar/rebaixar).
- **Combine** junta itens para produzir outro (variantes, Odin, catálogo).

## Refino

`refine.go` aplica, `refine_feedback.go` devolve ao cliente o resultado (sucesso, falha, quebra) e
`internal/refine` concentra as taxas em `internal/refine/{rate,sanc,amount,classebonus,itembonus}.go`
— chance, santuário, quanto sobe, e bônus por classe e por item.

As taxas **não** estão no código: vêm de `tmserver/content/Common/Settings/SancRate.txt` e
`tmserver/content/Common/Settings/CompRate.txt`. Arquivo de taxa quebrado é erro fatal no boot do tmServer — de
propósito, porque servidor subindo com taxa zero é pior que servidor fora do ar.

## Combine

`combine.go` orquestra; `combine_catalog.go` resolve o catálogo (o que combina com o quê),
`combine_odin.go` trata a variante Odin e `combine_variants.go` as demais. Combinação inválida é
recusada sem consumir item.

## Onde está

| Peça | Arquivo |
|---|---|
| Refino | `refine.go`, `refine_feedback.go` |
| Taxas de refino | `internal/refine/*`, `internal/ratetables` |
| Combine | `combine.go`, `combine_catalog.go`, `combine_odin.go`, `combine_variants.go` |
| Conteúdo de taxa | `tmserver/content/Common/Settings/{SancRate,CompRate}.txt` |
| Regra e fluxo | `docs/migration/game-rules.md` §3, `docs/migration/flows.md` §6 |

## Como verificar

```bash
go test ./tmserver/internal/refine/ ./tmserver/internal/handler/ -run 'Refine|Combine'
```

No jogo: refinar um item com taxa conhecida e conferir o feedback; depois tentar um combine inválido e
confirmar que **nenhum** item foi consumido.

!!! warning "UNVERIFIED"
    As taxas por faixa de refino são as do arquivo de conteúdo deste repositório; o servidor original
    pode ter valores diferentes. Comparar sem registrar o arquivo usado não prova nada — registre
    antes/depois em `docs/migration/game-rules.md` §3.

## Limite atual

`/refino` no portal está `soon: true`: não há tela nem RPC de administração de refino. Ajustar taxa
hoje é editar o arquivo de conteúdo e reiniciar o tmServer — o portal **não** escreve em `tmserver/content/`.
