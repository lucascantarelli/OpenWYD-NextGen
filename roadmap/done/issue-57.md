# Issue 57 — Migrar todos os blocos de NPCGener e remover o suporte ao arquivo

**Fase:** 7 · **Depende de:** 23 · **Bloqueia:** remoção de `Release/TMsrv/run/NPCGener.txt` · **Esforço:** L · **Estado:** ✅ concluída (04/11)

## Objetivo

Fechar a migração de NPCs: **todos** os blocos de spawn no Postgres e o tmServer deixando de ler
`Release/TMsrv/run/NPCGener.txt` — sem flag, sem caminho duplo, sem fallback silencioso.

## O que mudou

| Antes | Depois |
|---|---|
| `dbserver import-npcs` importava só `Merchant != 0` (548 de 6.099) | importa **todos** os blocos; o `npc_shop_item` só é gravado para quem vende (o `Carry[]` de monstro é a tabela de drop) |
| `spawnNPCs` parseava o arquivo no boot, registrava os slots e chamava `GenerateMob` | `registerNPCGenerators` registra um slot por **definição de conteúdo** do snapshot (`seg_x/seg_y`, `seg_range/seg_wait`, `fight/die action`, população, formação, `training_field`) |
| `-npc-editing` escolhia entre arquivo e banco (`skipMerchants`, `leader.rawMerchant`) | o banco é a **única** fonte; a flag, `skipMerchants`/`rawMerchant`, `Generator.DBManaged` e `DBManagedGeneratorCount` foram removidos |
| `ApplyNPCConfigBoot` validava `len(defs) == DBManagedGeneratorCount()` com retry | `FetchNPCConfig` (retry enquanto o dbserver sobe/semeia) + `ApplyNPCConfigBoot(w, snap)`, que falha se uma definição habilitada não tiver slot registrado |
| blocos de evento (água, Kefra, Nightmare, rune quest) vinham do arquivo | são registrados a partir do snapshot, com `DBManaged` fora de cena: `world.EventOwnedGenerator` marca quem o **código de evento** spawna, e o overlay não materializa esses blocos |

`Release/TMsrv/run/NPCGener.txt` **continua no repositório como insumo do importador**: é de lá que o
`dbserver` (re)seedá o catálogo no boot (`buildNPCDefinitions`). O runtime do jogo não o lê mais —
`content.LoadNPCGenerators` permanece apenas como parser de fixture nos testes de boot.

## Evidência (04/11)

```text
# boot do tmServer com as imagens novas (sem nenhuma flag de overlay)
npc generators registered generators=6099 slots=6099 event_owned=199 training_grounds=104
npc config applied at boot version=17 npcs=5161
tmserver listening addr=:8281 mtls=false
# (o log "NPCs spawned" não existe mais)

# banco
6100 definições / 6100 ativas / 6099 content        (6099 blocos + qa-ferreiro custom)

# login headless
TestE2ESmokeLogin  PASS  LOGIN OK — CNFAccountLogin, conn=30002, body=1996 bytes

# go
go build + go vet + go test ./tmserver/cmd/tmserver ./tmserver/internal/{handler,world,content}
             ./dbserver/cmd/dbserver            → todos ok
```

Os **739 blocos sem entidade** (6.099 slots − 5.161 materializados) já não spawnavam pelo arquivo:
têm `max_num_mob=0` (o `GenerateMob` do legado retorna nil em `CurrentNumMob >= 0`) ou posição
bloqueada no `AttributeMap` (salas de pergaminho/dungeon). Antes isso era silencioso; agora cada
caso aparece em `npc content generator produced no entities`. Não é regressão desta issue.

## Passos

1. ✅ `import-npcs` importa todos os blocos, com slug/`generator_data`/população preservados.
2. ✅ `ApplyNPCConfigBoot` valida contra os slots registrados; `DBManagedGeneratorCount` e
   `Generator.DBManaged` removidos.
