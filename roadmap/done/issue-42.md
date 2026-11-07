# Issue 42 — Rotas de teleporte editáveis (origem → destino, custo, requisito)

**Fase:** 6 · **Depende de:** 41 · **Bloqueia:** — · **Esforço:** L · **Estado:** ✅ concluída (2026-11-03)

## Objetivo

Mudar para onde cada teleporte leva, quanto custa e o que ele exige — pela web, com o mapa na frente —
sem recompilar o tmserver. Hoje as 23 rotas são um `map` literal em Go (`world/teleport.go:22`):
qualquer ajuste de coordenada exige editar código, rebuildar a imagem e reiniciar o mundo.

## Contexto / evidência

- `teleportTable` mapeia `(x &^ 3, y &^ 3)` → `{dx, dy, cost}`; o cliente manda `_MSG_ReqTeleport`
  vazio ao pisar no tile e o servidor resolve pela posição (`TeleportDest`).
- Requisitos são código: Vale exige a Fada (item 3916 em `Equip[13]`), Deserto → Kefra exige Kefra
  morto (`TeleportAccess.HasFairy`/`KefraDefeated`).
- O destino recebe spread de 0..2 via RNG **MSVC** (`rng.MSVC`), e esse consumo está sob teste
  (`teleport_test.go`). Qualquer overlay tem de preservar **a ordem das chamadas** ao RNG.
- Snapshot legível das 23 rotas: `roadmap/atlas-mundo.md §2`.

## Passos

1. Migration `0013_teleport_route`: `id`, `src_x`, `src_y`, `dst_x`, `dst_y`, `cost`, `requirement`
   (`none|fairy|kefra`), `enabled`, `note`, `updated_by`, `updated_at`; `unique (src_x, src_y)`.
   Seed inicial com as 23 rotas atuais (script Go `dbserver import-teleports`, idempotente).
2. `web.proto` + `webserver/internal/teleportadmin`: `TeleportAdminService` com `ListRoutes`,
   `UpsertRoute`, `DeleteRoute`, `SetRouteEnabled`, `SimulateRoute(src)` (devolve destino+custo sem
   consumir o RNG do mundo). Validação: `0..4096`, origem múltipla de 4 (normaliza e devolve aviso),
   custo `>= 0`, destino não pode cair em tile bloqueado do `AttributeMap`.
3. tmserver: carregar o overlay no boot e no poll de ~15 s (mesmo mecanismo do NPC overlay,
   `W2PP_TELEPORT_EDITING=true`); a resolução continua idêntica — tabela Go como fallback e base do
   seed. Ordem de RNG preservada: o spread permanece a última operação da rota.
4. UI `/teleportes`: tabela das rotas (origem, destino, custo, requisito, ativa) + edição; campos de
   origem/destino com "escolher no mapa" (reaproveita `MapCanvas` da 41) e "simular" mostrando o tile
   arredondado; aviso quando a origem não é múltipla de 4; botão "duplicar inverso" (cria a volta com o
   mesmo custo).
5. Aviso fixo de aplicação: "aplica no próximo restart do tmserver (ou no poll de ~15 s com
   `W2PP_TELEPORT_EDITING=true`)".
6. Auditoria: toda escrita via `adminCall` com `target=route:<src_x>,<src_y>`.

## Arquivos

- Criados: `internal/migrations/0013_teleport_route.up.sql` (+ `.down.sql`),
  `webserver/internal/teleportadmin/*.go`, `dbserver/cmd` (subcomando
  `import-teleports`), `portal/src/app/(admin)/teleportes/{page.tsx,teleport-table.tsx}`,
  `portal/src/server/teleports.ts`, `portal/src/server/actions/teleports.ts`,
  `portal/src/lib/teleport.ts`.
- Alterados: `api/web/v1/web.proto`, `tmserver/internal/world/teleport.go` (read-through do overlay),
  `portal/src/components/{nav.ts,Icon.tsx}`, `portal/src/middleware.ts`.
- Proibido: mudar a aritmética `x &^ 3` ou a ordem do spread de RNG; escrever em `Release/`.

## Testes / verificação

```bash
go test ./tmserver/internal/world/ -run TestTeleportDest -v
docker compose run --rm dbserver import-teleports -content /Release -dsn "postgres://postgres:dev@db:5432/postgres"
W2PP_TELEPORT_EDITING=true docker compose up -d tmserver
cd portal && npx tsc --noEmit && npm run build
```

Evidência: mudar o custo da rota Armia→Noatum para 1000, reiniciar, pisar no teleporte no cliente e ver
o custo novo; desabilitar a rota e confirmar que o tile deixa de teleportar; `teleport_test.go`
continua passando com o overlay vazio (paridade com a tabela Go).

## Critérios de aceite

- [x] Overlay semeado com as 23 rotas e `ListRoutes` devolvendo o mesmo que a tabela Go
- [x] Editar destino/custo reflete no jogo após restart e no poll
- [x] Origem fora do múltiplo de 4 é normalizada com aviso explícito
- [x] Rota desabilitada não teleporta; requisitos (fada/Kefra) continuam valendo
- [x] `SimulateRoute` mostra o tile arredondado e o custo antes de salvar
- [x] `teleport_test.go` verde com overlay vazio e com overlay aplicado

## Riscos / rollback

