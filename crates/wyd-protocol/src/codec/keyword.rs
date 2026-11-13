//! Transformação `pKeyWord` (semente/trans) e ofuscação do CPSock.
//!
//! Regras que não podem mudar (`CPSock.cpp:428-453` decode, `:556-584` encode):
//! - `Size`, `KeyWord` e `CheckSum` (offsets 0..4) ficam em claro.
//! - A transformação começa em [`KEYWORD_OFFSET`] (4) e vai até o `Size` declarado.
//! - Semente: `KEY_TABLE[key_word * 2]`; transformação: `KEY_TABLE[rst * 2 + 1]`.
//! - Aritmética em 8 bits com wrap (equivale ao `unsigned char` do C).
//! - `CheckSum = soma(cifrado) - soma(claro)`, ambos com wrap.
//!
//! Todo erro é tipado: bytes vindos da rede nunca causam panic.

use core::fmt;

use super::generated::keyword_table::KEY_TABLE;
use crate::wire::header::{Header, HEADER_SIZE};

/// Offset onde começa o payload ofuscado (o `HEADER` fica em claro).
pub const KEYWORD_OFFSET: usize = 4;

/// Erros de transformação de um frame.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TransformError {
    /// Buffer menor que o `HEADER`.
    BufferTooShort { needed: usize, got: usize },
    /// `Size` declarado inválido.
    InvalidSize { size: u16 },
    /// Buffer não tem exatamente o `Size` declarado.
    LengthMismatch { declared: usize, got: usize },
    /// Checksum estrito divergiu (política [`Strict`](super::checksum::ChecksumPolicy)).
    ChecksumMismatch { computed: u8, declared: u8 },
}

impl fmt::Display for TransformError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::BufferTooShort { needed, got } => {
                write!(f, "buffer curto: precisa de {needed} bytes, recebeu {got}")
            }
            Self::InvalidSize { size } => write!(f, "Size declarado inválido: {size}"),
            Self::LengthMismatch { declared, got } => {
                write!(f, "frame com {got} bytes, Size declarado {declared}")
            }
            Self::ChecksumMismatch { computed, declared } => {
                write!(
                    f,
                    "checksum divergiu: calculado {computed}, declarado {declared}"
                )
            }
        }
    }
}

impl std::error::Error for TransformError {}

/// Byte de transformação para a posição corrente da semente.
fn trans_at(pos: usize) -> u8 {
    KEY_TABLE[(pos % 256) * 2 + 1]
}

/// Confere se `packet` corresponde exatamente ao `Size` do seu cabeçalho.
fn frame_len(packet: &[u8]) -> Result<usize, TransformError> {
    if packet.len() < HEADER_SIZE {
        return Err(TransformError::BufferTooShort {
            needed: HEADER_SIZE,
            got: packet.len(),
        });
    }
    let header = Header::decode(packet).map_err(|_| TransformError::BufferTooShort {
        needed: HEADER_SIZE,
        got: packet.len(),
    })?;
    let declared = crate::wire::frame::validate_size(header.size)
        .map_err(|_| TransformError::InvalidSize { size: header.size })?;
    if declared != packet.len() {
        return Err(TransformError::LengthMismatch {
            declared,
            got: packet.len(),
        });
    }
    Ok(declared)
}

/// Desofusca o frame no lugar e devolve o checksum calculado.
///
/// O chamador compara o retorno com `Header::checksum` e decide a política (ver
/// [`super::checksum`]): o legado **tolerava** mismatch (`CPSock.cpp:457-466`).
pub fn decode_in_place(packet: &mut [u8]) -> Result<u8, TransformError> {
    let size = frame_len(packet)?;
    if packet.len() < HEADER_SIZE {
        return Err(TransformError::BufferTooShort {
            needed: HEADER_SIZE,
            got: packet.len(),
        });
    }
    let key_word = packet[2];
    let start = KEY_TABLE[usize::from(key_word) * 2] as usize;
    let mut sum_plain: u8 = 0;
    let mut sum_cipher: u8 = 0;

    for (offset, slot) in packet[KEYWORD_OFFSET..size].iter_mut().enumerate() {
        let index = KEYWORD_OFFSET + offset;
        let cipher = *slot;
        sum_cipher = sum_cipher.wrapping_add(cipher);
        let trans = trans_at(start + offset);
        let plain = match index & 0x3 {
            0 => cipher.wrapping_sub(trans << 1),
            1 => cipher.wrapping_add(trans >> 3),
            2 => cipher.wrapping_sub(trans << 2),
            _ => cipher.wrapping_add(trans >> 5),
        };
        *slot = plain;
        sum_plain = sum_plain.wrapping_add(plain);
    }

    Ok(sum_cipher.wrapping_sub(sum_plain))
}

