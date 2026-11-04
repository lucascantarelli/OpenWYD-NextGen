# Issue 33 — Catálogo de itens editável (overlay de `ItemList.csv`)

**Fase:** 6 · **Depende de:** 22, 32 · **Bloqueia:** — · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Editar nome de exibição e preço base de qualquer item pela web, guardando **overlay no Postgres** e
aplicando no boot do tmserver — `Release/Common/ItemList.csv` continua somente leitura.

## Contexto / evidência

- `ItemList.csv` é carregado por `content.LoadItemList` no boot
  (`tmserver/cmd/tmserver/main.go:635`) e é a fonte de nome, preço, `nUnique`, `nPos`, efeitos e
  refino (`tmserver/internal/content/catalog.go:12-59`). O `Release/` está montado `:ro`.
- O web-api **escaneia** o mesmo arquivo (`itemcatalog.Scan`) e serve `ListItems` read-only
  (`web.proto:186-205`) — hoje não há escrita nenhuma de catálogo.
- O padrão de escrita de conteúdo já existe e é o modelo: `npc_definition` (issue #?), 
  `mob_template_stat` (`-mob-stat-editing`), ambos lidos uma vez no boot.
- Preço **global** de item já tem caminho separado (`item_price` + `SetItemPrice`, issue 24). Esta
  issue cobre o **nome de exibição** e o **preço base do catálogo** (fallback quando não há override),
  não o override de preço.

## Passos

1. **Migração** `0035_item_catalog_override`:
   ```sql
   CREATE TABLE IF NOT EXISTS item_catalog_override (
     item_index   integer PRIMARY KEY CHECK (item_index > 0),
     display_name text    CHECK (display_name IS NULL OR char_length(display_name) BETWEEN 1 AND 40),
     base_price   integer CHECK (base_price IS NULL OR base_price BETWEEN 0 AND 1000000000),
     updated_by   bigint NOT NULL REFERENCES account(id),
     updated_at   timestamptz NOT NULL DEFAULT now()
   );
   CREATE TABLE IF NOT EXISTS item_catalog_audit (
     id bigserial PRIMARY KEY,
     actor_id bigint NOT NULL REFERENCES account(id),
     item_index integer NOT NULL,
     field text NOT NULL, old_value text, new_value text,
     created_at timestamptz NOT NULL DEFAULT now()
   );
   ```
   (`.down.sql` correspondente.)
2. **Proto**: `ItemAdminService` com `ListOverrides`, `UpsertOverride(item_index, display_name?,
   base_price?)`, `DeleteOverride(item_index)`; `moderator_id`, `AdminResult`.
3. **Store** `openwyd/webserver/internal/itemadmin`: upsert com auditoria por campo alterado; valida
   que `item_index` existe no catálogo varrido (senão `INVALID`).
4. **Aplicação no tmserver**: pacote leaf `tmserver/internal/itemcfg` + `Source` via dbServer
   (`GetItemCatalogOverrides`), aplicado **no boot** após `content.LoadItemList`, atrás de
   `-item-catalog-editing` / `W2PP_ITEM_CATALOG_EDITING` (default OFF, igual às outras).
5. **Aplicação no web-api**: `itemcatalog.Scan` passa a aplicar os overrides do banco antes de servir
   `ListItems`, para a UI e a loja verem o mesmo nome/preço que o jogo (senão a tela mente até o
   restart).
6. **Portal** `/itens/[index]`: botão "editar" abre form com `display_name` e `base_price`, mostra
   "valor do CSV" vs "valor atual", salvar/remover override; badge "override" na lista.
7. Aviso: "preço base é fallback; se houver override de preço (issue 24), ele vence".
8. **Não** permitir editar `nPos`, `nUnique`, efeitos, refino nem apagar item — fora de escopo por
   risco de quebrar paridade com o cliente (nome de item vem do `ItemList.csv`/cliente; mudar isso
   exige decisão de conteúdo, registrada em nova issue).

## Arquivos

- Criados: `internal/migrations/0035_item_catalog_override.{up,down}.sql`,
  `webserver/internal/itemadmin/{service.go,service_test.go}`,
  `tmserver/internal/itemcfg/itemcfg.go` (+ teste),
  `webserver/internal/grpcsrv/itemadmin.go`,
  `portal/src/app/(admin)/itens/[index]/edit.tsx`, `portal/src/server/actions/itemadmin.ts`.
- Alterados: `api/web/v1/web.proto`, `webserver/cmd/webserver/main.go`,
  `webserver/internal/itemcatalog/catalog.go`,
  `tmserver/cmd/tmserver/main.go`, `docker-compose.yaml` (flag opcional),
  `portal/src/app/(admin)/itens/page.tsx` (badge).
- Proibido: escrever `Release/Common/ItemList.csv`; mexer em `nUnique`/`nPos`.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test ./webserver/internal/itemadmin/ ./tmserver/internal/content/ ./tmserver/internal/itemcfg/"
```

```bash
# (comandos rodam na raiz do repositório)
W2PP_ITEM_CATALOG_EDITING=true docker compose up -d tmserver webserver
docker compose logs --tail 20 tmserver | grep -i "item catalog"
```

Evidência: renomear o item 1234 para "Espada de Teste" e preço 999999; a lista `/itens` mostra o novo
nome **sem** restart; reiniciar o tmserver e ver o nome novo na loja do jogo; remover o override e
voltar ao nome do CSV após o próximo boot.

## Critérios de aceite

- [ ] Overlay aplicado no boot do tmserver e refletido no web-api imediatamente
- [ ] Auditoria por campo alterado (quem, quando, de/para)
- [ ] `item_index` inexistente → `INVALID`; flag OFF = comportamento atual idêntico
- [ ] Nenhum arquivo de `Release/` alterado (`git status` limpo nessa árvore)
- [ ] Nome novo visível no cliente após restart; override removido volta ao CSV
- [ ] `go test` dos pacotes tocados verde

## Riscos / rollback

- **Risco:** nomes divergentes entre cliente e servidor confundirem jogadores. **Mitigação:** só
  `display_name` (o cliente exibe o nome do seu próprio `ItemList`); documentar que é cosmético do
  servidor e que o cliente pode continuar mostrando o nome antigo.
- **Risco:** flag ligada em produção sem revisão. **Mitigação:** default OFF + log `item catalog
  overrides applied count=N` no boot.
- **Rollback:** flag OFF e/ou `DeleteOverride` em lote; `.down.sql` remove as tabelas.
