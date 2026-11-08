# ADR-0009 — Conteúdo estático no PostgreSQL (fonte única), validado no boot; nenhum CSV/TXT em runtime

- **Estado:** aceita
- **Data:** 2026-11-08 (reescrita no mesmo dia: a primeira versão propunha CSV versionado
  e foi vetada pelo dono — arquivo plano é prática do servidor Go, não do projeto atual)
- **Decisores:** dono do projeto + agente (análise na FASE-03-4)
- **Governa:** `roadmap-rust/issues/FASE-04-2-migrations.md`,
  `FASE-04-3-persistencia-async.md`, `FASE-03-4-dados-estaticos.md`,
  `FASE-03-5-autoridade-servidor.md`, `FASE-07-1-gateway.md`, `FASE-10-1-equacoes.md`

## Contexto

`pre-release-rust.md` FASE 3 §3.2 manda **substituir os arquivos de texto plano arcaicos
por carregamento seguro de tabelas tratadas ou banco de dados relacional**, com validação
de schema na inicialização abortando com relatório claro; a FASE 4 §4.1/§4.2 manda
PostgreSQL com `FOREIGN KEY`/`CHECK`/índices e migrações versionadas (`sqlx-cli`/
`refinery`). Nenhuma das duas fases prevê arquivo plano como fonte em runtime.

O estado atual confirma o problema:

1. **Nenhum conteúdo na árvore Rust.** `crates/wyd-db/migrations` cria só tabelas de
   jogador. O catálogo que o tick precisa — `item_catalog` 3.220 linhas, `skill_catalog`
   151, `npc_template` 1.991, `npc_definition` 6.100, `npc_shop_item` 6.708,
   `world_item_seed` 49, `guild_zone` 5 — existe só como `internal/migrations/seed_*.up.sql`
   em `origin/main` (árvore Go removida em 2026-11) e como TXT em `tmserver/content/**`
   (`LevelItem.txt` 17.468 B, `gameconfig.txt` 2.467 B, `Guard.txt` 699 B…).
2. **O legado já sabia disso.** A migração `0008_platform_content.up.sql` (issue 59) tirou
   os TXT sem leitor para o banco (`guard_spawn`, `level_item`, `mob_merc`,
   `game_setting`, 52 retângulos de `Regions`) com proveniência em `content_source_file`
   (`path`, `sha256`, `size_bytes`, `rows`) — ou seja, o próprio Go já tratava arquivo
   como **insumo de importação**, não como fonte viva.
3. **Os `.bin` do cliente não servem como fonte.** `ItemList.bin` (910.004 B) e
   `SkillData.bin` (23.812 B) abrem com bytes repetidos (`5A 5A…`), `Mixlist.bin` é
   opaco, e não há leitor em `tools/client/` (`docs/cliente/inventario.md`: CONHECIDO
   parcial/UNVERIFIED). Servem para conferir `item_index`/nome, não para alimentar o tick.

A ADR-0006 difere `sqlx`/`refinery`; por isso o carregamento de conteúdo passa a ser
**dependência explícita da FASE-04-3**, não algo que a FASE-03-4 resolve sozinha.

## Decisão

Adotamos o **PostgreSQL como única fonte canônica** do conteúdo estático, em tabelas
tipadas com `PRIMARY KEY`/`FOREIGN KEY`/`CHECK` populadas por **migrações versionadas de
seed**, carregadas por `wyd-db` e validadas antes do primeiro tick, porque a diretriz do
projeto é modernizar (FASE 3 §3.2 + FASE 4 §4.1/§4.2) e arquivo plano em runtime
reintroduz a prática que estamos eliminando; nenhum `.csv`/`.txt` é lido pelo servidor em
momento algum.

Regras que decorrem da decisão:

- **Tabelas de conteúdo (base, somente leitura para a aplicação).** `item_catalog`,
  `skill_catalog`, `npc_template`, `npc_definition`, `npc_shop_item`, `world_item_seed`,
  `guild_zone`, `level_item`, `guard_spawn`, `mob_merc`, `game_setting`, `region` e
  `content_version`. Escrita só por migração; o portal nunca altera a base.
- **Overlays continuam separados.** `item_price`, `item_effect_override`,
  `drop_rate_override`, `content_rate_override` e edições de `npc_*` são linhas de
  overlay auditadas (`*_audit`), aplicadas **sobre** a base; chegam ao servidor como
  intenção na `delivery_queue` (ADR-0002/0004/0006), nunca por leitura no tick.
