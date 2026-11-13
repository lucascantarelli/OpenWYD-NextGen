//! Goldens byte-exatos contra `crates/wyd-protocol/tests/fixtures/cliente/*.json`.
//!
//! Cada teste congela um pacote de fio: decodifica, valida o pipeline
//! tamanho → opcode → payload e prova que o corpo recodificado é idêntico
//! ao `payloadHex` do fixture. Os fixtures são sintéticos (gerados por
//! `node tools/client/captura.mjs --gen` a partir dos offsets documentados),
//! NÃO captura de rede — o golden contra captura real aguarda a CLIENT-06.
//! Se um fixture mudar, este teste deve acompanhar (o JSON é lido em tempo
//! de teste, não copiado para cá).

use std::path::PathBuf;

use wyd_protocol::codec::{decode_in_place, encode_in_place, verify_checksum, ChecksumPolicy};
use wyd_protocol::{
    inspect, require_ported, split_frame, AccountLogin, Action, CharacterLogin, CreateCharacter,
    FrameError, Header, Opcode, PayloadError, ProtocolError, WirePayload, HEADER_SIZE,
};

/// Lê `crates/wyd-protocol/tests/fixtures/cliente/{id}.json` a partir do diretório da crate.
fn load_fixture(id: &str) -> (Vec<u8>, Vec<u8>) {
    let path: PathBuf = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("tests/fixtures/cliente") // relativo à crate: crates/wyd-protocol/tests/fixtures/cliente
        .join(format!("{id}.json"));
    let json = std::fs::read_to_string(&path).expect("fixture legível");
    (
        extract_hex(&json, "wireHex"),
        extract_hex(&json, "payloadHex"),
    )
}

/// Extrai um campo hexadecimal do JSON de schema fixo dos fixtures.
fn extract_hex(json: &str, key: &str) -> Vec<u8> {
    let needle = format!("\"{key}\": \"");
    let start = json
        .find(&needle)
        .unwrap_or_else(|| panic!("fixture tem {key}"))
        + needle.len();
    let end = json[start..]
        .find('"')
        .unwrap_or_else(|| panic!("{key} terminado"));
    hex_decode(&json[start..start + end])
}

fn hex_decode(hex: &str) -> Vec<u8> {
    assert!(hex.len().is_multiple_of(2), "hex com tamanho par");
    hex.as_bytes()
        .chunks(2)
        .map(|pair| {
            let s = std::str::from_utf8(pair).expect("hex ascii");
            u8::from_str_radix(s, 16).expect("dígito hex válido")
        })
        .collect()
}

/// Pipeline completo de recepção: corte → desofuscação → checksum → opcode.
///
/// O cabeçalho devolvido é lido do buffer já claro (`Type`/`ID`/`Tick`
/// trafegam ofuscados — o corte inicial só enxerga `Size`/`KeyWord`/`CheckSum`).
fn receive(wire: &[u8]) -> (Header, Vec<u8>) {
    let (_, frame_len) = split_frame(wire).expect("frame íntegro");
    assert_eq!(frame_len, wire.len(), "fixture contém um frame exato");
    let mut clear = wire.to_vec();
    let computed = decode_in_place(&mut clear).expect("desofuscação válida");
    let (header, _) = split_frame(&clear).expect("cabeçalho claro legível");
    verify_checksum(computed, header.checksum, ChecksumPolicy::Strict)
        .expect("checksum confere em Strict");
    (header, clear)
}

/// Resela pacote completo (cabeçalho + corpo) e prova byte-identidade com o fio.
///
/// Ida e volta selado → claro → selado sem perda: `encode_in_place` regrava
/// `KeyWord`/`CheckSum` e o resultado deve igualar o `wireHex` do fixture.
fn reseal(header: &Header, body: &[u8]) -> Vec<u8> {
    let mut out = vec![0u8; header.size as usize];
    header
        .encode(&mut out[..HEADER_SIZE])
        .expect("cabeçalho cabe");
    out[HEADER_SIZE..].copy_from_slice(body);
    encode_in_place(&mut out, header.key_word).expect("selagem válida");
    out
}

