//! Sessão única entre os canais, provada contra os containers de verdade.
//!
//! `WYD_LIVE_CPSOCK=192.168.100.3:8281` (canal 1) e `WYD_LIVE_CPSOCK_2=192.168.100.4:8281`
//! (canal 2). Sem as duas o teste **pula** — nada de passe silencioso.
//!
//! Roteiro, na ordem do jogo:
//! 1. conta entra no canal 1 e recebe `_MSG_CNFAccountLogin`; socket fica aberto (= em jogo);
//! 2. a mesma conta tenta o canal 2 → sem confirmação, conexão fechada (o personagem não
//!    existe em dois mundos ao mesmo tempo);
//! 3. o socket do canal 1 cai → a sessão é liberada e a mesma conta entra no canal 2
//!    (transitar entre canais sem perder dado: o personagem está no banco, não na memória).

#[path = "../common/mod.rs"]
mod common;

use std::io::{Read, Write};
use std::net::TcpStream;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

use wyd_protocol::codec::keyword::encode_in_place;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, Header};

fn enderecos() -> Option<(String, String)> {
    let um = std::env::var("WYD_LIVE_CPSOCK").ok();
    let dois = std::env::var("WYD_LIVE_CPSOCK_2").ok();
    match (um, dois) {
        (Some(a), Some(b)) if !a.trim().is_empty() && !b.trim().is_empty() => Some((a, b)),
        _ => {
            eprintln!(
                "SKIP: WYD_LIVE_CPSOCK/WYD_LIVE_CPSOCK_2 ausentes — sessão entre canais não testada"
            );
            None
        }
    }
}

fn conta(prefixo: &str) -> String {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("{prefixo}{}", nanos % 100_000_000_000_000);
    nome.chars().take(16).collect()
}

fn login_frame(version: i32, key_word: u8, conta: &str) -> Vec<u8> {
    let mut frame = vec![0u8; AccountLogin::WIRE_LEN];
    Header {
        size: AccountLogin::WIRE_LEN as u16,
        key_word,
        checksum: 0,
        message_type: Opcode::AccountLogin.wire(),
        id: 1,
        client_tick: 42,
    }
    .encode(&mut frame)
    .expect("cabeçalho cabe no frame");
    let mut account_name = [0u8; 16];
    let bytes = conta.as_bytes();
    let n = bytes.len().min(account_name.len());
    account_name[..n].copy_from_slice(&bytes[..n]);
    AccountLogin {
        account_password: [0u8; 12],
        account_name,
        zero: [0u8; 52],
        client_version: version,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut frame)
    .expect("corpo cabe no frame");
    encode_in_place(&mut frame, key_word).expect("ofuscação válida");
    frame
}

/// Conecta e faz o handshake; devolve `(socket, confirmou)`.
fn entra(endereco: &str, conta: &str) -> (TcpStream, bool) {
    let mut stream = TcpStream::connect(endereco).expect("conecta no canal");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout de leitura");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    stream
        .write_all(&login_frame(12_000, 11, conta))
        .expect("login");
    let mut reply = [0u8; HEADER_SIZE];
    match stream.read_exact(&mut reply) {
        Ok(()) => {
            let mut frame = reply;
            wyd_protocol::codec::keyword::decode_in_place(&mut frame).expect("desofusca resposta");
            let header = Header::decode(&frame).expect("cabeçalho da resposta");
            (
                stream,
                Opcode::from_wire(header.message_type) == Some(Opcode::CNFAccountLogin),
            )
        }
        Err(_) => (stream, false),
    }
}

#[test]
fn mesma_conta_nao_entra_em_dois_canais_e_libera_ao_sair() {
    let Some((canal1, canal2)) = enderecos() else {
        return;
    };
    let conta = conta("s");

    // 1) canal 1 confirma e mantém o socket aberto (sessão viva).
    let (sessao1, confirmou1) = entra(&canal1, &conta);
    assert!(
        confirmou1,
        "{canal1} confirmou o handshake da conta {conta}"
    );

    // 2) a mesma conta no canal 2 é recusada **enquanto** a primeira sessão vive.
    let (_sessao2, confirmou2) = entra(&canal2, &conta);
    assert!(
        !confirmou2,
        "{canal2} aceitou a mesma conta {conta} com sessão viva no canal 1"
    );

    // 3) saindo do canal 1, a sessão libera e o canal 2 passa a aceitar (transitar).
    drop(sessao1);
    let entrou = common::esperar_ate(Duration::from_secs(10), || {
        let (_sessao3, ok) = entra(&canal2, &conta);
        ok
    });
    assert!(
        entrou,
        "{canal2} não liberou a conta {conta} depois de a sessão do canal 1 cair"
    );
}
