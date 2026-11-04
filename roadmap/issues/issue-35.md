# Issue 35 — Curva de EXP e rebalanceamento em massa

**Fase:** 6 · **Depende de:** 25 · **Bloqueia:** — · **Esforço:** M · **Estado:** ⏳ aberta

## Objetivo

Ver e ajustar a curva de nível (`g_pNextLevel[0..400]`), simular "quantos mobs para subir de A a B" e
rebalancear a EXP de uma faixa de mobs em lote — com aplicação no boot do tmserver.

## Contexto / evidência

- A curva vive em `tmserver/internal/level` (`g_pNextLevel`; a variante Celestial
  `g_pNextLevel_2` é **placeholder** — não expor como editável antes de existir dado real).
- A EXP **por mob** já é editável individualmente na issue 25 (`AdminMobTemplateStat.exp`); o que
  falta é visão de curva + operação em lote.
- Rebalancear template em massa hoje é CLI: `tmserver/cmd/exptool -npc Release/TMsrv/run/npc
  [-dry-run|-inventory|-inventory-sample N]`; o boot avisa `unbalanced Exp` quando o catálogo não
  foi restampado.
- Multiplicadores que mexem na EXP efetiva: `-double-exp`, `-newbie-event`, `-kefra-live`,
  `-training-field-exp` (default 6) e `partydif`/`partybonus` legados — o evento global (issue 28) é o
  caminho sem restart para double/newbie.

## Passos

1. **Migração** `0037_level_curve`: `level_curve_override(level int PRIMARY KEY CHECK (level BETWEEN 1
   AND 400), next_exp bigint NOT NULL CHECK (next_exp > 0), updated_by, updated_at)` + auditoria.
2. **Proto**: `LevelCurveAdminService` com `GetLevelCurve` (curva efetiva + origem de cada faixa),
   `UpsertLevelCurve(level, next_exp)`, `ResetLevelCurve(level)`, `BulkRebalanceMobExp(filter, factor)`
   (pré-visualização obrigatória: `dry_run=true` devolve a lista de mudanças).
3. **tmserver**: `level.ApplyOverrides(map[int]int64)` chamado no boot atrás de
   `-level-curve-editing` / `W2PP_LEVEL_CURVE_EDITING` (OFF = curva do código). O bulk de EXP de mob
   grava `mob_template_stat` (mesmo overlay da issue 25), nunca os arquivos.
4. **Portal** `/exp`: gráfico da curva (nível × EXP acumulada) + tabela por faixa com "origem
   (código/override)"; editor por nível; simulador (mob selecionado + nível atual → mortes
   necessárias, considerando flags ativas exibidas ao lado).
5. Rebalance em lote: filtro (faixa de nível, clan/merchant, nome), fator (ex.: `×1.5`), **tabela de
   diff antes/depois** e só então aplicar; limite de 500 templates por operação e auditoria por
   template.
6. Selo de saúde: se houver templates com `exp=0` ou fora de faixa, mostrar contagem e link para
   `/mobs`.
7. Nota na tela: "curva e EXP só valem após restart do tmserver; double/newbie valem pelo evento
   global".

## Arquivos

- Criados: `internal/migrations/0037_level_curve.{up,down}.sql`,
  `tmserver/internal/levelcfg/{levelcfg.go,levelcfg_test.go}`,
  `webserver/internal/leveladmin/{service.go,service_test.go}`,
  `webserver/internal/grpcsrv/leveladmin.go`,
  `portal/src/app/(admin)/exp/page.tsx`, `portal/src/server/actions/level.ts`,
  `portal/src/components/exp/CurveChart.tsx`, `portal/src/lib/level.ts`.
- Alterados: `api/web/v1/web.proto`, `webserver/cmd/webserver/main.go`,
  `tmserver/cmd/tmserver/main.go`, `portal/src/components/nav.ts`.
- Proibido: expor `g_pNextLevel_2` (placeholder); reescrever templates em `Release/`.

## Testes / verificação

```bash
# (comandos rodam na raiz do repositório)
MSYS_NO_PATHCONV=1 docker run --rm -v "D:/dolorestec/wyd2/OpenWYD:/src" -w /src golang:1.26-alpine \
  sh -c "go build ./... && go test ./tmserver/internal/level/ ./tmserver/internal/levelcfg/ ./webserver/internal/leveladmin/"
```

Evidência: subir o nível 10 em 30% no override, reiniciar o tmserver, matar mobs no jogo e conferir
que o level-up acontece na EXP esperada; `dry_run` do bulk lista N templates e **nenhum** é alterado
antes da confirmação; `exptool -dry-run` continua reportando apenas o que resta fora do padrão.

## Critérios de aceite

- [ ] Curva exibida com origem por faixa e simulador coerente com `level.ForExpTier`
- [ ] Override aplicado no boot; flag OFF reproduz a curva atual
- [ ] Bulk exige pré-visualização e auditoria por template
- [ ] Curva Celestial marcada como não editável (placeholder)
- [ ] Evidência in-game de level-up com a curva nova
- [ ] `go test` dos pacotes tocados verde

## Riscos / rollback

- **Risco:** curva não monotônica travar level-up. **Mitigação:** validar `next_exp` crescente no
  store; recusar salvar faixa que quebre monotonicidade.
- **Risco:** bulk aplicar fator duas vezes. **Mitigação:** operação registrada por template com valor
  antigo/novo; `ResetLevelCurve`/`DeleteMobTemplateStat` para voltar.
- **Rollback:** flag OFF para a curva; `DeleteMobTemplateStat` em lote para os mobs.