#[test]
fn golden_account_login() {
    let (wire, _) = load_fixture("account_login");
    let (header, clear) = receive(&wire);
    assert_eq!(header.size, 116);
    assert_eq!(header.id, 1);
    assert_eq!(header.client_tick, 1001);

    let frame = inspect(&clear).expect("opcode conhecido");
    assert_eq!(frame.opcode, Opcode::AccountLogin);
    assert_eq!(require_ported(frame.opcode), Ok(116));

    let login = AccountLogin::decode(&clear).expect("corpo íntegro");
    assert_eq!(login.account_name_bytes(), b"GOLDEN");
    assert_eq!(&login.account_password, b"SYNTHETIC\x00\x00\x00");
    assert_eq!(login.client_version, 12_000);

    let mut body = vec![0u8; AccountLogin::WIRE_LEN];
    login.encode_into(&mut body).expect("tamanho correto");
    assert_eq!(
        reseal(&header, &body[HEADER_SIZE..]),
        wire,
        "ida e volta selado → claro → selado é byte-idêntica"
    );
}

#[test]
fn golden_action() {
    let (wire, _) = load_fixture("action");
    let (_, clear) = receive(&wire);

    let frame = inspect(&clear).expect("opcode conhecido");
    assert_eq!(frame.opcode, Opcode::Action);
    assert_eq!(require_ported(frame.opcode), Ok(52));

    let action = Action::decode(&clear).expect("corpo íntegro");
    assert_eq!((action.pos_x, action.pos_y), (1702, 1728));
    assert_eq!((action.target_x, action.target_y), (1703, 1730));

    let mut body = vec![0u8; Action::WIRE_LEN];
    action.encode_into(&mut body).expect("tamanho correto");
    let (header, _) = split_frame(&clear).expect("cabeçalho claro");
    assert_eq!(
        reseal(&header, &body[HEADER_SIZE..]),
        wire,
        "ida e volta selado → claro → selado é byte-idêntica"
    );
}

#[test]
fn golden_character_login() {
    let (wire, _) = load_fixture("character_login");
    let (_, clear) = receive(&wire);

    let frame = inspect(&clear).expect("opcode conhecido");
    assert_eq!(frame.opcode, Opcode::CharacterLogin);
    assert_eq!(require_ported(frame.opcode), Ok(20));

    let login = CharacterLogin::decode(&clear).expect("corpo íntegro");
    assert_eq!((login.slot, login.force), (0, 0));

    let mut body = vec![0u8; CharacterLogin::WIRE_LEN];
    login.encode_into(&mut body).expect("tamanho correto");
    let (header, _) = split_frame(&clear).expect("cabeçalho claro");
    assert_eq!(
        reseal(&header, &body[HEADER_SIZE..]),
        wire,
        "ida e volta selado → claro → selado é byte-idêntica"
    );
}

#[test]
fn golden_create_character() {
    let (wire, _) = load_fixture("create_character");
    let (_, clear) = receive(&wire);

    let frame = inspect(&clear).expect("opcode conhecido");
    assert_eq!(frame.opcode, Opcode::CreateCharacter);
    assert_eq!(require_ported(frame.opcode), Ok(36));

    let create = CreateCharacter::decode(&clear).expect("corpo íntegro");
    assert_eq!(create.slot, 0);
    assert_eq!(wyd_protocol::nul_trimmed(&create.mob_name), b"GOLDEN");
    assert_eq!(create.mob_class, 0);

    let mut body = vec![0u8; CreateCharacter::WIRE_LEN];
    create.encode_into(&mut body).expect("tamanho correto");
    let (header, _) = split_frame(&clear).expect("cabeçalho claro");
    assert_eq!(
        reseal(&header, &body[HEADER_SIZE..]),
        wire,
        "ida e volta selado → claro → selado é byte-idêntica"
    );
}

#[test]
fn golden_header_only() {
    let (wire, _) = load_fixture("header_only");
    let (header, _) = receive(&wire);
    assert_eq!(header.size, HEADER_SIZE as u16);
    assert_eq!(header.message_type, 819);
    assert_eq!(
        reseal(&header, &[]),
        wire,
        "header reselado é byte-idêntico"
    );
}

