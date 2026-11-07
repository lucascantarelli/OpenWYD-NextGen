//! Transporte async (esqueleto).
//!
//! Destino: `tokio` (+ `io_uring` no Linux) com buffers zero-copy e validacao
//! estrita de limites. Sem dependencias externas neste esqueleto para permitir
//! build offline.

/// Tamanho maximo aceito para um pacote CPSock neste esqueleto.
pub const MAX_PACKET_BYTES: usize = 4096;

/// Verifica o tamanho do buffer antes de qualquer parse.
#[must_use]
pub fn accepts_len(len: usize) -> bool {
    len > 0 && len <= MAX_PACKET_BYTES
}

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
