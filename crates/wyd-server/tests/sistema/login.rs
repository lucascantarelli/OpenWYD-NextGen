//! Gate de entrada **real**: senha da conta + senha numérica no listener CPSock.
//!
//! Cada teste conversa com `cpsock::serve_listener` de verdade (socket, frame ofuscado,
//! Argon2id no banco Postgres) — a única coisa trocada é o banco de desenvolvimento, que é
//! o mesmo da stack. Sem `WYD_TEST_DATABASE_URL`/`DATABASE_URL` os testes não rodam
//! (o CI não tem a stack); com a URL, cada caso prova o fluxo inteiro.

#[path = "../common/mod.rs"]
mod common;

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::auth::hash_secret;
use wyd_admin::hash_password;
use wyd_db::{Accounts, PgBackend, PoolConfig, Role, ServerChannel};
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, CreateCharacter, Header};
use wyd_server::auth::Authenticator;
use wyd_server::channels::World;
use wyd_server::characters::SelectionStore;
use wyd_server::cpsock;
use wyd_server::logs;
use wyd_server::session::SessionGate;

const SENHA: &str = "SenhaForte1";
const PIN: &str = "123456";

fn runtime() -> Arc<tokio::runtime::Runtime> {
    Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(1)
            .enable_all()
            .build()
            .expect("runtime"),
    )
}

/// Sobe listener real com autenticação e cerca; devolve (porta, autenticador, backend).
fn servidor(dsn: &str) -> (u16, Arc<Authenticator>, Arc<PgBackend>) {
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(dsn, PoolConfig::local()))
            .expect("conecta no Postgres de desenvolvimento"),
    );
    let auth = Arc::new(Authenticator::new(Arc::clone(&backend), Arc::clone(&rt)));
    let cerca = Arc::new(SessionGate::new(
        Arc::clone(&backend),
        Arc::clone(&rt),
        1,
        1,
        Duration::from_secs(120),
        Duration::from_secs(180),
    ));
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
    let selection = Arc::new(SelectionStore::new(Arc::clone(&backend), Arc::clone(&rt)));
    let auth_ret = Arc::clone(&auth);
    let listener = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
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
    (porta, auth_ret, backend)
}

/// A conta é única por execução: a cerca e o bloqueio são por login.
/// Sequência por processo: o relógio do Windows tem granularidade grossa e
/// testes paralelos podiam gerar o mesmo `login` (violação de UNIQUE).
static SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

fn conta_unica() -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let seq = SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
    format!("qa{:011}{:02}", nanos % 100_000_000_000, seq % 100)
}

fn nova_conta(rt: &tokio::runtime::Runtime, backend: &PgBackend, conta: &str) {
    rt.block_on(backend.create_account(conta, Role::Player))
        .expect("conta criada");
    let hash = hash_password(SENHA).expect("hash argon2id");
    rt.block_on(backend.set_credentials(conta, &hash, Role::Player))
        .expect("credencial");
}

