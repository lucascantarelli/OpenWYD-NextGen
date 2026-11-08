//! Payloads tipados do protocolo CPSock (round-trip encode/decode).
//!
//! Cada struct representa o pacote **inteiro** (o `HEADER` de 12 bytes continua no
//! início), mas apenas o corpo é lido/escrito — o cabeçalho é responsabilidade de
//! [`Header`](crate::wire::header::Header). Offsets e tamanhos vêm de
//! `docs/arquitetura/protocolo.md` §5.
//!
//! Regras de segurança do parser:
//! - nada de `unwrap`/`panic` em bytes de rede: toda entrada é validada por
//!   [`PayloadError::BadLength`] antes de qualquer leitura;
//! - somente layouts **documentados** viram tipo aqui. Pacote com layout
//!   UNVERIFIED (ex.: `MSG_Trade`) fica de fora até a captura real (FASE-08-2);
//! - strings do protocolo são `char[N]` em bytes crus — nada de UTF-8 assumido.
//!
//! Um domínio por arquivo (SRP); este `mod.rs` só tem o contrato, as macros e o
//! índice de tamanhos.

use core::fmt;

use crate::wire::opcode::Opcode;

/// Erros de payload — nenhum caminho de parse causa panic.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PayloadError {
    /// Buffer não tem exatamente o tamanho de fio do pacote.
    BadLength { expected: usize, got: usize },
    /// Opcode conhecido mas sem layout documentado (UNVERIFIED, FASE-08-2).
    Unported { opcode: Opcode },
}

impl fmt::Display for PayloadError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::BadLength { expected, got } => {
                write!(
                    f,
                    "tamanho de pacote inválido: esperado {expected}, recebido {got}"
                )
            }
            Self::Unported { opcode } => {
                write!(f, "opcode sem layout documentado: {}", opcode.const_name())
            }
        }
    }
}

impl std::error::Error for PayloadError {}

fn check_len(expected: usize, got: usize) -> Result<(), PayloadError> {
    if expected == got {
        Ok(())
    } else {
        Err(PayloadError::BadLength { expected, got })
    }
}

/// Contrato de um payload de tamanho fixo.
pub trait WirePayload: Sized {
    /// Opcode correspondente na tabela gerada.
    const OPCODE: Opcode;
    /// Tamanho total do pacote (cabeçalho + corpo).
    const WIRE_LEN: usize;

    /// Decodifica o pacote inteiro (com `HEADER`), validando o tamanho.
    fn decode(packet: &[u8]) -> Result<Self, PayloadError>;

    /// Escreve apenas o corpo em `packet` (o `HEADER` já deve estar lá).
    fn encode_into(&self, packet: &mut [u8]) -> Result<(), PayloadError>;
}

/// Leitura/escrita little-endian de um campo do pacote.
///
/// `read` só é chamado depois de [`check_len`] confirmar o tamanho do pacote, e
/// todo offset é menor que o tamanho declarado — logo não há acesso fora do buffer.
pub trait WireField: Sized {
    /// Tamanho do campo em bytes.
    const LEN: usize;
    /// Lê o campo em `off`.
    fn read(buf: &[u8], off: usize) -> Self;
    /// Escreve o campo em `off`.
    fn write(self, buf: &mut [u8], off: usize);
}

macro_rules! wire_scalar {
    ($ty:ty, $len:expr) => {
        impl WireField for $ty {
            const LEN: usize = $len;

            fn read(buf: &[u8], off: usize) -> Self {
                let mut raw = [0u8; $len];
                raw.copy_from_slice(&buf[off..off + $len]);
                <$ty>::from_le_bytes(raw)
            }

            fn write(self, buf: &mut [u8], off: usize) {
                buf[off..off + $len].copy_from_slice(&self.to_le_bytes());
            }
        }
    };
}

wire_scalar!(u8, 1);
wire_scalar!(i16, 2);
wire_scalar!(u16, 2);
wire_scalar!(i32, 4);
wire_scalar!(u32, 4);
wire_scalar!(i64, 8);

impl<const N: usize> WireField for [u8; N] {
    const LEN: usize = N;

    fn read(buf: &[u8], off: usize) -> Self {
        let mut raw = [0u8; N];
        raw.copy_from_slice(&buf[off..off + N]);
        raw
    }

    fn write(self, buf: &mut [u8], off: usize) {
        buf[off..off + N].copy_from_slice(&self);
    }
}

impl<const N: usize> WireField for [i32; N] {
    const LEN: usize = N * 4;

    fn read(buf: &[u8], off: usize) -> Self {
        let mut raw = [0i32; N];
        for (index, slot) in raw.iter_mut().enumerate() {
            *slot = i32::read(buf, off + index * 4);
        }
        raw
    }

    fn write(self, buf: &mut [u8], off: usize) {
        for (index, value) in self.iter().enumerate() {
            i32::write(*value, buf, off + index * 4);
        }
    }
}

/// Recorte de um `char[N]` até o primeiro `NUL` (o protocolo não é UTF-8).
#[must_use]
pub fn nul_trimmed(bytes: &[u8]) -> &[u8] {
    match bytes.iter().position(|byte| *byte == 0) {
        Some(end) => &bytes[..end],
        None => bytes,
    }
}

