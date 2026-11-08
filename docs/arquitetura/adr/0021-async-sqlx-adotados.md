# ADR-0021 — `tokio` + `sqlx` adotados: pool real e save fora do tick

- **Estado:** aceita
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (issue FASE-04-3)
- **Governa:** `crates/wyd-db/` (pool, adapters `pg`, fila de save), FASE-04-3, FASE-07-1
  (transporte Cold Path), FASE-15-2 (carga de banco)
- **Substitui parcialmente:** ADR-0006 (runtime async diferido) — a decisão de "std até haver
  rede" deixa de valer para o `wyd-db`; o resto da ADR-0006 continua histórico.

## Contexto

A ADR-0006 adiou `tokio`/`sqlx` porque não havia rede para baixar crates. A consequência ficou
visível na árvore: as portas (`Accounts`, `Characters`, `Grants`), o `PoolConfig` validado e a
`DeliveryQueue` existiam, mas **nenhuma linha falava com o Postgres** — o schema era aplicado por
`psql -f` e o servidor não conseguia gravar um personagem. Ou seja: a FASE-04 estava "concluída no
papel" e a persistência não existia de fato.

Rede medida em 2026-11-08 (`index.crates.io/config.json` → 200 e download de `.crate` → 200).
Com acesso liberado, manter as portas sem adapter passou a custar mais do que implementá-las:
todo o resto do roadmap (login, painel, carga de conteúdo no boot, carga de banco na FASE-15)
depende de escrita real no banco.

## Decisão

Adotamos **`tokio` (runtime) + `sqlx` 0.8 (driver Postgres)** no `wyd-db`, como *implementação*
das portas que já existiam — não como reescrita da arquitetura:

1. **Uma dependência nova só na borda de persistência.** `wyd-core` continua sem runtime:
   invariante 1 do `AGENTS.md` (nada de I/O no tick) segue intocada, e o compilador ajuda —
   `SaveQueue::enqueue` é `try_send` síncrono, então não existe `await` possível no quente.
2. **`default-features = false`**, com `runtime-tokio`, `postgres` e `tls-rustls-ring`: sem o
   conjunto de macros/derive (`query!` exige banco no build) e sem TLS nativo (OpenSSL no host
   Windows). SQL é sempre `sqlx::query` com bind — nada de string montada com entrada.
3. **Pool dimensionado pelo `PoolConfig` existente**, com `acquire_timeout` (contenção vira
   `PoolSaturated`) e `SET LOCAL statement_timeout` por transação (o teto não vaza para a conexão
   devolvida ao pool).
4. **Fila de save própria** (`transfer/save.rs`): canal limitado do `tokio`, coalescência por
   personagem (o último snapshot vence) e dreno de Cold Path com política periódica + shutdown.

### Adições à árvore (rito §7 do `AGENTS.md`)

A árvore §12 do `estrutura-e-padroes.md` não tinha arquivo para o adapter, o pool concreto, a
fila de save nem o teste de integração. Criados, e a árvore atualizada na mesma entrega:

| Arquivo | Por que existe |
|---|---|
| `src/pool/pg.rs` | pool `sqlx` + mapeamento `sqlx::Error → DbError` (adapter) |
| `src/repo/pg.rs` | `impl Accounts/Characters/Grants for PgBackend` (adapter das portas) |
| `src/transfer/save.rs` | `SavePolicy`, `SaveQueue`, `SaveDrain`, `FlushReport` |
| `tests/pg.rs` | integração real contra Postgres descartável |

Nenhum `mod.rs` ganhou lógica: os três continuam só declarando e re-exportando.

## Consequências

- **Positivas:** o servidor grava e lê personagem de verdade; a FASE-03-4 (carga de conteúdo no
  boot) e a FASE-07-1 (BFF do portal) deixam de estar bloqueadas por dependência; o erro de
  banco é tipado e não vaza DSN.
- **Negativas:** `Cargo.lock` cresce (~90 crates de `sqlx`/`tokio`); o build passa a exigir
  toolchain de C para `ring` (rustls); `cargo test --workspace` agora pode **pular** a integração
  quando `WYD_TEST_DSN` não está definido — pulo é anunciado com `SKIP` no stderr, nunca
  fake-pass.
- **Invariantes afetados:** nenhum relaxado. O save roda em tarefa separada, então "sem await de
  banco no tick" continua verdade por construção (`enqueue` não é `async`).

## Evidência (2026-11-08)

| Medida | Valor |
|---|---|
| `cargo clippy --workspace --all-targets -- -D warnings` | exit 0 |
| `cargo build --workspace` | exit 0 (58 s com `sqlx`) |
| `cargo test --workspace` | todos os alvos de teste, **0 failed** (exit 0) |
| Integração `tests/pg.rs` (Postgres 17, 28 migrations + seed) | **6/6 ok** |
| Custo de `enqueue` no Hot Path (N=20 000, dreno real em paralelo) | **p50 500 ns · p99 1 600 ns · max 257 µs** |
| Orçamento do tick a 20 Hz | 50 ms — o enqueue fica em **0,003%** do p99 |

O `max` de 257 µs é preempção do SO no meio da amostra (o dreno gravava no banco ao mesmo tempo),
não custo do caminho: p99 é a métrica que decide.

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| continuar com `psql -f` e portas sem adapter | não grava personagem; bloqueia FASE-03-4/07/15 |
| `postgres` (crate síncrono) + thread pool próprio | reinventa pool, timeout e transação; perderia `PgPool` maduro por nenhum ganho no Cold Path |
| `sqlx` com macros (`query!`) | exige banco no build (quebra build limpo e CI sem DB) |
| `diesel`/`sea-orm` | ORM fora do estilo do projeto (SQL explícito, modelo espelhado) |
| `tokio` no `wyd-core` | violaria o invariante 1 sem necessidade: o core não faz I/O |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/FASE-04-3-persistencia-async.md` (fechada),
  `FASE-03-4` (carga no boot), `FASE-07-1` (BFF), `FASE-15-2` (carga de banco)
- Substitui: `docs/arquitetura/adr/0006-async-diferido.md` (parcial — só a parte de runtime/sqlx)
- Docs atualizadas nesta ADR: `docs/arquitetura/estrutura-e-padroes.md` §12,
  `docs/rust/01-dependencias.md`, `docs/arquitetura/contrato-persistencia.md` §7–§8,
  `docs/rust/03-progresso.md`, `roadmap-rust/index.md`
- Evidência: comandos e números na tabela acima; log da integração em `crates/wyd-db/tests/pg.rs`

## Revisão

- Revisar quando: o pool virar gargalo medido (FASE-15-2), ou a carga de conteúdo no boot exigir
  outro caminho de leitura.
- Sinais de que a decisão envelheceu: `sqlx` pedir `query!` em massa (aí o build precisa de
  banco e a decisão muda), ou `tokio` vazar para dentro de `wyd-core`.
