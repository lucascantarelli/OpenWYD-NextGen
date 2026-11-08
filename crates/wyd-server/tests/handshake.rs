//! Handshake CPSock de verdade: socket, `INITCODE`, frame ofuscado e gate de versão.
//!
//! O teste sobe o listener em porta efêmera, conecta como o cliente faz e verifica os dois
//! lados do gate: versão do cliente oficial (12 000) recebe `_MSG_CNFAccountLogin`; versão
//! histórica (7 640) é recusada e a conexão fecha sem resposta.

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_db::ServerChannel;
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::encode_in_place;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, Header};
use wyd_server::channels::World;
use wyd_server::cpsock;

fn canal(channel: i64, host: &str) -> ServerChannel {
    ServerChannel {
        group_id: 1,
        channel,
        host: host.to_string(),
        status_url: if channel == 0 {
            "http://192.168.100.3/serv00.htm".to_string()
        } else {
            String::new()
        },
        enabled: true,
    }
}

fn world() -> Arc<World> {
    Arc::new(
        World::new(vec![
            canal(0, "192.168.100.3"),
            canal(1, "192.168.100.3"),
            canal(2, "192.168.100.4"),
        ])
        .expect("foto com dois canais"),
    )
}

/// Frame de login ofuscado, exatamente como o cliente o envia.
fn login_frame(version: i32, key_word: u8) -> Vec<u8> {
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
    account_name[..4].copy_from_slice(b"test");
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

fn serve() -> u16 {
    let listener = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
    let port = listener.local_addr().expect("endereço").port();
    let world = world();
    std::thread::spawn(move || cpsock::serve_listener(listener, world, VersionGate::default()));
    port
}

#[test]
fn versao_do_cliente_oficial_completa_o_handshake() {
    let port = serve();
    let mut stream = TcpStream::connect(("127.0.0.1", port)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    stream.write_all(&login_frame(12_000, 7)).expect("login");

    let mut reply = [0u8; HEADER_SIZE];
    stream.read_exact(&mut reply).expect("resposta do servidor");
    let mut frame = reply;
    let computed = wyd_protocol::codec::keyword::decode_in_place(&mut frame).expect("desofusca");
    assert_eq!(computed, frame[3], "checksum da resposta confere");
    let header = Header::decode(&frame).expect("cabeçalho da resposta");
    assert_eq!(
        Opcode::from_wire(header.message_type),
        Some(Opcode::CNFAccountLogin),
        "servidor confirma o login com _MSG_CNFAccountLogin"
    );
    assert_eq!(
        header.size as usize, HEADER_SIZE,
        "resposta só de cabeçalho"
    );
}

#[test]
fn versao_fora_do_gate_e_recusada_com_conexao_fechada() {
    let port = serve();
    let mut stream = TcpStream::connect(("127.0.0.1", port)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    stream
        .write_all(&login_frame(7_640, 3))
        .expect("login histórico");

    let mut buf = Vec::new();
    let lido = stream.read_to_end(&mut buf).unwrap_or(0);
    assert_eq!(lido, 0, "sem confirmação: o servidor fecha a conexão");
    assert!(buf.is_empty());
}

/// Prova contra os containers de verdade: `WYD_LIVE_CPSOCK=192.168.100.3:8281`.
///
/// Sem a variável o teste não roda (o CI não tem a stack), com ela o mesmo frame do cliente
/// atravessa o Docker e exige `_MSG_CNFAccountLogin` — handshake real, não in-process.
#[test]
fn handshake_contra_servidor_vivo() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        return;
    };
    let mut stream = TcpStream::connect(&endereco).expect("conecta no container");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    stream.write_all(&login_frame(12_000, 11)).expect("login");
    let mut reply = [0u8; HEADER_SIZE];
    stream
        .read_exact(&mut reply)
        .expect("resposta do container");
    let mut frame = reply;
    wyd_protocol::codec::keyword::decode_in_place(&mut frame).expect("desofusca");
    let header = Header::decode(&frame).expect("cabeçalho");
    assert_eq!(
        Opcode::from_wire(header.message_type),
        Some(Opcode::CNFAccountLogin),
        "{endereco} confirmou o handshake"
    );
}

/// Negativo vivo: `WYD_LIVE_CPSOCK_BAD=192.168.100.4:8281` envia 7 640 e exige conexão fechada.
#[test]
fn versao_historica_e_recusada_pelo_servidor_vivo() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK_BAD") else {
        return;
    };
    let mut stream = TcpStream::connect(&endereco).expect("conecta no container");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    stream.write_all(&login_frame(7_640, 5)).expect("login");
    let mut buf = Vec::new();
    let lido = std::io::Read::read_to_end(&mut stream, &mut buf).unwrap_or(1);
    assert_eq!(lido, 0, "{endereco} recusou 7640 e fechou sem resposta");
}

#[test]
fn initcode_errado_nao_entra_no_handshake() {
    let port = serve();
    let mut stream = TcpStream::connect(("127.0.0.1", port)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream
        .write_all(&0xDEAD_BEEFu32.to_le_bytes())
        .expect("lixo");
    let mut buf = Vec::new();
    let lido = stream.read_to_end(&mut buf).unwrap_or(0);
    assert_eq!(lido, 0, "conexão sem INITCODE é descartada");
}