- **Migrações de seed versionadas e idempotentes.**
  `crates/wyd-db/migrations/NNNN_conteudo_<dominio>.sql` (DDL + `CHECK`/FK) e
  `NNNN_seed_<tabela>.sql` (dados, `ON CONFLICT DO NOTHING`), com `down` que remove
  exatamente o que o `up` inseriu. Cada seed grava a linha de `content_source_file` do
  insumo legado (path/sha256/size/rows) — proveniência auditável, não dependência.
- **Geração, não digitação.** `tools/content/gen-seed.mjs` (Node) converte os seeds e TXT
  de `origin/main` em SQL de migração. O legado (`tmserver/content/**`, `.csv`, `.txt`) é
  **insumo do gerador e base de conhecimento**, nunca runtime; depois do seed revisado, o
  arquivo não é mais necessário ao servidor.
- **Seed = migração versionada, não importador de operação.** A escolha (2026-11-08) é
  emitir `crates/wyd-db/migrations/NNNN_seed_<tabela>.sql` gerado, idempotente
  (`ON CONFLICT DO NOTHING`) e com `down` próprio, em vez de um binário de importação
  rodado à mão no deploy. Motivos: o ambiente sobe com `sqlx`/`refinery` aplicando a
  mesma sequência em qualquer máquina (nenhum passo manual que alguém esquece); o hash
  do insumo fica no cabeçalho/`content_source_file`; rollback é migração, revisável em
  PR; e não existe caminho em que o servidor dependa de arquivo. **Mitigação de
  tamanho:** um arquivo por domínio (`0009` itens, `0010` skills), `INSERT` em lotes de
  200 linhas, colunas derivadas calculadas pelo gerador — o repositório ganha ~1–2 MB de
  SQL versionado em vez de um importador + passo de operação.
- **Boot.** Depois de aplicar migrações e conferir `content_version`, `wyd-db` carrega as
  tabelas em estruturas imutáveis (`ContentTables`, `Vec`/SoA ordenados) entregues ao
  `wyd-core` pela porta `ports/content.rs`. Nada de I/O de banco dentro do tick; o load
  acontece **antes** de abrir socket.
- **Validação tipada no boot.** O banco garante PK/FK/CHECK; o loader Rust valida o que
  SQL não expressa (faixas cruzadas, vocabulário fechado, contagem mínima, referência de
  efeito `EF_*`). Inconsistência → `ContentError { tabela, pk, coluna, motivo }` (até 20
  erros) e processo sai com código != 0. Nenhum default silencioso.
- **UNVERIFIED não vira gate.** Coluna declarada *aberta* (ex.: código `EF_*` sem
  vocabulário portado, chave de `game_setting` não interpretada) é registrada como
  `Unverified` no relatório e o boot segue — invariante 5 do `AGENTS.md`.
- **Determinismo.** O boot loga `content_version` (última migração de conteúdo) e a
  contagem por tabela; o servidor nunca "descobre" conteúdo em disco, então duas máquinas
  com o mesmo banco têm o mesmo mundo.

## Consequências

- **Positivas:** fonte única (nada de base no banco + cópia em arquivo); integridade
  declarativa (`FK`/`CHECK`/índice) em vez de validação caseira; reimport e rollback viram
  migração versionada; zero parser novo na árvore; o portal já lê o mesmo banco, então
  catálogo e overlays têm uma autoridade só; segue o que `pre-release-rust.md` §3.2/§4.1
  manda.
- **Negativas:** **o servidor não sobe sem PostgreSQL** — o boot passa a depender da
  FASE-04-3 (`sqlx`/`refinery`, ADR-0006); seed grande em SQL (~17 mil linhas somadas)
  aumenta o tempo de migração; rollback de conteúdo é migração, não `git checkout`;
  insumo legado precisa de gerador confiável com `sha256` para não perder dado na
  conversão.
- **Invariantes afetados:** `.csv`/`.txt`/`.bin` **nunca** são lidos em runtime; nenhuma
  consulta de conteúdo dentro do tick (load único no boot, tabelas imutáveis em memória);
  o portal **não** escreve nas tabelas de base (só overlay auditado); tabela de conteúdo
  nova exige migração + validador + teste, não arquivo novo.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| CSV/TXT versionado em `content/` lido no boot | prática do servidor Go que a FASE 3 §3.2 manda eliminar; criaria duas autoridades (arquivo + banco) e um parser próprio para dado que o Postgres já valida |
