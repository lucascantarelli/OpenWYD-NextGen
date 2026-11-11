---
name: wyd-rust-testing
description: Testa o servidor WYD em Rust sem o cliente gráfico (harness CPSock, goldens, determinismo, carga e bots de QA). Use ao validar mudança de protocolo/engine, reproduzir bug de pacote, montar bot de QA ou gerar carga no tick.
---

# Testes e bots Rust (harness + goldens + carga)

Sucessora Rust de `wyd-testing-bots`. Referências: `pre-release-rust.md` FASE 8/10/15,
`roadmap-rust/issues/FASE-09.md`, `docs/arquitetura/protocolo.md`, `docs/cliente/pacotes.md`.
Taxonomia, nomes de arquivo e higiene: `docs/arquitetura/testes.md` + skill
`rust-test-quality` (unidade/integração/contrato/golden/sistema).

## Regras

1. **Prove no headless primeiro.** Só abra o cliente depois do harness verde; se o harness passa e o cliente falha, o problema é cliente/rede.
2. **Use o codec real, nunca reimplemente.** Monte frames com `crates/wyd-protocol` (HEADER 12 bytes, ofuscação `pKeyWord`, `Strict`); asserte o campo do pacote S→C, não só o estado (classe de bug "estado certo, pacote errado").
3. **Goldens byte-exatos para protocolo.** Fixtures em `fixtures/cliente/*.json` validadas por `tools/client/captura.mjs --check`; divergência = bug, nunca ajuste silencioso de golden (`crates/wyd-protocol/tests/golden_pacotes.rs`).
4. **Determinismo antes de carga.** Mesmo seed → mesmo resultado (`crates/wyd-core/tests/integracao_determinism.rs`, `crates/wyd-sim/tests/integracao_determinism.rs`); só depois meça tick/carga e cite números.
5. **Bots com conta dedicada e limite.** Conta de bot dedicada, rate-limit do servidor respeitado (ver `crates/wyd-net/src/limits/`); bot descobre bug, teste reproduz sem rede.
6. **Nunca relaxe asserção para passar.** Sem `#[ignore]`, sem skip mudo; todo gate de sistema imprime `SKIP:` com as variáveis ausentes (skill `rust-test-quality` §3). Golden novo exige motivo no commit.

## Fluxos

- Unidade/integração/contrato/golden: `cargo test --workspace` (tudo o que não exige infraestrutura). Bytes no fio: `cargo test -p wyd-protocol -p wyd-net`.
- Sistema (opt-in): `WYD_TEST_DSN=... cargo test -p wyd-db --test sistema_pg`; no Windows use `-j 1` em `wyd-server` para evitar `LNK1102`.
- Goldens: `node tools/client/captura.mjs --gen && node tools/client/captura.mjs --check` (5/5 C→S hoje; S→C quando 08-6 liberar captura).
- Carga: benches por crate (`cargo bench --workspace`) + `wyd-sim` em lote; tick 20–50 Hz sem alocação quente, sem lock global.
- Aceitação: login → criar char → movimento → PvE → level-up → loja → cargo → trade → quest → refino → relogin (ver `docs/cliente/depuracao.md` §3).

## Higiene

- Sanitizar fixtures (conta/senha → `GOLDEN`/zeros, `sanitized: true`); nunca versionar `.pcap`, dumps ou logs.
- Não reiniciar stack no meio de teste alheio; derrubar só o que subiu; banco intacto (sem `down -v` sem pedido).
