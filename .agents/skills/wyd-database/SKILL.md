---
name: wyd-database
description: Audita, migra e opera o PostgreSQL do OpenWYD — esquema, migrations (incluindo o baseline/squash), seeds de referência determinísticos, índices, constraints, comentários, backup e restauração. Use ao criar/alterar tabela, escrever migration, investigar consulta lenta, revisar normalização/indexação/segurança do banco, ou ao rodar o squash de migrations.
---

# Banco de dados (PostgreSQL)

Banco único `postgres`, dono do estado do jogo persistido, do catálogo de NPCs e dos catálogos
de conteúdo (issue 58). O tmServer e o portal **não** falam SQL: tmServer via gRPC
(`ContentCatalogService`, `NpcConfigService`…), portal via web-api. SQL direto só em
migration, importador, seed e diagnóstico.

## 1. Invariantes (violar aqui quebra o jogo de forma silenciosa)

1. **A web nunca escreve estado de personagem.** Grants vão por `delivery_queue`; o tmServer drena.
   Migration/seed é a única escrita fora do caminho de jogo.
2. **Seed não carrega dado de jogador.** A allowlist está em `internal/store/seed.go`
   (`SeedTables`). `account`, `character`, `item`, `affect`, auditorias e `server_instance_flags`
   ficam fora. Um `pg_dump` genérico dentro do repositório é vazamento de conta.
3. **Colunas que apontam para jogador não entram no seed.** `npc_definition.updated_by → account(id)`
   é omitida (`playerRefColumns` no `seedgen`): num banco novo a conta não existe e a FK aborta.
4. **Seed determinístico.** Ordem por `to_jsonb(t)::text`, cabeçalho sem timestamp; regerar sem
   mudança tem de produzir diff vazio. Sem isso não existe "baseline" — só um dump aleatório.
5. **Blob de template é sempre canônico.** `npc_template.blob` tem `CHECK (length(blob) = 816)`;
   layouts 756/920 são normalizados no import (`savefmt.NormalizeMob`).
6. **Texto de conteúdo é Windows-1252** na origem: passe por `internal/textenc.ToUTF8` antes do banco.
7. **jsonb de lista nunca é `null`.** `json.Marshal(nil)` vira `null`, que `json_populate_record`
   relê como SQL NULL; use slice vazio e mantenha o `CHECK jsonb_typeof(...) = 'array'`.
8. **Migrations são append-only** até um squash explícito. Cada `.up.sql` roda em transação própria
   (`store.Migrate`, advisory lock `0x77327070`) e é registrado em `schema_migrations(version)`.

## 2. Comandos

```bash
# auditoria somente-leitura (tamanho, PK, FK sem índice, índice morto, comentários, tipos)
docker exec -i openwyd-db-1 psql -U postgres -d postgres -f - < scripts/db-audit.sql

# backup completo + esquema em texto (destino fora do repo por padrão)
./scripts/db-backup.sh

# consulta pontual
docker exec openwyd-db-1 psql -U postgres -d postgres -c "select count(*) from npc_definition"

# migrations: aplicadas no boot de dbserver/webserver/importadores (idempotente)
# importar catálogos de conteúdo (idempotente; -check compara banco × arquivos)
docker compose run --rm --entrypoint importcontent dbserver -content /Release
# gerar os seeds de referência a partir do banco populado
seedgen -dsn "postgres://postgres:<senha>@db:5432/postgres" -out internal/migrations

# restaurar um dump num banco NOVO (nunca por cima do que está rodando)
docker exec openwyd-db-1 psql -U postgres -q -c "CREATE DATABASE restore"
docker exec -i openwyd-db-1 pg_restore -U postgres -d restore --no-owner < dump
```

`make` não existe no PATH desta máquina: use
`/c/ProgramData/mingw64/mingw64/bin/mingw32-make`. Docker no Git Bash exige `MSYS_NO_PATHCONV=1`,
senão `/Release` é reescrito para um caminho do Git.

