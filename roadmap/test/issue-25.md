# Issue 25 — Editor de stats e equip de mobs

**Fase:** 6 · **Depende de:** 20, 21 · **Bloqueia:** 26, 35 · **Esforço:** L · **Estado:** 🟡 (lista + editor de stats com read-through, salvar e reverter ✅, verificado ao vivo; faltam o editor de `Equip[]`, o badge de override na lista e a evidência in-game)

## Objetivo

Editar os stats de combate de qualquer um dos 620 templates de `Release/TMsrv/run/npc/` — nível, HP/MP,
atributos, EXP, coin, skills e `Equip[16]` — com read-through do arquivo bruto e reversão para o
original.

## Contexto / evidência

- `MobTemplateAdminService` é o sucessor do `EDITAPPMOB` legado e edita **qualquer** template
  (monstros inclusos), não só mercadores (`web.proto:470-508`).
- `GetMobTemplateStat` faz **read-through**: sem override salvo, devolve os valores reais do arquivo
  (`overridden=false`), para o editor abrir com o valor de partida em vez de zeros.
- `UpsertMobTemplateStat` grava o override completo; `SetMobTemplateEquip` **exige** override
  existente; `DeleteMobTemplateStat` remove o override (volta ao arquivo, nunca apaga o arquivo).
- Aplicação no jogo: `-mob-stat-editing` / `W2PP_MOB_STAT_EDITING`, **uma vez no boot**
  (`tmserver/cmd/tmserver/main.go:106`). Sem restart, nada muda.
- `BaseScore`/`CurrentScore` não são modelados separados: o tmserver espelha o mesmo valor nos dois.
- `Carry[]` **não** está aqui (é loja de NPC, issue 23) — por isso o drop do mob não se edita nesta
  tela (issue 34).

## Passos

1. `/mobs`: `DataTable` com `template_name`, `display_name`, merchant, e badge "com override".
   Busca por nome; ordenação por nome.
2. `/mobs/[template]`: form em seções colapsáveis, espelhando `AdminMobTemplateStat`:
   - **Identidade**: `display_name` (vazio = mantém o do arquivo), `class`, `clan`, `merchant`,
     `level`, `direction`.
   - **Combate**: `damage`, `ac`, `attack_run`, `chaos_rate`.
   - **Atributos**: `str`, `intel`, `dex`, `con`, `special1..4`.
   - **Vitalidade**: `max_hp`, `hp`, `max_mp`, `mp`, `regen_hp`, `regen_mp`.
   - **Progressão**: `exp` (int64 — formatar pt-BR e mostrar o valor original ao lado), `coin`,
     `spx`, `spy`, `score_bonus`, `learned_skill`, `skill_bar1..4`.
   - **Resistências**: `resist1..4`.
   - **Equip**: 16 slots (`AdminMobTemplateEquipItem`: `slot, item_index, eff1..effv3`), combobox de
     item do catálogo.
3. Cada seção mostra `original` vs `novo` quando `overridden=true` (diff visual), com hint de que o
   valor original é o do arquivo.
4. Botão "reverter ao arquivo" (Dialog de confirmação) → `DeleteMobTemplateStat`.
5. Salvar: uma chamada `UpsertMobTemplateStat` com o objeto completo; se houver equip alterado,
   `SetMobTemplateEquip` **depois** do upsert (dependência explícita).
6. Aviso fixo após salvar: "aplica no próximo restart do tmserver" + link para `/operacao`.
7. Validação: inteiros nos intervalos do struct (level 0..255, hp/mp ≥ 0, atributos 0..65535,
   exp ≥ 0 e ≤ 2^31−1 na UI antes de mandar int64); o servidor decide.
8. Filtro "somente com override" para auditar o que foi mexido.

## Arquivos

- Criados: `portal/src/app/(admin)/mobs/page.tsx`,
  `portal/src/app/(admin)/mobs/[template]/page.tsx`,
  `portal/src/server/actions/mobs.ts`, `portal/src/lib/mob.ts` (tipos + zod + grupos de campos),
  `portal/src/components/mob/EquipEditor.tsx`, `portal/src/components/mob/StatSection.tsx`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: gravar bytes em `Release/TMsrv/run/npc/*` (é read-only); editar `Carry[]` por aqui.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
