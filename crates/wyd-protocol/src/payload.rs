//! Payloads tipados do protocolo CPSock (round-trip encode/decode).
//!
//! Cada struct representa o pacote **inteiro** (o `HEADER` de 12 bytes continua no
//! início), mas apenas o corpo é lido/escrito — o cabeçalho é responsabilidade de
//! [`crate::header::Header`]. Offsets e tamanhos vêm de
//! `.deprecated-go/docs/migration/protocol-spec.md` §3.5.
//!
//! Regras de segurança do parser:
//! - nada de `unwrap`/`panic` em bytes de rede: toda entrada é validada por
//!   [`PayloadError::BadLength`] antes de qualquer leitura;
//! - somente layouts **documentados** viram tipo aqui. Pacote com layout
//!   UNVERIFIED (ex.: `MSG_Trade`) fica de fora até a captura real (FASE-08-2);
//! - strings do protocolo são `char[N]` em bytes crus — nada de UTF-8 assumido.

use core::fmt;

use crate::opcodes::Opcode;

/// Erros de payload — nenhum caminho de parse causa panic.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PayloadError {
    /// Buffer não tem exatamente o tamanho de fio do pacote.
    BadLength { expected: usize, got: usize },
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

wire_payload!(
    AccountLogin,
    AccountLogin,
    116,
    [
        (12, account_password, [u8; 12]),
        (24, account_name, [u8; 16]),
        (40, zero, [u8; 52]),
        (92, client_version, i32),
        (96, db_need_save, i32),
        (100, adapter_name, [i32; 4]),
    ]
);

impl AccountLogin {
    /// Nome de conta útil (sem padding NUL).
    #[must_use]
    pub fn account_name_bytes(&self) -> &[u8] {
        nul_trimmed(&self.account_name)
    }
}

wire_payload!(
    CreateCharacter,
    CreateCharacter,
    36,
    [
        (12, slot, i32),
        (16, mob_name, [u8; 16]),
        (32, mob_class, i32),
    ]
);

wire_payload!(
    DeleteCharacter,
    DeleteCharacter,
    44,
    [
        (12, slot, i32),
        (16, mob_name, [u8; 16]),
        (32, password, [u8; 12]),
    ]
);

wire_payload!(
    CharacterLogin,
    CharacterLogin,
    20,
    [(12, slot, i32), (16, force, i32)]
);

wire_payload!(
    Action,
    Action,
    52,
    [
        (12, pos_x, i16),
        (14, pos_y, i16),
        (16, effect, i32),
        (20, speed, i32),
        (24, route, [u8; 24]),
        (48, target_x, i16),
        (50, target_y, i16),
    ]
);

wire_payload!(
    UseItem,
    UseItem,
    34,
    [
        (12, sour_type, i32),
        (16, sour_pos, i32),
        (20, dest_type, i32),
        (24, dest_pos, i32),
        (28, grid_x, u16),
        (30, grid_y, u16),
        (32, warp_id, u16),
    ]
);

wire_payload!(
    TradingItem,
    TradingItem,
    20,
    [
        (12, dest_place, u8),
        (13, dest_slot, u8),
        (14, src_place, u8),
        (15, src_slot, u8),
        (16, warp_id, i32),
    ]
);

wire_payload!(
    Buy,
    Buy,
    22,
    [
        (12, target_id, u16),
        (14, target_inven_pos, i16),
        (16, my_inven_pos, i16),
        (18, coin, i32),
    ]
);

wire_payload!(
    Sell,
    Sell,
    18,
    [(12, target_id, u16), (14, my_type, i16), (16, my_pos, i16),]
);

wire_payload!(
    ApplyBonus,
    ApplyBonus,
    18,
    [
        (12, bonus_type, i16),
        (14, detail, i16),
        (16, target_id, u16),
    ]
);

wire_payload!(
    AccountSecure,
    AccountSecure,
    32,
    [
        (12, numeric_token, [u8; 6]),
        (18, unknown, [u8; 10]),
        (28, change_numeric, i32),
    ]
);

wire_payload!(MessagePanel, MessagePanel, 140, [(12, text, [u8; 128])]);

impl MessagePanel {
    /// Texto do painel até o primeiro `NUL`.
    #[must_use]
    pub fn text_bytes(&self) -> &[u8] {
        nul_trimmed(&self.text)
    }
}

