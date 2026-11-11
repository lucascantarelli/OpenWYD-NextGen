//! Troca de PIN **pelo caminho do portal**: o gateway chama `wyd_admin::service::pin`,
//! que abre uma conexão CPSock e faz o `0x0FDE` em duas etapas no servidor de jogo.
//!
//! O teste usa a mesma função que a rota `account_web/change_pin` chama, contra um
//! servidor de verdade e o banco de verdade, e depois confere pelo fio que o PIN
//! antigo deixou de valer e que o hash gravado é o do PIN novo.

#[path = "../common/mod.rs"]
mod common;

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::auth::hash_secret;
use wyd_admin::hash_password;
use wyd_admin::service::pin::{change_pin, PinError};
use wyd_admin::verify_password;
use wyd_db::{Accounts, PgBackend, PoolConfig, Role, ServerChannel};
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::decode_in_place;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, Header};
use wyd_server::auth::Authenticator;
use wyd_server::channels::World;
use wyd_server::characters::SelectionStore;
use wyd_server::cpsock;
use wyd_server::session::SessionGate;

const SENHA: &str = "SenhaForte1";
const PIN_ATUAL: &str = "1122";
const PIN_NOVO: &str = "9988";

fn runtime() -> Arc<tokio::runtime::Runtime> {
    Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(1)
            .enable_all()
            .build()
            .expect("runtime"),
    )
}

fn servidor(backend: Arc<PgBackend>, rt: Arc<tokio::runtime::Runtime>) -> u16 {
    let auth = Arc::new(Authenticator::new(Arc::clone(&backend), Arc::clone(&rt)));
    let cerca = Arc::new(SessionGate::new(
        Arc::clone(&backend),
        Arc::clone(&rt),
        1,
        1,
        Duration::from_secs(120),
        Duration::from_secs(180),
    ));
    let selection = Arc::new(SelectionStore::new(Arc::clone(&backend), Arc::clone(&rt)));
    let world = Arc::new(
        World::new(vec![ServerChannel {
            group_id: 1,
            channel: 1,
            host: "127.0.0.1".to_string(),
            status_url: String::new(),
            enabled: true,
        }])
        .expect("canal"),
    );
    let listener = TcpListener::bind("127.0.0.1:0").expect("porta");
    let porta = listener.local_addr().expect("endereço").port();
    std::thread::spawn(move || {
        cpsock::serve_listener(
            listener,
            world,
            VersionGate::default(),
            Some(auth),
            Some(cerca),
            Some(selection),
        )
    });
    porta
}

fn conta_unica() -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    format!("pp{:013}", nanos % 10_000_000_000_000)
}

fn quadro(opcode: Opcode, len: usize) -> Vec<u8> {
    let mut frame = vec![0u8; len];
    Header {
        size: len as u16,
        key_word: 11,
        checksum: 0,
        message_type: opcode.wire(),
        id: 1,
        client_tick: 1,
    }
    .encode(&mut frame)
    .expect("header");
    frame
}

fn fechar(mut frame: Vec<u8>) -> Vec<u8> {
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

fn frame_login(conta: &str) -> Vec<u8> {
    let mut nome = [0u8; 16];
    nome[..conta.len()].copy_from_slice(conta.as_bytes());
    let mut senha = [0u8; 12];
    senha[..SENHA.len()].copy_from_slice(SENHA.as_bytes());
    let mut frame = quadro(Opcode::AccountLogin, AccountLogin::WIRE_LEN);
    AccountLogin {
        account_password: senha,
        account_name: nome,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut frame)
    .expect("corpo");
    fechar(frame)
}

fn frame_pin(pin: &str, change_numeric: i32) -> Vec<u8> {
    let mut token = [0u8; 6];
    token[..pin.len()].copy_from_slice(pin.as_bytes());
    let mut frame = quadro(Opcode::AccountSecure, AccountSecure::WIRE_LEN);
    AccountSecure {
        numeric_token: token,
        unknown: [0u8; 10],
        change_numeric,
    }
    .encode_into(&mut frame)
    .expect("corpo");
    fechar(frame)
}

fn ler_frame(stream: &mut TcpStream) -> Vec<u8> {
    let mut head = [0u8; HEADER_SIZE];
    stream.read_exact(&mut head).expect("frame");
    let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
    let mut frame = vec![0u8; size];
    frame[..HEADER_SIZE].copy_from_slice(&head);
    if size > HEADER_SIZE {
        stream.read_exact(&mut frame[HEADER_SIZE..]).expect("corpo");
    }
    frame
}

fn abrir(frame: &[u8]) -> Opcode {
    let mut limpo = frame.to_vec();
    let computado = decode_in_place(&mut limpo).expect("desofusca");
    assert_eq!(computado, limpo[3], "checksum");
    let header = Header::decode(&limpo[..HEADER_SIZE]).expect("header");
    Opcode::from_wire(header.message_type).expect("opcode")
}

/// Abre uma conexão de jogo já autenticada (login com senha).
fn conecta(porta: u16, conta: &str) -> TcpStream {
    let mut stream = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("init");
    stream.write_all(&frame_login(conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut stream)), Opcode::CNFAccountLogin);
    stream
}

fn pin(stream: &mut TcpStream, valor: &str, change: i32) -> Opcode {
    stream.write_all(&frame_pin(valor, change)).expect("pin");
    abrir(&ler_frame(stream))
}

#[test]
fn troca_de_pin_pelo_portal_funciona_e_invalida_o_antigo() {
    let Some(dsn) = common::dsn_ou_skip("PIN pelo portal") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let porta = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    // A primeira entrada não cadastra mais PIN (BUG-02): registra o atual direto.
    let pin_hash = hash_secret(PIN_ATUAL).expect("hash do PIN");
    rt.block_on(backend.set_pin_hash(&conta, &pin_hash))
        .expect("PIN cadastrado");

    // Primeiro acesso do jogo: valida o PIN atual já cadastrado.
    let mut primeira = conecta(porta, &conta);
    assert_eq!(pin(&mut primeira, PIN_ATUAL, 0), Opcode::AccountSecure);
    drop(primeira);

    let endereco = format!("127.0.0.1:{porta}");

    // PIN atual errado: recusa do servidor, nada gravado.
    assert_eq!(
        change_pin(&endereco, &conta, SENHA, "0000", PIN_NOVO),
        Err(PinError::Rejected)
    );

    // Caminho do portal: login + passo 1 + passo 2 no servidor de jogo.
    change_pin(&endereco, &conta, SENHA, PIN_ATUAL, PIN_NOVO).expect("troca pelo portal");

    // Banco: hash confere o novo e não o antigo.
    let gravado = rt
        .block_on(backend.pin_hash(&conta))
        .expect("hash")
        .expect("PIN cadastrado");
    assert!(verify_password(PIN_NOVO, &gravado), "hash novo vale");
    assert!(
        !verify_password(PIN_ATUAL, &gravado),
        "hash antigo não vale"
    );

    // Fio: PIN antigo recusado, PIN novo aceito (mesmo protocolo do jogo).
    let mut antigo = conecta(porta, &conta);
    assert_eq!(pin(&mut antigo, PIN_ATUAL, 0), Opcode::AccountSecureFail);
    drop(antigo);
    let mut novo = conecta(porta, &conta);
    assert_eq!(pin(&mut novo, PIN_NOVO, 0), Opcode::AccountSecure);
}
