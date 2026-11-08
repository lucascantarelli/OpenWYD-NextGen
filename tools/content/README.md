# tools/content — gerador de seed de conteúdo

Ferramenta **offline**: converte o conteúdo legado em migrações SQL. Nada aqui roda em
runtime — o servidor lê o PostgreSQL (ADR-0009).

## Uso

```bash
node tools/content/gen-seed.mjs --dry-run            # relatório + asserções, sem escrever
node tools/content/gen-seed.mjs --dry-run --verbose  # + matrizes do gameconfig e nomes coreanos
node tools/content/gen-seed.mjs                      # grava as migrações
```

Requer `git` com `origin/main` disponível: os insumos são lidos com `git show` — nenhum
arquivo do legado é copiado para a árvore.

## Saída

| Arquivo | Conteúdo |
|---|---|
| `crates/wyd-db/migrations/0009_seed_item_catalog.sql` | 3.220 itens + vocabulário `EF_*` |
| `crates/wyd-db/migrations/0010_seed_skill_catalog.sql` | 151 skills |
| `crates/wyd-db/migrations/0012_seed_mundo.sql` | 1.991 templates, 6.100 NPCs, 6.706 slots de loja, 161 prêmios por nível, 5 guardas (27 exigências), 1 mercador, 25 settings, 20 matrizes, 52 regiões, 49 itens de chão, 5 zonas de guilda + `content_source` |

Todos são **gerados**: não editar à mão (`NÃO EDITAR À MÃO` no cabeçalho). Mudança de
regra ou de parser = mudar este arquivo, regerar e revisar o diff de dados.

## Sanitização (modelo-banco.md §6.6)

O insumo é saída de servidor de 2003–2010: CP949, latin1, CRLF. As regras **verificam e
falham**; nenhum nome é "consertado" por chute.

| Regra | Efeito |
|---|---|
| UTF-8 estrito por insumo | byte inválido fora de linha `#`/`//` **aborta** o gerador |
| NFC canônico | `ã` composto ≠ `a`+U+0303 para o banco |
| BOM, U+200B–200F, U+2060, U+00AD, controles C0/C1 | removidos |
| NBSP (U+00A0, U+2007, U+202F) | vira espaço |
| espaços em rótulo | colapsados e aparados |

O relatório (`sanitizacao`) conta cada regra que disparou. Hoje só dispara o que é dado de
verdade: 4 bytes inválidos em comentário do `LevelItem.txt`, 157 linhas repetidas de
`level_item`, 1 retângulo de região invertido (canonizado, perda zero) e 2 regiões acima
do espaço 0..4095 do cliente (preservadas).

**CP949 lido como ISO-8859-1.** 20 nomes de `npc_template` vêm de arquivo coreano
(`ÇÇ´Ð½º` = 피닉스). A chave **não** é renomeada (é referência de cliente e de
`npc_definition`); o nome coreano vai em `name_recovered` quando a recuperação é total
(12 de 20 — os outros 8 perderam byte no legado). `Foema_Anciã` e `Troll_Caçador@@` são
português legítimo e ficam de fora da heurística.

## Parsers

| Insumo | Regra | Tabela |
|---|---|---|
| `seed_*_catalog.up.sql`, `seed_npc_*.up.sql` | blocos `$json$…$json$` | 1:1 |
| `LevelItem.txt` | `seq level classe type item [6 extras]`; bloco repetido deduplicado por `seq` | `level_item` |
| `Guard.txt` | `idx,npc` + N blocos `[8 números]` (cabeçalho do arquivo descreve 13 campos: bloco é UNVERIFIED) | `guard_spawn` + `guard_spawn_reward` |
| `MobMerc.txt` | blocos `# [n]` com `Chave: valor` | `mob_merc` |
| `gameconfig.txt` | várias chaves por linha; valor = números seguintes (`evstart` tem 4) | `game_setting` |
| `gameconfig.txt` (seções matriciais) | `Item Drop Bonus` 4×16, `Treasure` 8×2 | `game_matrix` |
| `Regions.txt` | `x1,y1,x2,y2 = Nome`; nome pode repetir | `region` |

## Asserções (o gerador não emite seed que o banco recusaria)

Contagens esperadas, `sha256(blob) = sha256` em cada template, template/NPC de loja
inexistente, PK `(npc_id, slot)` duplicada, `slot`/`quantity`/`class_id`/`type_id` fora de
faixa, guarda apontando para NPC inexistente, retângulo invertido, `game_setting`
duplicada. Qualquer falha → **nada é escrito** e o processo sai com código 1.

## Regras de classificação

Documentadas em `docs/wyd/itens.md` e `docs/wyd/skills.md`.

- `kind`/`primary_system`: regra por nome (`rule:name_v1`, confiança 60) ou `NULL` — nunca
  chute. `NULL` significa "não sabemos" e o portal mostra isso.
- `event_only`: só `true` quando o nome prova (cupom, ticket, evento…); senão `NULL`.
- `tradeable`: `false` se o item declara `EF_NOTRADE`.
- `class_mask`: **não preenchido** — o 4º campo do `ItemList` não tem semântica provada.
- skill: mapeada só por nome confirmado; fora dela, `unmapped`.

Cobertura (2026-11-08): 3.220 itens, 2.339 nomes distintos, 1.688 classificados, 99
`event_only`, 55 efeitos; 151 skills com 12 mapeadas.

**Duplicidade de nome não é erro:** 489 grupos cobrem 1.370 itens porque o catálogo tem
tiers `(Anct)` g5..g8, `Cupom_da_Sorte` ×78 (eventos distintos), placeholders
`NONAME`/`not_used` e itens de classe. Com nome+grade+preço+efeitos idênticos a contagem é
**0** — nada é deduplicado.

## Proveniência

Cada execução grava `content_source` (path, sha256, size_bytes, row_count, loaded_by,
transform) para os 12 insumos. É auditoria: o banco não depende do arquivo, e o boot pode
logar o hash do que carregou.
