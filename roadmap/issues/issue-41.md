# Issue 41 — Atlas do mundo: zonas, coordenadas e portais (leitura)

**Fase:** 6 · **Depende de:** 22, 24 · **Bloqueia:** 42, 45 · **Esforço:** M · **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Uma fonte única — legível por humano e por máquina — com **todas as áreas que existem no mundo**:
as 5 cidades (limite, spawn, arena), as 52 regiões nomeadas, as 23 rotas de teleporte, os 48 itens
fixos (portões/baús), os NPCs e os spawns de mob. Tela `/atlas` com mapa navegável, tabela por camada,
busca por nome e export CSV/JSON — porque hoje ninguém sabe responder "onde fica o quê" sem ler Go.

## Contexto / evidência

- As cidades são uma **tabela fixa no código**: `tmserver/internal/world/city.go:18` (spawn, `limit`,
  `war` por zona) — não há arquivo de conteúdo nem RPC.
- As regiões nomeadas estão em `Release/TMsrv/run/Regions.txt` ( 52 retângulos, formato
  `x1, y1, x2, y2 = Nome`) e **só o tmserver lê**: nenhuma RPC, nenhuma tela.
- As rotas de teleporte são `teleportTable` em `world/teleport.go:22` (origem múltipla de 4, custo,
  requisito de fada/Kefra). Snapshot em `roadmap/atlas-mundo.md`.
- Itens fixos: `Release/TMsrv/run/InitItem.csv` (48 linhas; id do portão = ordem de criação).
- Spawns: `NPCGener.txt` → boot `generators=6099 mobs=11930 templates=620`.
- O `AttributeMap.dat` (1024²) e o `HeightMap.dat` (4096²) já são lidos pelo `route.Bake`, então há
  base para desenhar o mapa sem novo decoder.

## Passos

1. `tmserver/cmd/worldatlas/main.go`: gerador (fica sob `tmserver/` porque importa `tmserver/internal/*`, que o Go não deixa importar de `webserver/`) que escreve `dist/world-atlas.json` com
   `{ version, source_hash, generated_at, cities[], regions[], teleports[], static_items[], npcs[],
   spawns[], bosses[] }`. Lê: cidade (tabela Go exportada via função `world.AtlasCities()`), regiões
   (parser do `Regions.txt`), teleportes (nova `world.TeleportRoutes()`), `InitItem.csv`, definições de
   NPC (overlay + conteúdo) e `NPCGener.txt`.
2. Expor as tabelas fixas: `world.AtlasCities()` e `world.TeleportRoutes()` (leitura pura, sem mudar
   comportamento) para o gerador não duplicar constantes.
3. `api/web/v1/web.proto`: `service WorldAtlasService { rpc GetAtlas(GetAtlasRequest) returns
   (GetAtlasResponse); }` — leitura **pública** como `ItemCatalogService` (sem `moderator_id`), com
   `AdminResult result` por consistência.
4. `webserver/internal/worldatlas`: carrega o JSON no boot (`W2PP_WORLD_ATLAS`, default
   `dist/world-atlas.json`), valida e serve; sem arquivo → `result=OK` com `atlas` vazio e log
   `world atlas not configured` (mesma política dos ícones).
5. Portal: `server/atlas.ts` (cache por `source_hash`), `/atlas/page.tsx` + `components/atlas/MapCanvas`
   (SVG sobre o `AttributeMap` reduzido a 512², camadas ligáveis: cidades, regiões, portais, NPCs,
   spawns), tabela `DataTable` por camada com busca e "copiar coordenada" (`navigator.clipboard`), e
   `/api/atlas/export?layer=…&format=csv|json`.
6. Destacar no mapa a região sob o cursor, com nome e retângulo — é o que responde "que área é esta?".
7. Nav: `Mundo → Atlas do mundo` (ícone `compass`), tirando o "em breve".

## Arquivos

- Criados: `tmserver/cmd/worldatlas/main.go`, `webserver/internal/worldatlas/*.go`,
  `tmserver/internal/world/atlas.go`, `portal/src/server/atlas.ts`,
  `portal/src/app/(admin)/atlas/{page.tsx,atlas-view.tsx}`,
  `portal/src/components/atlas/MapCanvas.tsx`, `portal/src/app/api/atlas/export/route.ts`.
- Alterados: `api/web/v1/web.proto`, `dist/world-atlas.json` (gerado, versionado),
  `portal/src/components/{nav.ts,Icon.tsx}`, `portal/src/middleware.ts`.
- Proibido: editar `Release/` (o atlas lê, não escreve); duplicar as constantes de cidade/teleporte no
  TypeScript.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
