//! Entidades na cena: `_MSG_CreateMob` (0x0364) e `_MSG_RemoveMob` (0x0165).
//!
//! Layout de `MSG_CreateMob` (232 B) levantado nas duas fontes que este repositório já usa
//! para os demais pacotes S→C (`docs/arquitetura/protocolo.md` §5.1):
//!
//! - `Basedef.h` (o mesmo cabeçalho do porte dos structs de login), `struct MSG_CreateMob`:
//!   `_MSG` (12 B) + `short PosX, PosY` + `unsigned short MobID` + `char MobName[16]` +
//!   `unsigned short Equip[16]` + `unsigned short Affect[32]` + `unsigned short Guild` +
//!   `char GuildMemberType` + `char Unknow[3]` + `STRUCT_SCORE Score` + `unsigned short
//!   CreateType` + `unsigned char AnctCode[16]` + `char Tab[26]` + `int Hold`;
//! - porte Go do mesmo cliente 7662 (`tmserver/internal/protocol/createmob.go`,
//!   `createMobSize = 232`, `EncodeCreateMobBody`), que fixa os offsets em teste
//!   (`createmob_test.go`: `MobID @16`, `MobName @18`, `Equip[0] @34`, `Affect[0] @66`,
//!   `Guild @130`, `Score.Level @136`, `CreateType @184`, `AnctCode[0] @186`).
//!
//! Alinhamento natural do MSVC x86 (sem `#pragma pack`): os dois recortes fecham em 232 B e
//! cada campo cai no offset do C — `Score` em +136 (múltiplo de 4), `Hold` em +228.
//!
//! `HEADER.ID` do `CreateMob` é **`ESCENE_FIELD` (30000)**, nunca o id da entidade: é o mesmo
//! pacote autoritativo de cena do invariante 4 do `AGENTS.md`. O id da criatura vai em
//! `MobID` (+16). Já o `_MSG_RemoveMob` (16 B) usa `HEADER.ID` = id da entidade e corpo
//! `int RemoveType` (0 fora de vista, 1 morte, 2 logout).
//!
//! O que **não** está provado aqui: a captura ao vivo (CLIENT-06) — o cliente 7662 ainda não
//! foi observado recebendo estes bytes. Este crate garante o layout das fontes acima, não o
//! comportamento do binário fechado; por isso `docs/cliente/pacotes.md` registra o status de
//! cada pacote separadamente.

use super::{check_len, nul_trimmed, PayloadError, WireField, WirePayload};
use crate::wire::opcode::Opcode;

wire_payload!(
    CreateMob,
    CreateMob,
    232,
    [
        (12, pos_x, u16),
        (14, pos_y, u16),
        (16, mob_id, u16),
        (18, mob_name, [u8; 16]),
        (34, equip, [u16; 16]),
        (66, affect, [u16; 32]),
        (130, guild, u16),
        (132, guild_member_type, u8),
        (133, unknow, [u8; 3]),
        (136, level, i32),
        (140, ac, i32),
        (144, damage, i32),
        (148, merchant, u8),
        (149, attack_run, u8),
        (150, direction, u8),
        (151, chaos_rate, u8),
        (152, max_hp, i32),
        (156, max_mp, i32),
        (160, hp, i32),
        (164, mp, i32),
        (168, strength, i16),
        (170, intelligence, i16),
        (172, dexterity, i16),
        (174, constitution, i16),
        (184, create_type, u16),
        (186, anct_code, [u8; 16]),
        (202, tab, [u8; 26]),
        (228, hold, i32),
    ]
);

impl CreateMob {
    /// `CreateType` de criatura que acabou de entrar na vista (`Basedef.h`; 0 = normal).
    pub const CREATE_JUST_ENTERED: u16 = 2;

    /// `CreateType` de quem já estava na vista (NPC revelado, outro jogador).
    pub const CREATE_NORMAL: u16 = 0;

    /// Nome do mob até o primeiro `NUL` (o template pode ocupar os 16 bytes).
    #[must_use]
    pub fn name_bytes(&self) -> &[u8] {
        nul_trimmed(&self.mob_name)
    }
}

wire_payload!(RemoveMob, RemoveMob, 16, [(12, remove_type, i32)]);

impl RemoveMob {
    /// `RemoveType` 0: saiu da vista (recarga que removeu/realocou o NPC).
    pub const OUT_OF_VIEW: i32 = 0;
    /// `RemoveType` 1: morte.
    pub const DEATH: i32 = 1;
    /// `RemoveType` 2: logout.
    pub const LOGOUT: i32 = 2;
}

#[cfg(test)]
mod tests {
    use super::super::round_trip;
    use super::*;
    use crate::wire::header::HEADER_SIZE;