wire_payload!(
    UpdateWeather,
    UpdateWeather,
    16,
    [(12, current_weather, i32)]
);

/// Um alvo de dano de `MSG_Attack` (`STRUCT_DAM`, 8 bytes).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct TargetDamage {
    /// Índice da conexão do alvo.
    pub target_id: i32,
    /// Dano aplicado.
    pub damage: i32,
}

impl TargetDamage {
    /// Tamanho em bytes (`STRUCT_DAM`).
    pub const WIRE_LEN: usize = 8;
}

/// `MSG_Attack`/`MSG_AttackOne`/`MSG_AttackTwo` — mesmo corpo, N alvos.
///
/// `N` = 13 (`MSG_Attack`, `MAX_TARGET`), 1 (`AttackOne`) ou 2 (`AttackTwo`);
/// o tamanho do pacote varia com o número de alvos (`protocol-spec` §3.5).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Attack<const N: usize> {
    pub unk_1: [u8; 4],
    pub current_hp: i32,
    pub unk_2: [u8; 4],
    pub current_exp: i64,
    pub unk_0: i16,
    pub pos_x: u16,
    pub pos_y: u16,
    pub target_x: u16,
    pub target_y: u16,
    pub attacker_id: u16,
    pub progress: u16,
    pub motion: u8,
    pub skill_parm: u8,
    pub double_critical: u8,
    pub flag_local: u8,
    pub rsv: i16,
    pub current_mp: i32,
    pub skill_index: i16,
    pub req_mp: i16,
    /// `Dam[N]` — `STRUCT_DAM` por alvo.
    pub dam: [TargetDamage; N],
}

/// Opcode de `MSG_Attack` para a aridade `N` (13 alvos = pacote cheio).
const fn attack_opcode<const N: usize>() -> Opcode {
    match N {
        1 => Opcode::AttackOne,
        2 => Opcode::AttackTwo,
        _ => Opcode::Attack,
    }
}

impl<const N: usize> WirePayload for Attack<N> {
    const OPCODE: Opcode = attack_opcode::<N>();
    const WIRE_LEN: usize = ATTACK_BODY_LEN + TargetDamage::WIRE_LEN * N;

    fn decode(packet: &[u8]) -> Result<Self, PayloadError> {
        check_len(Self::WIRE_LEN, packet.len())?;
        let mut dam = [TargetDamage::default(); N];
        for (index, slot) in dam.iter_mut().enumerate() {
            let off = ATTACK_BODY_LEN + index * TargetDamage::WIRE_LEN;
            *slot = TargetDamage {
                target_id: i32::read(packet, off),
                damage: i32::read(packet, off + 4),
            };
        }
        Ok(Self {
            unk_1: <[u8; 4]>::read(packet, 12),
            current_hp: i32::read(packet, 16),
            unk_2: <[u8; 4]>::read(packet, 20),
            current_exp: i64::read(packet, 24),
            unk_0: i16::read(packet, 32),
            pos_x: u16::read(packet, 34),
            pos_y: u16::read(packet, 36),
            target_x: u16::read(packet, 38),
            target_y: u16::read(packet, 40),
            attacker_id: u16::read(packet, 42),
            progress: u16::read(packet, 44),
            motion: u8::read(packet, 46),
            skill_parm: u8::read(packet, 47),
            double_critical: u8::read(packet, 48),
            flag_local: u8::read(packet, 49),
            rsv: i16::read(packet, 50),
            current_mp: i32::read(packet, 52),
            skill_index: i16::read(packet, 56),
            req_mp: i16::read(packet, 58),
            dam,
        })
    }