## 3. Escrever uma migration

O pacote tem **um baseline** (squash de 04/11/2026) e migrations incrementais a partir dele:

```
0001_schema.up.sql            esquema inteiro (pg_dump --schema-only, sem schema_migrations)
0001_schema.down.sql          DESTRUTIVO (DROP SCHEMA public CASCADE) — documenta o desfazer
seed_<tabela>.up.sql          dado de referência determinístico (12 arquivos, gerados por seedgen)
00NN_<assunto>.up.sql         toda mudança depois do baseline
_archived/                    staging local (gitignored) das 59 antigas — histórico é o git
```

O embed é `//go:embed *.sql` na **raiz** do pacote (`internal/migrations`), então seed novo vai na
raiz, nunca em subpasta. `scripts/db-baseline.sh` regenera o baseline (backup → dump → valida em
banco vazio → `--apply` arquiva, promove e adota) e `seedgen` regera os seeds a partir de um banco
populado:

```bash
bash scripts/db-baseline.sh            # dry-run: gera e valida, não mexe na raiz
docker compose run --rm --entrypoint seedgen dbserver -dsn "$W2PP_DB_DSN" -out internal/migrations
```

Depois do baseline, cada mudança nova segue tabela → seed → constraints/índices:

```
00NN_<tabela>.up.sql              CREATE TABLE (+ CHECK inline, PK, NOT NULL)
00NN+1_seed_<tabela>.up.sql       INSERT ... ON CONFLICT DO NOTHING (opcional)
00NN+2_<tabela>_constraints.up.sql FK, UNIQUE e índices de leitura
```

Por quê: o seed não paga manutenção de b-tree por linha, e a falha aparece na etapa certa.
Nunca renomeie um `.up.sql` já aplicado — o runner usa o nome como versão.

Regras de escrita:

- **Comentário de tabela e de coluna não são opcionais** quando a semântica não é óbvia
  (`COMMENT ON TABLE/COLUMN`); o audit mostra a cobertura.
- Índice só com leitor identificado (`WHERE`/`JOIN` real). Índice "preventivo" é custo de escrita
  e aparece como `idx_scan = 0` na auditoria.
- **Toda FK de coluna única usada em join precisa de índice na origem** — sem ele, update/delete do
  pai varre a filha inteira e trava.
- `TIMESTAMPTZ`, nunca `TIMESTAMP`; `TEXT` para texto (não `VARCHAR(n)` sem limite real);
  `BIGINT` para id, `NUMERIC` só para dinheiro quando houver fração.
- Identidade: `GENERATED ALWAYS AS IDENTITY` é o padrão novo; tabelas antigas com `BIGSERIAL`
  continuam como estão até o squash.

## 4. Diagnóstico rápido

| Sintoma | Ver |
|---|---|
| NPC/item errado no jogo | `npc_definition`, `npc_shop_item`, `item_catalog` — e o log `npc config applied at boot` |
| Catálogo vazio no boot | `importcontent -check`; o tmServer cai nos arquivos e loga o motivo |
| Consulta lenta no portal | `scripts/db-audit.sql` §14 (seq_scan) + `EXPLAIN (ANALYZE, BUFFERS)` |
| Dump/restore que não sobe | `-check` dos seeds, FKs para `account`, identidade `GENERATED ALWAYS` |
| Suspeita de dado de jogador em seed | `git grep -l '"pass_hash"' internal/migrations` (deve ser vazio) |

## 5. Não faça

- `DROP`/`TRUNCATE` fora de um squash autorizado e com backup recém-feito.
- `pg_dump` completo dentro do repositório.
- Migration que reescreve dado de jogador (correção de conteúdo é seed; de personagem é issue própria).
- `UPDATE` em massa sem `WHERE` em tabela viva durante teste humano — o restart/save do jogo
  transforma isso em bug fantasma.
