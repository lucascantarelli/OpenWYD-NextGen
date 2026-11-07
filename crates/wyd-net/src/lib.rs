//! Transporte Cold-ish (FASE-06): tetos e sanitização antes de qualquer parse.
//!
//! Destino: `tokio` (+ `io_uring` no Linux) com buffers zero-copy (exige rede +
//! ADR). O que já vale offline: `MAX_PACKET_BYTES` como teto, `limits/`
//! (rajada, porte de handshake, backpressure) e a ordem tamanho → opcode →
//! payload herdada de `wyd-protocol` (08-2). Input de rede nunca causa panic.

pub mod limits;

/// Tamanho maximo aceito para um pacote CPSock.
pub const MAX_PACKET_BYTES: usize = 4096;

/// Verifica o tamanho do buffer antes de qualquer parse.
#[must_use]
pub fn accepts_len(len: usize) -> bool {
    len > 0 && len <= MAX_PACKET_BYTES
}

pub use limits::{default_bucket, ConnectionGate, Deny, TokenBucket};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_empty_and_oversized_packets() {
        assert!(!accepts_len(0));
        assert!(accepts_len(1));
        assert!(accepts_len(MAX_PACKET_BYTES));
        assert!(!accepts_len(MAX_PACKET_BYTES + 1));
    }
}
