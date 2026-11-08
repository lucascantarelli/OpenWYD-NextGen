# Issue 03-4 — Fim dos TXT/CSV: validação de schema no boot

**Fase:** 3 · **Depende de:** FASE-04-3 (carga de conteúdo) · **Bloqueia:** 03-5 · **Esforço:** M

## Objetivo

Substituir o carregamento de TXT/CSV arcaicos por tabelas validadas, com o
servidor abortando no boot (erro claro) ante dado estático inconsistente.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.2: substituir texto plano por tabelas/banco;
validação de schema na inicialização com falha explícita, nunca default
silencioso.

## Análise (2026-11-08) — passo 1 concluído

Inventário completo em `docs/arquitetura/modelo-banco.md` §6. Resumo:

- **Tabelas de conteúdo não existem na árvore Rust.** `crates/wyd-db/migrations`
  cobre só jogador (contas, personagens, inventário, guildas, auditoria, fila).
- **Fonte histórica está em `origin/main`** (árvore Go removida do working tree):
  `internal/migrations/seed_*.up.sql` gerados por `tmserver/cmd/seedgen` —
  `item_catalog` 3.220, `skill_catalog` 151, `npc_template` 1.991,
  `npc_definition` 6.100, `npc_shop_item` 6.708, `world_item_seed` 49,
  `guild_zone` 5 e estados iniciais (castelo/Kefra/world event) com 1 linha.
  Cada linha já traz `sha256` de proveniência e `ON CONFLICT DO NOTHING`.
- **Configuração sem tabela própria** veio em `0008_platform_content.up.sql`
  (`guard_spawn`, `level_item`, `mob_merc`, `game_setting`, `Regions` → 52
  retângulos) com `content_source_file` (path/sha256/rows).
- **Cliente não é fonte**: `ItemList.bin`/`SkillData.bin` abrem com bytes
  repetidos (XOR), `Mixlist.bin`/`object.bin` opacos, e nenhum leitor existe em
  `tools/client/`. O seed Rust deve sair das linhas `item_catalog`/`skill_catalog`,
  não do `.bin`.
- **Docs de sistema/lore** (`docs/sistemas/*`, `docs/lore/*`, `docs/guias/*`)
  só existem em `origin/main`; a árvore atual perdeu os originais.

## Decisão (ADR-0009, 2026-11-08)

Fonte canônica do conteúdo estático: **PostgreSQL**, em tabelas tipadas com
`PRIMARY KEY`/`FOREIGN KEY`/`CHECK` + `content_version`, populadas por migrações
versionadas de seed (idempotentes, com `down`). `wyd-db` carrega e `wyd-core` valida
antes do primeiro tick (`ports/content.rs`), abortando com `{tabela, pk, coluna, motivo}`.
**Nenhum `.csv`/`.txt`/`.bin` é lido em runtime** — o legado é insumo de
`tools/content/gen-seed.mjs` e base de conhecimento. Alternativas descartadas (CSV/JSON
em repo, `.bin` do cliente, TXT legado, `include_str!`) e consequências estão na ADR.

## Passos

1. [x] Inventariar os arquivos planos consumidos pelo legado (`modelo-banco.md`
   §6, incluindo procedência e lacunas).
2. [x] Definir a fonte canônica — **ADR-0009**: PostgreSQL com migração de seed e
   validação tipada no boot; arquivo plano nunca é lido em runtime.
3. [x] Implementar a carga validada no boot (via `wyd-db`) com relatório por
   tabela/pk/coluna.
4. [x] Teste: dado corrompido aborta com mensagem que identifica o problema.

## Arquivos

- Criados: migrações de schema/seed em `crates/wyd-db/migrations/`, carga em
  `crates/wyd-db/src/repo/content.rs`, porta `crates/wyd-core/src/ports/content.rs` e
  testes de conteúdo inconsistente.
- Alterados: docs do formato em `docs/arquitetura/` (`modelo-banco.md` §6 com o
  inventário; `estrutura-e-padroes.md` §10/§15 e `visao-pastas.md` com a árvore de
  `content/`).
- Proibido tocar: conteúdo de jogo sem motivo, default
  silencioso (proibido por regra).

## Concluído (2026-11-08)

- **Porta:** `wyd_core::ports::content` — `ContentTables`/`ItemRow`/`SkillRow`/
  `SettingRow`/`ContentVersion`, `ContentSource` (trait async), `ContentIssue`
  (`{tabela, pk, coluna, motivo}`), `ContentError` com teto de 20 problemas e
  `ContentCounts` para o log de boot. Validação pura em `ContentTables::validate`:
  versão ≥ 1, catálogo de itens/skills não vazio, índice/chave repetido, nome vazio ou
  com caractere de controle, preço negativo, linhagem fora de `skill_lineage`.
- **Loader:** `wyd_db::repo::content` — `impl ContentSource for PgBackend`, uma
  transação de leitura (version, items, skills, settings, lineages) e `validate()`
  antes do commit; falha de leitura vira issue de infraestrutura (`tabela = "*"`),
  distinta de conteúdo inconsistente. `wyd-db` passou a depender de `wyd-core` (DIP;
  sem ciclo: `wyd-core` só depende de `wyd-protocol`).
