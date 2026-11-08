# Issue 04-3 — Persistência assíncrona fora do tick

**Fase:** 4 · **Depende de:** 04-2 · **Bloqueia:** FASE-07 · **Esforço:** L

## Objetivo

Implementar o pool async (`sqlx`) com transações curtas e fila de persistência,
sem nenhum I/O bloqueante no tick loop.

## Contexto / evidência

`pre-release-rust.md` FASE 4 §4.2: persistência async em segundo plano, sem I/O
bloqueante no tick. Regra Hot/Cold (`AGENTS.md`): banco é Cold Path.

## Passos

1. Implementar pool em `crates/wyd-db/src/` (timeouts, limite de conexões,
   transações curtas por operação).
2. Implementar a fila de persistência (drain periódico pelo core, sem lock
   global — canal/fila dedicada).
3. Definir política de save (periódico, on-demand, on-shutdown) e documentar.
4. Teste: latência de tick inalterada sob carga de saves (números anexados).

## Arquivos

- Criados: pool, fila e testes em `crates/wyd-db/src/`.
- Alterados: `crates/wyd-db/Cargo.toml` (sqlx via ADR).
- Proibido tocar: `crates/wyd-core/src/tick/` (nenhum await de banco no
  quente), segredos.

## Testes / verificação

```bash
cargo test -p wyd-db
make check
```

Saída esperada: `test result: ok`; teste de carga de saves sem degradar o tick.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-db - pool e transações.
- .agents/skills/wyd-rust-core - interface de drain pelo core.
- .agents/skills/rust-code-quality - timeouts, erros tipados.

## Documentação de referencia

- pre-release-rust.md (FASE 4 §4.2, §0.2 Hot/Cold).
- roadmap-rust/issues/FASE-04-2-migrations.md - schema base.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [ ] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Zero await de banco no tick; fila drenada com política documentada.
- [x] Save-on-shutdown sem perda (teste dedicado).

> Nota (2026-11-07): base pronta na FASE-04 (traits `Accounts`/`Characters`/
> `Grants`, `PoolConfig`, `DeliveryQueue` em memória). Resta a implementação
> `pg` + pool async — ver ADR-0006 (exige rede + vendor — fora do escopo desta entrega).

## Progresso 2026-11-08 — pool, adapters e save (issue fechada)

- **ADR:** `docs/arquitetura/adr/0021-async-sqlx-adotados.md` — `tokio` + `sqlx` 0.8 adotados no
  `wyd-db`; substitui parcialmente a ADR-0006 (que previa exatamente esta ADR filha). Rede
  revalidada antes de decidir (`index.crates.io` 200, download de `.crate` 200).
- **Arquivos criados** (e árvore §12 de `estrutura-e-padroes.md` atualizada na mesma entrega):
  `src/pool/pg.rs`, `src/repo/pg.rs`, `src/transfer/save.rs`, `tests/pg.rs`. Nenhum `mod.rs` com
  lógica.
- **Passos da issue:** (1) pool com `acquire_timeout` + `SET LOCAL statement_timeout` e transações
  curtas ✅; (2) fila de save drenada por tarefa Cold Path, sem lock global ✅; (3) política
  periódica/por lote/on-shutdown documentada em `contrato-persistencia.md` §7.1 ✅; (4) custo do
  lado do tick medido sob carga de saves ✅.
- **Números:** `enqueue` N=20 000 com dreno gravando em paralelo → p50 **500 ns**, p99 **1 600 ns**,
  max 257 µs (0,003% do tick de 50 ms no p99). Integração real **6/6 ok** em Postgres 17 com 28
  migrations + seed. `cargo test --workspace` 36 suítes / 0 failed; clippy `-D warnings` exit 0.
- **API ajustada:** `create` das duas portas virou `create_account`/`create_character` (E0034 no
  mesmo adapter); `DbError` ganhou `NotFoundKey`, `SaveQueueFull`, `SaveTimeout`.
- **Fora do escopo (decisão registrada):** runner embutido `sqlx::migrate!`/`refinery` — migrations
  continuam por `tools/content/migrate-cycle.mjs`/`psql -f`.

## Riscos / rollback

- Risco: perda de dados em crash entre saves → mitigação: janela de perda
  documentada + teste de shutdown.
- Rollback: `git checkout -- crates/wyd-db/`.
