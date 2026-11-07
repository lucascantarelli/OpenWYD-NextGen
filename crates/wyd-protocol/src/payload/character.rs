//! Criação, remoção e seleção de personagem.

use super::{check_len, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

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

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;

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
}
