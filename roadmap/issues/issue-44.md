# Issue 44 — Adds e efeitos de item (base, instância, refino e combine)

**Fase:** 6 · **Depende de:** 25, 33 · **Bloqueia:** — · **Esforço:** L · **Estado:** 🟡 (leitura ✅ + overlay de efeitos base ✅ + read-through no tmServer ✅; refino/combine pendentes)

## Feito — leitura de efeitos (fatia desta rodada)

- `OS` pares `EF_<nome>,valor` do `ItemList.csv` passaram a ser parseados no `itemcatalog`
  (`parseEffects`), resolvidos para o código de `ItemEffect.h` via `internal/itemeffects.Code`.
  Token fora do header é descartado, como o loader legado faz.
- `ItemCatalogEntry.effects` (código, rótulo em português, valor) e `ListItemsResponse.effect_types`
  (o vocabulário completo, 108 tipos) expostos pelo `ItemCatalogService` — a UI não reimplementa a
  tabela em TypeScript.
- `/itens/[index]` ganhou o card **Efeitos base** lendo esses pares.
- O seletor de adds da concessão (issue 43) deixou de pedir números: cada par é um `select` com
  `rótulo (código)`.
- **Validação**: `characteradmin.ValidateAdds` recusa código fora do `ItemEffect.h` (ex.: 98, no vão
  entre 91 e 100) antes de enfileirar — o cliente não interpretaria o add.

## Feito — overlay de efeitos base (escrita)

- Migration `0036_item_effect_override`: `item_effect_override(item_index, effect, value, updated_by,
  updated_at)` + `item_effect_audit(before/after/reason)`.
- **A chave é o código do efeito, não um slot 1..3.** O `ItemList.csv` real traz mais de três pares por
  item (a 1188 — Botas Douradas — tem seis: `EF_CLASS`, `EF_GRID`, `EF_AC`, `EF_RUNSPEED`,
  `EF_REGENMP`, `EF_ITEMLEVEL`) e `content.BaseEffects()` indexa por código. Um overlay posicional
  ignoraria metade dos adds e desalinharia os demais.
- `ItemEffectAdminService` (`ListEffects`, `UpsertEffect`, `DeleteEffect`) com `moderator_id` da sessão,
  `AdminResult` no corpo e `message` explicando a recusa.
- Validação: código precisa existir em `ItemEffect.h` (`itemeffects.Known`) e valor caber num short. Um
  código que o conteúdo não declara **pode** ser concedido (é assim que se dá um add novo); um código
  fora do header não.
- UI: `/itens/[index]` ganhou o editor — uma linha por efeito com valor, `salvar`, `voltar ao conteúdo`
  e motivo; e uma linha "adicionar efeito" limitada aos códigos que o item ainda não tem.
- Auditoria: `upsert` grava `before`/`after` lidos da própria tabela (não do que o cliente mandou);
  `delete` grava `after = NULL`, que é como se distingue "mudei" de "removi".

## Feito — read-through no tmServer (flag `-item-effect-editing`)

Com a flag desligada (default) o catálogo é exatamente o `ItemList.csv`; ligada, o jogo passa a ver o
que o portal salvou.

- `api/db/v1/db.proto`: `ItemEffectConfigService.ListItemEffectOverrides` (leitura pura) +
  `ItemEffectOverride{item_index, effect, value}` — o tmServer nunca escreve config.
- `dbserver/internal/grpcsrv/itemeffectconfig.go` sobre `store.ListItemEffectOverrides(ctx, 0)`
  (0 = todas as linhas), registrado em `dbserver/cmd/dbserver/main.go`.
- `tmserver/internal/dbclient/itemeffectconfig.go`: cliente de leitura, espelho do de teleporte.
- `tmserver/internal/content/effectoverlay.go`: `ApplyItemEffectOverrides` dobra as linhas no
  `map[int][]BaseEffect` — valor trocado quando o item já declara o código, par anexado quando não, e
  **código fora do allowlist de `BaseEffects` recusado e contado**: `EF_RANGE` e companhia têm leitor
  próprio, então injetá-los aqui prometeria um bônus que o cálculo de score nunca aplica. Copy-on-write:
  o mapa de origem não é mutado, e overlay vazio devolve o catálogo idêntico ao do CSV.
- `tmserver/cmd/tmserver/main.go`: `-item-effect-editing` / `W2PP_ITEM_EFFECT_EDITING` (default false,
  exige `-dbserver` e `-content`), aplicado **uma vez** logo antes de `handler.New`, como
  `-mob-stat-editing`: `ItemEffects` é snapshot, então editar no portal exige restart do tmServer.
  Precedência intacta — efeito da instância (refino) &gt; catálogo + overlay.
- Verificação feita sem tocar o servidor ao vivo: container one-off da imagem `openwyd-tmserver` na rede
  do compose em `:8282`/`:8088`. Com uma linha no overlay (`1188`/`EF_AC`=60):
  `item effect overlay applied rows=1 replaced=1 added=0 ignored=0`; com o overlay vazio:
  `rows=0 replaced=0 added=0 ignored=0`; sem a flag: nenhuma linha e catálogo do CSV. `go test` verde em
  `tmserver/internal/content` (dobra + imutabilidade), `tmserver/internal/dbclient` e
  `dbserver/internal/grpcsrv`.

Falta: `refine_rule`/`combine_recipe` e o simulador de refino com o RNG do MSVC.

## Objetivo

