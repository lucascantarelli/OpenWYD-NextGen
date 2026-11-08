---
name: wyd-rust-net
description: Desenvolve o transporte async do WYD em Rust (crates/wyd-net + wyd-protocol): tokio/io_uring, buffers zero-copy, parsers com validação estrita de limites, rate limiting e hardening de handshake. Use ao mexer em rede, protocolo CPSock, opcodes ou serialização binária.
---

# Transporte Rust (wyd-net)

Sucessora parcial da `wyd-server-dev` para rede. Referências:
`pre-release-rust.md` FASE 8, `roadmap-rust/issues/FASE-08.md`.

## Regras

1. **Validar antes de parsear.** Tamanho, opcode e limites primeiro; rejeitar com
   erro tipado, nunca `panic!`/`unwrap` em input de rede.
2. **Zero-copy e sem alocação no caminho quente.** Layouts `#[repr(C)]`/`zerocopy`,
   validação de limites; `MAX_PACKET_BYTES` como teto explícito.
3. **Servidor é autoridade.** Cliente nunca dita posição/dano/cooldown; pacotes
   autoritativos S→C usam `ESCENE_FIELD` (30000).
4. **DoS/rate-limit desde o handshake.** Backpressure e filas observáveis
   (ver FASE-06); sem `println!` como telemetria.

## Estrutura do crate (obrigatória)

`wyd-protocol` segue `docs/arquitetura/estrutura-e-padroes.md` §3 (ADR-0002) e
`wyd-net` segue o §11 do mesmo documento:
`wire/{header,frame,opcode}`, `codec/{keyword,checksum}`, `payload/<domínio>`,
`error.rs` único, tabelas em `generated/` e integração em `tests/`. `wyd-net`
segue: `session/`, `transport/`, `dispatch/`, `limits/`, `telemetry/`. Arquivo
novo fora dessa árvore exige ADR.

## Antes de mudar

Mapeie opcode/tamanho em `crates/wyd-protocol/src/wire/generated/opcode_table.rs + docs/arquitetura/protocolo.md §4` e espelhe com
teste byte-exato (`cargo test -p wyd-net -p wyd-protocol`).