- **Testes:** `crates/wyd-core/tests/content.rs` (7 casos, sem I/O) e
  `conteudo_estatico_carrega_e_aborta_com_dado_corrompido` em
  `crates/wyd-db/tests/pg.rs` (seed real + linha corrompida restaurada antes do assert).

## Testes / verificação

```bash
cargo test -p wyd-core 2>&1 | tail -n 3
make check
```

Saída esperada: `test result: ok` incluindo teste de dado corrompido (boot
abortado com relatório).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - carga no boot.
- .agents/skills/wyd-rust-db - tabelas como fonte (se for o destino).
- .agents/skills/rust-code-quality - erros tipados, sem default silencioso.

## Documentação de referencia

- pre-release-rust.md (FASE 3 §3.2).
- docs/rust/05-stack-docker.md - arquivos planos legados.

**Status:** 🚧 (2026-11-08) — esquema e seed de **mundo** entregues (0007–0012, verificado
em Postgres); falta a carga/validação no boot, que depende da FASE-04-3 (`sqlx`).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Resultados (esquema + seed, 2026-11-08)

- `crates/wyd-db/migrations/0007_conteudo_catalogo.sql`: tabelas de conteúdo
  (`item_catalog`, `skill_catalog`) + taxonomia (`game_system`, `game_class`,
  `skill_lineage`, `item_kind`, `item_effect`, `content_version`) com PK/FK/CHECK e GIN em
  `effect_names`.
- `0008_seed_taxonomia.sql`: 20 sistemas, 6 classes, 13 linhagens, 16 categorias de item
  (base em `docs/wyd/sistemas.md` e `docs/wyd/classes.md`).
- `0009_seed_item_catalog.sql` (gerado): 3.220 itens, 55 símbolos `EF_*`, 1.688 com
  `kind`/`primary_system`, 99 `event_only`, 156 não negociáveis. Os 1.532 restantes
  ficam `NULL` (nome não prova categoria).
- `0010_seed_skill_catalog.sql` (gerado): 151 skills; 12 mapeadas por nome confirmado
  (classe/linhagem/papel), 139 `unmapped` — o casamento nome↔índice ainda não foi provado.
- `tools/content/gen-seed.mjs` + `README.md`: gerador offline (lê `origin/main` via
  `git show`, emite lotes de 200 linhas); seed = **migração versionada**, não importador
  manual (ADR-0009).
- `docs/wyd/`: lore, classes, skills, sistemas e itens — fonte das regras de classificação
  e material para `/guias`.
- Prova: bloco `up` de 0007–0010 aplicado em banco descartável no Postgres do compose
  (`ON_ERROR_STOP=1`) → 3.220/151/55/20 e `content_version = 3`.
- Pendente: carga tipada + validação no boot (`wyd-db` → `ports/content.rs`), que depende
  da FASE-04-3 (`sqlx`); teste de conteúdo inconsistente abortando o boot.

## Critérios de aceite

- [ ] Zero `.csv`/`.txt`/`.bin` lido em runtime; conteúdo vem do banco e
  inconsistência aborta o boot com relatório claro.
- [ ] Teste de conteúdo inconsistente (banco descartável) verde; evidência anexada.

## Riscos / rollback

- Risco: formato destino incompatível com ferramenta de conteúdo → mitigação:
  conversor versionado + validador antes/depois.
- Rollback: `git checkout -- crates/ docs/`.

## Resultados (mundo, 2026-11-08)

- `0011_conteudo_mundo.sql`: `content_source`, `npc_template` (STRUCT_MOB 816 B com
  `sha256` conferido), `npc_definition`, `npc_shop_item`, `level_item`, `guard_spawn` +
  `guard_spawn_reward`, `mob_merc`, `game_setting` + `game_matrix`, `region`,
  `world_item_seed`, `guild_zone`.
- `0012_seed_mundo.sql` (gerado, 5,5 MB): 1.991 templates, 6.100 NPCs, 6.706 slots de
  loja, 161 linhas de `level_item`, 5 guardas com 27 exigências, 1 mercador, 25 settings,
  20 matrizes, 52 regiões, 49 itens de chão, 5 zonas de guilda, `content_version = 4`.
- Correções que a auditoria obrigou: `region.name` **não** é único (Erion e Armia têm dois
  retângulos cada) → id sintético; `LevelItem.txt` tem 318 linhas para 161 de conteúdo →
  deduplicado; `gameconfig` põe várias chaves na mesma linha e `evstart` tem 4 valores →
  `value_text`; `Guard.txt` tem blocos de 8 campos contra um cabeçalho de 13 → bloco cru
  preservado; `npc_shop_item.item_index` aceita `-25426` (2 linhas) → sem FK e sem CHECK.
- Prova: `up` de 0007–0012 no Postgres do compose (`wyd_seedcheck`, `ON_ERROR_STOP=1`) →
  contagens acima, `sha256(blob) = sha256` em 1.991/1.991, `content_fold` conferido em
  quatro nomes; `down` na ordem inversa deixa **0 tabelas** e remove `content_fold`.