Editar os "adds" dos itens de ponta a ponta: os **efeitos base** do catálogo (pares `EF_*` do
`ItemList.csv`), os **efeitos de instância** (os 3 pares `stEffect/stValue` de um item de jogador), os
**limites/custos de refino** e as **receitas de combine** — com simulador antes de salvar. Sem isso, a
UI edita nome e preço do item, mas não o que ele *faz*.

## Contexto / evidência

- Efeitos base vêm do catálogo: `content.ItemList.BaseEffects()` (`catalog.go:107`) lê os pares
  `EF_<nome>,valor` ao final de cada linha do `ItemList.csv`; o mapeamento nome→id está em
  `tmserver/internal/handler`/`ItemEffect.h`.
- Efeitos de instância: `STRUCT_ITEM` = 8 B (`sIndex`, `stEffect[3]`, `sValue[3]`) — é o "+10 crítico"
  que o jogador vê; hoje só entra por drop/refino/concessão.
- Refino: `tmserver/internal/refine` (cap e custo no código) + `SancRate.txt` (chance, issue 36).
- Combine: `tmserver/internal/combine` (`combine.go`, `odin.go`, `match.go`) — receitas para
  Arch/Anct/Odin; `CompRate.txt` dá a chance.
- `extraitem.csv` tem **1 item** fora do catálogo (linha `204`), usado por efeitos especiais.

## Passos

1. Migration `0014_item_effect`: `item_effect(item_index, slot 1..3, eff, val, updated_by, updated_at)`;
   `refine_rule(level, success_rate, cost, max_level)`; `combine_recipe(id, kind, inputs jsonb,
   output_item, output_eff jsonb, rate, enabled)`. Seeds a partir do conteúdo atual.
2. `web.proto` + `webserver/internal/itemeffectadmin`: `ItemEffectAdminService` com
   `ListItemEffects(item_index)`, `UpsertItemEffect`, `DeleteItemEffect`, `ListRefineRules`,
   `UpsertRefineRule`, `ListCombineRecipes`, `UpsertCombineRecipe`, `SimulateRefine(item, target)`
   (10k rolas no RNG MSVC, devolve distribuição e custo esperado).
3. Validação de efeito: contra a tabela `EF_*` conhecida (nome→id e faixa de valor por efeito);
   efeito desconhecido → `INVALID` com o nome rejeitado. Nunca aceitar `EF_AMOUNT` (61) — é derivado da
   quantidade.
4. tmserver: `W2PP_ITEM_EFFECT_EDITING=true` aplica o overlay de efeitos base/refino/combine no boot e
   no poll, mantendo a precedência: instância > overlay > catálogo.
5. UI: aba **Efeitos** em `/itens/[index]` — 3 pares com Combobox de efeito (nome + id), valor,
   preview do tooltip como o cliente mostra e aviso de que vale para todos os itens daquele índice.
   Tela `/refino` com tabela de chance/custo por nível, curva e simulador (alvo +9/+12, mostra
   tentativas esperadas e custo). Aba **Combine** listando receitas com edição e teste.
6. Concessão de item com adds: campos de efeito no `GrantDialog` da issue 43 (mesma validação).

## Arquivos

- Criados: `internal/migrations/0014_item_effect.*.sql`,
  `webserver/internal/itemeffectadmin/*.go`, `portal/src/app/(admin)/itens/[index]/effects.tsx`,
  `portal/src/app/(admin)/refino/{page.tsx,refine-table.tsx}`,
  `portal/src/server/item-effects.ts`, `portal/src/server/actions/item-effects.ts`,
  `portal/src/lib/item-effect.ts`, `portal/src/components/item/EffectPicker.tsx`.
- Alterados: `api/web/v1/web.proto`, `tmserver/internal/{content,refine,combine}` (read-through),
  `portal/src/components/{nav.ts,Icon.tsx}`, `portal/src/middleware.ts`.
- Proibido: editar `extraitem.bin`/`ItemEffect.h` pelo portal; quebrar a ordem de RNG do refino.

## Testes / verificação

```bash
go test ./tmserver/internal/refine/... ./tmserver/internal/combine/... -v
cd portal && npx tsc --noEmit && npm run build
```

Evidência: adicionar `EF_CRITICAL 10` ao item 1234 na aba Efeitos, reiniciar o tmserver e ver o bônus no
item no cliente; simular refino +9 e comparar com o log real de 10 refinos; editar uma receita de
combine e ver o resultado no jogo.

## Critérios de aceite

- [ ] Efeitos base editáveis por item (3 pares) com validação contra a tabela `EF_*`
- [ ] Efeito inválido → 422 com o nome do efeito rejeitado
- [ ] Regras de refino (chance, custo, cap) editáveis e simulador batendo com as taxas salvas
- [ ] Receitas de combine listadas e editáveis, refletindo no jogo
- [ ] Concessão de item com adds pela 43 usa a mesma validação
- [ ] `go test` de refine/combine verde com overlay vazio (paridade)

## Riscos / rollback

- **Risco:** efeito incoerente quebra o cliente (efeito fora de faixa). **Mitigação:** allowlist do
  `ItemEffect.h` + faixa por efeito + preview do tooltip.
- **Risco:** mudar o RNG do refino muda resultados históricos. **Mitigação:** overlay só altera taxa
  configurável, nunca a ordem das chamadas; simulador usa a mesma sequência do servidor.
- **Rollback:** deletar as linhas do overlay volta ao catálogo/código.