    fn encode_into(&self, packet: &mut [u8]) -> Result<(), PayloadError> {
        check_len(Self::WIRE_LEN, packet.len())?;
        <[u8; 4]>::write(self.unk_1, packet, 12);
        i32::write(self.current_hp, packet, 16);
        <[u8; 4]>::write(self.unk_2, packet, 20);
        i64::write(self.current_exp, packet, 24);
        i16::write(self.unk_0, packet, 32);
        u16::write(self.pos_x, packet, 34);
        u16::write(self.pos_y, packet, 36);
        u16::write(self.target_x, packet, 38);
        u16::write(self.target_y, packet, 40);
        u16::write(self.attacker_id, packet, 42);
        u16::write(self.progress, packet, 44);
        u8::write(self.motion, packet, 46);
        u8::write(self.skill_parm, packet, 47);
        u8::write(self.double_critical, packet, 48);
        u8::write(self.flag_local, packet, 49);
        i16::write(self.rsv, packet, 50);
        i32::write(self.current_mp, packet, 52);
        i16::write(self.skill_index, packet, 56);
        i16::write(self.req_mp, packet, 58);
        for (index, target) in self.dam.iter().enumerate() {
            let off = ATTACK_BODY_LEN + index * TargetDamage::WIRE_LEN;
            i32::write(target.target_id, packet, off);
            i32::write(target.damage, packet, off + 4);
        }
        Ok(())
    }
}

/// Corpo fixo de `MSG_Attack` antes de `Dam[]`.
const ATTACK_BODY_LEN: usize = 60;

/// `_MSG_Attack` com 13 alvos (`MAX_TARGET`).
pub type AttackFull = Attack<13>;
/// `_MSG_AttackOne` com 1 alvo.
pub type AttackOne = Attack<1>;
/// `_MSG_AttackTwo` com 2 alvos.
pub type AttackTwo = Attack<2>;

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

#[cfg(test)]
mod tests {
    use super::*;

    /// Codifica em `T::WIRE_LEN` bytes e decodifica de volta.
    fn round_trip<T>(value: T) -> T
    where
        T: WirePayload + PartialEq + core::fmt::Debug,
    {
        let mut buf = vec![0u8; T::WIRE_LEN];
        value
            .encode_into(&mut buf)
            .expect("encode com tamanho correto");
        T::decode(&buf).expect("decode do que acabou de ser codificado")
    }

    #[test]
    fn account_login_round_trip() {
        let mut account_name = [0u8; 16];
        account_name[..4].copy_from_slice(b"test");
        let login = AccountLogin {
            account_password: [1u8; 12],
            account_name,
            zero: [0u8; 52],
            client_version: 12_000,
            db_need_save: 1,
            adapter_name: [11, 22, 33, 44],
        };
        let mut buf = vec![0u8; AccountLogin::WIRE_LEN];
        login.encode_into(&mut buf).expect("tamanho correto");
        assert_eq!(&buf[24..28], b"test");
        assert_eq!(&buf[92..96], &12_000i32.to_le_bytes());

        let decoded = round_trip(login);
        assert_eq!(decoded, login);
        assert_eq!(decoded.account_name_bytes(), b"test");
        assert_eq!(AccountLogin::WIRE_LEN, 116);
        assert_eq!(AccountLogin::OPCODE.wire(), 525);
    }

    #[test]
    fn create_and_delete_character_round_trip() {
        let mut mob_name = [0u8; 16];
        mob_name[..3].copy_from_slice(b"Kae");
        let create = CreateCharacter {
            slot: 2,
            mob_name,
            mob_class: 3,
        };
        assert_eq!(round_trip(create), create);

        let delete = DeleteCharacter {
            slot: 1,
            mob_name,
            password: [9u8; 12],
        };
        assert_eq!(round_trip(delete), delete);
    }

    #[test]
    fn character_login_round_trip() {
        let login = CharacterLogin { slot: 3, force: 1 };
        assert_eq!(CharacterLogin::WIRE_LEN, 20);
        assert_eq!(round_trip(login), login);
    }

    #[test]
    fn action_round_trip_keeps_route_bytes() {
        let mut route = [0u8; 24];
        route[..3].copy_from_slice(&[1, 2, 3]);
        let action = Action {
            pos_x: 100,
            pos_y: -200,
            effect: 1,
            speed: 4,
            route,
            target_x: 150,
            target_y: -260,
        };
        let decoded = round_trip(action);
        assert_eq!(decoded, action);
        assert_eq!(decoded.pos_y, -200);
        assert_eq!(&decoded.route[..3], &[1, 2, 3]);
        assert_eq!(Action::WIRE_LEN, 52);
    }

