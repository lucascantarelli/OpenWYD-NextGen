# Execução do roadmap Rust — ciclo, gates e documentação viva

> Fonte: `pre-release-rust.md` FASE 2 (§16 ordem estratégica) + `start-rust.md`
> (branch `develop`, planejamento antes de código). Este documento é **normativo**:
> toda issue de `roadmap-rust/issues/` é executada neste ciclo.

## 1. Ciclo obrigatório por issue (uma por vez)

1. **Análise** — ler a issue inteira, as docs de referência citadas e o código
   afetado. Nada de implementar por suposição.
2. **Design** — decidir a forma da mudança em uma frase; se for estrutural
   (fronteira de crate, dependência, formato, concorrência), abrir ADR em
   `docs/arquitetura/adr/` **antes** de codar (FASE-13).
3. **Código** — menor mudança que cumpre o objetivo, nos arquivos listados na
   issue. Fora da lista = parar e registrar por quê.
4. **Teste** — o comando de verificação da própria issue, mais o gate completo.
5. **Docs** — atualizar `docs/rust/03-progresso.md` (status + evidência), o
   status em `roadmap-rust/index.md` e qualquer doc que a mudança invalidou.
6. **Fechamento** — commit em português, branch mergeada em `develop`, issue
   marcada com a evidência anexada (log/saída).

Gate único, executado ao fim de **toda** issue:

```bash
make check      # fmt --check + clippy -D warnings + build + test
```

Sem `make` no host Windows (Git Bash), os quatro comandos `cargo` equivalentes
valem como gate (registrar qual foi usado na evidência).

## 2. Fluxo de branch

- `develop` é a branch principal de desenvolvimento (`start-rust.md`).
- Uma issue = uma branch: `fase/NN-slug` (ex.: `fase/03-1-tick`), criada de
  `develop`, mergeada de volta com `--no-ff`.
- Commits pequenos, descritivos, em português, um por passo relevante.
- Proibido: force-push em `develop`/`main`, commit de segredo, `client/`,
  `target/` (nunca versionar build).

## 3. Definition of Done por tipo de issue

| Tipo | Pronto quando |
|---|---|
| Código | gate verde + teste da issue + doc de progresso atualizado |
| Documento | arquivo criado/atualizado + links resolvendo + revisão fria |
| Processo | regra escrita em `docs/rust/02-execucao.md` + referenciada no aceite das issues seguintes |
| Auditoria | 100% dos itens classificados, com evidência ou marcação **hipótese** |
| ADR | status explícito (proposta/aceita/substituída) + issues consumidoras vinculadas |

## 4. Revisão consolidada (a cada 10 issues concluídas)

Checklist fixo, registrado em `docs/rust/03-progresso.md`:

1. `make check` + `cargo test --all-targets`.
2. `cargo bench` comparado ao baseline anterior (sem baseline = criar agora). Baseline
   registrado na FASE-05-1 (2026-11-08): IA de NPC 88,6–205,9 µs/tick com 500 NPCs + 50
   jogadores; balde de transporte 1,4–8,1 ns/op e porte 234–595 ns por handshake.
3. Testes de integração entre módulos tocados nas 10 issues.
4. ADRs pendentes/propostas que precisam ser decididas.
5. Docs vivos: `03-progresso.md`, `roadmap-rust/index.md`, `docs/arquitetura/`.
6. Débito aberto pelas 10 issues (TODO registrado, nunca silencioso).

## 6. Release e imagem (FASE-05-2, ADR-0017)

`[profile.release]` do workspace: `lto = "thin"`, `codegen-units = 1`,
`strip = "symbols"`, `panic = "abort"`. Não existe `[profile.bench]`: o Cargo ignora `panic`
nesse perfil e declarar o bloco só gera warning de manifesto.

