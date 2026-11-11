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

impl CharacterLogin {
    /// Tamanho que o cliente 7662 envia no `_MSG_CharacterLogin` real.
    ///
    /// Prova viva (2026-11-11): o log do canal recusava a seleção do cliente com
    /// `esperado 20, recebido 36`. O corpo do 7662 tem 24 B (16 B após `force`), dos quais
    /// só `slot`/`force` foram observados; o resto fica UNVERIFIED e não vira campo.
    pub const WIRE_LEN_CLIENTE: usize = 36;

    /// Decodifica o `_MSG_CharacterLogin` do cliente, aceitando os dois tamanhos provados.
    ///
    /// O corpo extra do 7662 é descartado: `slot`/`force` estão nos mesmos offsets, então a
    /// leitura continua estrita sobre o que importa (nenhum campo é inventado).
    pub fn decode_client(frame: &[u8]) -> Result<Self, PayloadError> {
        match frame.len() {
            Self::WIRE_LEN_CLIENTE => Self::decode(&frame[..Self::WIRE_LEN]),
            _ => Self::decode(frame),
        }
    }
}

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

    #[test]
    fn character_login_do_cliente_7662_tem_36_bytes() {
        // O cliente real manda 24 B de corpo; os 16 B extras não são campos portados.
        let mut curto = vec![0u8; CharacterLogin::WIRE_LEN];
        CharacterLogin { slot: 2, force: 0 }
            .encode_into(&mut curto)
            .expect("corpo");
        let mut frame = vec![0u8; CharacterLogin::WIRE_LEN_CLIENTE];
        frame[..CharacterLogin::WIRE_LEN].copy_from_slice(&curto); // 16 B de cauda UNVERIFIED
        let lido = CharacterLogin::decode_client(&frame).expect("36 bytes aceitos");
        assert_eq!(lido.slot, 2);
        assert_eq!(lido.force, 0);
        assert_eq!(
            CharacterLogin::decode_client(&frame[..CharacterLogin::WIRE_LEN]).map(|p| p.slot),
            Ok(2),
            "o tamanho curto dos testes continua valendo"
        );
        assert!(CharacterLogin::decode_client(&frame[..25]).is_err());
    }
}
