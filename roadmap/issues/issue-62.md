# Issue 62 — webserver ainda lê os catálogos de arquivo (fase 5 incompleta)

**Fase:** 5 (resíduo) · **Depende de:** 58 · **Esforço:** M · **Estado:** ⏳ aberta

## Sintoma

A cada boot o `webserver` (web-api gRPC, `:7600`) emite quatro avisos procurando arquivos que não
existem mais:

```
level=WARN msg="npc template scan failed; merchant picker will be empty"
           err="npctemplates: npctemplate: read /Release/TMsrv/run/npc: no such file or directory"
level=WARN msg="item catalog scan failed; item picker will be empty"
           err="itemcatalog: open /Release/Common/ItemList.csv: no such file or directory"
level=WARN msg="mob template scan failed; template picker will be empty"
           err="mobtemplates: npctemplate: read /Release/TMsrv/run/npc: no such file or directory"
level=WARN msg="drop catalog scan failed; DropTool endpoints will be empty"
           err="droptool: item catalog: itemcatalog: open /Release/Common/ItemList.csv: ..."
```

`tmserver`, `dbserver`, `binserver`, `portal` e `logpipe`: **zero** ocorrências. O ruído é de um
serviço só.

## Causa

A fase 5 da issue 58 tirou `Common/ItemList.csv`, `Common/SkillData.csv` e `TMsrv/run/npc/*` do
repositório — o `tmServer` passou a buscar tudo no banco (`FetchItemCatalog`/`FetchSkillCatalog`/
`FetchNPCTemplates` via `tmserver/internal/dbclient`), mas o **webserver não foi migrado**. Ele ainda
roda o caminho antigo em `webserver/cmd/webserver/main.go:138-200`, sob `if *contentDir != ""`:

| Linha | Chamada | Lê | Resultado hoje |
|---:|---|---|---|
| 139 | `npctemplates.Scan` | `Release/TMsrv/run/npc/` | diretório ausente → **WARN** |
| 147 | `itemcatalog.Scan` | `Release/Common/ItemList.csv` | arquivo ausente → **WARN** |
| 167 | `skillcatalog.Scan` | `Release/Common/SkillData.csv` | nunca roda (aninhado no sucesso do item) |
| 176 | `mobtemplates.Scan` | `Release/TMsrv/run/npc/` | ausente → **WARN** |
| 191 | `droptool.LoadContentExclusions` + `Scan` | item catalog | ausente → **WARN** |
| 184 | `os.ReadFile(.../npc/<nome>)` | template por nome | falha em toda leitura |

O compose entrega `W2PP_CONTENT: /Release` e `command: ["-addr", ":7600", "-content", "/Release"]`
com `./Release:/Release:ro`, então o caminho nunca é pulado.

## Impacto (silencioso, porque é "degradação graciosa")

Sem catálogo, o portal perde — sem erro na tela:

- picker de item e de NPC mercador (sobra entrada manual por índice);
- picker de template de mob;
- nomes e ícones de item na ficha do personagem;
- nomes de skill (ficam como índice cru);
- overlay de efeito de item (mescla com o que o "CSV declara" — agora nada);
- validação do editor de `Carry[]` por template;
- endpoints do DropTool.

Isso explica a sensação de "a UI ficou mais pobre depois da fase 5" sem nenhum 500.

## Correção

O `webserver` **já tem** `store.New(pool)` em mãos e o `internal/store` já expõe o que falta:

- `Store.ListItemCatalog` (`internal/store/contentcatalog.go:167`)
- `Store.ListSkillCatalog` (`:202`)
- `Store.ListNPCTemplates` (`:232`, blob canônico de 816 B)

Passos:

1. Construir `itemcatalog.Catalog`/`skillcatalog.Catalog` a partir dessas leituras (mesma conversão
   que `tmserver/internal/dbclient/contentcatalog.go` faz para o tmServer), mantendo `Version` e
   `ApplyIcons`.
2. Trocar `readMobTemplate` para `npctemplate` vindo do banco (`ListNPCTemplates`), não de `os.ReadFile`.
3. `droptool.Scan` recebe o catálogo em memória; `ItemDropList.txt` continua sendo arquivo (regra de
   drop, não catálogo) — só a parte de item sai do disco.
4. Remover do `main.go` os `if *contentDir != ""` que só existiam para os catálogos; `contentDir`
   fica apenas para `attributemap` (`AttributeMap.dat`) e `ratesadmin` (`Common/Settings/*`), que
   **continuam em arquivo por design**.
5. Atualizar o help da flag `-content` e o comentário do compose (o mount `./Release:/Release:ro`
   permanece, mas por causa de rates/mapas).

## Testes/verificação

- Boot do `webserver` sem nenhum dos quatro WARN; em vez disso, `scanned item catalog count=3220`
  (ou o log equivalente vindo do banco).
- `ListMerchantTemplates` e `ListItemCatalog` devolvendo listas não vazias com o banco populado.
- Teste de unidade do serviço com catálogo injetado (o padrão `SetItemCatalog` já existe em
  `npcadmin`/`characterAdmin`).
- Boot com banco vazio: erro explícito pedindo o seed, não WARN silencioso.

## Critérios de aceite

- [ ] Nenhum `Scan` de catálogo em `webserver/cmd/webserver/main.go`.
- [ ] Quatro WARN extinguidos e features de picker/ficha funcionando com dado do banco.
- [ ] `-content` documentado como "rates e mapas", não como catálogo.
- [ ] Teste cobrindo catálogo vindo do store.

## Riscos/rollback

Risco: o `webserver` quebrar o boot quando o banco está vazio — comportamento desejado é falha
explícita, como no tmServer ("item_catalog is empty: seed the content catalogue"). Rollback: manter
o scan atrás de uma flag temporária `-legacy-content-scan` (removida na mesma entrega seguinte, para
não repetir o defeito de duas fontes vivas).