#[test]
fn golden_handshake_offsets_confirmam_a_adr_0020() {
    // A ADR-0020 apoia o gate do login em três medidas deste pacote: `Size` total de
    // 116 B, `ClientVersion` como i32 no offset 92 e valor 12 000. Aqui elas são
    // conferidas contra os bytes do golden — se o layout ou o valor mudar, o gate
    // documentado na ADR deixa de valer e este teste acusa.
    let (wire, payload) = load_fixture("account_login");
    let (header, clear) = receive(&wire);

    assert_eq!(header.size as usize, AccountLogin::WIRE_LEN);
    assert_eq!(payload.len(), AccountLogin::WIRE_LEN);
    // `payloadHex` é o frame **claro inteiro** (cabeçalho + corpo), com o campo
    // `CheckSum` (offset 3) zerado: ele só existe depois da selagem. Todo o resto
    // do descortinado precisa bater byte a byte com o fixture.
    assert_eq!(&clear[..3], &payload[..3], "Size/KeyWord em claro");
    assert_eq!(
        clear[3], header.checksum,
        "CheckSum calculado no descortinado"
    );
    assert_eq!(clear[4..], payload[4..], "Type/ID/Tick e corpo claros");

    // Offsets declarados em `crates/wyd-protocol/src/payload/auth.rs:8-14`.
    assert_eq!(
        &payload[12..24],
        b"SYNTHETIC\x00\x00\x00",
        "offset 12: senha"
    );
    assert_eq!(
        &payload[24..40],
        b"GOLDEN\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
        "offset 24: nome da conta"
    );
    assert_eq!(&payload[40..92], &[0u8; 52][..], "offset 40: zero de 52 B");
    assert_eq!(
        &payload[92..96],
        &12_000i32.to_le_bytes(),
        "offset 92: ClientVersion do cliente oficial (ADR-0020)"
    );
    assert_eq!(
        &payload[96..100],
        &0i32.to_le_bytes(),
        "offset 96: db_need_save"
    );
    assert_eq!(
        &payload[100..116],
        &[0u8; 16][..],
        "offset 100: adapter_name"
    );

    let login = AccountLogin::decode(&clear).expect("corpo íntegro");
    assert_eq!(
        login.client_version,
        i32::from(wyd_protocol::PROTOCOL_VERSION)
    );
    assert_eq!(login.client_version, 12_000);
}

#[test]
fn golden_tampered_checksum_byte_is_rejected() {
    let (wire, _) = load_fixture("header_only");
    // Campo CheckSum (offset 3) é claro no fio: corromper 1 byte invalida.
    let mut tampered = wire.clone();
    tampered[3] ^= 0x01;
    let (header, _) = split_frame(&tampered).expect("corte não depende do checksum");
    let mut clear = tampered.clone();
    let computed = decode_in_place(&mut clear).expect("desofuscação ainda válida");
    assert_eq!(
        verify_checksum(computed, header.checksum, ChecksumPolicy::Strict),
        Err(wyd_protocol::TransformError::ChecksumMismatch {
            computed,
            declared: header.checksum,
        })
    );
}

#[test]
fn golden_truncated_frame_is_rejected() {
    let (wire, _) = load_fixture("action");
    let short = &wire[..wire.len() - 1];
    assert_eq!(
        split_frame(short),
        Err(FrameError::FrameIncomplete {
            declared: 52,
            available: 51,
        })
    );
}

#[test]
fn golden_unknown_opcode_is_rejected_after_decode() {
    let (wire, _) = load_fixture("action");
    let (_, mut clear) = receive(&wire);
    // Type claro nos offsets 4..6: 0xFFFF não existe na tabela gerada.
    clear[4..6].copy_from_slice(&0xFFFFu16.to_le_bytes());
    assert_eq!(
        inspect(&clear),
        Err(ProtocolError::Frame(FrameError::UnknownOpcode {
            value: 0xFFFF
        }))
    );
}

#[test]
fn golden_unported_opcode_never_parses() {
    // Opcode conhecido (tabela) mas sem layout documentado: gate UNVERIFIED.
    assert_eq!(
        require_ported(Opcode::Trade),
        Err(PayloadError::Unported {
            opcode: Opcode::Trade
        })
    );
}
