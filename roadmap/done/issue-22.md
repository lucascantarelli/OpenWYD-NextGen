# Issue 22 — Catálogo de itens com ícones (leitura)

**Fase:** 6 · **Depende de:** 20 · **Bloqueia:** 24, 33 · **Esforço:** M · **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Uma tela de catálogo com os 3.220 itens do `ItemList.csv`: busca por nome/índice, filtro por slot e
grade, ícone real vindo do pack, e detalhe do item — tudo somente leitura.

## Contexto / evidência

- `ItemCatalogService.ListItems` é **público** (sem `moderator_id`): o catálogo é conteúdo imutável
  lido do `Release/` no boot (`web.proto:185-205`). Resposta inclui `catalog_version` e
  `icon_pack_version`.
- `ItemCatalogEntry`: `item_index, name, icon_key, display_name, slot_mask, slots[], grade, mesh,
  texture, icon_url` (`docs/integrations/item-icons-nextjs.md`).
- `icon_key` é **opaco** ("iNNNN"); `icon_url` é a URL pública do storage-manager. Nunca concatenar
  base+versão+chave no front.
- O pack é gerado por `webserver/cmd/itemiconpack -client <dir> -out dist/item-icons -content Release`
  a partir de `itemicon.bin` + `UI/itemiconNN.wyt` do **cliente local**; o web-api o carrega via
  `W2PP_ITEM_ICONS_MANIFEST`.
- Sem manifesto, o serviço sobe em "fallback-only" (log `item icon manifest not configured`) — a UI
  não pode quebrar por isso.

## Passos

1. `src/server/catalog.ts`: `getCatalog()` chamando `ListItems` uma vez e memoizando
   (`unstable_cache` com tag `catalog:<catalog_version>`), devolvendo
   `{ items, byIndex: Map, catalogVersion, iconPackVersion }`.
2. `/itens`: `DataTable` virtualizada (3.2k linhas) com colunas ícone (32×32), `#`, nome, display,
   slots, grade. Busca com debounce 150ms em nome/display/índice; filtros de slot (multi) e grade.
3. `ItemIcon` conforme o guia: começa no fallback (iniciais do slot), troca para a imagem quando
   carrega, `onError` mantém o fallback **sem loop**; `alt=""` + nome acessível no `<td>`.
4. `/itens/[index]`: painel de detalhe (drawer) com todos os campos, `mesh`/`texture`/`slot_mask`
   crus (para conferência) e links: "ver preço" (issue 24), "quem dropa" (issue 26),
   "usar em loja" (issue 23).
5. Ordenação por índice (default), nome, grade. Paginação 100/200/500.
6. Estado vazio explícito quando `items.length === 0` (web-api sem `-content`): explicar que o
   serviço está sem catálogo e como subir (`-content /Release`).

## Arquivos

- Criados: `portal/src/app/(admin)/itens/page.tsx`,
  `portal/src/app/(admin)/itens/[index]/page.tsx`,
  `portal/src/server/catalog.ts`, `portal/src/components/ItemIcon.tsx`, `portal/src/lib/item.ts`.
- Alterados: `portal/src/components/nav.ts` (tirar "em breve" de `/itens`), `portal/.env.example`
  (comentário sobre `W2PP_ITEM_ICONS_MANIFEST`).
- Proibido: gerar/baixar ícones em runtime; reescrever `icon_key`.

## Testes / verificação

```bash
# gerar o pack a partir do cliente local (uma vez)
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2:/w" -w /w/OpenWYD golang:1.26-alpine \
  go run ./webserver/cmd/itemiconpack -client ./client -out dist/item-icons -content Release
# subir o web-api com o manifesto e reiniciar
W2PP_ITEM_ICONS_MANIFEST=/Release/../dist/item-icons/manifest.json docker compose up -d webserver
docker compose logs --tail 10 webserver | grep -i "item icon"
```

```bash
cd portal && npx tsc --noEmit && npm run build
```

