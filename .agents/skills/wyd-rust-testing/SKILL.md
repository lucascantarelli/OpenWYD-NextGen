---
name: wyd-rust-testing
description: Testa o servidor WYD em Rust sem o cliente gráfico (harness CPSock, goldens, determinismo, carga e bots de QA). Use ao validar mudança de protocolo/engine, reproduzir bug de pacote, montar bot de QA ou gerar carga no tick.
---

# Testes e bots Rust (harness + goldens + carga)

Norma: `docs/arquitetura/testes.md` (taxonomia, gate de infraestrutura, higiene),
`docs/arquitetura/protocolo.md`, `docs/cliente/depuracao.md` e
`docs/rust/07-ia-e-bots.md` (bots/LLM). A matriz por crate e os comandos completos estão
nesses docs; a skill guarda só o critério de escolha.

## Decisões específicas

- Prove no headless antes do cliente gráfico. Use o codec real de `crates/wyd-protocol`
  (HEADER, ofuscação, `Strict`); não reimplemente frame no teste.
- Asserte o pacote S→C e o estado. “Estado certo, pacote errado” é bug de contrato.
- Golden é byte-exato e versionado; divergência é bug, não ajuste silencioso. Fixtures
  sanitizadas (`GOLDEN`/zeros), nunca `.pcap`/dump/log.
- Determinismo antes de carga: mesmo seed → mesmo resultado; depois meça tick/carga e cite
  números. Bot usa conta dedicada e respeita rate limit.
- Teste de sistema só com gate explícito (`WYD_TEST_DSN`, socket, processo); `SKIP` nomeia
  o que faltou e não é prova. Sem `#[ignore]`/`return` mudo ou `sleep` de sincronização.

## Prova

`cargo test --workspace`; sistema conforme `docs/arquitetura/testes.md`; goldens por
`node tools/client/captura.mjs --check`. Feche com `make check`.
