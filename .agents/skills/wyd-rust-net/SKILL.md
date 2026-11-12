---
name: wyd-rust-net
description: Desenvolve o transporte async do WYD em Rust (crates/wyd-net + wyd-protocol): tokio/io_uring, buffers zero-copy, parsers com validação estrita de limites, rate limiting e hardening de handshake. Use ao mexer em rede, protocolo CPSock, opcodes ou serialização binária.
---

# Transporte Rust (wyd-net)

Norma: `docs/arquitetura/protocolo.md` (HEADER, ofuscação, opcodes, payloads),
`docs/arquitetura/estrutura-e-padroes.md` §3 e §11 (árvores) e ADR-0005. A árvore e a
tabela de opcodes são normativas; não as copie para a skill.

## Decisões específicas

- Entrada de rede é hostil: valide tamanho, opcode e limites antes de interpretar; erro
  tipado, sem `panic!`/`unwrap`. O teto de pacote é explícito (`MAX_PACKET_BYTES`).
- Layout e opcode seguem `crates/wyd-protocol/src/wire/generated/opcode_table.rs` +
  `protocolo.md` §4. Qualquer mudança de fio exige golden byte-exato.
- Caminho quente não aloca buffer por pacote; `Strict` é a política de checksum da stack
  nova. Não relaxe a validação para “aceitar” um cliente desconhecido.
- Rate limit/backpressure são observáveis; telemetria é `tracing`/métricas, não `println!`.

## Prova

`cargo test -p wyd-net -p wyd-protocol`; depois `make check`.