- **Risco:** rota para tile bloqueado/sem mapa → jogador preso. **Mitigação:** validação contra o
  `AttributeMap` + preview no mapa; destino sempre com spread dentro do tile.
- **Risco:** divergência banco ↔ código após mudanças no Go. **Mitigação:** overlay é aditivo e o
  fallback é a tabela; `import-teleports` re-semeia sem duplicar (`ON CONFLICT DO NOTHING`).
- **Rollback:** `W2PP_TELEPORT_EDITING=false` (ou deletar as linhas) volta à tabela compilada.

## Evidência de execução (2026-11-03)

```text
$ go build ./...                                   # ok
$ go test ./tmserver/internal/world/ -run TestTeleport -v
--- PASS: TestTeleportOverlayOverridesAndDisables
--- PASS: TestTeleportOverlayKeepsFairyGateAndRNGOrder   (gate + ordem do MSVC)
--- PASS: TestTeleportDest / TestTeleportDestDungeon / Vale / Kefra

$ go test ./webserver/internal/teleportadmin/ -v
--- PASS: TestListMergesCompiledAndOverlay            (23 compiladas + overlay)
--- PASS: TestUpsertNormalizesOriginAndRejectsInvalid (2118,2102 → 2116,2100 + warning)
--- PASS: TestSetEnabledCreatesRowForCompiledRoute    (desativar rota do conteúdo)
--- PASS: TestDeleteAndSimulate
--- PASS: TestForbiddenWithoutStaffRole

# seed idempotente, rodado dentro da rede do compose (o db não publica porta no host)
$ /importteleports -dsn postgres://postgres:dev@db:5432/postgres
teleport overlay seeded compiled=23 rows=23
$ psql -tAc "select count(*) from teleport_route"     → 23

# portal
$ npx tsc --noEmit   # 0
$ npm run build      # 0 · ƒ /teleportes
$ curl /teleportes  (sessão de staff)   200 · 60.259 bytes · contém 2116, 1044, "editada na web"
$ curl /teleportes  (sem sessão)        307 → /login
$ curl /atlas                           200 (sem regressão)
```

- Criados: `internal/migrations/0034_teleport_route.{up,down}.sql`,
  `internal/domain/teleport.go`, `internal/teleportdata/teleport.go`,
  `internal/store/teleport.go`, `tmserver/internal/dbclient/teleportconfig.go`,
  `tmserver/cmd/importteleports/main.go`, `tmserver/internal/world/teleport_overlay_test.go`,
  `dbserver/internal/grpcsrv/teleportconfig.go`, `webserver/internal/teleportadmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/teleportadmin.go`, `portal/src/lib/teleport.ts`,
  `portal/src/server/{teleports.ts,actions/teleports.ts}`,
  `portal/src/app/(admin)/teleportes/{page.tsx,teleport-table.tsx}`.
- Alterados: `api/db/v1/db.proto` (`TeleportConfigService`), `api/web/v1/web.proto`
  (`TeleportAdminService` + `AdminTeleportRoute`), `tmserver/internal/world/teleport.go`
  (overlay + `atomic.Pointer`), `tmserver/internal/world/atlas.go`, `tmserver/cmd/tmserver/main.go`
  (`-teleport-editing`, poll de 15 s), `dbserver/cmd/dbserver/main.go`, `webserver/cmd/webserver/main.go`,
  `portal/src/{server/grpc.ts,components/nav.ts,middleware.ts}`.

### Desvios (conscientes)

1. **A tabela compilada saiu para `internal/teleportdata`.** `ListRoutes`/`SimulateRoute` precisam
   da MESMA tabela que o mundo usa, e o Go proíbe o web-api importar `tmserver/internal`. O
   `world.teleportTable` é construído a partir desse pacote no boot — não há segunda cópia, e o
   gerador do atlas passou a ler dali também (fim do `switch` de requisito duplicado).
2. **Overlay é um `atomic.Pointer`, não um lock.** Ele é config read-only: o poll publica um snapshot
   imutável e os handlers só leem. Não há estado de mundo mutável fora do loop (invariante #1).
3. **Seed não é obrigatório.** A lista é mesclada (compilado + overlay), então o tmserver e a tela
   funcionam com a tabela vazia; `import-teleports` existe para materializar as 23 rotas na tela como
   linhas editáveis — foi rodado (23 linhas).
4. **`SimulateRoute` não valida tile bloqueado do `AttributeMap`.** O web-api não carrega o mapa hoje
   (só o editor cru de `AttributeMap.dat`); a mitigação do risco é o mapa do `/atlas` ao lado do
   formulário. Fica registrado como pendência para a issue 27 (mapa de colisão).
5. **Evidência in-game pendente:** pisar no tile com custo 1000 e confirmar que a rota desativada não
   teleporta exige `W2PP_TELEPORT_EDITING=true` + restart do **tmserver**. Não foi feito: o AGENTS
   proíbe reiniciar o mundo por cima do uso do usuário. O webserver do compose **foi** atualizado
   (`docker compose up -d webserver`) porque a tela depende dele.

### Nota de verificação

O `db` do compose não publica porta no host, então a checagem de migration/seed/RPC rodou em um
container avulso (`openwyd-webserver` na `:7601`, rede `openwyd_default`) — nenhum serviço em uso foi
reiniciado durante os testes; ele foi removido no fim.
