//! Transporte Cold-ish (FASE-06): tetos e sanitização antes de qualquer parse.
//!
//! Destino: `tokio` (+ `io_uring` no Linux) com buffers zero-copy (exige rede +
//! ADR). O que já vale offline: a faixa de pacote válida
//! (`MIN_PACKET_BYTES..=MAX_PACKET_BYTES` = 12..=8192), `limits/`
//! (rajada, porte de handshake, backpressure) e a ordem tamanho → opcode →
//! payload herdada de `wyd-protocol` (08-2). Input de rede nunca causa panic.

pub mod error;
pub mod limits;
pub mod metrics;
pub mod session;

/// Teto de um pacote CPSock: **8192 B** (`CPSock.h:38`, `protocolo.md` §1).
///
/// O número não é duplicado aqui: a fonte é `MAX_MESSAGE_SIZE` do
/// `wyd-protocol`, para o net não divergir do enquadramento que valida o `Size`.
pub const MAX_PACKET_BYTES: usize = wyd_protocol::MAX_MESSAGE_SIZE;

/// Piso de um pacote CPSock: `HEADER` puro de 12 B (`MIN_MESSAGE_SIZE`).
pub const MIN_PACKET_BYTES: usize = wyd_protocol::MIN_MESSAGE_SIZE;

/// Verifica se o buffer tem tamanho de pacote CPSock válido antes de qualquer parse.
///
/// Não repete a faixa: delega para `validate_size` do `wyd-protocol` — a **mesma** função que o
/// caminho real de leitura usa (`split_frame` → `validate_size`), então este pre-check barato
/// nunca aceita o que o enquadramento recusaria. Semântica documentada em
/// `docs/arquitetura/protocolo.md` §1: `Size` é o tamanho **total do pacote** (os 12 B do `HEADER`
/// estão dentro dele), válido em `[12, 8192]`. Um `len` acima de `u16::MAX` nem cabe no campo
/// `Size` e por isso é recusado.
#[must_use]
pub fn accepts_len(len: usize) -> bool {
    u16::try_from(len).is_ok_and(|size| wyd_protocol::wire::frame::validate_size(size).is_ok())
}

pub use error::NetError;
pub use limits::{default_bucket, ConnectionGate, Deny, TokenBucket};
pub use metrics::{NetMetrics, NET};
pub use session::{VersionGate, VersionPolicy};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_empty_and_oversized_packets() {
        assert!(!accepts_len(0));
        assert!(!accepts_len(MIN_PACKET_BYTES - 1));
        assert!(accepts_len(MIN_PACKET_BYTES));
        assert!(accepts_len(MAX_PACKET_BYTES));
        assert!(!accepts_len(MAX_PACKET_BYTES + 1));
    }

    #[test]
    fn packet_range_matches_the_cpsock_contract() {
        // `CPSock.h:38` e `protocolo.md` §1: Size em [12, 8192].
        assert_eq!(MIN_PACKET_BYTES, 12);
        assert_eq!(MAX_PACKET_BYTES, 8192);
        assert!(accepts_len(8192), "8192 B é pacote válido no legado");
        assert!(!accepts_len(11), "11 B não cabe um HEADER de 12 B");
        assert!(accepts_len(12), "HEADER puro (MSG_STANDARD) é válido");
        assert!(!accepts_len(8193), "acima do teto do CPSock");
    }

    #[test]
    fn limites_nomeiam_qual_borda_falhou() {
        assert!(
            accepts_len(8192),
            "limite superior: 8192 B é o maior pacote válido (MAX_PACKET_BYTES)"
        );
        assert!(
            !accepts_len(8193),
            "limite superior: 8193 B excede MAX_PACKET_BYTES = 8192"
        );
        assert!(
            accepts_len(12),
            "limite inferior: 12 B é o HEADER puro (MIN_PACKET_BYTES)"
        );
        assert!(
            !accepts_len(11),
            "limite inferior: 11 B não cabe o HEADER de 12 B"
        );
        assert!(
            !accepts_len(usize::from(u16::MAX) + 1),
            "acima de u16::MAX o valor nem cabe no campo Size"
        );
    }

    #[test]
    fn pre_check_concorda_com_o_enquadramento_real() {
        use wyd_protocol::wire::frame::{split_frame, validate_size};
        use wyd_protocol::wire::header::Header;

        // Mesma resposta do caminho de leitura (`split_frame` → `validate_size`) em toda borda.
        for len in [0usize, 11, 12, 13, 8191, 8192, 8193, 70_000] {
            let pelo_net = accepts_len(len);
            let pelo_fio = u16::try_from(len)
                .map(|size| validate_size(size).is_ok())
                .unwrap_or(false);
            assert_eq!(
                pelo_net, pelo_fio,
                "divergência em {len} B: pre-check e enquadramento precisam concordar"
            );
        }

        // Um frame de 8192 B de verdade (HEADER + corpo zerado) atravessa o enquadramento.
        let mut buf = vec![0u8; 8192];
        Header {
            size: 8192,
            key_word: 0,
            checksum: 0,
            message_type: 525,
            id: 1,
            client_tick: 0,
        }
        .encode(&mut buf)
        .expect("cabeçalho cabe nos 12 B iniciais");
        assert!(accepts_len(buf.len()), "limite superior real: 8192 B");
        let (_, declarado) = split_frame(&buf).expect("frame de 8192 B é aceito pelo fio");
        assert_eq!(declarado, 8192, "tamanho consumido é o Size declarado");

        buf[0..2].copy_from_slice(&8193u16.to_le_bytes());
        assert!(!accepts_len(8193usize), "8193 B recusado no pre-check");
        assert!(
            split_frame(&buf).is_err(),
            "8193 B recusado no enquadramento"
        );
    }
}
