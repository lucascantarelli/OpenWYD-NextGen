//! Contratos de wire do WYD (CPSock): cabeçalho, opcodes, payloads tipados e
//! a ofuscação do legado.
//!
//! Tudo aqui é derivado de fonte canônica verificável:
//! - `.deprecated-go/docs/migration/protocol-spec.md` (especificação de fio);
//! - `.deprecated-go/Source/Code/Basedef.h` (198 constantes `_MSG_*`);
//! - `.deprecated-go/Source/Code/CPSock.cpp` (tabela `pKeyWord[512]`).
//!
//! Árvore normativa em `docs/arquitetura/estrutura-e-padroes.md` §3.
//!
//! Regras do crate (Hot Path):
//! - nenhum `panic`/`unwrap` em bytes vindos da rede — erros tipados;
//! - layout é contrato: offsets explícitos, little-endian, sem `repr(Rust)`;
//! - o que não está documentado **não é inventado**: fica marcado UNVERIFIED e
//!   fora dos tipos até a captura real (FASE-08-2).

pub mod codec;
pub mod error;
pub mod payload;
pub mod wire;

pub use codec::{
    decode_in_place, encode_in_place, verify_checksum, ChecksumPolicy, TransformError,
    KEYWORD_OFFSET,
};
pub use error::ProtocolError;
pub use payload::{
    declared_len, nul_trimmed, require_ported, AccountLogin, AccountSecure, Action, ApplyBonus,
    Attack, AttackFull, AttackOne, AttackTwo, Buy, CharacterLogin, CreateCharacter,
    DeleteCharacter, MessagePanel, PayloadError, Sell, TargetDamage, TradingItem, UpdateWeather,
    UseItem, WireField, WirePayload,
};
pub use wire::{
    inspect, split_frame, validate_size, Flow, FrameError, Header, HeaderError, Opcode, ValidFrame,
    APP_VERSION, FLAG_CLIENT2GAME, FLAG_DB2GAME, FLAG_DB2NP, FLAG_GAME2CLIENT, FLAG_GAME2DB,
    FLAG_NEW, FLAG_NP2DB, HEADER_SIZE, INITCODE, MAX_MESSAGE_SIZE, MIN_MESSAGE_SIZE, SKIPCHECKTICK,
};

/// Versão de cliente esperada pelo servidor (medida em 2026-11-04: o cliente 7662 envia 12000).
pub const PROTOCOL_VERSION: u16 = 12_000;

/// Pacote autoritativo de cena (barra de EXP/HP só atualiza com este ID).
pub const ESCENE_FIELD: u16 = 30_000;

/// Retorna `true` quando o pacote é autoritativo de cena.
#[must_use]
pub fn is_authoritative_scene(id: u16) -> bool {
    id == ESCENE_FIELD
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scene_is_authoritative_only_for_escene_field() {
        assert!(is_authoritative_scene(ESCENE_FIELD));
        assert!(!is_authoritative_scene(0));
        assert_eq!(PROTOCOL_VERSION, 12_000);
    }

    #[test]
    fn opcode_catalog_covers_every_legacy_constant() {
        assert_eq!(
            Opcode::ALL.len(),
            198,
            "Basedef.h tem 198 constantes _MSG_*"
        );

        let mut wires: Vec<u16> = Opcode::ALL.iter().map(|op| op.wire()).collect();
        wires.sort_unstable();
        wires.dedup();
        assert_eq!(
            wires.len(),
            Opcode::ALL.len(),
            "nenhum valor de fio duplicado na tabela"
        );
    }

    #[test]
    fn opcode_wire_values_match_basedef() {
        assert_eq!(Opcode::AccountLogin.wire(), 525); // 0x020D
        assert_eq!(Opcode::AccountLogin.const_name(), "_MSG_AccountLogin");
        assert_eq!(Opcode::CharacterLogin.wire(), 531);
        assert_eq!(Opcode::Attack.wire(), 871);
        assert_eq!(Opcode::AttackOne.wire(), 925);
        assert_eq!(Opcode::AttackTwo.wire(), 926);
        assert_eq!(Opcode::MessagePanel.wire(), 257);
        assert_eq!(Opcode::Ping.wire(), 928);
        assert_eq!(Opcode::AccountSecure.wire(), 4062); // 0x0FDE
        assert_eq!(Opcode::DBAccountLogin.wire(), 2051); // 0x0803
        assert_eq!(Opcode::NPReqIDPASS.wire(), 4097); // 0x1001
        assert_eq!(
            Opcode::from_wire(0x020D).map(Opcode::const_name),
            Some("_MSG_AccountLogin")
        );
    }

    #[test]
    fn opcode_lookup_round_trips_for_every_entry() {
        for opcode in Opcode::ALL {
            let resolved = Opcode::from_wire(opcode.wire());
            assert_eq!(
                resolved.map(Opcode::wire),
                Some(opcode.wire()),
                "from_wire deve resolver o valor de fio de {:?}",
                opcode.const_name()
            );
        }
        assert_eq!(Opcode::from_wire(0xFFFF), None);
    }

    #[test]
    fn flows_reflect_direction_flags() {
        assert!(Opcode::AccountLogin.flows().contains(&Flow::Client2Game));
        assert!(Opcode::MessagePanel.flows().contains(&Flow::Game2Client));
        assert!(Opcode::DBAccountLogin.flows().contains(&Flow::Game2Db));
        assert!(Opcode::ExpMsgPanel.flows().is_empty());
    }
}
