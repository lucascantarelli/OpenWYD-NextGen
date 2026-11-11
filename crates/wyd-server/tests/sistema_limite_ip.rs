//! Limite de conexões simultâneas por IP: 4–5 janelas do jogo passam, abuso não.

use std::io::{ErrorKind, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_db::ServerChannel;
use wyd_net::VersionGate;
use wyd_protocol::wire::frame::INITCODE;
use wyd_server::channels::World;
use wyd_server::cpsock::{self, ConnectionLimit};

fn servidor(limite: usize) -> u16 {
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
        cpsock::serve_listener_with_limit(
            listener,
            world,
            VersionGate::default(),
            None,
            None,
            None,
            ConnectionLimit::new(limite),
        )
    });
    porta
}

/// `true` quando o servidor fechou a conexão sem responder nada (recusa por limite).
fn recusada(stream: &mut TcpStream) -> bool {
    let mut buffer = [0u8; 1];
    match stream.read(&mut buffer) {
        Ok(0) => true,
        Ok(_) => false,
        Err(error) => matches!(
            error.kind(),
            ErrorKind::ConnectionReset | ErrorKind::BrokenPipe
        ),
    }
}

fn conecta(porta: u16) -> TcpStream {
    let stream = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_millis(700)))
        .expect("timeout");
    stream
}

#[test]
fn segunda_conexao_do_mesmo_ip_e_recusada_e_volta_ao_sair() {
    let porta = servidor(1);
    // Primeira conexão ocupa a única vaga do IP.
    let mut primeira = conecta(porta);
    // Dá tempo do accept registrar a vaga.
    std::thread::sleep(Duration::from_millis(200));
    let mut segunda = conecta(porta);
    assert!(
        recusada(&mut segunda),
        "segunda conexão do mesmo IP tem de ser fechada sem resposta"
    );
    assert!(
        wyd_server::metrics::ip_limit_refusals() >= 1,
        "recusa por IP conta no /metrics (wyd_ip_limit_refusals_total)"
    );
    // A primeira continua viva: recebe o CNF do handshake ao pedir login.
    primeira
        .write_all(&INITCODE.to_le_bytes())
        .expect("initcode");
    drop(primeira);
    // Sem a primeira, a vaga volta e uma nova conexão não é recusada. A liberação acontece na
    // thread do accept (depois do EOF), então a espera é **observada** com prazo: recusar de
    // novo não é erro — continuar recusando depois do prazo é.
    let limite = std::time::Instant::now() + Duration::from_secs(5);
    let mut terceira = loop {
        let mut tentativa = conecta(porta);
        if !recusada(&mut tentativa) {
            break tentativa;
        }
        assert!(
            std::time::Instant::now() < limite,
            "vaga não voltou após a primeira conexão sair"
        );
        std::thread::sleep(Duration::from_millis(50));
    };
    let _ = terceira.write_all(&INITCODE.to_le_bytes());
}
