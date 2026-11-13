//! Criação e deleção de personagem ponta-a-ponta contra o servidor real: pacote entra,
//! banco muda, pacote de confirmação volta — e deleção com nome divergente não apaga nada.

#[path = "../common/mod.rs"]
mod common;

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::hash_password;
use wyd_db::{Accounts, Characters, PgBackend, PoolConfig, Role, ServerChannel};
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::decode_in_place;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, CreateCharacter, DeleteCharacter, Header};
use wyd_server::auth::Authenticator;
use wyd_server::channels::World;
use wyd_server::characters::SelectionStore;
use wyd_server::cpsock;
use wyd_server::personagens::{Selchar, NEW_CHARACTER_LEN};
use wyd_server::session::SessionGate;

const SENHA: &str = "SenhaForte1";
const PIN: &str = "2468";

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

/// Login único por chamada (relógio + contador do processo: dois testes paralelos leem o
/// mesmo tick de ~100 ns no Windows e colidiriam em `accounts_login_unique`).
fn conta_unica() -> String {
    use std::sync::atomic::{AtomicU32, Ordering};
    static SEQ: AtomicU32 = AtomicU32::new(0);
    let seq = SEQ.fetch_add(1, Ordering::Relaxed) % 1000;
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    format!("cr{:010}{seq:03}", nanos % 10_000_000_000)
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

fn frame_pin() -> Vec<u8> {
    let mut token = [0u8; 6];
    token[..PIN.len()].copy_from_slice(PIN.as_bytes());
    let mut frame = quadro(Opcode::AccountSecure, AccountSecure::WIRE_LEN);
    AccountSecure {
        numeric_token: token,
        unknown: [0u8; 10],
        change_numeric: 0,
    }
    .encode_into(&mut frame)
    .expect("corpo");
    fechar(frame)
}

fn nome16(texto: &str) -> [u8; 16] {
    let mut nome = [0u8; 16];
    nome[..texto.len()].copy_from_slice(texto.as_bytes());
    nome
}

fn frame_create(slot: i32, nome: &str, classe: i32) -> Vec<u8> {
    let mut frame = quadro(Opcode::CreateCharacter, CreateCharacter::WIRE_LEN);
    CreateCharacter {
        slot,
        mob_name: nome16(nome),
        mob_class: classe,
    }
    .encode_into(&mut frame)
    .expect("corpo");
    fechar(frame)
}

fn frame_delete(slot: i32, nome: &str) -> Vec<u8> {
    let mut frame = quadro(Opcode::DeleteCharacter, DeleteCharacter::WIRE_LEN);
    DeleteCharacter {
        slot,
        mob_name: nome16(nome),
        password: [0u8; 12],
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
    abrir_com_corpo(frame).0
}

/// Desofusca o frame e devolve `(opcode, corpo)` — mesma checagem de checksum do `abrir`.
fn abrir_com_corpo(frame: &[u8]) -> (Opcode, Vec<u8>) {
    let mut limpo = frame.to_vec();
    let computado = decode_in_place(&mut limpo).expect("desofusca");
    assert_eq!(computado, limpo[3], "checksum");
    let header = Header::decode(&limpo[..HEADER_SIZE]).expect("header");
    (
        Opcode::from_wire(header.message_type).expect("opcode"),
        limpo[HEADER_SIZE..].to_vec(),
    )
}

fn responde(stream: &mut TcpStream, frame: Vec<u8>) -> Opcode {
    stream.write_all(&frame).expect("envia");
    abrir(&ler_frame(stream))
}

#[test]
fn cria_e_deleta_personagem_pelo_protocolo() {
    let Some(dsn) = common::dsn_ou_skip("personagens CRUD") else {
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
    let conta_id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;

    let mut stream = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("init");
    stream.write_all(&frame_login(&conta)).expect("login");
    let _ = ler_frame(&mut stream);
    assert_eq!(responde(&mut stream, frame_pin()), Opcode::AccountSecure);

    // Criação: confirmação do servidor e linha no banco.
    let nome = format!("H{:05}", conta_id % 100_000);
    stream
        .write_all(&frame_create(1, &nome, 2))
        .expect("envia criação");
    let (opcode, corpo) = abrir_com_corpo(&ler_frame(&mut stream));
    assert_eq!(opcode, Opcode::CNFNewCharacter);
    assert_eq!(corpo.len(), NEW_CHARACTER_LEN, "MSG_CNFNewCharacter");
    let sel = Selchar::decode(&corpo[4..]).expect("sel@4");
    assert_eq!(
        sel.name[1][..nome.len()],
        *nome.as_bytes(),
        "nome no slot 1"
    );
    assert_eq!(sel.equip[1][0].index, 21, "rosto BeastMaster do BaseMob");
    assert_eq!(sel.coin[1], 5_000_000, "ouro do BaseMob");
    // O fio é 0-based (o cliente soma 1); o banco guarda 1.
    assert_eq!(sel.score[1].level, 0, "nível inicial MORTAL no fio");
    assert_eq!((sel.score[1].max_hp, sel.score[1].hp), (71, 71));
    let personagens = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster");
    assert_eq!(personagens.len(), 1, "personagem criado no banco");
    let criado = &personagens[0];
    assert_eq!(criado.name, nome);
    assert_eq!((criado.slot, criado.class), (1, 2));
    // BeastMaster nível 1: BaseSIDCHM {6,6,9,5,70,55} + incrementos {1,2}.
    assert_eq!((criado.hp, criado.hp_max), (71, 71));
    assert_eq!((criado.mp, criado.mp_max), (57, 57));
    assert_eq!(
        (
            criado.strength,
            criado.intelligence,
            criado.dexterity,
            criado.constitution
        ),
        (6, 6, 9, 5)
    );
    // Pontos são ganhos por nível **subido**: recém-criado (nível 1) não tem nenhum.
    assert_eq!((criado.score_bonus, criado.skill_bonus), (0, 0));
    // Ouro e ponto de save vêm do BaseMob da classe (5.000.000 e 2096,2096).
    assert_eq!(criado.gold, 5_000_000, "ouro do template");
    assert_eq!(
        (criado.pos_x, criado.pos_y),
        (2096, 2096),
        "spawn do template"
    );

    // Deleção com nome divergente é recusada e não apaga.
    assert_eq!(
        responde(&mut stream, frame_delete(1, "NomeErrado")),
        Opcode::DeleteCharacterFail
    );
    assert_eq!(
        rt.block_on(backend.list_by_account(conta_id))
            .expect("roster")
            .len(),
        1,
        "recusa não pode apagar"
    );

    // Deleção correta: confirmação com o roster redesenhado e roster vazio (soft delete).
    stream
        .write_all(&frame_delete(1, &nome))
        .expect("envia remoção");
    let (opcode_delete, corpo_delete) = abrir_com_corpo(&ler_frame(&mut stream));
    assert_eq!(opcode_delete, Opcode::CNFDeleteCharacter);
    assert_eq!(
        corpo_delete.len(),
        NEW_CHARACTER_LEN,
        "MSG_CNFDeleteCharacter carrega o STRUCT_SELCHAR"
    );
    let roster = Selchar::decode(&corpo_delete[4..]).expect("sel@4");
    assert!(
        roster.name[1].iter().all(|byte| *byte == 0),
        "slot removido volta zerado para o cliente redesenhar"
    );
    assert!(
        rt.block_on(backend.list_by_account(conta_id))
            .expect("roster")
            .is_empty(),
        "personagem deletado sai do roster"
    );
}

/// Quatro personagens em quatro slots distintos: o cliente pede **sempre** o slot 0 (tela
/// dessincronizada) e o servidor aloca o primeiro livre, sem sobrescrever ninguém. O corpo de
/// `CNFNewCharacter` traz o roster inteiro em cada passo e o quinto pedido é recusado.
#[test]
fn cria_quatro_personagens_em_slots_distintos_sem_sobrescrever() {
    let Some(dsn) = common::dsn_ou_skip("personagens CRUD") else {
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
    let conta_id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;
    let sufixo = format!("{:06}", conta_id % 1_000_000);

    let mut stream = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    stream.write_all(&INITCODE.to_le_bytes()).expect("init");
    stream.write_all(&frame_login(&conta)).expect("login");
    let _ = ler_frame(&mut stream);
    assert_eq!(responde(&mut stream, frame_pin()), Opcode::AccountSecure);

    let classes = [0, 1, 2, 3];
    let iniciais = ['K', 'F', 'B', 'I'];
    let mut nomes: Vec<String> = Vec::new();
    for (indice, classe) in classes.into_iter().enumerate() {
        let nome = format!("{}{}", iniciais[indice], &sufixo[1..]);
        stream
            .write_all(&frame_create(0, &nome, classe))
            .expect("envia criação");
        let (opcode, corpo) = abrir_com_corpo(&ler_frame(&mut stream));
        assert_eq!(opcode, Opcode::CNFNewCharacter, "criação {indice}");
        let sel = Selchar::decode(&corpo[4..]).expect("sel@4");
        assert_eq!(
            sel.name[indice][..nome.len()],
            nome.as_bytes()[..],
            "personagem {indice} no slot {indice} do pacote"
        );
        for (anterior, preservado) in nomes.iter().enumerate() {
            assert_eq!(
                sel.name[anterior][..preservado.len()],
                preservado.as_bytes()[..],
                "slot {anterior} preservado depois da criação {indice}"
            );
        }
        nomes.push(nome);
    }

    let roster = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster");
    assert_eq!(roster.len(), 4, "quatro personagens vivos");
    let slots: Vec<i64> = roster.iter().map(|personagem| personagem.slot).collect();
    assert_eq!(slots, vec![0, 1, 2, 3], "slots 0..3, um por personagem");
    for (indice, nome) in nomes.iter().enumerate() {
        let slot = i64::try_from(indice).expect("slot");
        let personagem = roster
            .iter()
            .find(|personagem| personagem.slot == slot)
            .expect("personagem do slot");
        assert_eq!(&personagem.name, nome, "slot {slot} sem sobrescrita");
        assert_eq!(personagem.class, slot, "classe do slot {slot}");
    }

    // Conta cheia: recusa explícita, nunca sobrescrita.
    assert_eq!(
        responde(
            &mut stream,
            frame_create(0, &format!("Z{}", &sufixo[1..]), 0)
        ),
        Opcode::NewCharacterFail,
        "quinto personagem recusado com os quatro slots ocupados"
    );
    assert_eq!(
        rt.block_on(backend.list_by_account(conta_id))
            .expect("roster")
            .len(),
        4,
        "recusa não cria nem troca ninguém"
    );
}