| Medida (2026-11-08, host Windows/Docker 29.9.2) | Valor |
|---|---|
| `wyd-launcher` release antes das flags | 323.072 bytes |
| `wyd-launcher` release com LTO thin + cgu=1 + strip + abort | **296.448 bytes** (−8,2%) |
| `cargo build --release --workspace` | exit 0 (rebuild completo 13,89 s) |
| imagem `wyd/launcher:check` (distroless `cc-debian12:nonroot`) | **37.847.971 bytes / 36,1 MiB**, 20 camadas, `User=nonroot:nonroot` |
| smoke no container (`check /jogo /jogo/manifest.toml`) | exit 0 — `1 íntegros, 0 com problema` |
| conteúdo da imagem | 1.792 arquivos, único binário de app `usr/local/bin/wyd-launcher`, 0 `/bin/sh`, 0 `rustc`/`cargo`/`gcc`/`apt`/`dpkg` |
| bench IA (`wyd-bench tick`) | 85,6–189,7 µs/tick (p99 139,9–397,5) contra baseline 205,86 — sem regressão |
| bench transporte | `TokenBucket` 2,7–2,8 ns/op; `ConnectionGate` 100,1–100,8 ns/handshake |

Comandos: `make release`, `make docker`, `make docker-smoke` (exige Docker; o smoke usa
`target/smoke` com manifesto gerado localmente, fora do git). Build do portal valida sintaxe
por `docker build --check`; o build completo é do artefato do portal.

## 7. Telemetria (FASE-06-1, ADR-0018)

**Log estruturado:** `tracing` + `tracing-subscriber` (`fmt`, sem `env-filter`) em `wyd-core`.
Nível por `WYD_LOG` (fallback `RUST_LOG`; `error|warn|info|debug|trace|off`, default `info`).
O binário chama `wyd_core::telemetry::init_from_env()` uma vez no boot.

**Métricas:** registro em atomics (`wyd_core::metrics::Metrics`) com histograma de tick de
buckets fixos (0,5 ms–1 s) e exporter `MetricsServer` servindo `GET /metrics` em texto Prometheus
0.0.4 em thread própria (`std::net::TcpListener`). O laço do servidor faz, por tick:
`metrics.record_tick(micros, entidades)`; a amostra de memória (`mem::sample_rss_bytes()`, só Linux)
entra a cada ~20 ticks, fora do quente. Rede: `wyd_net::metrics::NET` publica um fragmento que
entra por `Metrics::render(&net.render_prometheus())`; **exige `--features metrics`** no build.

| Métrica | Tipo |
|---|---|
| `wyd_ticks_total`, `wyd_handshakes_total`, `wyd_packets_total`, `wyd_packets_rejected_total{reason}`, `wyd_memory_samples_total` | counter |
| `wyd_entities_active`, `process_resident_memory_bytes`, `wyd_connections_pending`, `wyd_sessions_open` | gauge |
| `wyd_tick_duration_microseconds` | histogram (P50/P95/P99 por bucket) |

**Smoke (sem `make`, host Windows):**

```bash
WYD_LOG=debug cargo test -p wyd-core --test telemetry -- --nocapture
WYD_LOG=info  cargo test -p wyd-core --test telemetry -- --nocapture   # nenhuma linha DEBUG
cargo test -p wyd-net --features metrics --test metrics -- --nocapture # fragmento de rede
```

**Medições desta passada (2026-11-08):** log `INFO resumo de telemetria ticks=40 … rss_bytes=0`
(RSS 0 = sem `/proc` no Windows; em Linux é lido de `/proc/self/statm`), `GET /metrics` com
1.257 bytes e `wyd_ticks_total 40`; `Cargo.lock` 7→24 pacotes; `wyd-launcher` release 296.448 bytes
e imagem distroless 37.847.971 bytes (inalterados — o launcher não linka `wyd-core`); benches tick
81,27–82,22 µs/tick (p99 88–116), `TokenBucket` 2,7 ns sem a feature e 24,3–24,8 ns com ela,
`ConnectionGate` 98,1 ns.

## 5. Documentação viva (regra permanente)

- **`docs/rust/03-progresso.md`** é o painel de execução: muda a cada issue.
- **`roadmap-rust/index.md`** carrega o status de cada issue.
- **`AGENTS.md` §0** é a fonte única de invariantes; se uma execução mudar um
  invariante, o ADR e o `AGENTS.md` mudam na mesma entrega.
- Documento que descreve comportamento deve ser atualizado **na entrega que
  muda o comportamento**, nunca "depois". Doc desatualizada é defeito.
- Toda afirmação técnica não confirmada leva a marca **hipótese** (ou
  UNVERIFIED, no caso de protocolo), como em `docs/`.
