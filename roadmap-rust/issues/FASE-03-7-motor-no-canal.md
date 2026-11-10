# Issue 03-7 — Motor no canal: tick real dentro do `wyd-server`

**Fase:** 3 · **Depende de:** 03-1 ✅, 03-6 ✅, 04-3 ✅ · **Bloqueia:** 03-8, 03-9 · **Esforço:** M
**Status:** ✅ entregue em 2026-11-09

## Objetivo

Fazer o processo do canal **rodar o motor**: um `World` de `wyd-core` por processo, tick
determinístico a 20 Hz (ou 50 Hz configurável), métricas reais e desligamento limpo — sem I/O
dentro do tick.

## Contexto / evidência

Medição desta revisão (2026-11-09): `crates/wyd-server/Cargo.toml` **não depende de
`wyd-core`** e `crates/wyd-server/src/main.rs` não tem laço de tick. O processo entrega
CPSock, login, seleção de personagem, status e `/metrics` do processo, mas o mundo não simula.

Consequências medidas:

- `wyd_ticks_total`/`wyd_tick_duration_microseconds`/`wyd_entities_active` **não são
  publicados** — e isso está correto: a ADR-0022 §7 proíbe publicá-los zerados (“seria
  mentira”). O painel do motor fica vazio por ausência de loop, não por falta de dashboard;
- `FASE-03-5`/`03-6` provam as regras em teste, mas nenhuma delas roda em processo;
- os benches de tick (88,6–205,9 µs) medem a lib, não o serviço.

## Passos

1. `crates/wyd-server/Cargo.toml`: dependência `wyd-core` (sem features novas).
2. `crates/wyd-server/src/world.rs`: dono do `World`/`ComponentStore` do canal, com
   `Tick` (acumulador + clamp) e `Clock` reais; **um por processo**, sem `mutex` global.
3. `main.rs`: fio de tick com desligamento por sinal (SIGINT/SIGTERM via `ctrlc` handler
   simples em std é aceitável; nada de `tokio` no tick), publicando por tick
   `metrics.record_tick(micros, entidades)` — as métricas de motor passam a ser **reais**.
4. Nenhum I/O no laço: rede e banco continuam nas threads atuais; o tick lê/escreve apenas
   estado em memória e enfileira eventos para a borda.
5. `WYD_TICK_HZ` (default 20, faixa 20–50) validado no boot; valor fora da faixa é erro fatal.
6. Teste de processo: subir o binário com mundo vazio e provar `wyd_ticks_total > 0` e
   p99 dentro do orçamento após 2 s (`tests/tick_processo.rs`, sem banco: `--no-db` para
   testes? ver risco).
7. Bench **do serviço** (não só da lib) anexado: 0 entidades e 1 000 entidades.

## Arquivos

- Criados: `crates/wyd-server/src/world.rs`, `crates/wyd-server/tests/tick_processo.rs`.
- Alterados: `crates/wyd-server/Cargo.toml`, `crates/wyd-server/src/main.rs`,
  `crates/wyd-server/src/lib.rs`, `docs/arquitetura/estrutura-e-padroes.md` §16,
  `docs/rust/05-stack-docker.md` §1/§4, `docs/rust/03-progresso.md`.
- Proibido tocar: `crates/wyd-core/` (regra de jogo fica lá; aqui só se **chama**),
  persistência no tick, `client/`.

## Testes / verificação

```bash
cargo test -p wyd-server --test tick_processo
cargo bench -p wyd-core --bench tick
curl -s localhost:9898/metrics | grep -E 'wyd_ticks_total|wyd_entities_active'
cargo clippy --workspace --all-targets -- -D warnings
```

Saída esperada: `wyd_ticks_total` crescendo, `wyd_entities_active 0`, tick p99 abaixo de 10%
do orçamento com mundo vazio.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core — tick, determinismo, sem alocação quente.
- .agents/skills/wyd-rust-obs — métricas do motor (nomes já fechados na FASE-06-1).
- .agents/skills/software-architecture — raiz de composição (ADR-0022 §1).