fn frame_login(conta: &str, senha: &str) -> Vec<u8> {
    let mut nome = [0u8; 16];
    nome[..conta.len()].copy_from_slice(conta.as_bytes());
    let mut senha_bytes = [0u8; 12];
    let s = senha.as_bytes();
    senha_bytes[..s.len().min(12)].copy_from_slice(&s[..s.len().min(12)]);
    let mut frame = vec![0u8; AccountLogin::WIRE_LEN];
    Header {
        size: AccountLogin::WIRE_LEN as u16,
        key_word: 11,
        checksum: 0,
        message_type: Opcode::AccountLogin.wire(),
        id: 1,
        client_tick: 1,
    }
    .encode(&mut frame)
    .expect("header");
    AccountLogin {
        account_password: senha_bytes,
        account_name: nome,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut frame)
    .expect("corpo");
    encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

fn frame_pin(pin: &str) -> Vec<u8> {
    let mut token = [0u8; 6];
    let b = pin.as_bytes();
    token[..b.len()].copy_from_slice(b);
    let mut frame = vec![0u8; AccountSecure::WIRE_LEN];
    Header {
        size: AccountSecure::WIRE_LEN as u16,
        key_word: 11,
        checksum: 0,
        message_type: Opcode::AccountSecure.wire(),
        id: 1,
        client_tick: 2,
    }
    .encode(&mut frame)
    .expect("header");
    AccountSecure {
        numeric_token: token,
        unknown: [0u8; 10],
        change_numeric: 0,
    }
    .encode_into(&mut frame)
    .expect("corpo do PIN");
    encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

fn frame_criacao(slot: i32, nome: &str) -> Vec<u8> {
    let mut campo = [0u8; 16];
    campo[..nome.len()].copy_from_slice(nome.as_bytes());
    let mut frame = vec![0u8; CreateCharacter::WIRE_LEN];
    Header {
        size: CreateCharacter::WIRE_LEN as u16,
        key_word: 11,
        checksum: 0,
        message_type: Opcode::CreateCharacter.wire(),
        id: 1,
        client_tick: 3,
    }
    .encode(&mut frame)
    .expect("header");
    CreateCharacter {
        slot,
        mob_name: campo,
        mob_class: 0,
    }
    .encode_into(&mut frame)
    .expect("corpo");
    encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

fn conectar(porta: u16) -> TcpStream {
    let stream = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    stream
}

/// Lê um frame inteiro (cabeçalho + corpo declarado no `Size`) e devolve o opcode.
fn ler_opcode(stream: &mut TcpStream) -> Opcode {
    let mut head = [0u8; HEADER_SIZE];
    stream.read_exact(&mut head).expect("resposta do servidor");
    let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
    let mut frame = vec![0u8; size];
    frame[..HEADER_SIZE].copy_from_slice(&head);
    if size > HEADER_SIZE {
        stream.read_exact(&mut frame[HEADER_SIZE..]).expect("corpo");
    }
    let computed = decode_in_place(&mut frame).expect("desofusca");
    assert_eq!(computed, frame[3], "checksum da resposta");
    let header = Header::decode(&frame).expect("header");
    Opcode::from_wire(header.message_type).expect("opcode conhecido")
}

/// Login de verdade: INITCODE + frame; devolve o opcode da resposta (ou None se fechou).
fn login(stream: &mut TcpStream, conta: &str, senha: &str) -> Option<Opcode> {
    stream.write_all(&INITCODE.to_le_bytes()).ok();
    stream.write_all(&frame_login(conta, senha)).ok();
    let mut head = [0u8; HEADER_SIZE];
    match stream.read_exact(&mut head) {
        Ok(()) => {
            let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
            let mut frame = vec![0u8; size];
            frame[..HEADER_SIZE].copy_from_slice(&head);
            if size > HEADER_SIZE {
                stream.read_exact(&mut frame[HEADER_SIZE..]).expect("corpo");
            }
            let computed = decode_in_place(&mut frame).expect("desofusca");
            assert_eq!(computed, frame[3]);
            let header = Header::decode(&frame).expect("header");
            Opcode::from_wire(header.message_type)
        }
        Err(_) => None,
    }
}

fn entrada(conta: &str) -> (u16, Arc<PgBackend>, Arc<tokio::runtime::Runtime>) {
    let dsn = common::dsn_ou_skip("login do canal")
        .expect("WYD_TEST_DATABASE_URL/DATABASE_URL para o gate de login");
    let (porta, _auth, backend) = servidor(&dsn);
    let rt = runtime();
    nova_conta(&rt, &backend, conta);
    // Conta com PIN já cadastrado: é o caso de "PIN errado", não de primeira entrada.
    let hash = hash_secret(PIN).expect("hash do PIN");
    rt.block_on(backend.set_pin_hash(conta, &hash))
        .expect("PIN cadastrado");
    (porta, backend, rt)
}

/// Conta nova **sem** PIN (primeira entrada).
fn entrada_sem_pin(conta: &str) -> (u16, Arc<PgBackend>, Arc<tokio::runtime::Runtime>) {
    let dsn = common::dsn_ou_skip("login do canal")
        .expect("WYD_TEST_DATABASE_URL/DATABASE_URL para o gate de login");
    let (porta, _auth, backend) = servidor(&dsn);
    let rt = runtime();
    nova_conta(&rt, &backend, conta);
    (porta, backend, rt)
}

#[test]
fn senha_e_pin_corretos_chegam_na_selecao() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let (porta, backend, rt) = entrada(&conta);
    let mut s = conectar(porta);
    assert_eq!(login(&mut s, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    s.write_all(&frame_pin(PIN)).expect("PIN");
    assert_eq!(ler_opcode(&mut s), Opcode::AccountSecure, "PIN aceito");

    let nome = format!("Ch{}", &conta[2..]);
    s.write_all(&frame_criacao(1, &nome)).expect("criação");
    assert_eq!(
        ler_opcode(&mut s),
        Opcode::CNFNewCharacter,
        "seleção alcançada depois do PIN"
    );
    let personagens = rt
        .block_on(wyd_db::Characters::list_by_account(
            &*backend,
            rt.block_on(backend.find_by_login(&conta))
                .expect("conta")
                .id,
        ))
        .expect("lista");
    assert!(personagens.iter().any(|p| p.name == nome), "criou no banco");
}

#[test]
fn senha_errada_e_recusada_sem_sessao() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let (porta, _backend, _rt) = entrada(&conta);
    let mut s = conectar(porta);
    assert_eq!(
        login(&mut s, &conta, "senha-errada"),
        Some(Opcode::NewAccountFail),
        "senha errada recebe recusa, nunca CNF"
    );
    // A recusa vem com `NewAccountFail` + painel explicativo antes de fechar; o que
    // importa para a cerca é que o login correto seguinte não encontre sessão presa.
    let mut resto = Vec::new();
    let _ = s.read_to_end(&mut resto);

    // Se houvesse sessão meio-aberta, este login correto bateria na cerca.
    let mut ok = conectar(porta);
    assert_eq!(login(&mut ok, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    ok.write_all(&frame_pin(PIN)).expect("PIN");
    assert_eq!(ler_opcode(&mut ok), Opcode::AccountSecure);
}

#[test]
fn pin_errado_e_recusado_sem_sessao() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let (porta, _backend, _rt) = entrada(&conta);
    let mut s = conectar(porta);
    assert_eq!(login(&mut s, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    s.write_all(&frame_pin("0000")).expect("PIN errado");
    assert_eq!(ler_opcode(&mut s), Opcode::AccountSecureFail);
    let mut resto = Vec::new();
    assert_eq!(s.read_to_end(&mut resto).unwrap_or(0), 0, "conexão fechada");

    let mut ok = conectar(porta);
    assert_eq!(login(&mut ok, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    ok.write_all(&frame_pin(PIN)).expect("PIN correto");
    assert_eq!(
        ler_opcode(&mut ok),
        Opcode::AccountSecure,
        "sem sessão presa"
    );
}

/// Primeira entrada sem PIN: o cliente 7662 não manda `_MSG_AccountSecure` e o canal
/// não exige cadastro (BUG-02) — a seleção acontece direto.
#[test]
fn primeira_entrada_sem_pin_vai_direto_para_a_selecao() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let (porta, backend, rt) = entrada_sem_pin(&conta);
    assert!(
        rt.block_on(backend.pin_hash(&conta))
            .expect("pin_hash")
            .is_none(),
        "conta nova começa sem PIN"
    );

    let mut s = conectar(porta);
    assert_eq!(login(&mut s, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    let nome = format!("Ch{}", &conta[2..]);
    s.write_all(&frame_criacao(1, &nome))
        .expect("criação sem PIN");
    assert_eq!(
        ler_opcode(&mut s),
        Opcode::CNFNewCharacter,
        "seleção sem PIN"
    );

    let personagens = rt
        .block_on(wyd_db::Characters::list_by_account(
            &*backend,
            rt.block_on(backend.find_by_login(&conta))
                .expect("conta")
                .id,
        ))
        .expect("lista");
    assert!(personagens.iter().any(|p| p.name == nome), "criou no banco");
    assert!(
        rt.block_on(backend.pin_hash(&conta))
            .expect("pin_hash")
            .is_none(),
        "PIN continua opcional"
    );
}

#[test]
fn cinco_pins_errados_bloqueiam_a_conta() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let (porta, _backend, _rt) = entrada(&conta);
    for i in 0..5 {
        let mut s = conectar(porta);
        assert_eq!(login(&mut s, &conta, SENHA), Some(Opcode::CNFAccountLogin));
        s.write_all(&frame_pin("0000")).expect("PIN errado");
        assert_eq!(
            ler_opcode(&mut s),
            Opcode::AccountSecureFail,
            "falha {i} recusada"
        );
    }
    // Sexta tentativa, agora com o PIN certo: continua bloqueada.
    let mut s = conectar(porta);
    assert_eq!(login(&mut s, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    s.write_all(&frame_pin(PIN)).expect("PIN certo");
    assert_eq!(
        ler_opcode(&mut s),
        Opcode::AccountSecureFail,
        "bloqueio ativo"
    );
    let linhas = logs::tail(512).join("\n");
    assert!(linhas.contains("bloqueado"), "log registra o bloqueio");
}

#[test]
fn pin_e_senha_nunca_aparecem_no_log() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let pin_secreto = "987654";
    let (porta, _backend, _rt) = entrada_sem_pin(&conta);
    let mut s = conectar(porta);
    assert_eq!(login(&mut s, &conta, SENHA), Some(Opcode::CNFAccountLogin));
    s.write_all(&frame_pin(pin_secreto)).expect("PIN");
    assert_eq!(ler_opcode(&mut s), Opcode::AccountSecure);
    let mut s = conectar(porta);
    let _ = login(&mut s, &conta, SENHA);
    s.write_all(&frame_pin("000000")).ok();
    let _ = ler_opcode(&mut s);

    let linhas = logs::tail(512).join("\n");
    assert!(!linhas.contains(pin_secreto), "PIN novo não vaza no log");
    assert!(!linhas.contains(SENHA), "senha não vaza no log");
    assert!(!linhas.contains("$argon2"), "hash não vaza no log");
}

/// Bloqueio de moderação vale na entrada do canal (WEB-03).
///
/// Senha correta + conta bloqueada → nenhum `CNF` (recusa antes da sessão).
/// Liberar volta a autenticar sem exigir novo cadastro de PIN.
#[test]
fn conta_bloqueada_e_recusada_e_volta_a_logar_apos_desbloqueio() {
    let Some(_dsn) = common::dsn_ou_skip("login do canal") else {
        return;
    };
    let conta = conta_unica();
    let (porta, backend, rt) = entrada(&conta);
    let id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;
    rt.block_on(backend.set_blocked(&conta, true, "gate 07-3", id))
        .expect("bloqueio");

    let mut s = conectar(porta);
    assert_eq!(
        login(&mut s, &conta, SENHA),
        Some(Opcode::NewAccountFail),
        "conta bloqueada recebe recusa, nunca CNF"
    );
    drop(s);

    rt.block_on(backend.set_blocked(&conta, false, "", id))
        .expect("liberação");
    let mut s = conectar(porta);
    assert_eq!(
        login(&mut s, &conta, SENHA),
        Some(Opcode::CNFAccountLogin),
        "desbloqueada volta a logar"
    );
}
