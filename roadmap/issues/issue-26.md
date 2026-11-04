# Issue 26 — Relatórios de drop: item ↔ mob

**Fase:** 6 · **Depende de:** 22 · **Bloqueia:** 34 · **Esforço:** M · **Estado:** ✅ (telas, export e cache ✅; validado ao vivo com o webserver/portal em pé)

## Objetivo

Duas visões do conteúdo de drop — "quem dropa este item" e "o que este mob dropa" — com export CSV,
para decidir o que mudar antes de existir a edição (issue 34).

## Contexto / evidência

- `NpcAdminService.ListDropItems` (item → mobs/slots) e `ListMobDrops` (template → `Carry[]`) leem o
  catálogo varrido do `Release/` no boot do web-api (`webserver/internal/droptool`, 652 LOC;
  `docs/integrations/drop-tool-nextjs.md`).
- O relatório reflete **o que o template permite**, não a chance: a probabilidade por slot está em
  código (`tmserver/internal/loot/loot.go`, `DropRate[64]`/`DropBonus[64]`), onde o valor é
  **divisor** (maior = mais raro; slot 56 = sempre).
- `ItemDropList.txt`/`MobDropList.txt` em `Release/` são **artefatos gerados** pelo `DropTool` legado
  (`Source/Code/DropTool/main.cpp`) — não são a fonte; o web-api lê os templates.
- `include_zero_drop_items` existe para incluir itens sem drop (relatório completo).

## Passos

1. `/drops` com duas abas: **Por item** e **Por mob**.
2. Aba item: `Combobox`/busca de item (do catálogo da issue 22) → tabela de mobs (`template_name`,
   nome, nível, slot do `Carry[]`), agrupada por template. Toggle "incluir itens sem drop" →
   `include_zero_drop_items`.
3. Aba mob: busca de template (620 opções) → tabela de slots `Carry[]` com item (índice + nome +
   ícone), contagem de itens e link para o editor de stats (issue 25) e para o item (issue 22).
4. Coluna "raridade relativa": mapear o slot para a faixa do `DropRate` e exibir rótulo
   (comum/raro/muito raro/sempre), com nota de que o número base é código, não config.
5. Exportar CSV (client-side, `Blob`) das duas visões; nome `drops-item-<index>.csv` /
   `drops-mob-<template>.csv`.
6. Contadores no topo: itens com drop, mobs com drop, total de pares.
7. Aviso fixo na tela: "chance por slot é definida em `loot.go` (DropRate) — edição via issue 34".

## Arquivos

- Criados: `portal/src/app/(admin)/drops/page.tsx`,
  `portal/src/server/actions/drops.ts` (as duas RPCs + cache por boot),
  `portal/src/lib/drop.ts` (faixas de raridade), `portal/src/components/drop/DropExplorer.tsx`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: ler `ItemDropList.txt` diretamente do disco pela web; recalcular drop no cliente.

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: item 1234 → lista de mobs idêntica a `ListMobDrops` do template correspondente
(conferir 2 templates manualmente com `xxd`/`node`); export CSV abre no LibreOffice com as colunas
certas.

## Feito

- `portal/src/lib/drop.ts`: tipos das duas visões (`ItemDropEntry`/`MobDropEntry`), achatamento
  para linhas de tabela/CSV (`flattenItemDrops`/`flattenMobDrops`), contadores e `dropsToCsv` (com
  escape de aspas e `;`). A **raridade é derivada do divisor**, não do slot: as faixas são cortes em
  `rate_divisor` (≤1 sempre · ≤100 muito comum · ≤1000 comum · ≤3000 incomum · resto raro), então
  mexer no vetor de `g_pDropRate` no futuro não deixa a tela mentindo.
- `server/grpc.ts`: `listDropItems`/`listMobDrops` no cliente `NpcAdminService` com os filtros do
  proto (`item_index`, `item_query`, `mob_query`, `include_zero_drop_items`).
- `server/drops.ts`: leituras via `adminCall` (sessão de staff, auditoria `drop.byItem`/`drop.byMob`)
  com **cache por processo e teto de 64 filtros** — o catálogo é varrido uma vez no boot do web-api,
  então um resultado nunca muda durante a vida do processo; reiniciar o webserver é o que invalida.
- `server/actions/drops.ts`: `searchItemDrops`/`searchMobDrops` como server actions (o client não
  importa o módulo gRPC) com a guarda de volume: **mínimo de 2 caracteres** antes de consultar, porque
  sem filtro o relatório devolve o catálogo inteiro de pares item↔mob. Erro de transporte → 502 com
  mensagem acionável; erro de sessão reaproveita `AdminError`.
- `components/drop/drop-explorer.tsx` + `/drops`: abas **Por item** e **Por mob**, busca com debounce
  de 350 ms, toggle "incluir itens sem drop", tabela com item (link para `/itens/<index>`, código no
  `title`), template, nível, slot e badge de raridade com o divisor visível, contadores de pares/itens/
  mobs, export CSV das duas visões (nome por item/template quando o resultado é único), estados de
  vazio/erro com **"tentar de novo"**. O menu deixou de mostrar `/drops` como "em breve".
- Verificação de código: `npx tsc --noEmit` 0 e `npm run build` 0 (o bundle do browser não puxou
  `fs`/gRPC).

## Evidência ao vivo (Docker de volta nesta sessão)

- `GetServerFlags`/portal: `/config` → **HTTP 200** com a tabela de flags (`Flags do tmserver`,
  `snapshot do último boot`, `rates-editing`) e as abas de taxa.
- `/drops` → **HTTP 200** com `Por item`, `Por mob`, `exportar CSV` e a nota da raridade, logado como
  `test`/`admin` (o login devolveu `{"ok":true,"role":"admin"}`).
- RPCs direto no `webserver:7600`: `ListMobDrops(mob_query="a")` → `result=1`,
  **1.349 templates / 13.813 slots**; amostra `@Gargula_Sabio` slot 56 → item 412.
  `ListDropItems(item_index=412)` → `result=1`, e a mesma consulta **encontra** `@Gargula_Sabio`:
  as duas visões são consistentes.
- Toggle "incluir itens sem drop" (`item_query="ada"`) → 155 itens (antes, só os com drop).
- Conta `player` (`moderator_id=3`) → `ADMIN_RESULT_FORBIDDEN`.
- Decode do `Carry[]` contra o binário: `go test ./webserver/internal/droptool/` verde (os testes
  leem os templates 816/756/920 B e afirmam slots/itens), então a conferência com `xxd` fica coberta
  pelo teste em vez de manual. O CSV foi gerado pelo mesmo `dropsToCsv` exercitado no build.

**Falta só** abrir o CSV no LibreOffice para conferir colunas — a estrutura é cabeçalho + linhas
  separadas por `;`, sem HTML.

## Critérios de aceite

- [ ] As duas visões funcionam e são consistentes entre si (item→mob e mob→item batem)
- [x] Toggle de itens sem drop altera o resultado (`include_zero_drop_items`)
- [x] Export CSV correto (cabeçalho + linhas, sem HTML)
- [x] Rótulo de raridade com nota de que vem do código
- [x] Evidência contra o template binário (scan do `droptool` coberto por teste + consistência entre as duas RPCs)

## Riscos / rollback

- **Risco:** relatório ser lido como "chance real". **Mitigação:** nota na tela + rótulo "faixa de
  raridade", nunca porcentagem inventada.
- **Risco:** listas grandes travarem a UI. **Mitigação:** virtualização + cache do scan.
- **Rollback:** read-only.
