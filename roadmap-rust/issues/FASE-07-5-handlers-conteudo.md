# Issue 07-5 — Handlers de leitura de conteúdo no gateway (npc/mob/item/efeito/mapa/teleporte)

**Fase:** 7 · **Depende de:** 07-1 🚧 · **Bloqueia:** — · **Esforço:** M

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

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] As rotas de leitura listadas respondem com dado do Postgres.
- [ ] Paginação com teto; parâmetro inválido vira `result:3`.
- [ ] Paridade de contrato continua verde (16 serviços / 72 métodos).
- [ ] Nenhuma leitura abre transação longa nem escreve auditoria de leitura.

## Riscos / rollback

- Risco: listar 6.100 NPCs de uma vez derrubar o gateway → mitigação: paginação obrigatória.
- Risco: divergência entre tipo Rust e coluna → mitigação: teste com banco real em CI
  (`seed-boot` já existe).
- Rollback: remover a ligação das rotas; elas voltam a `501` auditado.
