//! Fixtures compartilhadas: monta pacote (header + corpo + ofuscação) e reabre.
//!
//! Usa só a API pública do crate — é o contrato que a integração testa.

use wyd_protocol::{
    decode_in_place, encode_in_place, split_frame, verify_checksum, ChecksumPolicy, Header,
    WirePayload, HEADER_SIZE,
};

/// Pacote selado pronto para o fio: header + corpo codificado + ofuscação.
#[must_use]
pub fn sealed_packet<T>(payload: &T, id: u16, client_tick: u32, key_word: u8) -> Vec<u8>
where
    T: WirePayload,
{
    let mut buf = vec![0u8; T::WIRE_LEN];
    Header {
        size: T::WIRE_LEN as u16,
        key_word: 0,
        checksum: 0,
        message_type: T::OPCODE.wire(),
        id,
        client_tick,
    }
    .encode(&mut buf)
    .expect("header cabe no buffer do próprio tamanho");
    payload
        .encode_into(&mut buf)
        .expect("corpo cabe no próprio tamanho");
    encode_in_place(&mut buf, key_word).expect("ofuscação de pacote bem formado");
    assert!(buf.len() > HEADER_SIZE);
    buf
}

/// Reabre um pacote do fio: desofusca, enquadra (checksum estrito) e decodifica.
///
/// A ordem importa: o `Type` trafega ofuscado (o loop do CPSock vai de 4 até
/// `Size`), então o enquadramento só vale depois do `decode`.
pub fn open_packet<T>(wire: &mut [u8]) -> (Header, T)
where
    T: WirePayload,
{
    let computed = decode_in_place(wire).expect("desofuscação válida");
    let (header, len) = split_frame(wire).expect("frame válido");
    assert_eq!(len, T::WIRE_LEN);
    assert_eq!(header.message_type, T::OPCODE.wire());
    verify_checksum(computed, header.checksum, ChecksumPolicy::Strict)
        .expect("checksum de pacote íntegro");
    let body = T::decode(wire).expect("payload válido");
    (header, body)
}
