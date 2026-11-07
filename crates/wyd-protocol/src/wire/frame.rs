//! Enquadramento CPSock: gate de conexão e corte por `Size`.
//!
//! Todo pacote começa com o [`Header`](super::header::Header) de 12 bytes; o
//! campo `Size` declara o tamanho total. Fonte: `CPSock.h:38-40` e
//! `CPSock.cpp:397-406`.

use core::fmt;

use super::header::{Header, HEADER_SIZE, MAX_MESSAGE_SIZE, MIN_MESSAGE_SIZE};
use super::opcode::Opcode;
use crate::ProtocolError;

/// Gate de conexão (`CPSock.h:40`): primeiros 4 bytes de toda conexão nova.
pub const INITCODE: u32 = 0x1F11_F311;

/// Erros de enquadramento — nunca panic em bytes de rede.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FrameError {
    /// Buffer menor que o `HEADER` de 12 bytes.
    BufferTooShort { needed: usize, got: usize },
    /// `Size` fora de `[12, 8192]`.
    SizeOutOfRange { size: usize },
    /// O frame declarado ainda não chegou inteiro.
    FrameIncomplete { declared: usize, available: usize },
    /// `Type` sem constante conhecida em `Basedef.h`.
    UnknownOpcode { value: u16 },
}

impl fmt::Display for FrameError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::BufferTooShort { needed, got } => {
                write!(f, "buffer curto: precisa de {needed} bytes, recebeu {got}")
            }
            Self::SizeOutOfRange { size } => {
                write!(
                    f,
                    "Size {size} fora da faixa [{MIN_MESSAGE_SIZE}, {MAX_MESSAGE_SIZE}]"
                )
            }
            Self::FrameIncomplete {
                declared,
                available,
            } => write!(
                f,
                "frame incompleto: declarado {declared} bytes, disponível {available}"
            ),
            Self::UnknownOpcode { value } => {
                write!(f, "opcode desconhecido: {value:#06x}")
            }
        }
    }
}

impl std::error::Error for FrameError {}

/// Valida `Size` contra `[HEADER_SIZE, MAX_MESSAGE_SIZE]` (`CPSock.cpp:397-406`).
pub fn validate_size(size: u16) -> Result<usize, FrameError> {
    let size = size as usize;
    if (MIN_MESSAGE_SIZE..=MAX_MESSAGE_SIZE).contains(&size) {
        Ok(size)
    } else {
        Err(FrameError::SizeOutOfRange { size })
    }
}

/// Frame com tamanho e opcode validados — porta de entrada de todo parse.
///
/// Ordem obrigatória tamanho → opcode → payload: só depois daqui o chamador
/// consulta [`crate::payload::declared_len`] e decodifica o corpo.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ValidFrame {
    /// Cabeçalho já validado (buffer já desofuscado — ver abaixo).
    pub header: Header,
    /// Opcode resolvido na tabela gerada.
    pub opcode: Opcode,
}

/// Valida um buffer contra o pipeline tamanho → opcode, sem alocar.
///
/// Pré-condição: `buf` já passou por `decode_in_place` (`Type` trafega
/// ofuscado — o loop do CPSock vai de 4 até `Size`). Fluxo completo:
/// `split_frame` (corte, bytes selados) → `decode_in_place` → `inspect` →
/// `require_ported` → `T::decode`. Tudo é empréstimo do buffer de entrada
/// (parse zero-copy por empréstimo); o primeiro erro tipado interrompe antes
/// de qualquer parse de corpo.
pub fn inspect(buf: &[u8]) -> Result<ValidFrame, ProtocolError> {
    let (header, _) = split_frame(buf)?;
    let opcode = Opcode::from_wire(header.message_type).ok_or(FrameError::UnknownOpcode {
        value: header.message_type,
    })?;
    Ok(ValidFrame { header, opcode })
}

/// Lê o cabeçalho e devolve também o tamanho do frame a consumir do stream.
pub fn split_frame(buf: &[u8]) -> Result<(Header, usize), FrameError> {
    if buf.len() < HEADER_SIZE {
        return Err(FrameError::BufferTooShort {
            needed: HEADER_SIZE,
            got: buf.len(),
        });
    }
    let header = Header::decode(buf).map_err(|_| FrameError::BufferTooShort {
        needed: HEADER_SIZE,
        got: buf.len(),
    })?;
    let declared = validate_size(header.size)?;
    if buf.len() < declared {
        return Err(FrameError::FrameIncomplete {
            declared,
            available: buf.len(),
        });
    }
    Ok((header, declared))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample() -> Header {
        Header {
            size: 20,
            key_word: 42,
            checksum: 0xAB,
            message_type: 525,
            id: 7,
            client_tick: 1_234_567,
        }
    }

    #[test]
    fn size_bounds_follow_cpsock_limits() {
        assert_eq!(validate_size(12), Ok(12));
        assert_eq!(validate_size(8192), Ok(8192));
        assert!(validate_size(11).is_err());
        assert!(validate_size(8193).is_err());
        assert_eq!(
            validate_size(0),
            Err(FrameError::SizeOutOfRange { size: 0 })
        );
    }

    #[test]
    fn split_frame_reports_incomplete_frame() {
        let mut buf = [0u8; HEADER_SIZE];
        sample()
            .encode(&mut buf)
            .expect("cabeçalho cabe em 12 bytes");
        assert_eq!(
            split_frame(&buf),
            Err(FrameError::FrameIncomplete {
                declared: 20,
                available: HEADER_SIZE
            })
        );
    }

    #[test]
    fn split_frame_returns_declared_length() {
        let mut buf = [0u8; 20];
        sample().encode(&mut buf).expect("cabeçalho");
        let (decoded, len) = split_frame(&buf).expect("frame completo");
        assert_eq!(decoded, sample());
        assert_eq!(len, 20);
    }

    #[test]
    fn inspect_accepts_known_opcode_and_rejects_unknown() {
        // Buffer claro por construção (o teste de fio ofuscado vive em tests/).
        let mut buf = [0u8; HEADER_SIZE];
        Header {
            size: 12,
            key_word: 0,
            checksum: 0,
            message_type: 525,
            id: 1,
            client_tick: 7,
        }
        .encode(&mut buf)
        .expect("cabeçalho");
        let valid = inspect(&buf).expect("opcode conhecido");
        assert_eq!(valid.opcode, Opcode::AccountLogin);
        assert_eq!(valid.header.id, 1);

        buf[4..6].copy_from_slice(&0xFFFFu16.to_le_bytes());
        assert_eq!(
            inspect(&buf),
            Err(crate::ProtocolError::Frame(FrameError::UnknownOpcode {
                value: 0xFFFF
            }))
        );
    }

    #[test]
    fn split_frame_rejects_short_buffer() {
        assert_eq!(
            split_frame(&[0u8; 11]),
            Err(FrameError::BufferTooShort {
                needed: HEADER_SIZE,
                got: 11
            })
        );
    }
}
