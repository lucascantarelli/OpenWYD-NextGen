# Issue 07-6 — Handlers de operação no gateway (rates/daily/donate/evento/admin de conta)

**Fase:** 7 · **Depende de:** 07-1 🚧, 07-3 ✅ · **Bloqueia:** — · **Esforço:** M

## Objetivo

Tirar do `501` as rotas de **operação** que hoje só existem no contrato: rates de conteúdo,
recompensa diária, donate/saldo, evento de mundo, receita e administração de conta — cada
escrita com auditoria e sem tocar estado vivo.

## Contexto / evidência

Rotas declaradas no portal e sem handler (respondem `501` auditado): `content_rates`,
`daily_reward_admin`, `donate_admin`, `donate_revenue_admin`, `world_event_admin`,
`account_admin.list_accounts`/`get_account`, `character_admin.grant_item`. O padrão de escrita
auditada já existe em `api/moderation.rs`/`api/economy.rs` (`PersistentSink` → `audit_log`).
O invariante 7 proíbe escrever personagem vivo: item/nível vão por `delivery_queue`.

## Passos

1. `content_rates`: `get_rates` (leitura de `game_setting`/`game_matrix`) e `upsert_rate`/
   `reset_rate` com `content_source` e auditoria; valor fora da faixa → `result:3`.
2. `daily_reward_admin`: CRUD em tabela de recompensa com `enabled`; nunca entrega direto.
3. `donate_admin`: CRUD de loja + `credit_donate_balance` **como grant na `delivery_queue`**
   (nunca `UPDATE` em saldo de personagem), idempotência por chave.
4. `donate_revenue_admin`: leitura agregada (sem PII além do necessário).
5. `world_event_admin` / `eventos`: estado do evento como linha de configuração; efeito só no
   próximo boot/tick que lê o dado (nenhum push para o Hot Path).
6. `account_admin.list_accounts`/`get_account`: projeção sem hash/PIN/segredo.
7. `character_admin.grant_item`: enfileira item (`delivery_queue`), recusa se o inventário do
   Hot Path ainda não consome (resposta explícita “fila apenas”), sem simular entrega.
8. Testes: cada escrita grava `audit_log` com autor da sessão; jogador recebe `result:2`;
   grant repetido devolve o mesmo `grant_id`; nenhuma rota altera tabela de personagem vivo.

## Arquivos

- Criados: `crates/wyd-admin/src/api/{rates,daily,donate,world_event}.rs`,
  `crates/wyd-admin/tests/operacao.rs`.
- Alterados: `crates/wyd-admin/src/api/{mod,accounts,grants}.rs`, `transport/mod.rs`,
  `crates/wyd-db/src/repo/{content,economy,grants}.rs`, `crates/wyd-db/migrations/00NN_*.sql`
  (só se faltar tabela), `docs/arquitetura/banco-catalogo.md`, `wyd-portal/README.md`.
- Proibido tocar: `characters.gold`/inventário vivo por SQL direto, `crates/wyd-core`.

## Testes / verificação

```bash
WYD_TEST_DATABASE_URL=... cargo test -p wyd-admin --test operacao
cargo test -p wyd-admin -p wyd-db
node tools/content/check-migrations.mjs      # se houver migration nova
```

Saída esperada: rotas de operação respondem `result:1` com efeito auditado; grant de item é
fila, não mutação; jogador nunca passa.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-game-admin — grants, papéis, auditoria.
- .agents/skills/wyd-rust-db — migration, ledger, idempotência.
- .agents/skills/wyd-web-admin-ui — BFF e `adminCall`.

## Documentação de referencia

- `AGENTS.md` §3 invariante 7 (a web nunca escreve estado vivo).
- `docs/arquitetura/adr/0023` (moderação/auditoria/economia) e `0022` (gateway).
- `docs/rust/08-checklist-funcionalidades.md` 11.2/11.3.

## Entrega (2026-11-10)

Escopo ajustado ao que o banco sustenta (D-B07: sem `ALTER TABLE` no `up`):
`content_rates` já tinha handler (fase anterior) e não reabriu; `donate_revenue_admin/*`
e `daily_reward_admin/set_reward_item_enabled` seguem `501` auditado — não há tabela
nem coluna que os sustente e a série proíbe inventar. `credit_donate_balance` credita
`account_wallet` (saldo de **conta**, nunca personagem vivo) com idempotência pela tabela
própria `donate_credit` (`0042_credito_donate.sql`, sem `ALTER`); `grant_item` enfileira
na `delivery_queue` e o dreno entrega via `claim_item` (slot livre de `Carry` 16–79;
mochila cheia pende com `attempts`, nunca duplica nem perde).

## Subtasks

- [x] Codificar issue (2026-11-10)
- [x] Testar issue (2026-11-10)
- [x] Revisar issue (2026-11-10)
- [x] Documentar issue (2026-11-10)
- [ ] Versionar issue, mergear branch e fechar issue (proibido nesta entrega: sem commit/push)
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Rotas de operação deixam de responder `501` (2026-11-10: daily/donate/evento/
  contas/`grant_item` com handler; `501` só onde não há tabela — ver Entrega).
- [x] Toda escrita audita autor, alvo e antes/depois quando aplicável (2026-11-10:
  `log_action` em cada escrita; leitura não audita, como em 07-5).
- [x] Item/saldo só por `delivery_queue`, com idempotência (2026-11-10: `grant_item` é
  fila; `claim_item` drena; `donate_credit` impede crédito duplo; teste falha se
  `characters`/`character_items` mudarem fora do dreno).
- [x] Nenhuma rota lê ou devolve hash/PIN/token (2026-11-10: `AccountListRow` sem
  segredo; teste asserta ausência de `password`/`pin` no JSON).

## Riscos / rollback

- Risco: `credit_donate_balance` virar escrita direta em personagem → mitigação: teste que
  falha se `characters` mudar.
- Risco: rate nova alterar economia sem revisão → mitigação: `content_source` obrigatório e
  valor default explícito.
- Rollback: desligar as rotas (voltam a `501`) sem migration destrutiva.