    fn sample() -> CreateMob {
        let mut mob_name = [0u8; 16];
        mob_name[..7].copy_from_slice(b"Ciclope");
        let mut equip = [0u16; 16];
        equip[0] = 831;
        let mut affect = [0u16; 32];
        affect[0] = (31 << 8) | 7;
        let mut anct_code = [0u8; 16];
        anct_code[0] = 43;
        let mut tab = [0u8; 26];
        tab[0] = 1;
        CreateMob {
            pos_x: 100,
            pos_y: 200,
            mob_id: 1001,
            mob_name,
            equip,
            affect,
            guild: 7,
            guild_member_type: 3,
            unknow: [0; 3],
            level: 171,
            ac: 40,
            damage: 55,
            merchant: 1,
            attack_run: 82,
            direction: 3,
            chaos_rate: 0,
            max_hp: 15_000,
            max_mp: 0,
            hp: 15_000,
            mp: 0,
            strength: 10,
            intelligence: 11,
            dexterity: 12,
            constitution: 13,
            create_type: CreateMob::CREATE_JUST_ENTERED,
            anct_code,
            tab,
            hold: 0,
        }
    }

    #[test]
    fn create_mob_round_trip_keeps_the_layout_of_basedef_h() {
        let mob = sample();
        assert_eq!(CreateMob::WIRE_LEN, 232);
        assert_eq!(CreateMob::OPCODE.wire(), 0x0364);
        assert_eq!(CreateMob::OPCODE.const_name(), "_MSG_CreateMob");

        // O corpo começa em 12: todo offset do C é `12 + o offset do corpo` do porte Go.
        let mut packet = vec![0u8; CreateMob::WIRE_LEN];
        mob.encode_into(&mut packet).expect("encode do corpo");
        assert_eq!(u16::from_le_bytes([packet[12], packet[13]]), 100);
        assert_eq!(u16::from_le_bytes([packet[14], packet[15]]), 200);
        assert_eq!(u16::from_le_bytes([packet[16], packet[17]]), 1001);
        assert_eq!(&packet[18..25], b"Ciclope");
        assert_eq!(u16::from_le_bytes([packet[34], packet[35]]), 831);
        assert_eq!(u16::from_le_bytes([packet[66], packet[67]]), (31 << 8) | 7);
        assert_eq!(u16::from_le_bytes([packet[130], packet[131]]), 7);
        assert_eq!(packet[132], 3);
        assert_eq!(
            i32::from_le_bytes(packet[136..140].try_into().unwrap()),
            171
        );
        assert_eq!(packet[148], 1, "Score.Merchant");
        assert_eq!(packet[149], 82, "Score.AttackRun");
        assert_eq!(
            i32::from_le_bytes(packet[152..156].try_into().unwrap()),
            15_000
        );
        assert_eq!(
            u16::from_le_bytes([packet[184], packet[185]]),
            CreateMob::CREATE_JUST_ENTERED
        );
        assert_eq!(packet[186], 43);
        assert_eq!(i32::from_le_bytes(packet[228..232].try_into().unwrap()), 0);

        let decoded = round_trip(mob);
        assert_eq!(decoded, mob);
        assert_eq!(decoded.name_bytes(), b"Ciclope");
        assert_eq!(CreateMob::WIRE_LEN - HEADER_SIZE, 220, "corpo do CreateMob");
    }

    #[test]
    fn remove_mob_is_the_entity_id_in_the_header_plus_the_type() {
        assert_eq!(RemoveMob::WIRE_LEN, 16);
        assert_eq!(RemoveMob::OPCODE.wire(), 0x0165);
        let pacote = RemoveMob {
            remove_type: RemoveMob::OUT_OF_VIEW,
        };
        // Corpo de 4 bytes: `RemoveType` em +12 do pacote (o HEADER carrega o id).
        let mut packet = vec![0u8; RemoveMob::WIRE_LEN];
        pacote.encode_into(&mut packet).expect("encode do corpo");
        assert_eq!(
            i32::from_le_bytes(packet[12..16].try_into().unwrap()),
            RemoveMob::OUT_OF_VIEW
        );
        assert_eq!(RemoveMob::WIRE_LEN - HEADER_SIZE, 4, "corpo do RemoveMob");
        assert_eq!(round_trip(pacote), pacote);
    }

    #[test]
    fn wrong_length_never_panics() {
        for len in [0usize, 11, 12, 231, 233, 8192] {
            let buf = vec![0u8; len];
            assert!(CreateMob::decode(&buf).is_err());
            assert!(RemoveMob::decode(&buf).is_err());
        }
        let mut curto = vec![0u8; CreateMob::WIRE_LEN - 1];
        assert!(sample().encode_into(&mut curto).is_err());
        assert!(matches!(
            CreateMob::decode(&curto),
            Err(PayloadError::BadLength { .. })
        ));
    }
}