    #[test]
    fn use_item_and_trading_item_round_trip() {
        let use_item = UseItem {
            sour_type: 1,
            sour_pos: 2,
            dest_type: 3,
            dest_pos: 4,
            grid_x: 5,
            grid_y: 6,
            warp_id: 7,
        };
        assert_eq!(UseItem::WIRE_LEN, 34);
        assert_eq!(round_trip(use_item), use_item);

        let trading = TradingItem {
            dest_place: 1,
            dest_slot: 2,
            src_place: 3,
            src_slot: 4,
            warp_id: 99,
        };
        assert_eq!(TradingItem::WIRE_LEN, 20);
        assert_eq!(round_trip(trading), trading);
    }

    #[test]
    fn buy_sell_apply_bonus_round_trip() {
        let buy = Buy {
            target_id: 12,
            target_inven_pos: -1,
            my_inven_pos: 7,
            coin: 5000,
        };
        assert_eq!(Buy::WIRE_LEN, 22);
        assert_eq!(round_trip(buy), buy);

        let sell = Sell {
            target_id: 12,
            my_type: 2,
            my_pos: 8,
        };
        assert_eq!(Sell::WIRE_LEN, 18);
        assert_eq!(round_trip(sell), sell);

        let bonus = ApplyBonus {
            bonus_type: 0,
            detail: 3,
            target_id: 55,
        };
        assert_eq!(ApplyBonus::WIRE_LEN, 18);
        assert_eq!(round_trip(bonus), bonus);
    }

    #[test]
    fn account_secure_round_trip() {
        let secure = AccountSecure {
            numeric_token: [1, 2, 3, 4, 5, 6],
            unknown: [0u8; 10],
            change_numeric: 1,
        };
        assert_eq!(AccountSecure::WIRE_LEN, 32);
        assert_eq!(round_trip(secure), secure);
    }

    #[test]
    fn message_panel_and_weather_round_trip() {
        let mut text = [0u8; 128];
        text[..5].copy_from_slice(b"hello");
        let panel = MessagePanel { text };
        let decoded = round_trip(panel);
        assert_eq!(decoded, panel);
        assert_eq!(decoded.text_bytes(), b"hello");
        assert_eq!(MessagePanel::WIRE_LEN, 140);

        let weather = UpdateWeather { current_weather: 3 };
        assert_eq!(UpdateWeather::WIRE_LEN, 16);
        assert_eq!(round_trip(weather), weather);
    }

    macro_rules! attack {
        ($ty:ident, $dam:expr) => {
            $ty {
                unk_1: [1, 2, 3, 4],
                current_hp: 500,
                unk_2: [5, 6, 7, 8],
                current_exp: 9_000_000,
                unk_0: -1,
                pos_x: 100,
                pos_y: 200,
                target_x: 110,
                target_y: 210,
                attacker_id: 7,
                progress: 3,
                motion: 1,
                skill_parm: 2,
                double_critical: 1,
                flag_local: 0,
                rsv: 0,
                current_mp: 300,
                skill_index: 42,
                req_mp: 10,
                dam: $dam,
            }
        };
    }

    #[test]
    fn attack_variants_round_trip_with_their_target_counts() {
        let full = attack!(
            AttackFull,
            std::array::from_fn(|index| TargetDamage {
                target_id: index as i32 + 1,
                damage: 100 + index as i32,
            })
        );
        assert_eq!(AttackFull::WIRE_LEN, 164);
        assert_eq!(AttackFull::OPCODE.wire(), 871);
        let decoded = round_trip(full);
        assert_eq!(decoded, full);
        assert_eq!(decoded.dam[12].target_id, 13);

        let one = attack!(
            AttackOne,
            [TargetDamage {
                target_id: 9,
                damage: 77,
            }]
        );
        assert_eq!(AttackOne::WIRE_LEN, 68);
        assert_eq!(AttackOne::OPCODE.wire(), 925);
        let decoded = round_trip(one);
        assert_eq!(decoded.dam[0].target_id, 9);
        assert_eq!(decoded.dam[0].damage, 77);

        let two = attack!(
            AttackTwo,
            [
                TargetDamage {
                    target_id: 9,
                    damage: 77,
                },
                TargetDamage {
                    target_id: 10,
                    damage: 88,
                },
            ]
        );
        assert_eq!(AttackTwo::WIRE_LEN, 76);
        assert_eq!(AttackTwo::OPCODE.wire(), 926);
        let decoded = round_trip(two);
        assert_eq!(decoded.dam[1].damage, 88);
    }

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
