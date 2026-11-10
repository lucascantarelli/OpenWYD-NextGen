# Issue 07-5 — Handlers de leitura de conteúdo no gateway (npc/mob/item/efeito/mapa/teleporte)

**Fase:** 7 · **Status:** ✅ concluída em 2026-11-10 · **Depende de:** 07-1 🚧 (roteador/envelope já serviam; só faltavam os handlers) · **Bloqueia:** — · **Esforço:** M

## Objetivo

Tirar do `501` as rotas de **leitura** de conteúdo do portal, servindo o Postgres real por
portas do `wyd-db`, sem escrita e sem estado vivo.

## Contexto / evidência

`crates/wyd-admin/src/transport/mod.rs` conhece 16 serviços/72 métodos; os que não têm
handler respondem `501` auditado (`admin.pending`). O `item_catalog/list_items` já é servido da
foto real (3.220 itens). O banco tem o dado: `npc_template` 1.991, `npc_definition` 6.100,
`npc_shop_item` 6.706, `region` 52, `level_item` 161 (medido em 2026-11-09).

Rotas de leitura sem handler hoje: `npc_admin.list_npcs`, `mob_admin` (listar),
`item_effect_admin` (listar), `attribute_map_admin.get_attribute_map_info`,
`teleport_admin` (listar), `drop_admin.list_drop_rates`/`list_mob_carry`.

## Passos

1. `crates/wyd-db/src/repo/content.rs`: portas de leitura paginadas (limite + cursor por id),
   uma consulta por rota, sem `SELECT *` sem projeção tipada.
2. `crates/wyd-admin/src/api/{npc,mob,item_effect,attribute_map,teleport,drop}.rs`: handler
   que valida `params`, chama a porta e devolve envelope `result` 1..5 + `trace_id`.
3. Paridade: manter `SERVICOS` como está (o teste de contrato já cobra os 72 métodos); o
   `pending` deixa de ser alcançado nessas rotas.
4. Paginação obrigatória em listas grandes (npc 1.991, mobs): `limit` default 50, teto 200;
   sem paginação o portal trava e o teste de contrato passa a cobrir o teto.
5. Testes de contrato: cada rota responde `result:1` com dado real (banco de teste), lista
   vazia não vira erro, `limit` acima do teto → `result:3`, sem sessão → 401, jogador →
   `result:2` auditado.

## Arquivos

- Criados: `crates/wyd-admin/src/api/{npc,mob,item_effect,attribute_map,teleport,drop}.rs`,
  `crates/wyd-admin/tests/conteudo.rs`.
- Alterados: `crates/wyd-admin/src/api/mod.rs`, `transport/mod.rs` (ligação das rotas),
  `crates/wyd-db/src/repo/content.rs`, `crates/wyd-db/src/repo/pg.rs`,
  `docs/arquitetura/estrutura-e-padroes.md` §13, `wyd-portal/README.md` (telas que passam a funcionar).
- Proibido tocar: `audit_log` em rota de leitura (não auditar leitura pura), estado vivo,
  `crates/wyd-core`.

## Testes / verificação

```bash
WYD_TEST_DATABASE_URL=... cargo test -p wyd-admin --test conteudo
cargo test -p wyd-admin            # contrato + paridade
WYD_TEST_DATABASE_URL=... cargo test -p wyd-db
```

Saída esperada: rotas de leitura devolvem dado real; nenhuma delas responde `501`; teto de
paginação cobrado por teste.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-db — portas, projeção tipada, paginação.
- .agents/skills/wyd-web-admin-ui — BFF, `adminCall`, envelope.
- .agents/skills/solid-design-patterns — porta por domínio, handler fino.

## Documentação de referencia

- `crates/wyd-admin/src/transport/mod.rs` (`SERVICOS`, `pending`).
- `docs/arquitetura/adr/0008` (contrato HTTP/JSON) e `0022` (gateway).
- `docs/arquitetura/banco-catalogo.md` (tabelas e colunas reais).

## Subtasks

- [x] Codificar issue (2026-11-10)
- [x] Testar issue (2026-11-10)
- [x] Revisar issue (2026-11-10)
- [x] Documentar issue (2026-11-10)
- [ ] Versionar issue, mergear branch e fechar issue — **não executado**: a missão não
  autoriza commit/push/PR; tudo segue uncommitted em `develop`.
