//! `HEADER` de 12 bytes do CPSock e o enquadramento por `Size`.
//!
//! Layout canônico (little-endian, sem padding), de
//! `docs/arquitetura/protocolo.md` §1:
//!
//! | offset | campo | tipo |
//! |---:|---|---|
//! | 0 | `Size` | u16 |
//! | 2 | `KeyWord` | u8 |
//! | 3 | `CheckSum` | u8 |
//! | 4 | `Type` | u16 |
//! | 6 | `ID` | u16 |
//! | 8 | `ClientTick` | u32 |

use core::fmt;

/// Tamanho do `HEADER` em bytes.
pub const HEADER_SIZE: usize = 12;
/// Menor pacote válido: um `HEADER` puro (`MSG_STANDARD`).
pub const MIN_MESSAGE_SIZE: usize = HEADER_SIZE;
/// `MAX_MESSAGE_SIZE` de `CPSock.h:38`.
pub const MAX_MESSAGE_SIZE: usize = 8192;
/// Tick interno que o cliente nunca deve usar (`Basedef.h:232`).
pub const SKIPCHECKTICK: u32 = 235_543_242;
/// Versão declarada em `Basedef.h:102` (o cliente medido envia 12000 — ver `PROTOCOL_VERSION`).
pub const APP_VERSION: i32 = 7640;

/// Erros de cabeçalho/enquadramento — nunca panic em bytes de rede.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HeaderError {
    /// Buffer menor que o `HEADER` de 12 bytes.
    BufferTooShort { needed: usize, got: usize },
    /// `Size` fora de `[12, 8192]`.
    SizeOutOfRange { size: usize },
    /// O frame declarado ainda não chegou inteiro.
    FrameIncomplete { declared: usize, available: usize },
}

impl fmt::Display for HeaderError {
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
        }
    }
}

impl std::error::Error for HeaderError {}

/// Cabeçalho de 12 bytes comum a todo pacote CPSock.
///
/// `repr(C)` fixa o layout na ordem declarada (sem reordenação do compilador);
/// os offsets de fio são provados por teste (`repr_c_matches_wire_offsets`).
/// O parse continua por leitura explícita little-endian (sem `unsafe`, sem
/// dependência externa) — veja a decisão na issue FASE-08-2.
#[repr(C)]
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Header {
    /// Tamanho total do pacote, incluindo o próprio `HEADER`.
    pub size: u16,
    /// Índice (0–255) na tabela `pKeyWord` usado na ofuscação.
    pub key_word: u8,
    /// Checksum aditivo de 1 byte do payload.
    pub checksum: u8,
    /// `Type` já com os bits de direção (`Opcode::wire()`).
    pub message_type: u16,
    /// Índice de conexão/jogador (`pUser[]`).
    pub id: u16,
    /// Timestamp/tick do cliente.
    pub client_tick: u32,
}

impl Header {
    /// Lê o `HEADER` do início do buffer; não valida `Size` (use [`Self::frame_len`]).
    pub fn decode(buf: &[u8]) -> Result<Self, HeaderError> {
        if buf.len() < HEADER_SIZE {
            return Err(HeaderError::BufferTooShort {
                needed: HEADER_SIZE,
                got: buf.len(),
            });
        }
        Ok(Self {
            size: u16::from_le_bytes([buf[0], buf[1]]),
            key_word: buf[2],
            checksum: buf[3],
            message_type: u16::from_le_bytes([buf[4], buf[5]]),
            id: u16::from_le_bytes([buf[6], buf[7]]),
            client_tick: u32::from_le_bytes([buf[8], buf[9], buf[10], buf[11]]),
        })
    }

    /// Escreve o `HEADER` no início de `out` (exige ao menos [`HEADER_SIZE`] bytes).
    pub fn encode(&self, out: &mut [u8]) -> Result<(), HeaderError> {
        if out.len() < HEADER_SIZE {
            return Err(HeaderError::BufferTooShort {
                needed: HEADER_SIZE,
                got: out.len(),
            });
        }
        out[0..2].copy_from_slice(&self.size.to_le_bytes());
        out[2] = self.key_word;
        out[3] = self.checksum;
        out[4..6].copy_from_slice(&self.message_type.to_le_bytes());
        out[6..8].copy_from_slice(&self.id.to_le_bytes());
        out[8..12].copy_from_slice(&self.client_tick.to_le_bytes());
        Ok(())
    }

    /// Tamanho validado do frame declarado por este cabeçalho.
    pub fn frame_len(&self) -> Result<usize, HeaderError> {
        super::frame::validate_size(self.size).map_err(|_| HeaderError::SizeOutOfRange {
            size: self.size as usize,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample() -> Header {
        Header {
            size: 116,
            key_word: 42,
            checksum: 0xAB,
            message_type: 525,
            id: 7,
            client_tick: 1_234_567,
        }
    }

    #[test]
    fn header_round_trip_is_byte_exact() {
        let header = sample();
        let mut buf = [0u8; HEADER_SIZE];
        header.encode(&mut buf).expect("buffer de 12 bytes");
        assert_eq!(&buf[0..2], &116u16.to_le_bytes());
        assert_eq!(buf[2], 42);
        assert_eq!(buf[3], 0xAB);
        assert_eq!(&buf[4..6], &525u16.to_le_bytes());
        assert_eq!(&buf[6..8], &7u16.to_le_bytes());
        assert_eq!(Header::decode(&buf).expect("cabeçalho válido"), header);
    }

    #[test]
    fn header_rejects_short_buffer() {
        let err = Header::decode(&[0u8; 11]).expect_err("11 bytes não é HEADER");
        assert_eq!(
            err,
            HeaderError::BufferTooShort {
                needed: HEADER_SIZE,
                got: 11
            }
        );
        let mut out = [0u8; 4];
        assert!(sample().encode(&mut out).is_err());
    }

    #[test]
    fn repr_c_matches_wire_offsets() {
        use core::mem::{offset_of, size_of};
        assert_eq!(size_of::<Header>(), HEADER_SIZE);
        assert_eq!(offset_of!(Header, size), 0);
        assert_eq!(offset_of!(Header, key_word), 2);
        assert_eq!(offset_of!(Header, checksum), 3);
        assert_eq!(offset_of!(Header, message_type), 4);
        assert_eq!(offset_of!(Header, id), 6);
        assert_eq!(offset_of!(Header, client_tick), 8);
    }

    #[test]
    fn frame_len_delegates_to_frame_validation() {
        assert_eq!(sample().frame_len(), Ok(116));
        assert_eq!(
            Header {
                size: 11,
                ..sample()
            }
            .frame_len(),
            Err(HeaderError::SizeOutOfRange { size: 11 })
        );
    }
}