/// Declara um payload de tamanho fixo com offsets explícitos do protocolo.
macro_rules! wire_payload {
    ($name:ident, $opcode:ident, $len:expr, [$(($off:expr, $field:ident, $ty:ty)),* $(,)?]) => {
        #[derive(Debug, Clone, Copy, PartialEq, Eq)]
        pub struct $name {
            $(pub $field: $ty,)*
        }

        impl WirePayload for $name {
            const OPCODE: Opcode = Opcode::$opcode;
            const WIRE_LEN: usize = $len;

            fn decode(packet: &[u8]) -> Result<Self, PayloadError> {
                check_len(Self::WIRE_LEN, packet.len())?;
                Ok(Self {
                    $($field: <$ty as WireField>::read(packet, $off),)*
                })
            }

            fn encode_into(&self, packet: &mut [u8]) -> Result<(), PayloadError> {
                check_len(Self::WIRE_LEN, packet.len())?;
                $(<$ty as WireField>::write(self.$field, packet, $off);)*
                Ok(())
            }
        }
    };
}

pub mod auth;
pub mod character;
pub mod combat;
pub mod commerce;
pub mod inventory;
pub mod movement;
pub mod ui;

pub use auth::{AccountLogin, AccountSecure};
pub use character::{CharacterLogin, CreateCharacter, DeleteCharacter};
pub use combat::{Attack, AttackFull, AttackOne, AttackTwo, TargetDamage};
pub use commerce::{Buy, Sell};
pub use inventory::{ApplyBonus, TradingItem, UseItem};
pub use movement::Action;
pub use ui::{MessagePanel, UpdateWeather};

/// Exige layout documentado para o opcode (etapa payload do pipeline).
///
/// `Unported` nunca vira parse: é o gate contra layout UNVERIFIED.
pub fn require_ported(opcode: Opcode) -> Result<usize, PayloadError> {
    declared_len(opcode).ok_or(PayloadError::Unported { opcode })
}

/// Tamanho documentado do pacote, quando o layout já foi portado para um tipo.
///
/// `None` = layout ainda não documentado/UNVERIFIED (vai para a FASE-08-2).
#[must_use]
pub fn declared_len(opcode: Opcode) -> Option<usize> {
    match opcode {
        Opcode::AccountLogin => Some(AccountLogin::WIRE_LEN),
        Opcode::CreateCharacter => Some(CreateCharacter::WIRE_LEN),
        Opcode::DeleteCharacter => Some(DeleteCharacter::WIRE_LEN),
        Opcode::CharacterLogin => Some(CharacterLogin::WIRE_LEN),
        Opcode::Action => Some(Action::WIRE_LEN),
        Opcode::UseItem => Some(UseItem::WIRE_LEN),
        Opcode::TradingItem => Some(TradingItem::WIRE_LEN),
        Opcode::Buy => Some(Buy::WIRE_LEN),
        Opcode::Sell => Some(Sell::WIRE_LEN),
        Opcode::ApplyBonus => Some(ApplyBonus::WIRE_LEN),
        Opcode::AccountSecure => Some(AccountSecure::WIRE_LEN),
        Opcode::MessagePanel => Some(MessagePanel::WIRE_LEN),
        Opcode::UpdateWeather => Some(UpdateWeather::WIRE_LEN),
        Opcode::Attack => Some(AttackFull::WIRE_LEN),
        Opcode::AttackOne => Some(AttackOne::WIRE_LEN),
        Opcode::AttackTwo => Some(AttackTwo::WIRE_LEN),
        _ => None,
    }
}

/// Codifica em `T::WIRE_LEN` bytes e decodifica de volta (uso dos testes de domínio).
#[cfg(test)]
pub(crate) fn round_trip<T>(value: T) -> T
where
    T: WirePayload + PartialEq + core::fmt::Debug,
{
    let mut buf = vec![0u8; T::WIRE_LEN];
    value
        .encode_into(&mut buf)
        .expect("encode com tamanho correto");
    T::decode(&buf).expect("decode do que acabou de ser codificado")
}

#[cfg(test)]
mod tests {
    use super::auth::AccountLogin;
    use super::movement::Action;
    use super::*;

    #[test]
    fn wrong_length_is_rejected_without_panic() {
        for len in [0usize, 11, 12, 115, 117, 8192] {
            let buf = vec![0u8; len];
            assert!(AccountLogin::decode(&buf).is_err());
        }
        let action = Action {
            pos_x: 0,
            pos_y: 0,
            effect: 0,
            speed: 0,
            route: [0u8; 24],
            target_x: 0,
            target_y: 0,
        };
        let mut too_long = vec![0u8; Action::WIRE_LEN + 1];
        assert!(action.encode_into(&mut too_long).is_err());
        let mut too_short = vec![0u8; Action::WIRE_LEN - 1];
        assert!(action.encode_into(&mut too_short).is_err());
    }

    #[test]
    fn require_ported_rejects_unverified_layout() {
        assert_eq!(require_ported(Opcode::AccountLogin), Ok(116));
        assert_eq!(
            require_ported(Opcode::Trade),
            Err(PayloadError::Unported {
                opcode: Opcode::Trade
            })
        );
    }

    #[test]
    fn declared_len_covers_every_portable_message() {
        assert_eq!(declared_len(Opcode::AccountLogin), Some(116));
        assert_eq!(declared_len(Opcode::Attack), Some(164));
        assert_eq!(declared_len(Opcode::AttackOne), Some(68));
        assert_eq!(declared_len(Opcode::AttackTwo), Some(76));
        assert_eq!(declared_len(Opcode::MessagePanel), Some(140));
        assert_eq!(declared_len(Opcode::War), None);
    }

    #[test]
    fn nul_trimmed_stops_at_first_zero() {
        assert_eq!(nul_trimmed(b"abc\0def"), b"abc");
        assert_eq!(nul_trimmed(b"abcdef"), b"abcdef");
        assert!(nul_trimmed(b"\0abc").is_empty());
    }
}