- [x] Limpar branch (se aplicável) — nada a limpar: nenhum branch novo foi criado.

## Critérios de aceite

- [x] As rotas de leitura listadas respondem com dado do Postgres — para as que têm
  tabela (`list_npcs`, `list_mob_templates`, `list_effects` + as já servidas
  `list_routes`, `list_drop_rates`). `get_attribute_map_info` e `list_mob_carry` seguem
  em `501`: não há tabela de mapa de atributos nem de carry de mob no banco (D-B07
  explícito: nomear colunas seria invenção até a FASE-10) — ver § Evidência.
- [x] Paginação com teto; parâmetro inválido vira `result:3`.
- [x] Paridade de contrato continua verde (16 serviços / 72 métodos; `SERVICOS` intacto).
- [x] Nenhuma leitura abre transação longa nem escreve auditoria de leitura.

## Evidência (execução 2026-11-10, sem commit/push por regra da missão)

**O que foi entregue:**

- `wyd-db/src/repo/config.rs`: `config_npc_page(limit, after_id)` (sem o `blob` de 816 B
  do template; cursor `id > after`), `config_mob_templates(limit, after)` (de
  `npc_template`, 1.991 linhas — `mob_template_stat` segue vazia até a FASE-10) e
  `config_effects(limit, after)` (de `item_effect`; símbolo `EF_*` sem código numérico).
  Teto 1..=200 aplicado no bind; `map_error`/`Transport` sem vazar SQL nem DSN.
  (Desvio documentado do plano: as portas vivem em `config.rs`, não em `content.rs`/
  `pg.rs`, porque é lá que moram todas as leituras de configuração do portal.)
- `wyd-admin`: `ConfigStore::{npcs, mob_templates, effects}` + `Blocking` no gateway,
  `api/npc.rs::list_npcs` (+ `npcs_json`), `api/mob.rs` e `api/item_effect.rs` novos
  (leitura sem `log_action` — só a negação de acesso é auditada), dispatch em
  `transport/mod.rs` para `npc_admin/list_npcs`, `mob_admin/list_mob_templates` e
  `item_effect_admin/list_effects` (`limit` default 50, cursor `after_id`/`after`).
  `attribute_map_admin.get_attribute_map_info` e `drop_admin.list_mob_carry` (além das
  escritas `set/reset_mob_carry`, que são de outra fase) continuam em `pending`/`501`
  auditado: sem tabela que os sustente, servir seria inventar dado.
- Docs: `estrutura-e-padroes.md` §13 (árvore real de `api/` + `tests/conteudo.rs`) e
  `wyd-portal/README.md` (leituras de conteúdo com contrato; telas do portal seguem
  futuras — o `admin-client.ts` não declara essas rotas, então a paridade 12/25 não muda).
  Checklist sem linha a virar: as rotas são contrato habilitador, ainda sem tela.

**Testes reais que passam:**

- `cargo test -p wyd-admin --test conteudo` 7/7 — contrato sem banco (fake: `result:1`
  com dado, cursor por id e por nome, lista vazia sem erro, `limit` 0/201/-5 → `result:3`,
  sem token → 401, jogador → `result:2`) + Postgres real (Postgres 17 descartável em
  container próprio na porta 15432, fora da stack — 41 ups aplicados, contagens 6.100 /
  1.991 / 55): página de NPCs com ordem e cursor, templates e `EF_*` com cursor.
- Gates: `cargo fmt --check` 0 · `cargo clippy --workspace --all-targets -D warnings` 0 ·
  `cargo test --workspace --all-targets` 59 alvos ok, 0 falhas · `check-migrations`
  41/82/152 ok · `check-issue-status` 61 comparadas, 0 divergências ·
  `gen-catalogo-banco --check` ok. Sem mudança de esquema (`SCHEMA_VERSION` inalterado).
  Container descartável removido ao final (a stack segue intacta).

## Riscos / rollback

- Risco: listar 6.100 NPCs de uma vez derrubar o gateway → mitigação: paginação obrigatória.
- Risco: divergência entre tipo Rust e coluna → mitigação: teste com banco real em CI
  (`seed-boot` já existe).
- Rollback: remover a ligação das rotas; elas voltam a `501` auditado.