| JSON/TOML em arquivo | mesmo problema de fonte dupla; parser só existe no Cold Path (`wyd-admin/transport/json.rs`) e `toml` exigiria dependência nova |
| Ler os `.bin` do cliente | XOR/opaco, sem leitor, não revisável em diff; serve para conferência de `item_index`, não como fonte |
| Manter os TXT do legado | é exatamente o que a decisão elimina (sem schema, sem erro por linha, UNVERIFIED misturado ao dado) |
| Seed em memória gerado no build (`include_str!`) | arquivo embutido continua sendo arquivo, sem migração, sem proveniência e exige recompilar para corrigir dado |
| Importar TXT em runtime, uma vez, e gravar no banco | o servidor passaria a depender do arquivo no primeiro boot de cada ambiente; a importação é offline (gerador → migração) |
| Importador binário de uso único, rodado por operação | exige passo manual em cada ambiente, sem `down` e sem diff em PR; o schema e o dado passariam a ter históricos separados — descartado em favor da migração gerada |

## Rastreabilidade

- Issues consumidoras: `FASE-04-2` (migrações de schema/seed), `FASE-04-3` (`sqlx` +
  carga no boot), `FASE-03-4` (validador tipado e relatório de inconsistência),
  `FASE-03-5` (autoridade usa as tabelas), `FASE-07-1` (portal lê base + overlays),
  `FASE-10-1` (fórmulas).
- Docs atualizadas nesta ADR: `docs/arquitetura/modelo-banco.md` §6,
  `docs/arquitetura/estrutura-e-padroes.md` §10 (`ports/content.rs`) e §15
  (`tools/content/` gera SQL), `docs/arquitetura/visao-pastas.md`.
- Evidência: `git show origin/main:internal/migrations/{0001_schema,seed_item_catalog,
  seed_skill_catalog,0008_platform_content}.up.sql`; `crates/wyd-db/migrations` com 6
  arquivos de jogador e zero de conteúdo; `docs/cliente/inventario.md` §3 para o status
  dos `.bin`.

## Revisão

- Revisar quando: `sqlx`/`refinery` entrarem (ADR-0006) e o custo real de boot for medido;
  alguma tabela de conteúdo passar de ~1 M linhas; surgir conteúdo por servidor (não
  global).
- Sinais de que a decisão envelheceu: alguém propondo arquivo para "facilitar edição";
  tempo de migração de seed dominando o deploy; validador Rust virando duplicata do que o
  SQL já garante.

## Emenda 2026-11-08 — sanitização, normalização e proveniência

A implementação (0007–0012) fixou três pontos que a decisão original deixava em aberto:

1. **Proveniência em tabela, não em comentário.** `content_source` (`path`, `sha256`,
   `size_bytes`, `row_count`, `loaded_by`, `transform`) guarda o byte-a-byte consumido pelo
   gerador — 12 linhas cobrem os 12 insumos. Substitui o `content_source_file` previsto
   acima. Apagar o legado não altera o banco: é auditoria, não dependência.
2. **Sanitização é verificação dura, não reparo.** UTF-8 estrito por arquivo (byte inválido
   fora de comentário **aborta**), NFC canônico, CRLF→LF, sem BOM/invisível/controle/NBSP.
   O insumo estava limpo nesses pontos e o gerador prova isso a cada execução. Exceção
   registrada: 20 nomes de `npc_template` em CP949 lido como ISO-8859-1 — a chave é
   preservada (referência de cliente e de `npc_definition`) e o coreano vai em
   `name_recovered` quando a recuperação é total (12 de 20).
3. **Chave de busca derivada.** `content_fold(text)` é `IMMUTABLE STRICT PARALLEL SAFE` e
   alimenta colunas geradas (`item_catalog.name_fold`, `region.name_fold`). Sem `unaccent`
   (STABLE, não indexável) e sem `citext` (não ignora acento). É o que permite o portal
   achar "capa de akelonia" em `Capa_de_Akelonia`, mantendo NFC no valor canônico.

Consequência para o boot: `content_version` (4) e `content_source.sha256` formam o
manifesto que o servidor loga antes do primeiro tick. Regras e mapa de colunas em
`docs/arquitetura/modelo-banco.md` §6.6–§6.7; revisão de uso do Postgres em §7.
