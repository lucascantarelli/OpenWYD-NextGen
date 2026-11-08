# Issue 05-1 — Pipeline CI (fmt, clippy, test, bench, audit, deny)

**Fase:** 5 · **Depende de:** FASE-01 · **Bloqueia:** 05-2 · **Esforço:** M
**Estado:** ✅ entregue em 2026-11-08

## Objetivo

Pipeline verde e espelhável localmente (`make ci`) com fmt, clippy, testes,
benches, `cargo audit` e `cargo deny`.

## O que foi entregue

| Peça | Conteúdo |
|---|---|
| `.github/workflows/rust.yml` | Job `rust`: `fmt --all --check`, `clippy --workspace --all-targets -D warnings`, `build`, `test --workspace --all-targets`; job `bench`: compila todos os benches e roda os dois medidos; job `supply-chain`: instala `cargo-audit`/`cargo-deny` via `taiki-e/install-action` e roda `cargo audit` + `cargo deny check`. `deny.toml` e `Makefile` entraram nos `paths` do gatilho |
| `deny.toml` | Política v2: advisories (yanked em `warn`), lista fechada de licenças, `wildcards = "deny"`, `multiple-versions = "warn"`, registro permitido só o crates.io, `private = { ignore = true }` para as crates `publish = false` do jogo |
| `Makefile` | `clippy` passou a `--all-targets`; novos `test-all`, `bench-build`, `deny`; `ci` = `check test-all bench-build audit deny` (espelho do workflow) |
| `crates/wyd-core/benches/tick.rs` | Custo da IA de NPC no orçamento do tick (ADR-0016): 500 NPCs + 50 jogadores, falha o job acima de 10% do tick |
| `crates/wyd-net/benches/limits.rs` | Custo dos limites de transporte (balde por conexão e porte de handshake), `harness = false`, sem dependência nova |
| Manifestos | `publish = false` nas 7 crates e `version = "0.1.0"` nas dependências internas por path (sem wildcard) |

## Evidência medida (árvore final, 2026-11-08)

```text
cargo test --workspace                  exit 0 — 30 alvos verdes, 0 falhas
cargo test --workspace --all-targets    exit 0 — 23 alvos verdes (inclui doc-tests/benches)
cargo clippy --workspace --all-targets  exit 0 — 0 erro, 0 warning
cargo fmt --all --check                 exit 0
cargo audit                             exit 0 — 7 crate dependencies, 1422 advisories carregados
cargo deny check                        exit 0 — advisories ok, bans ok, licenses ok, sources ok
                                          (só warnings `license-not-encountered` da lista fechada)
cargo bench -p wyd-core --bench tick    exit 0 — 205,86 µs/tick (p99 420,10) = 0,412% do tick
                                          de 50 ms (p99 0,840%); piso sem jogador 29,14 µs
cargo bench -p wyd-net --bench limits   exit 0 — TokenBucket 8,1 ns/op; ConnectionGate
                                          234,3 ns por handshake
check-migrations                        exit 0 — 24 migrations, 59 tabelas, 140 INSERTs
gen-catalogo-banco --check              exit 0 — catálogo em dia
verify-schema                           exit 0 — 59 tabelas, 457 colunas, 59 PKs
migrate-cycle                           exit 0 — up/down/up idêntico, 50,9 s
```

**Prova de que o pipeline falha ante vulnerabilidade** (passo 4 da issue): foi gerado um
lockfile descartável com `time 0.1.44` (RUSTSEC-2020-0071) fora do workspace e
`cargo audit --file .tmp-audit-proof/Cargo.lock` saiu com **código 1** apontando
`error: 1 vulnerability found!` e o ID do advisory. O diretório temporário foi removido
depois da prova; nenhum artefato ficou na árvore.

Ferramentas instaladas no host para a verificação local: `cargo-audit 0.22.2` e
`cargo-deny 0.20.2` (o CI instala as mesmas via `taiki-e/install-action`, sem depender do host).

## O que segue fora do escopo / UNVERIFIED

- O job de CI roda em `ubuntu-24.04`; o host de desenvolvimento é Windows. A paridade foi
  verificada pelos mesmos comandos, não por execução do workflow em runner.
- `multiple-versions = "warn"` é decisão de revisão: o primeiro crate externo (tokio/sqlx)
  pode exigir ADR se trouxer duas versões de uma dependência transitiva.
- Sem dependência externa hoje, `licenses`/`advisories` não exercitam a lista de licenças
  (daí os warnings `license-not-encountered`); isso deixa de valer no primeiro ADR de
  dependência.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - gates, fmt/clippy, dependências e audit.
- .agents/skills/github-actions-docs - sintaxe do workflow.
- .agents/skills/rust-code-quality - lints como erro.

## Documentação de referencia

- docs/rust/02-execucao.md (§1 ciclo, §4 revisão consolidada com baseline de bench)
- docs/rust/01-dependencias.md
- pre-release-rust.md FASE 5 §5.1
- docs/arquitetura/adr/0016-ia-npc-deterministica.md (bench da IA)

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue (árvore só com alterações locais, por instrução)
- [ ] Limpar branch (se aplicável)
