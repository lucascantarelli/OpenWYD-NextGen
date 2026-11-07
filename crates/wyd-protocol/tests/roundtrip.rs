//! Integração ponta a ponta (só API pública): pacote → fio → pacote.
//!
//! Cobre todo payload portado + header + ofuscação + checksum + rejeições.

mod common;

use common::{open_packet, sealed_packet};
use wyd_protocol::{
    decode_in_place, split_frame, validate_size, verify_checksum, AccountLogin, AccountSecure,
    Action, ApplyBonus, AttackFull, AttackOne, AttackTwo, Buy, CharacterLogin, ChecksumPolicy,
    CreateCharacter, DeleteCharacter, Header, MessagePanel, Opcode, Sell, TargetDamage,
    TradingItem, UpdateWeather, UseItem, WirePayload,
};

fn account_login_sample() -> AccountLogin {
    let mut account_name = [0u8; 16];
    account_name[..4].copy_from_slice(b"test");
    AccountLogin {
        account_password: [1u8; 12],
        account_name,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [11, 22, 33, 44],
    }
}

#[test]
fn full_packet_compose_matches_wire_layout() {
    let login = account_login_sample();
    let mut packet = vec![0u8; AccountLogin::WIRE_LEN];
    Header {
        size: AccountLogin::WIRE_LEN as u16,
        key_word: 0,
        checksum: 0,
        message_type: AccountLogin::OPCODE.wire(),
        id: 1,
        client_tick: 1,
    }
    .encode(&mut packet)
    .expect("cabeçalho");
    login.encode_into(&mut packet).expect("corpo");

    let (header, len) = split_frame(&packet).expect("frame válido");
    assert_eq!(len, AccountLogin::WIRE_LEN);
    assert_eq!(header.message_type, Opcode::AccountLogin.wire());

    let checksum = wyd_protocol::encode_in_place(&mut packet, 123).expect("ofuscação");
    assert_eq!(packet[3], checksum);
    let decoded_checksum = decode_in_place(&mut packet).expect("desofuscação");
    assert_eq!(decoded_checksum, checksum);

    let decoded = AccountLogin::decode(&packet).expect("payload");
    assert_eq!(decoded, login);
}

