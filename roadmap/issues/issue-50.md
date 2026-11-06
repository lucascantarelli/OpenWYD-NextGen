# Issue 50 — Conteúdo do servidor no banco (CSV vira seed)

**Fase:** 8 · **Depende de:** 22–34 (overlays), 04 · **Esforço:** L · **Estado:** 🟡 catálogos já em banco (issue 58) e proveniência entregue em 06/11; falta o `-content-source=db` das configs restantes e o hash no CI

## Objetivo

`tmserver` carregar catálogos e configs **do Postgres**, não de `Release/*.csv|txt`, mantendo um
importador idempotente que usa a árvore de conteúdo como seed. `Release/` deixa de ser obrigatório em
runtime (só para importar e para os mapas binários).

## Contexto

- Hoje o boot lê `ItemList.csv` (18 referências no Go), `NPCGener.txt` (13), `SkillData.csv`,
  `InitItem.csv`, `Regions.txt`, `Guard.txt`, `gameconfig.txt`, rates, `CastleQuest.txt` e os
  `.dat` de mapa — ver [`docs/analysis/persistencia-conteudo.md`](../../docs/analysis/persistencia-conteudo.md).
- Já existem 8 overlays auditados (`npc_*`, `item_price`, `mob_*`, `drop_rate_override`,
  `content_rate_override`, `teleport_route`, `world_event_config`, `daily_reward_item`), provando o
  padrão "arquivo base + banco".
- `HeightMap.dat`/`AttributeMap.dat` (17 MiB) **não** vão para o banco (lidos inteiros e baked).

## Entregue em 06/11/2026

- **Passo 1 (proveniência):** `content_source_file(path, sha256, size_bytes, rows, imported_at)` em
  `internal/migrations/0008_platform_content.up.sql`; o importador grava um registro por arquivo
  (`addSource` + `RecordContentSources`) e o `-check` compara o hash, então editar conteúdo sem
  reimportar reprova.
- Os cinco conjuntos de configuração da issue 59 entram pelo mesmo importador, com o mesmo `-check`.
- Descoberta que muda o plano: `ItemList.csv`, `SkillData.csv`, `InitItem.csv` e `npc/` **já não existem**
  na árvore — a issue 58 os substituiu por seeds SQL. O importador os mantém apenas para reprocessar
  uma árvore externa de backup (`-sets all`); o default é o que ainda é arquivo.

## O que ainda falta

- `-content-source=db|files` no tmServer para as configs que continuam sendo lidas de arquivo.
- Paridade por hash no CI (hoje só o importador sabe comparar).
- Remover `./content:/content:ro` dos serviços que passarem a ler tudo do banco (mapas ficam).

## Passos

1. Migrações `content_seed(arquivo, sha256, versao, imported_at)` e `content_item`, `content_skill`,
   `content_spawn` (schema espelhando os campos que o Go já usa).
2. Comando `tmserver -import-content=Release` (ou `make content-import`) idempotente: grava seed,
   compara hash, não sobrescreve overlay.
3. `-content-source=db|files` no `tmserver` (default `files` até o golden test passar; depois `db`).
4. Golden test de paridade: hash do catálogo + 10 itens/mobs/skills sorteados, arquivo × banco.
5. Medir tempo de boot antes/depois e registrar no doc.
6. Remover o volume `./Release:/Release:ro` dos serviços que não precisam mais.

## Arquivos

- `internal/migrations/0043_content_seed*.sql` (e seguintes), `internal/store/content*.go`,
  `tmserver/cmd/tmserver/main.go`, `internal/itemcatalog`, `internal/skillcatalog`,
  `tmserver/internal/world` (loader), `Makefile`, `docker-compose.yaml`,
  `docs/analysis/persistencia-conteudo.md`, skill nova `wyd-content-import`.

## Verificação

```bash
docker compose run --rm tmserver -import-content=/Release
docker compose exec -T db psql -U postgres -c "select count(*) from content_item;"   # 3.220
docker compose exec -T db psql -U postgres -c "select count(*) from content_spawn;"  # 6.103 geradores
go test -run TestContentParity ./internal/... ./tmserver/...
```

## Critérios de aceite

- [ ] Importar duas vezes não duplica nada (hash estável)
- [ ] Overlay continua vencendo o seed, com auditoria
- [ ] Boot com banco populado e `Release/` ausente funciona (exceto mapas)
- [ ] Paridade: catálogo idêntico ao CSV atual (hash + amostra conferida)
- [ ] Tempo de boot medido e documentado

## Riscos / rollback

- **Risco:** divergência silenciosa entre CSV e banco. **Mitigação:** golden test no CI + `-content-source=files`
  como chave de rollback imediata.
- **Rollback:** flag volta para `files`; tabelas `content_*` ficam inertes.
