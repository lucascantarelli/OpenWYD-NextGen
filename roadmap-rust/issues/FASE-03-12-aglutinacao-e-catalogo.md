# FASE-03-12 — Aglutinação item a item e teto de pilha no catálogo

## Objetivo

Fechar a classificação de aglutinação dos 3.220 itens e derivar `max_stack` do dado, para o drop
empilhar poeira/joia/poção e **nunca** empilhar set, arma ou acessório.

## O que já é verdade

| Item | Teto por slot | Fonte |
|---|---|---|
| Poção (Poderosa/Grande/Divina) | 120 | `EF_AMOUNT` 120 (`BaseMob` 401/406; loja "120x Poção Poderosa") |
| Moeda de Ouro (evento) | 100 | guia WYD Brazuca |
| Pack de poeira | 10 | loja ("contém 10 unidades") |
| Set, arma, escudo, elmo, bota, luva, calça, capa, acessório | 1 | uma instância por slot |
| Resto de Ori/Lac, poeira, joia | agrupa | drop lists; teto por item a confirmar |

O **core** já respeita isso: `ItemRules::max_stack` é consultado por `inventory::move_item` e
`inventory::deposit`, com testes. O que **não** existe é a implementação de produção do trait a
partir do `item_catalog` — hoje só há mocks de teste. Enquanto isso, o drop não tem teto vindo do
dado.

## Problema medido (2026-11-10)

`item_catalog` (2026-11-10): `stackable = true` em **368**, `false` em **1.320** e **nulo em
1.532** — e essas 1.532 são exatamente as linhas **sem `kind`**, cuja amostra traz `item_index`
1/6/7/8/9/11/16/17 com nome `TransKnight`/`Foema` (nome de classe). É **ruído de importação**,
não item à espera de classificação: a primeira tarefa é classificar ou descartar essas linhas.

Segundo problema: o `EF_AMOUNT` só aparece em **94** linhas e **não** está em `effects[0]` — em
poção, `effects[0]` é a cura (50/200/300). O teto de 120 da poção vem do `BaseMob`, não do
catálogo agregado. Derivar de `effects` produziria teto 50/200/300.

## Escopo

1. Derivar `stackable`/`max_stack` de `client/InitItem.bin` e `client/ItemList.bin`
   (`docs/cliente/tabelas-bin.md`), com o `EF_AMOUNT` como teto e `confidence`/`classification_source`
   preenchidos — sem editar artefato gerado à mão, pelo pipeline de `tools/`.
2. Regra explícita para equipamento: `kind in (weapon, shield, armor, helmet, boots, gloves, cloak,
   accessory)` **força** `max_stack = 1` mesmo que o binário diga outra coisa (refino/Anct/gema
   distinguem instâncias).
3. Coluna `max_stack` no catálogo (ou derivar no carregamento) — sem número mágico no core.
4. Teste de catálogo: nenhuma linha com `stackable = true` e `max_stack = 1`; nenhuma linha de
   equipamento com `max_stack > 1`; nulo proibido após a migração.

## DoD

- Migração + seed regenerados; `stackable` nulo = 0 no relatório.
- Testes acima verdes e `make check` verde.
- `docs/wyd/itens.md` §5/§8 e `docs/wyd/sistemas/drops.md` §8 com os números finais.

## Risco

Binário pode não carregar a flag de aglutinação (só o `EF_AMOUNT` do template). Se não carregar,
a classificação vem do `kind` + lista explícita revisada por item, e o `EF_AMOUNT` continua sendo
o teto.