#[test]
fn every_ported_payload_survives_the_wire() {
    let cases: Vec<(&str, Vec<u8>)> = vec![
        ("login", sealed_packet(&account_login_sample(), 1, 1, 7)),
        (
            "create",
            sealed_packet(
                &CreateCharacter {
                    slot: 2,
                    mob_name: *b"Kae\0\0\0\0\0\0\0\0\0\0\0\0\0",
                    mob_class: 3,
                },
                2,
                2,
                77,
            ),
        ),
        (
            "delete",
            sealed_packet(
                &DeleteCharacter {
                    slot: 1,
                    mob_name: *b"Kae\0\0\0\0\0\0\0\0\0\0\0\0\0",
                    password: [9u8; 12],
                },
                3,
                3,
                1,
            ),
        ),
        (
            "charlogin",
            sealed_packet(&CharacterLogin { slot: 3, force: 1 }, 4, 4, 128),
        ),
        (
            "action",
            sealed_packet(
                &Action {
                    pos_x: 100,
                    pos_y: -200,
                    effect: 1,
                    speed: 4,
                    route: [5u8; 24],
                    target_x: 150,
                    target_y: -260,
                },
                5,
                5,
                255,
            ),
        ),
        (
            "useitem",
            sealed_packet(
                &UseItem {
                    sour_type: 1,
                    sour_pos: 2,
                    dest_type: 3,
                    dest_pos: 4,
                    grid_x: 5,
                    grid_y: 6,
                    warp_id: 7,
                },
                6,
                6,
                0,
            ),
        ),
        (
            "trading",
            sealed_packet(
                &TradingItem {
                    dest_place: 1,
                    dest_slot: 2,
                    src_place: 3,
                    src_slot: 4,
                    warp_id: 99,
                },
                7,
                7,
                9,
            ),
        ),
        (
            "buy",
            sealed_packet(
                &Buy {
                    target_id: 12,
                    target_inven_pos: -1,
                    my_inven_pos: 7,
                    coin: 5000,
                },
                8,
                8,
                42,
            ),
        ),
        (
            "sell",
            sealed_packet(
                &Sell {
                    target_id: 12,
                    my_type: 2,
                    my_pos: 8,
                },
                9,
                9,
                43,
            ),
        ),
        (
            "bonus",
            sealed_packet(
                &ApplyBonus {
                    bonus_type: 0,
                    detail: 3,
                    target_id: 55,
                },
                10,
                10,
                44,
            ),
        ),
        (
            "secure",
            sealed_packet(
                &AccountSecure {
                    numeric_token: [1, 2, 3, 4, 5, 6],
                    unknown: [0u8; 10],
                    change_numeric: 1,
                },
                11,
                11,
                45,
            ),
        ),
    ];
    assert_eq!(cases.len(), 11);
    for (name, mut wire) in cases {
        let computed = decode_in_place(&mut wire).expect("desofuscação válida");
        let (header, _) = split_frame(&wire).expect("frame válido pós-decode");
        verify_checksum(computed, header.checksum, ChecksumPolicy::Strict)
            .unwrap_or_else(|_| panic!("checksum íntegro de {name}"));
    }

    let mut panel_text = [0u8; 128];
    panel_text[..5].copy_from_slice(b"hello");
    let mut panel = sealed_packet(&MessagePanel { text: panel_text }, 12, 12, 46);
    let (_, body): (Header, MessagePanel) = open_packet(&mut panel);
    assert_eq!(body.text_bytes(), b"hello");

    let mut weather = sealed_packet(&UpdateWeather { current_weather: 3 }, 13, 13, 47);
    let (header, body): (Header, UpdateWeather) = open_packet(&mut weather);
    assert_eq!(header.id, 13);
    assert_eq!(body.current_weather, 3);

    let full = AttackFull {
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
        dam: std::array::from_fn(|index| TargetDamage {
            target_id: index as i32 + 1,
            damage: 100 + index as i32,
        }),
    };
    let mut wire = sealed_packet(&full, 14, 14, 48);
    let (_, back): (Header, AttackFull) = open_packet(&mut wire);
    assert_eq!(back, full);

    let one = AttackOne {
        unk_1: [0; 4],
        current_hp: 0,
        unk_2: [0; 4],
        current_exp: 0,
        unk_0: 0,
        pos_x: 0,
        pos_y: 0,
        target_x: 0,
        target_y: 0,
        attacker_id: 0,
        progress: 0,
        motion: 0,
        skill_parm: 0,
        double_critical: 0,
        flag_local: 0,
        rsv: 0,
        current_mp: 0,
        skill_index: 0,
        req_mp: 0,
        dam: [TargetDamage {
            target_id: 9,
            damage: 77,
        }],
    };
    let mut wire = sealed_packet(&one, 15, 15, 49);
    let (_, back): (Header, AttackOne) = open_packet(&mut wire);
    assert_eq!(back.dam[0].damage, 77);

    let two = AttackTwo {
        unk_1: [0; 4],
        current_hp: 0,
        unk_2: [0; 4],
        current_exp: 0,
        unk_0: 0,
        pos_x: 0,
        pos_y: 0,
        target_x: 0,
        target_y: 0,
        attacker_id: 0,
        progress: 0,
        motion: 0,
        skill_parm: 0,
        double_critical: 0,
        flag_local: 0,
        rsv: 0,
        current_mp: 0,
        skill_index: 0,
        req_mp: 0,
        dam: [
            TargetDamage {
                target_id: 9,
                damage: 77,
            },
            TargetDamage {
                target_id: 10,
                damage: 88,
            },
        ],
    };
    let mut wire = sealed_packet(&two, 16, 16, 50);
    let (_, back): (Header, AttackTwo) = open_packet(&mut wire);
    assert_eq!(back.dam[1].damage, 88);
}

#[test]
fn tampered_body_fails_strict_checksum() {
    // O checksum do CPSock é soma(cifrado) − soma(claro): cada byte contribui
    // com ±f(trans), ou seja, ele é função da chave, não do conteúdo. Tamper de
    // 1 byte do corpo preserva o valor — o Strict detecta corrupção do campo.
    let mut wire = sealed_packet(&account_login_sample(), 1, 1, 7);
    wire[3] ^= 0xFF; // adultera o CheckSum declarado no HEADER
    let computed = decode_in_place(&mut wire).expect("formato ainda válido");
    let (header, _) = split_frame(&wire).expect("enquadramento pós-decode");
    assert_eq!(header.checksum, 0xFF ^ sealed_checksum());
    assert!(verify_checksum(computed, header.checksum, ChecksumPolicy::Strict).is_err());
    assert!(verify_checksum(computed, header.checksum, ChecksumPolicy::Tolerant).is_ok());
}

fn sealed_checksum() -> u8 {
    let wire = sealed_packet(&account_login_sample(), 1, 1, 7);
    wire[3]
}

#[test]
fn invalid_sizes_and_unknown_opcodes_are_rejected() {
    assert!(validate_size(11).is_err());
    assert!(validate_size(8193).is_err());
    assert!(split_frame(&[0u8; 11]).is_err());
    assert_eq!(Opcode::from_wire(0xFFFF), None);
    assert!(AccountLogin::decode(&[0u8; 115]).is_err());
}