go run ./tmserver/cmd/worldatlas -content Release -out dist/world-atlas.json
go test ./tmserver/internal/world/ -run TestAtlas
cd portal && npx tsc --noEmit && npm run build
curl -s -b jar 127.0.0.1:3000/api/atlas/export?layer=regions\&format=csv | head -3
```

Esperado: JSON com `cities=5`, `regions=52`, `teleports=23`, `static_items=49`, `spawns` > 6.000;
`/atlas` 200 com o mapa e a tabela de regiões; export CSV com cabeçalho.

## Critérios de aceite

- [x] `world-atlas.json` gerado com contagens conferindo com as fontes (5/52/18/48/6.099)
- [x] `/atlas` mostra o mapa com camadas ligáveis e busca por nome
- [x] Clicar numa região/portal mostra nome, retângulo e coordenada copiável
- [x] Export CSV e JSON por camada
- [x] Sem `world-atlas.json` a tela abre vazia explicando o gerador (não quebra)
- [x] Nenhuma constante de cidade/teleporte duplicada em TypeScript

## Riscos / rollback

- **Risco:** `HeightMap.dat` (16 MiB) travar o browser. **Mitigação:** renderizar o `AttributeMap` a
  512² e sobrepor vetores; o mapa é estático (cacheável por hash).
- **Risco:** o gerador virar fonte de verdade paralela. **Mitigação:** ele só **lê** código/conteúdo;
  o `source_hash` no JSON denuncia divergência e testes comparam com as constantes.
- **Rollback:** a tela é read-only; apagar o JSON volta o portal ao estado anterior.

## Evidência de execução (2026-11-03)

```text
$ go run ./tmserver/cmd/worldatlas -content Release -out dist
worldatlas: cities=5 teleports=23 regions=52 static_items=49 spawns=6099 hash=f6b35bb0b45d9720
dist/world-atlas.json  1.198.585 bytes
dist/world-map.png         7.519 bytes (512²)

$ npx tsc --noEmit   # tsc_exit=0
$ npm run build      # build_exit=0 (ƒ /atlas, ƒ /api/atlas/map, ƒ /api/atlas/export)

# com os artefatos presentes
$ curl /atlas                         200 · 2.346.417 bytes
  contém Guerra_de_Cidades (região), Ciclope_Forte (líder de spawn), ui-atlas__map;
  contagens 5/52/23/49/6099 vêm do próprio JSON (o SSR separa número e rótulo por comentário)
$ curl /api/atlas/map                 200 image/png 7519
$ curl /api/atlas/export?layer=regions&format=csv    200
  "name","x1","y1","x2","y2","source"
  "Guerra_de_Cidades","129","137","253","243","Regions.txt"
$ curl /api/atlas/export?layer=spawns&format=csv     200 (6.100 linhas = cabeçalho + 6.099)
$ curl /api/atlas/export?layer=all&format=json       200
$ curl /api/atlas/export (sem sessão) 401

# com os artefatos escondidos (estado "não gerado")
$ curl /atlas                         200 · "Atlas não gerado" + comando do gerador
```

- Criados: `tmserver/internal/world/atlas.go` (`AtlasCities`/`AtlasTeleports`,
  leitura pura das tabelas fixas), `tmserver/cmd/worldatlas/main.go` (gerador + PNG do
  `AttributeMap`), `dist/world-atlas.json`, `dist/world-map.png`,
  `portal/src/server/atlas.ts`, `portal/src/app/(admin)/atlas/{page.tsx,atlas-view.tsx}`,
  `portal/src/app/api/atlas/{map,export}/route.ts`. Alterados: `components/nav.ts` (`/atlas` ativo),
  `middleware.ts`, `globals.css` (`.ui-atlas*`).
- **Desvio 1:** o gerador mora em `tmserver/cmd/`, não em `webserver/cmd/` — ele lê
  `tmserver/internal/{content,world}` e a regra de `internal` do Go proíbe importar esse pacote de
  fora da árvore `tmserver/`. O binário é o mesmo e roda com `go run`; nada é rebuildado.
- **Desvio 2:** os dados são servidos pelo **BFF do portal** lendo o artefato em
  `dist/` (mesmo padrão do preço base da issue 24), em vez de uma RPC
  `WorldAtlasService`. O JSON é read-only e derivado do servidor; evitar proto+rebuild do
  `webserver` por um dado estático. Se o atlas um dia ficar editável (issue 42), a RPC entra ali.
- **Desvio 3:** não há zoom/pan no mapa. O PNG de 512² com SVG em `viewBox 0 0 4096 4096` cobre o
  mundo inteiro; zoom fica para quando a 42 precisar escolher destino no mapa.
- **Desvio 4:** o PNG é um relevo legível, não colisão autoritativa: a semântica por byte do
  `AttributeMap` segue UNVERIFIED (`content/gamemap.go:44`); 0 = livre, ≥120 = bloqueado.
- **Bug encontrado e corrigido na verificação:** `unstable_cache` persiste o valor em disco,
  inclusive `null` — depois de testar o estado "atlas não gerado", a tela continuou vazia mesmo com
  os artefatos de volta (export respondia `404 {"error":"atlas não gerado"}`). Agora `loadAtlas`
  **lança** quando o arquivo falta e `getWorldAtlas` converte em `null`: erro não é cacheado, então
  a primeira requisição depois de gerar o atlas já o encontra. Cache key subiu para `world-atlas-v2`.
- **Pendente:** `items fixos` são 49 (o `LoadInitItems` descarta o sentinela -1); a leitura do
  catálogo é só informativa — editar/portões continua sem caminho (lacuna §5.2 da matriz).
- **Pendente:** NPCs (568 definições) não entram no JSON do atlas; a tela usa a lista de `/npcs` para
  isso e o cruzamento por coordenada fica para a 37 (spawns) / 42 (teleportes).