## Documentação de referencia

- `docs/arquitetura/adr/0022-binarios-de-servico.md` §1/§7 (raiz de composição e canais).
- `docs/rust/02-execucao.md` §7 (telemetria) e `crates/wyd-core/src/tick/scheduler.rs`.
- `AGENTS.md` §3 invariante 1 (sem I/O no tick).

## Concluído (2026-11-09)

- `crates/wyd-server/src/world.rs` (novo): `Engine` dono do `ComponentStore` + `Grid` +
  `Scheduler`, `advance(elapsed_ms)` medindo cada passo com `Metrics::record_tick`, spawn e
  despawn que mantêm armazém e grade coerentes (posição fora do mapa é `Err`, sem meia-entrada),
  e `EngineThread` com parada por `AtomicBool` + `join` no `Drop`.
- `main.rs`: `WYD_TICK_HZ` validado no boot (20–50; fora da faixa é erro fatal com exit 2),
  registro do motor anexado ao `ServiceMetrics` **antes** do primeiro scrape, thread
  `wyd-tick` subindo antes do `serve_listener` e `stop_and_join` no retorno.
- `metrics.rs`: `/metrics` passou a publicar `wyd_ticks_total`,
  `wyd_tick_duration_microseconds` e `wyd_entities_active` vindos do registro real; o ADR-0022 §7
  foi atualizado para registrar que deixaram de ser “lacuna declarada”.
- `Cargo.toml`: dependência `wyd-core` + `[[bench]] tick`; `benches/tick.rs` mede o **serviço**.

**Evidência:**

| Verificação | Resultado |
|---|---|
| `cargo test -p wyd-server --test tick_processo` | 4/4 (tick contínuo, scrape com motor, rollback de spawn, Hz inválido) |
| `cargo bench -p wyd-server --bench tick` (release) | mundo vazio **0,38 µs/tick**; 1 000 entidades **0,73 µs/tick** = **0,001%** do orçamento de 50 ms |
| `/metrics` de processo real (portas alternativas, banco da stack, 50 Hz) | **174 ticks em ~3,5 s**, `wyd_tick_duration_microseconds_count 174`, `wyd_entities_active 0`; boot `schema applied=0 skipped=31` |
| `cargo clippy --workspace --all-targets -D warnings` / `fmt --check` | 0 / 0 |

**O que ficou para as issues donas:** personagem entrando como entidade + save na saída (03-8);
uso da grade para visibilidade e `ESCENE_FIELD` (03-9). O laço não toca rede, banco nem
`tokio`; a borda continuará entregando intenção por fila na 03-8 (nenhum mutex global).

**Limitação declarada:** o teste de processo roda o `EngineThread` **no harness**, não o
binário inteiro — subir o binário exige `DATABASE_URL` e as migrations, que já são cobertas
pelo healthcheck do compose. A prova do `/metrics` foi feita rodando o binário localmente em
portas alternativas (sem reiniciar os canais da stack).

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] O processo do canal executa tick contínuo e publica métricas de motor reais.
- [x] Nenhuma chamada de rede/banco dentro do laço de tick (revisão + `grep` no módulo).
- [x] Desligamento limpo não deixa thread pendurada nem tick órfão.
- [x] Boot com `WYD_TICK_HZ` inválido falha com mensagem clara.

## Riscos / rollback

- Risco: tick competindo com a thread de aceite por CPU em máquina de dev → mitigação:
  `WYD_TICK_HZ` configurável e aflinity/prioridade só em produção (documentado).
- Risco: teste de processo exigir `DATABASE_URL` → mitigação: modo `--no-db` explícito que
  sobe só o tick, ou teste que pula com aviso como os demais.
- Rollback: remover o laço (`git checkout -- crates/wyd-server`); conexão/login/status seguem
  funcionando como hoje.
