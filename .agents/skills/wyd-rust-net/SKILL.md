---
name: wyd-rust-net
description: Desenvolve o transporte async do OpenWYD em Rust (crates/wyd-net + wyd-protocol): tokio/io_uring, buffers zero-copy, parsers com validação estrita de limites, rate limiting e hardening de handshake. Use ao mexer em rede, protocolo CPSock, opcodes ou serialização binária.
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

## Antes de mudar

Mapeie opcode/tamanho em `docs/migration/handlers/` e espelhe com teste byte-exato
(`cargo test -p wyd-net -p wyd-protocol`).
