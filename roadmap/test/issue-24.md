# Issue 24 — Preços globais de itens

**Fase:** 6 · **Depende de:** 23 · **Bloqueia:** — · **Esforço:** S · **Estado:** 🟡 UI ✅, rota HTTP ✅ (05/11) — falta a evidência in-game e o cache por `catalog_version`

## Objetivo

Editar o **preço global** de qualquer item numa tela só, ver quais itens têm override e voltar ao
preço do `ItemList.csv` com um clique.

## Contexto / evidência

- `NpcAdminService.SetItemPrice(item_index, price)`: `price >= 0` define, `price < 0` **limpa** o
  override (volta ao preço do catálogo). `ListItemPrices` retorna só os itens com override — item
  ausente usa a base do conteúdo (`web.proto:220-250`).
- O preço é **global**, não por loja: `AdminNpcShopItem` não carrega preço (comentário no proto).
- O catálogo tem 3.220 itens; a tela precisa de busca por nome e por índice.

## Passos

1. `/precos` (ou aba dentro de `/itens`, issue 22 — decidir e registrar): `DataTable` com `item_index`,
   nome, ícone, preço base do catálogo, preço atual, "tem override" e ação.
2. Botão "novo preço": `Combobox` de item (busca por nome/índice) + input de preço (inteiro ≥ 0,
   hint do preço base) + salvar via `adminCall`.
3. Ação "remover override" → `SetItemPrice(itemIndex, -1)`, com confirmação inline.
4. Filtro "só com override" (usa `ListItemPrices` como fonte primária; junta com `ListItems` para
   nome/ícone).
5. Mostrar contagem e um aviso: "preço global — afeta todas as lojas que não fixam preço".
6. Otimização: cachear o catálogo no server (`unstable_cache`/`fetch` memo com tag
   `catalog:<catalog_version>`) para não baixar 3.2k itens a cada visita.

## Arquivos

- Criados: `portal/src/app/(admin)/precos/page.tsx`,
  `portal/src/server/actions/prices.ts`, `portal/src/lib/price.ts`.
- Alterados: `portal/src/components/nav.ts`.
- Proibido: editar `Release/Common/ItemList.csv` (conteúdo é read-only; a issue 33 trata de catálogo
  editável via overlay).

## Testes / verificação

```bash
cd portal && npx tsc --noEmit && npm run build
```

Evidência: definir 1500000 para o item 1234, abrir a loja no jogo e ver o preço; remover o override e
ver o preço base voltar. Conferir no banco:
```bash
docker compose exec -T db psql -U postgres -d postgres -c \
  "select item_index, price from item_price where item_index=1234;"
```

## Critérios de aceite

- [x] Definir e limpar override funcionam; `ListItemPrices` reflete na hora
- [x] Preço base do catálogo visível ao lado do override
- [x] Busca por nome e por índice; item inexistente → `INVALID` tratado como 422
- [ ] Uma chamada de catálogo por sessão (cache por `catalog_version`), não por página
- [ ] Evidência no jogo + consulta SQL coladas

## Riscos / rollback

- **Risco:** confundir preço global com preço de loja. **Mitigação:** texto explícito e hint no campo;
  a loja (issue 23) não edita preço.
- **Rollback:** `SetItemPrice(idx, -1)` em todos os overrides listados.

## Evidência de execução (2026-11-03)

```text
$ npx tsc --noEmit    # tsc_exit=0
$ npm run build       # build_exit=0   ƒ /precos 5.17 kB / 121 kB first load

# probe /api/price-probe (rota temporária, removida depois)
SetItemPrice(1234, 1500000) → {ok:true}
  psql: select item_index, price from item_price where item_index=1234
        1234 | 1500000
SetItemPrice(1234, -1)      → {ok:true}
  psql: select count(*) … where item_index=1234 → 0
SetItemPrice(999999, 1500)  → {ok:true}     ← ver "Pendente" abaixo
player (conta teste)        → {ok:false,status:403,"sua conta não tem permissão para esta ação"}

# páginas
/precos 200 ("Preços globais") · /precos sem cookie 307 · /itens 200
```

- Criados: `app/(admin)/precos/{page.tsx,price-table.tsx}`, `server/prices.ts`,
  `server/actions/prices.ts`, `lib/price.ts`. Alterados: `server/grpc.ts` (`ListItemPrices`/
  `SetItemPrice`), `components/{nav.ts,Icon.tsx}` (entrada e ícone `tag`), `middleware.ts`,
  `api/search/items/route.ts` (hint com o preço base), `globals.css` (reuso de `.ui-banner`).
- **Decisão:** tela própria `/precos`, não aba de `/itens`. A tabela lista só quem **tem override**
  (`ListItemPrices` como fonte primária) e carrega o catálogo apenas para nome/ícone; item novo entra
  pelo Combobox de busca. Assim a página não paga 3,2 mil linhas para mostrar overrides.
- **Preço base não vem por RPC:** `ItemCatalogEntry` não carrega preço, então o base é lido da 6ª
  coluna do `Release/Common/ItemList.csv` — a mesma que `content.ItemList.Prices()` usa
  (`catalog.go:35`) — no host, com `unstable_cache` (15 min). Se o arquivo não estiver acessível, a
  tela avisa e continua editando overrides.
- **Pendente (critério 3):** o backend **aceita** `item_index` fora do catálogo —
  `SetItemPrice(999999, 1500)` devolveu OK e foi limpo logo depois. A UI só oferece item do picker,
  mas o "422 para item inexistente" que a issue previa não existe do lado do servidor.
- **Pendente (critério 4):** não contei chamadas ao web-api; o cache é por tag (`item-base-prices`) e
  o catálogo por `item-catalog`, não por `catalog_version` como a issue pedia.- **Pendente (critério 5):** evidência no jogo não feita — exige `W2PP_NPC_EDITING=true` no tmserver e
a loja aberta no cliente; a consulta SQL está colada acima.

## Evidência de execução (2026-11-05) — rota HTTP na imagem nova

Stack reconstruída do `HEAD` (`docs/analysis/deploy-2026-11-05.md`). O cookie sai `Secure`
(`NODE_ENV=production`), então o token do `set-cookie` vai no header `Cookie:` — com `curl -b` ele não
volta por `http://`.

```text
POST /api/items/price {"itemIndex":907,"price":12345} -> 200 {"ok":true}
  sql: select item_index, price from item_price where item_index=907  ->  907 | 12345
POST /api/items/price {"itemIndex":907,"price":-1}    -> 200 {"ok":true}
  sql: select count(*) from item_price where item_index=907           ->  0
POST /api/items/price {"itemIndex":999999,"price":100} -> 422 {"ok":false,"error":"dados inválidos para esta ação","status":422}
POST /api/items/price {"itemIndex":-3,"price":-9}      -> 422 {"error":"índice de item inválido"}
POST /api/items/price  (sem sessão)                    -> 401 {"error":"sessão expirada: entre novamente"}

GET /precos -> 200 (22.234 B)
GET /api/search/items?q=espada -> 200 option #907 "Espada Bastarda"
```

- **Resolvido:** o 422 para item inexistente que a issue previa agora **existe** (validação
  `itemCatalog.Has` em `npcadmin.SetItemPrice`), e a rota `POST /api/items/price` é o caminho
  verificável sem navegador.
- **Continua pendente:** cache por `catalog_version` (critério 4) e o preço visto na loja in-game
  (critério 5).