Saída esperada no log: `loaded item icon manifest version=… mapped=… icons=…`; na tela, item 1 com
ícone e nome. Com o manifesto ausente, a tela abre com fallback e nenhum erro no console.

## Critérios de aceite

- [x] 3.220 itens listados; contagem igual ao log `items=3220` do tmserver
- [ ] Busca por nome e por índice em < 100ms numa lista já carregada
- [ ] Ícone real quando há `icon_url`; fallback estável quando não há
- [ ] Uma única chamada `ListItems` por versão de catálogo (cache por tag)
- [x] Detalhe do item navegável por link direto (`/itens/1234`)
- [ ] Estado "sem catálogo" explicado, sem stack trace

## Riscos / rollback

- **Risco:** 3.2k ícones travarem a primeira renderização. **Mitigação:** `loading="lazy"`, largura e
  altura fixas, virtualização.
- **Risco:** `icon_url` expirada/404. **Mitigação:** fallback no `onError` + contador de falhas no
  console do servidor (não no cliente).
- **Rollback:** rota é read-only; remover a página não afeta o servidor.

## Evidência de execução (2026-11-03)

```text
$ npx tsc --noEmit    # tsc_exit=0
$ npm run build       # build_exit=0
  ƒ /itens         3.08 kB / 106 kB first load
  ƒ /itens/[index]  933 B  / 107 kB first load

$ curl /itens                      (sem cookie)  307 → /login
$ curl /itens                      (admin)       200  987.769 bytes
  "3.220 itens" · "ItemBrowser" · "pack de ícones ausente (fallback)"
$ curl /itens/1                    (admin)       200  campos #, slot_mask, mesh, texture, icon_key
$ curl /itens/999999               (admin)       404
$ curl /painel | grep href="/itens"               link ativo (sem "em breve")
```

- Criados: `app/(admin)/itens/page.tsx`, `app/(admin)/itens/item-browser.tsx`,
  `app/(admin)/itens/[index]/page.tsx`, `components/ItemIcon.tsx`, `lib/item.ts`.
  Alterados: `server/catalog.ts` (`getCatalog`/`getCatalogItem` + `unstable_cache`),
  `components/nav.ts` (`/itens` deixa de ser "em breve"), `globals.css` (classes do catálogo),
  `.env.example` (`AUDIT_LOG_DIR`, `W2PP_ITEM_ICONS_MANIFEST`).
- **Desvio 1:** o detalhe é **rota própria** (`/itens/1234`), não um drawer sobre a lista. A rota é
  linkável e sobrevive a reload — o critério de "link direto" é o que importa aqui; um drawer exigiria
  carregar a lista inteira para abrir por cima.
- **Desvio 2:** não há biblioteca de virtualização. A tabela pagina em 100/200/500 e só a página
  visível entra no DOM (100 linhas), o que resolve o mesmo problema sem dependência nova. O custo é o
  **payload**: 987 KB de HTML/RSC para levar os 3,2 mil itens ao filtro client-side (medido acima).
  Se incomodar, o próximo passo é filtrar no servidor com `searchParams`.
- **Desvio 3:** `unstable_cache` guarda a resposta serializada de `ListItems` na tag
  `item-catalog` (5 min); o índice `byIndex` é derivado depois porque `Map` não serializa. Não deu
  para contar chamadas ao web-api no log para provar "uma por versão" — o cache é por tag, não por
  `catalog_version` como a issue pedia.
- **Pendente:** o pack de ícones não existe nesta máquina (`dist/item-icons/` ausente e a
  pasta do cliente não está no workspace), então `icon_url` vem vazio e a tela mostrou **fallback**
  (sigla + borda por grau). O caminho da imagem está implementado conforme o guia (overlay, `onError`
  sem loop), mas não foi visto com ícone real. Comando para gerar segue na issue.
- **Pendente:** busca "< 100 ms" não foi cronometrada em browser; os critérios correspondentes
  seguem desmarcados, junto de "estado sem catálogo" (implementado, mas exigiria subir o
  `webserver` sem `-content` para exercitar).