3. ✅ `spawnNPCs` e o parser no boot saíram; `skipMerchants`, `rawMerchant` e o log
   `merchant_blocks_skipped` também. O audit de EXP por template (issue #43) foi preservado
   (`auditMobTemplates`) e o catálogo de layouts continua logado.
4. ✅ `-npc-editing`/`W2PP_NPC_EDITING` removida do código, do compose, do `.env(.example)`, das
   skills e da UI do portal.
5. ✅ Re-seed nos ambientes existentes coberto pelo seed de boot do dbserver + `import-npcs` manual.
6. ⏳ **Pendente numa issue de fechamento:** remover o arquivo do conteúdo exige um dump inicial do
   catálogo (o dbserver hoje o lê no boot). Até lá o arquivo é insumo, não fonte de runtime. O
   `cmd/worldatlas` (ferramenta de inspeção de conteúdo, não runtime) ainda o lê para publicar os
   spawns no atlas — quando ele passar a ler `npc_definition`, some o último leitor fora do importador.

## Arquivos

- Alterados: `dbserver/cmd/dbserver/main.go` (+ teste de catálogo real: 6.099), `tmserver/cmd/tmserver/main.go`
  (`registerNPCGenerators`, `auditMobTemplates`, boot sem flag), `tmserver/internal/handler/npcconfig.go`
  (`FetchNPCConfig`, boot/validate sem `DBManaged`), `tmserver/internal/world/generator.go`
  (`EventOwnedGenerator`, sem `DBManaged`), `tmserver/cmd/tmserver/{main_test,kefra_test,npccatalog_test}.go`,
  `docker-compose*.yaml`, `.env.example`, `README.md`, `docs/migration/config-ops.md`,
  `docs/migration/npc-editing-plan.md`, `docs/migration/npc-default-roster.md`,
  `docs/integrations/npc-admin-nextjs.md`, `docs/analysis/roteiros-qa.md`, skills do portal/ops/economia.

## Testes / verificação

```bash
go build ./... && go vet ./tmserver/... ./dbserver/...
go test ./tmserver/cmd/tmserver ./tmserver/internal/{handler,world,content} ./dbserver/cmd/dbserver
docker compose build tmserver dbserver && docker compose up -d tmserver
docker compose logs --tail 20 tmserver | grep -E "npc generators registered|npc config applied"
docker run --rm --network openwyd_default ... TestE2ESmokeLogin   # W2PP_E2E_VERSION=12000
```

## Critérios de aceite

- [x] Boot registra os 6.099 blocos e materializa 5.161 NPCs sem nenhuma flag e sem log `NPCs spawned`
- [x] `grep -rn "W2PP_NPC_EDITING\|-npc-editing" --include=*.go --include=*.yaml` vazio (só sobra
      histórico em `roadmap/issues/*` e planos datados)
- [x] Nenhuma flag no código, no compose e nas skills
- [x] `npc generators registered` + `npc config applied at boot` no log, com `event_owned=199` e
      `training_grounds=104` (Campo de Treino intacto)
- [x] `TestE2ESmokeLogin` verde depois da mudança de boot
- [ ] Paridade de drop/EXP medida em jogo (roteiro do `wyd-economy`) — exige sessão humana

## Riscos / rollback

- **RNG:** a população agora é gerada depois do seed de itens do mundo, num único stream; não há
  ordem legada de boot a preservar (o próprio `spawnNPCs` documentava isso). O roteiro de drop segue
  pendente de medição em jogo.
- **Volume:** snapshot de 6.099 definições (~5 MB de templates resolvidos) e ~2,5 min de boot até
  `tmserver listening` em Docker Desktop (era ~10 s com 568). Se pesar, o `Snapshot` pode devolver só
  `template_name` e resolver os bytes sob demanda.
- **Rollback:** `git revert` desta entrega volta ao caminho do arquivo; o catálogo no banco é
  compatível nos dois sentidos (slugs e índices são os mesmos).
