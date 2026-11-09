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
use wyd_protocol::{AccountLogin, CharacterLogin, CreateCharacter, DeleteCharacter, Header};
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

/// Nome de personagem único e ASCII (o banco tem UNIQUE parcial por nome vivo).
fn nome_unico() -> [u8; 16] {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.subsec_nanos())
        .unwrap_or(0);
    let texto = format!("QA{nanos:08}");
    let mut nome = [0u8; 16];
    nome[..texto.len()].copy_from_slice(texto.as_bytes());
    nome
}

/// Prova **viva** (gated): `WYD_LIVE_CPSOCK=127.0.0.1:8281 WYD_LIVE_ACCOUNT=<conta real>`.
///
/// Exige um canal com a stack de pé e a conta com linha em `accounts`. O personagem é criado,
/// selecionado e removido no mesmo teste — o banco é o de desenvolvimento, mas nada fica
/// pendurado: o `DeleteCharacter` faz o soft-delete que o cliente faria.
#[test]
fn selecao_contra_servidor_vivo() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        return;
    };
    let Ok(conta) = std::env::var("WYD_LIVE_ACCOUNT") else {
        return;
    };
    let mut nome = [0u8; 16];
    let bytes = conta.trim().as_bytes();
    assert!(bytes.len() <= nome.len(), "conta cabe em char[16]");
    nome[..bytes.len()].copy_from_slice(bytes);

    let mut stream = TcpStream::connect(&endereco).expect("conecta no canal");
    stream
        .set_read_timeout(Some(Duration::from_secs(5)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("INITCODE");
    enviar(
        &mut stream,
        11,
        Opcode::AccountLogin,
        AccountLogin::WIRE_LEN,
        |f| {
            AccountLogin {
                account_password: [0u8; 12],
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
    assert_eq!(expect_reply(&mut stream), Opcode::CNFAccountLogin);

    let personagem = nome_unico();
    enviar(
        &mut stream,
        11,
        Opcode::CreateCharacter,
        CreateCharacter::WIRE_LEN,
        |f| {
            CreateCharacter {
                slot: 3,
                mob_name: personagem,
                mob_class: 0,
            }
            .encode_into(f)
            .expect("corpo da criação");
        },
    );
    assert_eq!(
        expect_reply(&mut stream),
        Opcode::CNFNewCharacter,
        "{endereco} criou o personagem da conta {conta}"
    );

    enviar(
        &mut stream,
        11,
        Opcode::CharacterLogin,
        CharacterLogin::WIRE_LEN,
        |f| {
            CharacterLogin { slot: 3, force: 1 }
                .encode_into(f)
                .expect("corpo da seleção");
        },
    );
    assert_eq!(
        expect_reply(&mut stream),
        Opcode::CNFCharacterLogin,
        "{endereco} selecionou o personagem"
    );

    enviar(
        &mut stream,
        11,
        Opcode::DeleteCharacter,
        DeleteCharacter::WIRE_LEN,
        |f| {
            DeleteCharacter {
                slot: 3,
                mob_name: personagem,
                password: [0u8; 12],
            }
            .encode_into(f)
            .expect("corpo da remoção");
        },
    );
    assert_eq!(
        expect_reply(&mut stream),
        Opcode::CNFDeleteCharacter,
        "{endereco} removeu o personagem de teste"
    );
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
