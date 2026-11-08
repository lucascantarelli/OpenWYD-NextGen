# Issue 03-4 — Fim dos TXT/CSV: validação de schema no boot

**Fase:** 3 · **Depende de:** FASE-04 · **Bloqueia:** 03-5 · **Esforço:** M

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

## Passos

1. [x] Inventariar os arquivos planos consumidos pelo legado (`modelo-banco.md`
   §6, incluindo procedência e lacunas).
2. Definir o formato destino (tabelas versionadas ou banco — ver FASE-04) e o
   validador de schema.
3. Implementar a carga validada no boot com relatório de erro por
   arquivo/linha/campo.
4. Teste: dado corrompido aborta com mensagem que identifica o problema.

## Arquivos

- Criados: validador de schema + testes (módulo em `wyd-core` ou `wyd-db`,
  conforme decisão).
- Alterados: docs do formato em `docs/arquitetura/` (`modelo-banco.md` §6 já
  recebeu o inventário desta issue).
- Proibido tocar: conteúdo de jogo sem motivo, default
  silencioso (proibido por regra).

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

**Status:** 🚧 (2026-11-08) — inventário de dados estáticos concluído e
registrado; falta decidir o formato destino (tabelas em `wyd-db` vs. arquivo
versionado) e implementar o validador de boot.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Zero TXT/CSV lido sem validação; inconsistência aborta com relatório claro.
- [ ] Teste de corrupção verde; evidência anexada.

## Riscos / rollback

- Risco: formato destino incompatível com ferramenta de conteúdo → mitigação:
  conversor versionado + validador antes/depois.
- Rollback: `git checkout -- crates/ docs/`.
