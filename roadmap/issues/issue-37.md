# Issue 37 — Spawns e localização de mobs

**Fase:** 6 · **Depende de:** 25, 33 · **Bloqueia:** — · **Esforço:** L · **Estado:** ⏳ aberta

## Objetivo

Ver e editar onde os mobs nascem (geradores do `NPCGener.txt`: 6.099 geradores, ~11.930 mobs no boot),
com mapa de calor das coordenadas, sem tocar no arquivo de conteúdo.

## Contexto / evidência

- O boot reporta `NPCs spawned generators=6099 mobs=11930 templates=620` e
  `training-grounds generators marked generators=104` — número que serve de baseline para qualquer
  edição.
- `NPCGener.txt` (`Release/TMsrv/run/`) é a fonte; `docs/migration/npc-generator-inventory.md` já
  inventaria os geradores e `npc-editing-plan.md §9.2` explica que o mundo roda **uma grade única**
  (não há mapa por `map_id`): posição vem de `pos_x/pos_y`.
- O overlay de NPC (`npc_definition` / `npccfg.Definition`) já carrega os campos de gerador:
  `MinuteGenerate, MinGroup, MaxGroup, MaxNumMob, Formation, SegX/SegY/SegRange/SegWait[5],
  RouteType, FollowerTemplate` — o modelo para copiar.
- `route.Bake` usa `AttributeMap.dat` (altura 127 = bloqueado): um gerador em célula bloqueada não
  anda — a UI deve avisar.

## Passos

1. **Migração** `0039_spawn_override`:
   ```sql
   CREATE TABLE IF NOT EXISTS spawn_generator_override (
     generator_index integer PRIMARY KEY,
     template_name   text    NOT NULL,
     pos_x smallint NOT NULL, pos_y smallint NOT NULL,
     min_group smallint, max_group smallint, max_num_mob smallint,
     formation smallint, minute_generate smallint,
     seg_x smallint[], seg_y smallint[], seg_range smallint[], seg_wait smallint[],
     route_type smallint, enabled boolean NOT NULL DEFAULT true,
     updated_by bigint NOT NULL REFERENCES account(id),
     updated_at timestamptz NOT NULL DEFAULT now()
   );
   ```
2. **Parsers**: web-api lê `NPCGener.txt` com o mesmo parser do `tmserver/internal/content` (ou
   pacote comum) e serve `ListSpawnGenerators` (paginado, filtro por template/região/coordenada).
3. **Proto**: `SpawnAdminService` com `ListSpawnGenerators(filter)`, `GetSpawnGenerator(index)`,
   `UpsertSpawnGenerator(...)`, `SetSpawnGeneratorEnabled(index, enabled)`,
   `SpawnCoverage()` (contagem por template e por faixa de coordenada para o mapa de calor).
4. **tmserver**: aplica `spawn_generator_override` ao carregar `NPCGener.txt`, atrás de
   `-spawn-editing` / `W2PP_SPAWN_EDITING` (OFF = arquivo puro; log `spawn_overrides_applied=N`).
   Gerador com `enabled=false` é simplesmente não instanciado (mesma semântica de `NPCGener` com
   linha removida).
5. **Portal** `/spawns`:
   - tabela paginada: índice, template, posição, grupo (`min..max`), `max_num_mob`, formação,
     `enabled`, badge "override";
   - **mapa de calor** em SVG 4096×4096 (células agregadas 64×64), com a camada de bloqueio do
     `AttributeMap` desenhada por cima (via `ListMapZones` + info de atributo) — apenas visual;
   - form de edição com validação de coordenada `0..4095` e aviso quando a célula está bloqueada;
   - "duplicar gerador" (novo índice livre, offset de posição) e "desativar".
6. Filtro "geradores do Campo de Treino" (os 104 marcados) para não quebrar a EXP de treino.
7. Aviso de restart + link para `/operacao`; auditoria por gerador.
8. **Não** migrar automaticamente `NPCGener.txt` para o banco: o overlay é aditivo e reversível; a
   migração em massa fica para uma issue futura, se for decidida.

## Arquivos

- Criados: `internal/migrations/0039_spawn_override.{up,down}.sql`,
  `tmserver/internal/spawncfg/{spawncfg.go,spawncfg_test.go}`,
  `webserver/internal/spawnadmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/spawnadmin.go`,
  `portal/src/app/(admin)/spawns/page.tsx`, `portal/src/app/(admin)/spawns/[index]/page.tsx`,
  `portal/src/server/actions/spawns.ts`, `portal/src/components/spawn/Heatmap.tsx`.
- Alterados: `api/web/v1/web.proto`, `webserver/cmd/webserver/main.go`,
  `tmserver/cmd/tmserver/main.go`, `portal/src/components/nav.ts`,
  `docs/migration/npc-generator-inventory.md` (nota do overlay).
- Proibido: escrever `Release/TMsrv/run/NPCGener.txt`; mexer nos 104 geradores de treino sem aviso.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test ./tmserver/internal/spawncfg/ ./webserver/internal/spawnadmin/"
```

```bash
W2PP_SPAWN_EDITING=true docker compose up -d tmserver
docker compose logs --tail 30 tmserver | grep -E "NPCs spawned|spawn_overrides_applied"
```

Saída esperada: `generators=6099` com override desativado de 1 gerador → `generators=6098` e
`mobs` reduzido na mesma proporção; mover um gerador de teste e ver o mob nascer na coordenada nova.

## Critérios de aceite

- [ ] Lista paginada com filtro por template/região e contagem por gerador
- [ ] Mapa de calor coerente com as posições (sem coordenada fora de 0..4095)
- [ ] Aviso de célula bloqueada no form (usa o attribute map)
- [ ] Override aplicado no boot com log; flag OFF = baseline `generators=6099 mobs=11930`
- [ ] Duplicar/desativar funcionam e são auditados
- [ ] Evidência no jogo: mob novo na posição nova

## Riscos / rollback

- **Risco:** gerador duplicado em coordenada idêntica gerar "mobs empilhados". **Mitigação:** o form
  avisa colisão com outro gerador do mesmo template.
- **Risco:** desativar geradores de treino e quebrar a EXP inicial. **Mitigação:** filtro/aviso
  explícito e contador separado.
- **Rollback:** `SetSpawnGeneratorEnabled(true)`/delete do override; flag OFF restaura o arquivo.