W2PP_MOB_STAT_EDITING=true docker compose up -d tmserver
docker compose logs --tail 30 tmserver | grep -i "mob stat\|overlay"
```

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: abrir um template intocado e ver `overridden=false` com valores não-zero iguais ao arquivo
(conferir 2 campos com `xxd` no offset conhecido de `STRUCT_MOB`); alterar `exp`, salvar, reiniciar
tmserver e matar o mob no jogo para ver a EXP nova; reverter e conferir o valor antigo de volta.

## Feito

- `portal/src/lib/mob.ts`: tipos do `AdminMobTemplateStat`, schema zod com os **mesmos intervalos** da
  checagem do web-api (`mobTemplateStatInRange`: u8/i8/i16/u16 por campo) e os grupos de campos da
  ficha do `EDITAPPMOB` (identidade, combate, atributos, vitalidade, progressão, resistências).
  `changedFields` dá o diff da sessão; `toStatValues` projeta o stat para o proto.
- `server/grpc.ts` + `server/mobs.ts`: cliente `MobTemplateAdminService` e leituras `ListMobTemplates`
  e `GetMobTemplateStat` via `adminCall` (sessão de staff, auditoria `mob.templates`/`mob.get`).
- `server/actions/mobs.ts`: `saveMobStat` (valida no zod, **devolve o `Equip[]` carregado** porque o
  upsert substitui o override inteiro) e `revertMobStat` (remove o override).
- `/mobs` e `/mobs/[template]`: tabela com busca por template/nome e badge de merchant, e o editor com
  badge "override no banco" vs "valores do arquivo", campo de nome, seis seções numéricas, contador e
  destaque dos campos alterados na sessão, tabela de `Equip[]` (informativa, enviada intacta), salvar,
  "reverter ao arquivo" com confirmação, aviso de restart e `RetryButton` nos erros.
- `nav.ts`: `/mobs` deixou de estar "em breve".
- **Flag `overridden` na listagem**: `MobTemplateFile.overridden` (campo 4 do proto),
  `mobtemplates.File.Overridden` preenchido por `ListTemplates` numa única consulta
  (`ListMobTemplateStats`) e cópia do snapshot de boot — a listagem não muta o que foi escaneado, então
  um segundo request nunca vê flag velha. No portal, coluna "Override" com badge "no banco", contador
  no cabeçalho e checkbox "somente com override" (client-side: a lista inteira já vem numa chamada).

## Evidência ao vivo (stack no ar)

- `ListMobTemplates(1)` → `result=1`, **1.991 templates**, **656** com merchant (> 0).
- `GetMobTemplateStat("@@Gargula")` → `result=1`, `overridden=false` (read-through),
  `level=399`, `hp=14000/14000`, `exp=2990849`, `equip=2` — valores reais do arquivo, não zeros.
- `ListMobTemplates(3)` (conta `player`) → `ADMIN_RESULT_FORBIDDEN`.
- `GetMobTemplateStat("nao_existe_template_xyz")` → `ADMIN_RESULT_NOT_FOUND`.
- `/mobs` → **HTTP 200** com sessão de `test`/`admin`; `next build` com `/mobs` (2,24 kB) e
  `/mobs/[template]` (3,67 kB); `tsc --noEmit` e `npm test` (8/8) verdes.
- **Flag `overridden` ao vivo** (DB recém-migrado, sem overrides): `ListMobTemplates` → `overrides=[]`;
  `UpsertMobTemplateStat("@@Gargula")` → `result=1`, listagem passa a `overrides=[@@Gargula]` com
  `overridden=true`; `DeleteMobTemplateStat` → `result=1` e volta a `overrides=[]`/`false`. O portal
  renderiza "somente com override" em `/mobs` (HTTP 200).
- `go test ./webserver/internal/mobtemplateadmin/ ./webserver/internal/grpcsrv/ ./webserver/internal/mobtemplates/`
  verdes, incluindo `TestListTemplatesMarksOverrides` (flag por template e ausência de flag velha após
  delete).
- **Equip ao vivo**: com override criado, `SetMobTemplateEquip(slot 5, item 412)` → `result=1` e
  `GetMobTemplateStat` devolve `equip=[[5,412]]`; slot duplicado → `3` (INVALID), slot 16 → `3`,
  conta `player` → `2` (FORBIDDEN); `DeleteMobTemplateStat` limpa. `/mobs/@@Gargula` → **HTTP 200**
  com o editor (14 slots vazios "adicionar item" e o botão "salvar equip"); `next build` com
  `/mobs/[template]` em 5,53 kB.

## Pendente

- Evidência in-game (mudar `exp`/`Equip[]`, reiniciar com `W2PP_MOB_STAT_EDITING=true` e matar o mob).

## Editor de `Equip[]`

- `components/mob/equip-editor.tsx`: grade dos 16 slots com picker de item (`Combobox` sobre
  `/api/search/items`, o catálogo já cacheado), badge de diff (`novo/removido/trocado`), salvar e
  descartar. Trocar o item **zera os efeitos do slot** — `eff1/effv1…` descrevem o item anterior e
  herdá-los seria dado inventado; os efeitos aparecem, mas a edição deles é a issue 44.
- Sem override de stats o botão fica bloqueado e a tela explica: `SetMobTemplateEquip` exige a linha
  em `mob_template_stat`. A action também traduz o `NOT_FOUND` do web-api em "salve os stats do
  template antes de editar o equip" (HTTP 409 na UI).
- **`saveMobStat` deixou de aceitar equip do browser**: ele relê o equip do servidor e regrava junto.
  Antes, o formulário mandava a cópia que carregou — editar o equip e depois salvar os stats apagava a
  alteração (o upsert substitui o override inteiro).
- Nomes de item resolvidos no BFF (`getCatalog` por `catalog_version`); catálogo fora do ar não impede
  editar stats, o slot cai em `item #N`.

## Critérios de aceite

- [x] Read-through funciona: template sem override abre com os valores do arquivo
- [x] Badge "com override" na lista **e** filtro "somente com override" (o diff continua sendo o da
  sessão: o proto só devolve o override, nunca o valor do arquivo, e a tela diz isso)
- [x] `exp` tratado como int64 (sem perda de precisão no JSON)
- [x] Equip editável (`SetMobTemplateEquip`), com slot único validado duas vezes (zod + web-api) e
  stats relendo o equip do servidor no save
- [x] Reverter remove o override e o valor volta ao do arquivo
- [ ] Aviso de restart em toda escrita ✅ + evidência no jogo ⏳

## Riscos / rollback

- **Risco:** zerar stats de um mob por engano. **Mitigação:** `DeleteMobTemplateStat` sempre disponível
  + diff antes de salvar + auditoria com valores antigos/novos no log.
- **Risco:** confundir `merchant` do template com `merchant` do NPC (issue 23). **Mitigação:** label
  "merchant do template (não é o NPC)" e link cruzado.
- **Rollback:** reverter override de todos os templates alterados (`DeleteMobTemplateStat`).
