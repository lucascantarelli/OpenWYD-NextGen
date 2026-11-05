# LEGACY-01 — Inventário do C++ legado e arquivamento do que já foi substituído

**Fase:** revisão · **Depende de:** — · **Esforço:** S · **Estado:** ✅ concluída (05/11/2026)

## Objetivo

Classificar cada projeto de `Source/Code` e arquivar em `Source/Code/_legacy/` o que já foi
substituído por Go/portal, deixando em `Source/Code` só a referência de wire/layout/fórmula.

## Contexto/evidência

`docs/analysis/migracao-legado.md` já trazia o mapa C++ → Go, mas as ferramentas substituídas
continuavam misturadas com as referências. Critério aplicado: **a ferramenta ainda consegue operar no
runtime atual?** Não — os alvos (`npc/`, `ItemList.csv`, `NPCGener.txt` como fonte viva, dump de conta
em arquivo) saíram do runtime na fase 5/issue 58.

## O que foi feito (05/11/2026)

Movidos com `git mv` para `Source/Code/_legacy/` (onde o `DropTool` já estava desde a issue 61):

| Projeto | Substituto |
|---------|-----------|
| `ExpTool` | curva em `tmserver/internal/exptool` + `/mobs` |
| `NPTool` | `/npcs` sobre `npc_definition` (issue 57) |
| `EDITAPPMOB` | `/mobs/[template]` |
| `EDITAPPSHOP` | `/npcs/[id]` |
| `SearchPass` | argon2id (`internal/secret`); não há texto plano |
| `ZerarSkill` | `/personagens/[id]` + `delivery_queue` |
| `AttributeMap_Editor` | `/mapa` + `/api/mapa/download` |

Ficaram em `Source/Code/`: `TMSrv`, `DBSrv`, `BISrv`, `Basedef.*`, `CPSock.*`, `ItemEffect.h`
(referência citada por `docs/migration/`) e `ClientPatch_v7662` (dependência do cliente, REVIEW-07).

Citações vivas atualizadas para `_legacy/`: `api/web/v1/web.proto` (+ gerado), `tmserver/internal/exptool/exptool.go`,
`docs/integrations/attributemap-editor-nextjs.md`, `docs/migration/mob-template-editing-plan.md`,
`docs/migration/prompts/agent-prompt-pk-nick.md`, `roadmap/issues/issue-27.md`.
`Source/Code/_legacy/README.md` documenta cada substituto e o critério de remoção definitiva.

## Verificação executada

```bash
git mv Source/Code/{ExpTool,NPTool,EDITAPPMOB,EDITAPPSHOP,SearchPass,ZerarSkill,AttributeMap_Editor} Source/Code/_legacy/
git grep -n -E "Source/Code/(ExpTool|NPTool|EDITAPPMOB|EDITAPPSHOP|SearchPass|ZerarSkill|AttributeMap_Editor)" -- ':!Source/Code/_legacy'
# restou apenas docs/agents/*-2026-06-19* (relatório datado, mantido como retrato)
du -sh Source/Code/_legacy     # 468 KB
```

Prova de que os substitutos respondem (stack local, sessão `test`/`admin`):

```
POST /api/auth/login -> 200 {"ok":true,"role":"admin"}
/mobs 200 (19.807 B)   /mobs/Rei_Harabard 200 (20.088 B)
/npcs 200 (2.420.293 B) /mapa 200 (125.378 B)
/personagens 200 (19.171 B) /drops 200 (20.854 B) /drops/carry 200 (22.787 B)
```

## Critérios de aceite

- [x] Toda ferramenta substituída em `_legacy/`, com README de substituição.
- [x] Zero citação viva apontando para o caminho antigo.
- [x] Substitutos provados por requisição real.
- [x] `docs/analysis/migracao-legado.md` com a seção de arquivamento.

## Próximo passo (fora desta issue)

Remoção definitiva de `_legacy/` quando a paridade UNVERIFIED fechar (coeficientes Dex/Str,
`DoubleCritical`/`ParryRate`, `_AUTH_GAME`) — aí também saem `TMSrv`/`DBSrv`/`BISrv`.
