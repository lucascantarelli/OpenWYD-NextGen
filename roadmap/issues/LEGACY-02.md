# LEGACY-02 — Mover o conteúdo de runtime para dentro do servidor Go

**Fase:** remoção de legado · **Depende de:** — · **Esforço:** M · **Estado:** ⏳ planejada

## Objetivo

O servidor Go deixar de depender de `Release/` (árvore de distribuição do servidor Win32) para subir:
mapas, taxas, `gameconfig.txt`, `BaseMob`, `BaseSummon` e `serv00.htm` passam a morar em
`tmserver/content/`, e o que é resíduo do Win32 sai do caminho de build.

## Contexto/evidência (06/11/2026)

Análise completa em [`docs/analysis/release-dependencias.md`](../../docs/analysis/release-dependencias.md):

- `Release/` = 141 arquivos, 55 MB, três subárvores (`Common/`, `DBsrv/`, `TMsrv/`).
- Categoria 1 (runtime): `HeightMap.dat` 16 MiB, `AttributeMap.dat` 1 MiB, `gameconfig.txt`,
  `Regions/Guard/LevelItem.txt`, `Settings/*.txt` (5), `BaseMob/{TK,FM,BM,HT}`, `BaseSummon/`,
  `serv00.htm`.
- Categoria 2 (insumo de import): `npc/` (1.991 templates), `ItemList.csv`, `SkillData.csv`,
  `InitItem.csv`, `NPCGener.new.txt` — o runtime já lê do Postgres desde a issue 58.
- Categoria 3 (resíduo): `.bat` launchers, `W2 - Editor de Contas.exe`, `TesteDie/AttackDieTeste`,
  `Guild*.txt`/`Ranking.txt`, `ImportUser*`, `serverlist.bin` com IP privado.
- 68 arquivos Go citam caminho de `Release/`; `docker-compose.yaml` monta `/Release:ro` em `tmserver` e
  `webserver`; `-content` recebe o caminho por flag.

## Passos

1. Criar `tmserver/content/` com a categoria 1; mudar o **default** de `-content` para esse caminho
   mantendo a flag (nada de caminho mágico).
2. `go test ./...` e corrigir fixtures (esperado: só caminho, nenhuma lógica).
3. Compose/Dockerfiles: montar `tmserver/content` no lugar de `Release`; conferir no boot
   `walkability grid baked dim=4096` e `NPCs spawned`.
4. Arquivar a categoria 3 em `Release/_legacy/` (ou fora do repo) e remover `Release/` das imagens do
   servidor; manter apenas o insumo de import onde o `importcontent` espera.
5. Atualizar `AGENTS.md` §4, skills `wyd-server-ops`/`wyd-database` e `config-ops.md` no mesmo commit.

## Testes/verificação

```bash
go build ./... && go test ./...
docker compose up -d --build tmserver && docker compose logs tmserver | grep -E "walkability|NPCs spawned"
docker compose run --rm --entrypoint importcontent dbserver -content /content -dsn "..." -check
```

## Critérios de aceite

- [ ] `tmserver` sobe com `-content` apontando para dentro do módulo, sem `Release/` montado.
- [ ] `go test ./...` verde sem fixture lendo caminho de distribuição do Win32.
- [ ] Nenhum `.bat`, `.exe` ou estado de servidor de teste no caminho de build.
- [ ] `AGENTS.md` §4 e `config-ops.md` descrevem a nova árvore.

## Riscos/rollback

Risco: imagem do servidor sem os `.dat` sobe "saudável" e só quebra no movimento/drop. Mitigação: o
gate de boot já avisa falta de mapa; adicionar teste que falha se `HeightMap.dat` não for encontrado.
Rollback: reverter o default de `-content` para `/Release` — a flag continua existindo justamente para
isso.