/// Ofusca o frame no lugar, grava `KeyWord`/`CheckSum` no cabeçalho e devolve o checksum.
///
/// O `HEADER` já deve estar escrito (inclusive `Size`) antes da chamada.
pub fn encode_in_place(packet: &mut [u8], key_word: u8) -> Result<u8, TransformError> {
    let size = frame_len(packet)?;
    if packet.len() < HEADER_SIZE {
        return Err(TransformError::BufferTooShort {
            needed: HEADER_SIZE,
            got: packet.len(),
        });
    }
    packet[2] = key_word;
    let start = KEY_TABLE[usize::from(key_word) * 2] as usize;
    let mut sum_plain: u8 = 0;
    let mut sum_cipher: u8 = 0;

    for (offset, slot) in packet[KEYWORD_OFFSET..size].iter_mut().enumerate() {
        let index = KEYWORD_OFFSET + offset;
        let plain = *slot;
        sum_plain = sum_plain.wrapping_add(plain);
        let trans = trans_at(start + offset);
        let cipher = match index & 0x3 {
            0 => plain.wrapping_add(trans << 1),
            1 => plain.wrapping_sub(trans >> 3),
            2 => plain.wrapping_add(trans << 2),
            _ => plain.wrapping_sub(trans >> 5),
        };
        *slot = cipher;
        sum_cipher = sum_cipher.wrapping_add(cipher);
    }

    let checksum = sum_cipher.wrapping_sub(sum_plain);
    packet[3] = checksum;
    Ok(checksum)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::wire::header::{Header, HEADER_SIZE};

    fn packet(body: &[u8], key_word: u8) -> Vec<u8> {
        let size = (HEADER_SIZE + body.len()) as u16;
        let mut buf = vec![0u8; usize::from(size)];
        Header {
            size,
            key_word,
            checksum: 0,
            message_type: 0x0333,
            id: 1,
            client_tick: 7,
        }
        .encode(&mut buf)
        .expect("cabeçalho cabe no buffer");
        buf[HEADER_SIZE..].copy_from_slice(body);
        buf
    }

    #[test]
    fn key_table_has_expected_size() {
        assert_eq!(KEY_TABLE.len(), 512);
    }

    #[test]
    fn encode_decode_round_trip_for_several_keys() {
        let plain_body: Vec<u8> = (0..=200u8).collect();
        for key_word in [0u8, 1, 77, 128, 255] {
            let mut buf = packet(&plain_body, key_word);
            let checksum = encode_in_place(&mut buf, key_word).expect("ofuscação válida");
            assert_eq!(buf[3], checksum);
            assert_ne!(
                &buf[HEADER_SIZE..],
                &plain_body[..],
                "payload precisa ser ofuscado"
            );

            let computed = decode_in_place(&mut buf).expect("desofuscação válida");
            assert_eq!(computed, checksum);
            assert_eq!(&buf[HEADER_SIZE..], &plain_body[..]);
        }
    }

    #[test]
    fn size_field_stays_in_clear_and_payload_is_transformed() {
        let mut buf = packet(&[1, 2, 3, 4], 9);
        let size_before = buf[..2].to_vec();
        encode_in_place(&mut buf, 9).expect("ofuscação");
        assert_eq!(&buf[..2], &size_before[..], "Size nunca é transformado");
        assert_eq!(buf[2], 9, "KeyWord é gravado no cabeçalho");
        assert_ne!(
            &buf[HEADER_SIZE..],
            &[1, 2, 3, 4],
            "corpo precisa ser ofuscado"
        );
    }

    #[test]
    fn header_only_packet_round_trips_and_keeps_checksum() {
        // O loop do CPSock vai de 4 até Size, então um pacote só de HEADER também
        // passa pela transformação (Type/ID/ClientTick) — e precisa voltar igual.
        let mut buf = packet(&[], 5);
        let plain = buf.clone();
        let checksum = encode_in_place(&mut buf, 5).expect("pacote só com header");
        assert_eq!(buf[3], checksum);
        assert_eq!(decode_in_place(&mut buf).expect("decode"), checksum);
        assert_eq!(&buf[..2], &plain[..2], "Size inalterado");
        assert_eq!(
            &buf[KEYWORD_OFFSET..],
            &plain[KEYWORD_OFFSET..],
            "encode seguido de decode devolve o corpo original"
        );
    }

    #[test]
    fn transform_rejects_short_and_mismatched_buffers() {
        assert_eq!(
            decode_in_place(&mut [0u8; 5]),
            Err(TransformError::BufferTooShort {
                needed: HEADER_SIZE,
                got: 5
            })
        );
        let mut buf = packet(&[0u8; 4], 0);
        buf.push(0);
        assert_eq!(
            decode_in_place(&mut buf),
            Err(TransformError::LengthMismatch {
                declared: HEADER_SIZE + 4,
                got: HEADER_SIZE + 5
            })
        );
    }
}
