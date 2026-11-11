//! Sessão do canal **depois** do handshake (FASE-08-2): a conexão não fecha no `CNF`.
//!
//! O defeito corrigido: o laço antigo lia só o cabeçalho e encerrava no primeiro pacote — o
//! cliente era desconectado (e a conta liberada) antes de escolher personagem. Aqui o listener
//! roda em processo e o teste conversa com ele como o cliente faz:
//!
//! 1. `INITCODE` + `_MSG_AccountLogin` → `_MSG_CNFAccountLogin`;
//! 2. pacote que o processo não serve (`_MSG_War`, `_MSG_Ping`) → **sem resposta** e conexão viva;
//! 3. `_MSG_CharacterLogin`/`_MSG_CreateCharacter` → `*Fail` (o processo de teste roda sem
//!    banco: recusar é o correto, inventar sucesso não) e a conexão **continua** aberta depois;
//! 4. `Size` inválido → única condição que encerra a conexão.

use std::io::{ErrorKind, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_db::ServerChannel;
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{
    AccountLogin, AccountSecure, CharacterLogin, CreateCharacter, DeleteCharacter, Header,
};
use wyd_server::channels::World;
use wyd_server::cpsock;

fn canal(channel: i64, host: &str) -> ServerChannel {
    ServerChannel {
        group_id: 1,
        channel,
        host: host.to_string(),
        status_url: String::new(),
        enabled: true,
    }
}

fn world() -> Arc<World> {
    Arc::new(
        World::new(vec![canal(1, "192.168.100.3"), canal(2, "192.168.100.4")])
            .expect("foto com dois canais"),
    )
}

/// Listener em processo, sem cerca de sessão e **sem** porta de personagem.
fn serve() -> u16 {
    let listener = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
    let port = listener.local_addr().expect("endereço").port();
    let world = world();
    std::thread::spawn(move || {
        cpsock::serve_listener(listener, world, VersionGate::default(), None, None, None)
    });
    port
}

fn frame(key_word: u8, opcode: Opcode, len: usize) -> Vec<u8> {
    let mut frame = vec![0u8; len];
    Header {
        size: u16::try_from(len).expect("frame cabe em u16"),
        key_word,
        checksum: 0,
        message_type: opcode.wire(),
        id: 1,
        client_tick: 7,
    }
    .encode(&mut frame)
    .expect("cabeçalho cabe no frame");
    frame
}

fn login_frame(conta: &str) -> Vec<u8> {
    let mut nome = [0u8; 16];
    let bytes = conta.as_bytes();
    nome[..bytes.len()].copy_from_slice(bytes);
    let mut frame = frame(11, Opcode::AccountLogin, AccountLogin::WIRE_LEN);
    AccountLogin {
        account_password: [0u8; 12],
        account_name: nome,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut frame)
    .expect("corpo do login");
    encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

/// Conecta, faz o login e exige `_MSG_CNFAccountLogin` antes de devolver o socket.
fn conectado(port: u16) -> TcpStream {
    let mut stream = TcpStream::connect(("127.0.0.1", port)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout de leitura");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    stream.write_all(&login_frame("selcao01")).expect("login");
    assert_eq!(expect_reply(&mut stream), Opcode::CNFAccountLogin);
    stream
}

/// Lê um frame inteiro do servidor e devolve o opcode (checksum conferido).
fn expect_reply(stream: &mut TcpStream) -> Opcode {
    let mut head = [0u8; HEADER_SIZE];
    stream.read_exact(&mut head).expect("resposta do servidor");
    let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
    let mut reply = vec![0u8; size];
    reply[..HEADER_SIZE].copy_from_slice(&head);
    if size > HEADER_SIZE {
        stream.read_exact(&mut reply[HEADER_SIZE..]).expect("corpo");
    }
    let computed = decode_in_place(&mut reply).expect("resposta ofuscada");
    assert_eq!(computed, reply[3], "checksum da resposta confere");
    let header = Header::decode(&reply).expect("cabeçalho da resposta");
    assert!(header.size as usize >= HEADER_SIZE, "frame com cabeçalho");
    Opcode::from_wire(header.message_type).expect("opcode conhecido")
}

/// Não deve haver resposta em 300 ms — o pacote foi tolerado, não respondido.
fn expect_silencio(stream: &mut TcpStream) {
    stream
        .set_read_timeout(Some(Duration::from_millis(300)))
        .expect("timeout curto");
    let mut buf = [0u8; HEADER_SIZE];
    match stream.read(&mut buf) {
        Err(error) if matches!(error.kind(), ErrorKind::WouldBlock | ErrorKind::TimedOut) => {}
        Ok(0) => panic!("conexão fechada: o pacote tolerado derrubou a sessão"),
        Ok(n) => panic!("resposta inesperada de {n} byte(s) para pacote tolerado"),
        Err(error) => panic!("erro de leitura inesperado: {error}"),
    }
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout de volta ao normal");
}

#[test]
fn sessao_sobrevive_ao_pacote_tolerado_e_a_selecao_sem_banco() {
    let port = serve();
    let mut stream = conectado(port);

    // Pacote fora do escopo deste processo (guerra) e `_MSG_Ping`: tolerados, sem resposta.
    stream
        .write_all(&frame(9, Opcode::War, HEADER_SIZE))
        .expect("guerra");
    expect_silencio(&mut stream);
    stream
        .write_all(&frame(9, Opcode::Ping, HEADER_SIZE))
        .expect("ping");
    expect_silencio(&mut stream);

    // Seleção com o processo sem banco: recusa tipada, nunca sucesso inventado.
    let mut login_char = frame(9, Opcode::CharacterLogin, CharacterLogin::WIRE_LEN);
    CharacterLogin { slot: 0, force: 1 }
        .encode_into(&mut login_char)
        .expect("corpo do CharacterLogin");
    encode_in_place(&mut login_char, 9).expect("ofuscação");
    stream.write_all(&login_char).expect("seleção");
    assert_eq!(expect_reply(&mut stream), Opcode::CharacterLoginFail);

    // Depois de três pacotes a conexão continua viva: a sessão não morreu no handshake.
    let mut criar = frame(9, Opcode::CreateCharacter, CreateCharacter::WIRE_LEN);
    let mut nome = [0u8; 16];
    nome[..4].copy_from_slice(b"Kael");
    CreateCharacter {
        slot: 1,
        mob_name: nome,
        mob_class: 0,
    }
    .encode_into(&mut criar)
    .expect("corpo do CreateCharacter");
    encode_in_place(&mut criar, 9).expect("ofuscação");
    stream.write_all(&criar).expect("criação");
    assert_eq!(expect_reply(&mut stream), Opcode::NewCharacterFail);

    // E a seleção responde de novo — prova de que o laço está vivo, não em fim de vida.
    stream.write_all(&login_char).expect("seleção de novo");
    assert_eq!(expect_reply(&mut stream), Opcode::CharacterLoginFail);
}

#[test]
fn tamanho_invalido_encerra_a_conexao() {
    let port = serve();
    let mut stream = conectado(port);
    // `Size` acima do teto de 8192: framing inválido é o único motivo para fechar.
    let mut head = [0u8; HEADER_SIZE];
    head[..2].copy_from_slice(&8193u16.to_le_bytes());
    stream.write_all(&head).expect("cabeçalho inválido");

    let mut buf = Vec::new();
    let lido = stream.read_to_end(&mut buf).unwrap_or(0);
    assert_eq!(lido, 0, "servidor fecha sem resposta: {buf:?}");
}

/// Envia um frame de domínio já com o corpo escrito e ofuscado.
fn enviar<F>(stream: &mut TcpStream, chave: u8, opcode: Opcode, len: usize, corpo: F)
where
    F: FnOnce(&mut [u8]),
{
    let mut frame = frame(chave, opcode, len);
    corpo(&mut frame);
    encode_in_place(&mut frame, chave).expect("ofuscação");
    stream.write_all(&frame).expect("pacote");
}

/// Handshake real (INITCODE + senha + PIN) no socket já aberto.
fn autenticar(stream: &mut TcpStream, conta: &str, senha: &str, pin: &str) {
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    enviar(
        stream,
        11,
        Opcode::AccountLogin,
        AccountLogin::WIRE_LEN,
        |f| {
            let mut nome = [0u8; 16];
            let bytes = conta.trim().as_bytes();
            assert!(bytes.len() <= nome.len(), "conta cabe em char[16]");
            nome[..bytes.len()].copy_from_slice(bytes);
            let mut senha_bytes = [0u8; 12];
            let bytes = senha.as_bytes();
            assert!(bytes.len() <= senha_bytes.len(), "senha cabe em char[12]");
            senha_bytes[..bytes.len()].copy_from_slice(bytes);
            AccountLogin {
                account_password: senha_bytes,
                account_name: nome,
                zero: [0u8; 52],
                client_version: 12_000,
                db_need_save: 1,
                adapter_name: [0; 4],
            }
            .encode_into(f)
            .expect("corpo do login");
        },
    );
    assert_eq!(
        expect_reply(stream),
        Opcode::CNFAccountLogin,
        "senha da conta aceita"
    );
    enviar(
        stream,
        11,
        Opcode::AccountSecure,
        AccountSecure::WIRE_LEN,
        |f| {
            let mut token = [0u8; 6];
            let bytes = pin.as_bytes();
            assert!((4..=6).contains(&bytes.len()), "PIN tem 4 a 6 dígitos");
            token[..bytes.len()].copy_from_slice(bytes);
            AccountSecure {
                numeric_token: token,
                unknown: [0u8; 10],
                change_numeric: 0,
            }
            .encode_into(f)
            .expect("corpo do PIN");
        },
    );
    assert_eq!(
        expect_reply(stream),
        Opcode::AccountSecure,
        "PIN aceito ou cadastrado na primeira entrada"
    );
}

/// Lê `wyd_entities_active{canal="<canal>"}` do Prometheus por HTTP (sem dependência nova).
fn ler_gauge_entidades(prometheus: &str, canal: &str) -> Option<f64> {
    let alvo = prometheus
        .trim()
        .trim_start_matches("http://")
        .trim_end_matches('/');
    let mut stream = TcpStream::connect(alvo).ok()?;
    stream.set_read_timeout(Some(Duration::from_secs(5))).ok()?;
    let pedido = format!(
        "GET /api/v1/query?query=wyd_entities_active%7Bcanal%3D%22{canal}%22%7D HTTP/1.1\r\nHost: {alvo}\r\nConnection: close\r\n\r\n"
    );
    stream.write_all(pedido.as_bytes()).ok()?;
    let mut resposta = String::new();
    stream.read_to_string(&mut resposta).ok()?;
    let corpo = resposta.split("\r\n\r\n").nth(1)?;
    let marca = "\"value\":[";
    let inicio = corpo.find(marca)? + marca.len();
    let resto = &corpo[inicio..];
    let abre = resto.find('"')? + 1;
    let fecha = resto[abre..].find('"')? + abre;
    resto[abre..fecha].parse().ok()
}

/// Espera o gauge satisfazer `aceita`, devolvendo o último valor lido.
fn esperar_gauge(
    prometheus: &str,
    canal: &str,
    aceita: impl Fn(f64) -> bool,
    limite: Duration,
) -> Option<f64> {
    let inicio = std::time::Instant::now();
    let mut ultimo = None;
    while inicio.elapsed() < limite {
        if let Some(valor) = ler_gauge_entidades(prometheus, canal) {
            ultimo = Some(valor);
            if aceita(valor) {
                return Some(valor);
            }
        }
        std::thread::sleep(Duration::from_secs(2));
    }
    ultimo
}

/// Prova **viva** (gated) do gauge de entidades, pelo fluxo real do cliente.
///
/// Requer a stack de pé e estes ambientes:
/// `WYD_LIVE_CPSOCK=127.0.0.1:8281`, `WYD_LIVE_ACCOUNT`, `WYD_LIVE_PASSWORD` e, opcionalmente,
/// `WYD_LIVE_PIN` (default `2468`, cadastrado na primeira entrada), `WYD_LIVE_CHARACTER_SLOT`
/// (default 0), `WYD_LIVE_CANAL` (default 1), `WYD_LIVE_PROMETHEUS` (default
/// `127.0.0.1:9090`), `WYD_LIVE_HOLD_SECS` (default 20) e `WYD_LIVE_CREATE_NAME` (cria o
/// personagem pelo `_MSG_CreateCharacter` no slot e o remove no fim, pelo protocolo).
///
/// O personagem é o persistido no slot ou o criado pelo protocolo. A sessão fica aberta
/// (ping a cada 5 s) por pelo menos um scrape de 15 s; o teste exige o gauge > 0 no canal e,
/// depois do FIN, a volta a 0.
#[test]
fn sessao_viva_publica_entidade_no_gauge() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        eprintln!(
            "SKIP: WYD_LIVE_CPSOCK ausente — gauge de sessão viva não provado (requer WYD_LIVE_ACCOUNT/WYD_LIVE_PASSWORD)"
        );
        return;
    };
    let conta = std::env::var("WYD_LIVE_ACCOUNT").expect("WYD_LIVE_ACCOUNT com a conta real");
    let senha = std::env::var("WYD_LIVE_PASSWORD").expect("WYD_LIVE_PASSWORD da conta real");
    let pin = std::env::var("WYD_LIVE_PIN").unwrap_or_else(|_| "2468".to_string());
    let slot: i32 = std::env::var("WYD_LIVE_CHARACTER_SLOT")
        .ok()
        .and_then(|valor| valor.parse().ok())
        .unwrap_or(0);
    let canal = std::env::var("WYD_LIVE_CANAL").unwrap_or_else(|_| "1".to_string());
    let prometheus =
        std::env::var("WYD_LIVE_PROMETHEUS").unwrap_or_else(|_| "127.0.0.1:9090".to_string());
    let espera: u64 = std::env::var("WYD_LIVE_HOLD_SECS")
        .ok()
        .and_then(|valor| valor.parse().ok())
        .unwrap_or(20);
    let criar = std::env::var("WYD_LIVE_CREATE_NAME")
        .ok()
        .filter(|nome| !nome.trim().is_empty());

    let mut stream = TcpStream::connect(&endereco).expect("conecta no canal");
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    autenticar(&mut stream, &conta, &senha, &pin);

    let criado = criar.map(|nome| {
        let bytes = nome.as_bytes();
        assert!(bytes.len() <= 16, "nome do personagem cabe em char[16]");
        let mut campo = [0u8; 16];
        campo[..bytes.len()].copy_from_slice(bytes);
        enviar(
            &mut stream,
            11,
            Opcode::CreateCharacter,
            CreateCharacter::WIRE_LEN,
            |f| {
                CreateCharacter {
                    slot,
                    mob_name: campo,
                    mob_class: 0,
                }
                .encode_into(f)
                .expect("corpo da criação");
            },
        );
        assert_eq!(
            expect_reply(&mut stream),
            Opcode::CNFNewCharacter,
            "{endereco} criou o personagem {nome} no slot {slot}"
        );
        campo
    });

    enviar(
        &mut stream,
        11,
        Opcode::CharacterLogin,
        CharacterLogin::WIRE_LEN,
        |f| {
            CharacterLogin { slot, force: 1 }
                .encode_into(f)
                .expect("corpo da seleção");
        },
    );
    assert_eq!(
        expect_reply(&mut stream),
        Opcode::CNFCharacterLogin,
        "{endereco} encarnou o personagem do slot {slot}"
    );

    let durante = esperar_gauge(
        &prometheus,
        &canal,
        |valor| valor > 0.0,
        Duration::from_secs(45),
    )
    .unwrap_or_else(|| {
        panic!("wyd_entities_active{{canal=\"{canal}\"}} continuou 0 com a sessão aberta")
    });
    eprintln!(
        "PASS: sessão no canal {canal} publicou wyd_entities_active={durante} (personagem slot {slot})"
    );

    let inicio = std::time::Instant::now();
    while inicio.elapsed() < Duration::from_secs(espera) {
        std::thread::sleep(Duration::from_secs(5));
        enviar(&mut stream, 11, Opcode::Ping, HEADER_SIZE, |_| {});
    }
    drop(stream);

    let depois = esperar_gauge(
        &prometheus,
        &canal,
        |valor| valor == 0.0,
        Duration::from_secs(45),
    )
    .unwrap_or_else(|| {
        panic!("wyd_entities_active{{canal=\"{canal}\"}} não voltou a 0 após o logout")
    });
    assert_eq!(depois, 0.0, "gauge zerado depois do FIN");
    eprintln!("PASS: logout zerou wyd_entities_active{{canal=\"{canal}\"}}");

    if let Some(nome) = criado {
        // Remove o personagem pelo mesmo protocolo do cliente; a sessão anterior já saiu.
        std::thread::sleep(Duration::from_secs(1));
        let mut limpeza = TcpStream::connect(&endereco).expect("reconecta para remover");
        limpeza
            .set_read_timeout(Some(Duration::from_secs(10)))
            .expect("timeout da remoção");
        autenticar(&mut limpeza, &conta, &senha, &pin);
        enviar(
            &mut limpeza,
            11,
            Opcode::DeleteCharacter,
            DeleteCharacter::WIRE_LEN,
            |f| {
                DeleteCharacter {
                    slot,
                    mob_name: nome,
                    password: [0u8; 12],
                }
                .encode_into(f)
                .expect("corpo da remoção");
            },
        );
        assert_eq!(
            expect_reply(&mut limpeza),
            Opcode::CNFDeleteCharacter,
            "personagem de prova removido pelo protocolo"
        );
    }
}

#[test]
fn resposta_repete_a_chave_do_frame_de_entrada() {
    let port = serve();
    let listener = TcpStream::connect(("127.0.0.1", port)).expect("conecta");
    let mut stream = listener;
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");

    // Login com outra chave (3): a confirmação tem de voltar ofuscada com a mesma chave, e o
    // `Size` de 12 é o contrato do corpo UNVERIFIED.
    let mut nome = [0u8; 16];
    nome[..8].copy_from_slice(b"selchave");
    let mut login = frame(3, Opcode::AccountLogin, AccountLogin::WIRE_LEN);
    AccountLogin {
        account_password: [0u8; 12],
        account_name: nome,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut login)
    .expect("corpo");
    encode_in_place(&mut login, 3).expect("ofuscação");
    stream.write_all(&login).expect("login");

    let mut head = [0u8; HEADER_SIZE];
    stream.read_exact(&mut head).expect("resposta");
    assert_eq!(head[2], 3, "KeyWord da resposta é o do frame de entrada");
    let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
    let mut reply = vec![0u8; size];
    reply[..HEADER_SIZE].copy_from_slice(&head);
    if size > HEADER_SIZE {
        stream.read_exact(&mut reply[HEADER_SIZE..]).expect("corpo");
    }
    decode_in_place(&mut reply).expect("desofusca com a mesma chave");
    let header = Header::decode(&reply).expect("cabeçalho");
    assert_eq!(
        Opcode::from_wire(header.message_type),
        Some(Opcode::CNFAccountLogin)
    );
}
