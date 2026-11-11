//! Flood contido, legítimo preservado; sanitização estrita ponta a ponta.
//!
//! Pipeline tamanho → opcode → payload (08-2) + tetos (06-2) contra entrada
//! hostil: truncado, desconhecido, aleatório. Nada aqui pode causar panic —
//! o teste falharia — e todo excedente é recusado com erro tipado.

use wyd_net::accepts_len;
use wyd_net::limits::{default_bucket, ConnectionGate, Deny};
use wyd_protocol::codec::{decode_in_place, encode_in_place, verify_checksum, ChecksumPolicy};
use wyd_protocol::{
    inspect, require_ported, split_frame, Action, FrameError, Header, Opcode, PayloadError,
    ProtocolError, WirePayload,
};

/// Pacote selado válido (Action) construído pela API — o "legítimo".
fn sealed_action() -> Vec<u8> {
    let mut pkt = vec![0u8; Action::WIRE_LEN];
    Header {
        size: Action::WIRE_LEN as u16,
        key_word: 0,
        checksum: 0,
        message_type: 876,
        id: 1,
        client_tick: 7,
    }
    .encode(&mut pkt[..12])
    .expect("cabeçalho cabe");
    Action {
        pos_x: 1702,
        pos_y: 1728,
        effect: 0,
        speed: 0,
        route: [0u8; 24],
        target_x: 1703,
        target_y: 1730,
    }
    .encode_into(&mut pkt)
    .expect("corpo cabe");
    encode_in_place(&mut pkt, 5).expect("selagem válida");
    pkt
}

/// Recepção estrita: corte → desofuscação → checksum → opcode → payload.
fn receive_strict(wire: &[u8]) -> Result<Action, ProtocolError> {
    if !accepts_len(wire.len()) {
        return Err(ProtocolError::Frame(FrameError::SizeOutOfRange {
            size: wire.len(),
        }));
    }
    let (header, _) = split_frame(wire)?;
    let mut clear = wire.to_vec();
    let computed = decode_in_place(&mut clear)?;
    verify_checksum(computed, header.checksum, ChecksumPolicy::Strict)?;
    split_frame(&clear)?;
    let frame = inspect(&clear)?;
    let _ = require_ported(frame.opcode)?;
    Ok(Action::decode(&clear)?)
}

#[test]
fn flood_is_contained_and_legitimate_is_preserved() {
    // Atacante: rajada de 10 000 pacotes no mesmo ms, teto 100.
    let mut attacker = default_bucket();
    let mut ok = 0u32;
    let mut denied = 0u32;
    for _ in 0..10_000 {
        match attacker.try_consume(1_000) {
            Ok(()) => ok += 1,
            Err(Deny::RateLimited) => denied += 1,
            Err(e) => panic!("negação inesperada: {e}"),
        }
    }
    assert_eq!((ok, denied), (100, 9_900));
    // Legítimo (balde próprio): tudo passa no mesmo instante.
    let mut legit = default_bucket();
    for _ in 0..50 {
        legit.try_consume(1_000).expect("legítimo preservado");
    }
    eprintln!("flood: atacante 100/10000 passa; legítimo 50/50 passa");
}

#[test]
fn handshake_flood_per_ip_keeps_legitimate_out() {
    let mut gate = ConnectionGate::local();
    for _ in 0..3 {
        gate.admit(0x0A000001, 0).expect("atacante conecta");
    }
    assert_eq!(
        gate.admit(0x0A000001, 0),
        Err(Deny::TooManyPerIp { limit: 3 })
    );
    gate.admit(0x0A000002, 0).expect("legítimo conecta");
}

#[test]
fn legitimate_packet_passes_strict_pipeline() {
    let wire = sealed_action();
    let action = receive_strict(&wire).expect("legítimo passa");
    assert_eq!((action.pos_x, action.pos_y), (1702, 1728));
}

#[test]
fn every_truncation_is_rejected_before_payload() {
    let wire = sealed_action();
    for cut in 0..wire.len() {
        assert!(
            receive_strict(&wire[..cut]).is_err(),
            "corte em {cut} bytes deve ser recusado"
        );
    }
}

#[test]
fn unknown_opcode_is_rejected_after_decode() {
    let wire = sealed_action();
    // O Type trafega selado: decodifica, troca o Type claro, reinspeciona.
    let (header, _) = split_frame(&wire).expect("corte");
    let mut clear = wire.clone();
    let computed = decode_in_place(&mut clear).expect("desofusca");
    verify_checksum(computed, header.checksum, ChecksumPolicy::Strict).expect("checksum");
    clear[4..6].copy_from_slice(&0xFFFFu16.to_le_bytes());
    assert_eq!(
        inspect(&clear),
        Err(ProtocolError::Frame(FrameError::UnknownOpcode {
            value: 0xFFFF
        }))
    );
    assert_eq!(
        require_ported(Opcode::Trade),
        Err(PayloadError::Unported {
            opcode: Opcode::Trade
        })
    );
}

#[test]
fn fuzz_minimo_never_panics_and_oversize_always_rejected() {
    // LCG determinístico: 5000 buffers hostis, 0..5000 bytes.
    let mut state = 0xF00DCAFEu64;
    let mut lcg = || {
        state = state
            .wrapping_mul(6364136223846793005)
            .wrapping_add(1442695040888963407);
        (state >> 33) as usize
    };
    let mut oversized = 0u32;
    let mut rejected_early = 0u32;
    for _ in 0..5_000 {
        let n = lcg() % 5001;
        let buf: Vec<u8> = (0..n).map(|_| (lcg() % 256) as u8).collect();
        if !accepts_len(buf.len()) {
            oversized += 1;
            assert!(receive_strict(&buf).is_err());
            continue;
        }
        // Pode passar ou falhar — o que não pode é causar panic.
        let _ = receive_strict(&buf);
        rejected_early += u32::from(receive_strict(&buf).is_err());
    }
    assert!(oversized > 0, "fuzz cobre excedentes");
    eprintln!(
        "fuzz: 5000 buffers, {oversized} excedentes recusados, {rejected_early} recusas precoces"
    );
}
