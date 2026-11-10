# FASE-03-15 — Baldes do catálogo, teto por item e o que a classificação revelou

**Status:** entrega concluída em 2026-11-10 (dados); correções de catálogo pendentes.
**Depende de:** FASE-03-12 (aglutinação e catálogo).
**Fecha o pré-requisito de:** FASE-03-11 (drop ponta a ponta).

## Problema

1.532 linhas de `item_catalog` estavam sem `kind` — e também **sem `stackable` e sem `EF_AMOUNT`**.
A regra conservadora "sem classificação ⇒ teto 1" fazia item real não empilhar: `416 Antídoto`
(pack de 5), `419 Resto_de_Oriharucon`, `420 Resto_de_Lactolerium`, rações `3373…`. A receita que
consome 10 restos não conseguiria formar pilha, e o drop nunca empilharia esses itens.

Além disso, a extração de `EF_AMOUNT` usada antes era inválida: `item_catalog.fields` é **array
plano** `[índice, nome, peso, "0.0.0.0.0", flag, preço, 0, 0, 0, "EF_*", valor, …]`, não objeto.
`fields->>'EF_AMOUNT'` devolve NULL em toda linha; a leitura correta é pelo **par** rótulo→valor.

## Entrega

`crates/wyd-db/migrations/0039_classificacao_e_teto.sql` cria duas views derivadas (sem dado
materializado, sem risco de ficar velhas):

- **`item_bucket`** — cada linha em `ruido` / `real` / `placeholder`, com `reason` auditável;
- **`item_stack_cap`** — `cap` (120 ou 1), `contained` (`EF_AMOUNT`, conteúdo de pack) e `source`.

Critério (nenhum item real é rebaixado sem evidência):

| Balde | Regra |
|---|---|
| `placeholder` | `name_fold` em `noname` / `not used` / `n a` |
| `ruido` | nome igual a NPC, região, classe ou template de monstro **e** preço 0 **e** `unique_id` 0 **e** ausência em toda tabela que consome item |
| `real` | todo o resto — inclusive as 310 linhas sem uso conhecido |

Teto: `stackable` verdadeiro ⇒ 120; falso ⇒ 1; ruído/placeholder ⇒ 1; `EF_AMOUNT > 0` ⇒ 120
(pack empilha); poeira/resto/fragmento ⇒ 120; resto ⇒ 1.

## Números reais (banco do compose, 2026-11-10)

```
bucket      | count        reason            | count
real        |  1414        preco-ou-unique   |  703
ruido       |    96        usado-em-tabela   |  401
placeholder |    22        sem-uso-conhecido |  310
                           nome-de-npc       |   72
                           nome-de-classe    |   24
                           nome-placeholder  |   22
```

```
source                   | cap | count
default:sem-prova        |   1 |  1359
catalogo:nao_aglutinavel |   1 |  1320
catalogo:aglutinavel     | 120 |   368
catalogo:ruido           |   1 |    96
legado:ef_amount         | 120 |    53
catalogo:placeholder     |   1 |    22
familia:material         | 120 |     2
```

Dos 1.532 sem `kind`: **55** em teto 120, **1.477** em teto 1. Dos 94 com `EF_AMOUNT`: 53 por
`legado:ef_amount`, 35 já `stackable`, 6 placeholder = 94.

## Código

- `wyd_db::PgBackend::item_stack_caps` lê a view (`StackCapRow`), sem SQL no tick.
- `wydd-core::systems::inventory::StackCaps::from_caps` copia `(índice, teto, conteúdo)`.
- `crates/wyd-db/tests/stack_caps.rs` (gated por `WYD_TEST_DSN`) prova no banco: poeira 120, resto
  120, antídoto 120/5, kit 120/10, arma 1, placeholder 1.

## Erros de catálogo revelados (não corrigidos por chute)

| Item | Problema | Ação |
|---|---|---|
| `400` / `406` | mesmo nome `Poção_de_Cura_M.`; 406 é mana (preço 400 = 401) | renomear pelo `BaseItem` do cliente |
| `416 Antídoto` | real, `EF_AMOUNT 5`, sem `kind`/`stackable` | classificar como `consumable` |
| `419` / `420 Resto_*` | reais e com preço, `fields` sem efeito | recuperar os `EF_*` do insumo |
| `3385 not_used` | placeholder com preço 50.000 e `EF_AMOUNT 20` | decidir: placeholder ou item real mal nomeado |
| `2433`–`2435 N/A` | placeholder com `EF_AMOUNT 5` | idem |
| `4045 NONAME` | placeholder com preço 4.000.000 | idem |

## Definition of Done

- [x] Três baldes fecham 1.532 com consulta reexecutável.
- [x] Lista de aglutináveis com fonte por item (`item_stack_cap.source`).
- [x] Consulta de produção lê `fields` como array e roda no banco real.
- [x] Docs: `docs/wyd/itens.md` §5.1–§5.3 e `docs/wyd/sistemas/aglutinacao.md` §2, §6–§8.
- [x] Gates: fmt, clippy, test, `git diff --check`.
- [ ] Renomear/reclassificar os 6 itens da tabela de erros (próxima issue).
