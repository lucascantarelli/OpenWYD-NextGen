# Issue 59 (aceitação em banco) — importar as configs de conteúdo no Postgres vivo

**Fase:** 7 · **Depende de:** 59 (código entregue em 06/11) · **Esforço:** S · **Estado:** 🟡 só falta rodar contra o banco

## Por que esta issue existe

O código da issue 59 está pronto e provado neste host **até a fronteira do banco**: tabelas, parsers,
upserts e `-check` compilam, o golden test fixa as contagens dos arquivos e o dry-run do importador
imprime os números reais. O que não dá para provar sem container é a escrita: `psql` e Postgres não
existem nesta máquina e a missão proíbe subir stack aqui. Este roteiro é o que fecha a issue.

## O que já está provado (06/11/2026)

```bash
go test -race -count=1 ./tmserver/internal/content/ ./tmserver/cmd/importcontent/   # ok
go run ./tmserver/cmd/importcontent -content tmserver/content                      # dry-run
# guards=5 level_items=318 mob_mercs=1 settings=26 drop_matrix=64 regions=52
```

## Roteiro (com stack local de pé)

```bash
# 1. A migração nova precisa existir no schema aplicado
docker compose exec -T db psql -U openwyd_app -c "\d level_item"
docker compose exec -T db psql -U openwyd_app -c "select version from schema_migrations order by version desc limit 3;"

# 2. Importar (idempotente: rodar duas vezes não muda contagem)
docker compose run --rm openwyd-nextgen1-tmsrv \
  /app/importcontent -content /content -dsn "$W2PP_DB_DSN"
docker compose run --rm openwyd-nextgen1-tmsrv \
  /app/importcontent -content /content -dsn "$W2PP_DB_DSN"

# 3. Contagens esperadas (medidas nos arquivos)
docker compose exec -T db psql -U openwyd_app -c "select count(*) from guard_spawn;"      # 5
docker compose exec -T db psql -U openwyd_app -c "select count(*) from level_item;"       # 318
docker compose exec -T db psql -U openwyd_app -c "select count(*) from mob_merc;"         # 1
docker compose exec -T db psql -U openwyd_app -c "select count(*) from game_setting;"     # 26
docker compose exec -T db psql -U openwyd_app -c "select count(*) from game_drop_bonus;"  # 64
docker compose exec -T db psql -U openwyd_app -c "select count(*) from region;"           # 52

# 4. Paridade e proveniência
docker compose run --rm openwyd-nextgen1-tmsrv \
  /app/importcontent -content /content -dsn "$W2PP_DB_DSN" -check     # exit 0
docker compose exec -T db psql -U openwyd_app -c \
  "select path, rows, imported_at from content_source_file order by path;"

# 5. Prova de drift: mexer no arquivo e conferir que o -check reprova
#    (editar Regions.txt numa cópia e rodar de novo: deve sair 1)
```

## Critérios de aceite

- [ ] As 6 tabelas existem com as colunas da migração `0008_platform_content`
- [ ] Importar duas vezes mantém 5/318/1/26/64/52 (idempotência)
- [ ] `-check` sai 0 com o banco em sincronia e 1 após editar um arquivo sem reimportar
- [ ] `content_source_file` tem uma linha por arquivo importado, com `sha256` e `rows`
- [ ] Nenhum erro de jsonb/nulo nos upserts (`rules`, `extra`, `items` vazios gravam `[]`, não `null`)

## Arquivos

`internal/migrations/0008_platform_content.up.sql`, `internal/store/contentsettings.go`,
`tmserver/internal/content/settings.go`, `tmserver/cmd/importcontent/main.go`.

## Riscos

- **Rodar o importador contra o banco de um teste humano em andamento** mexe em conteúdo lido pelo
  servidor. Mitigação: rodar com a stack ociosa e conferir `content_source_file.imported_at`.
- **`game_drop_bonus` é limpa e reescrita** na mesma transação do `game_setting`: se um import falhar
  no meio, a transação inteira volta — não fica matriz nova com pares velhos.
