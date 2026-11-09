//! `STRUCT_SELCHAR` e `STRUCT_MOB` saindo do banco: login real, frame decodificado com a
//! tabela de keyword/opcode do crate e comparação campo a campo com o estado persistido.
//!
//! O teste também grava dois artefatos de captura (`tmp/captura/selchar-a.json` e
//! `...-b.json`) — o estado A e o estado B depois de variar **um** campo (level) — para o
//! `tools/client/diff-captura.mjs` produzir o artefato de diff.

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::hash_password;
use wyd_db::{Accounts, Character, Characters, PgBackend, PoolConfig, Role, ServerChannel};
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::decode_in_place;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, CharacterLogin, Header};
use wyd_server::auth::Authenticator;
use wyd_server::channels::World;
use wyd_server::characters::SelectionStore;
use wyd_server::cpsock;
use wyd_server::personagens::{decode_character_body, StructMob};
use wyd_server::session::SessionGate;

const SENHA: &str = "SenhaForte1";
const PIN: &str = "2468";

fn dsn() -> String {
    ["WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty())
        .expect("WYD_TEST_DATABASE_URL/DATABASE_URL para o teste de selchar")
}

fn runtime() -> Arc<tokio::runtime::Runtime> {
    Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(1)
            .enable_all()
            .build()
            .expect("runtime"),
    )
}

fn servidor(
    backend: Arc<PgBackend>,
    rt: Arc<tokio::runtime::Runtime>,
) -> (u16, Arc<SelectionStore>) {
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
    let para_servidor = Arc::clone(&selection);
    std::thread::spawn(move || {
        cpsock::serve_listener(
            listener,
            world,
            VersionGate::default(),
            Some(auth),
            Some(cerca),
            Some(para_servidor),
        )
    });
    (porta, selection)
}

fn conta_unica() -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    format!("sc{:013}", nanos % 10_000_000_000_000)
}

fn frame_login(conta: &str) -> Vec<u8> {
    let mut nome = [0u8; 16];
    nome[..conta.len()].copy_from_slice(conta.as_bytes());
    let mut senha = [0u8; 12];
    senha[..SENHA.len()].copy_from_slice(SENHA.as_bytes());
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
        account_password: senha,
        account_name: nome,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut frame)
    .expect("corpo");
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

fn frame_pin(pin: &str) -> Vec<u8> {
    let mut token = [0u8; 6];
    token[..pin.len()].copy_from_slice(pin.as_bytes());
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
    .expect("corpo");
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

fn frame_selecao(slot: i32) -> Vec<u8> {
    let mut frame = vec![0u8; CharacterLogin::WIRE_LEN];
    Header {
        size: CharacterLogin::WIRE_LEN as u16,
        key_word: 11,
        checksum: 0,
        message_type: Opcode::CharacterLogin.wire(),
        id: 1,
        client_tick: 3,
    }
    .encode(&mut frame)
    .expect("header");
    CharacterLogin { slot, force: 1 }
        .encode_into(&mut frame)
        .expect("corpo");
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

/// Lê um frame inteiro: `Size` no offset 0 (claro), corpo ofuscado.
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

/// Desofusca o frame inteiro e confere o checksum (mesma tabela do crate).
fn abrir(frame: &[u8]) -> (Opcode, Vec<u8>) {
    let mut limpo = frame.to_vec();
    let computado = decode_in_place(&mut limpo).expect("desofusca");
    assert_eq!(computado, limpo[3], "checksum do frame");
    let header = Header::decode(&limpo[..HEADER_SIZE]).expect("header");
    (
        Opcode::from_wire(header.message_type).expect("opcode"),
        limpo[HEADER_SIZE..].to_vec(),
    )
}

fn gravar_captura(nome: &str, frame: &[u8]) {
    let dir = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../tmp/captura");
    std::fs::create_dir_all(&dir).expect("tmp/captura");
    let documento = serde_json_lite(frame);
    std::fs::write(dir.join(nome), documento).expect("artefato");
}

/// JSON mínimo da captura (`frames[]` com direção e hex), sem dependência nova.
fn serde_json_lite(frame: &[u8]) -> String {
    let hex: String = frame.iter().map(|b| format!("{b:02x}")).collect();
    format!("{{\"frames\":[{{\"direcao\":\"s2c\",\"hex\":\"{hex}\"}}]}}\n")
}

#[test]
fn selchar_e_mob_saem_do_banco_e_voltam_pelo_decoder() {
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn(), PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("K{:06}", nanos % 1_000_000);
    let mut nome_esperado = [0u8; 16];
    nome_esperado[..nome.len()].copy_from_slice(nome.as_bytes());
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    let conta_id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;
    let personagem = Character {
        id: 0,
        account_id: conta_id,
        slot: 0,
        name: nome.clone(),
        class: 1,
        level: 41,
        exp: 123_456,
        hp: 900,
        mp: 300,
        pos_x: 120,
        pos_y: 88,
        gold: 1_500,
        hp_max: 1_100,
        mp_max: 320,
        pvp_points: 7,
        ..Character::default()
    };
    rt.block_on(backend.create_character(&personagem))
        .expect("personagem");
    let direto = selection.roster(&conta).expect("roster direto");
    assert_eq!(direto.len(), 1, "roster vê o personagem antes do handshake");

    // Estado A: login + PIN (primeira entrada) + leitura do CNFAccountLogin.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    let cnf = ler_frame(&mut s);
    let (opcode, corpo) = abrir(&cnf);
    assert_eq!(opcode, Opcode::CNFAccountLogin);
    gravar_captura("selchar-a.json", &cnf);

    let (count, registros) = decode_character_body(&corpo).expect("selchar");
    assert_eq!(count, 1);
    let lido = &registros[0];
    let banco = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster")
        .remove(0);
    // Campo a campo: o pacote tem de ser o estado persistido.
    assert_eq!(lido.name, nome_esperado);
    assert_eq!(lido.char_id, banco.id);
    assert_eq!(lido.account_id, banco.account_id);
    assert_eq!(u8::try_from(banco.slot).unwrap(), lido.slot);
    assert_eq!(u8::try_from(banco.class).unwrap(), lido.class, "classe");
    assert_eq!(lido.citizen, 2, "citizen = canal + 1");
    assert_eq!(u16::try_from(banco.level).unwrap(), lido.level, "level");
    assert_eq!(u32::try_from(banco.exp).unwrap(), lido.exp, "exp");
    assert_eq!(u32::try_from(banco.hp).unwrap(), lido.hp, "hp");
    assert_eq!(u32::try_from(banco.mp).unwrap(), lido.mp, "mp");
    assert_eq!(u32::try_from(banco.hp_max).unwrap(), lido.hp_max, "hp_max");
    assert_eq!(u32::try_from(banco.mp_max).unwrap(), lido.mp_max, "mp_max");
    assert_eq!(u32::try_from(banco.gold).unwrap(), lido.gold, "gold");
    assert_eq!(
        u32::try_from(banco.pvp_points).unwrap(),
        lido.pvp,
        "pvp_points"
    );
    assert_eq!(lido.guild_id, 0, "sem guilda: 0, não lixo");
    assert_eq!(u16::try_from(banco.pos_x).unwrap(), lido.pos_x, "pos_x");
    assert_eq!(u16::try_from(banco.pos_y).unwrap(), lido.pos_y, "pos_y");

    // PIN (cadastro) e seleção: o CNFCharacterLogin carrega o STRUCT_MOB.
    s.write_all(&frame_pin(PIN)).expect("pin");
    let (opcode_pin, _) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_pin, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção");
    let (opcode_mob, corpo_mob) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_mob, Opcode::CNFCharacterLogin);
    assert_eq!(corpo_mob.len(), wyd_server::personagens::MOB_LEN);
    let mob = StructMob::decode(&corpo_mob).expect("mob");
    assert_eq!(mob.name, nome_esperado);
    assert_eq!((mob.char_id, mob.account_id), (banco.id, banco.account_id));
    assert_eq!((mob.slot, mob.class, mob.citizen), (0, 1, 2));
    assert_eq!((mob.level, mob.exp), (41, 123_456));
    assert_eq!((mob.hp, mob.mp), (900, 300));
    assert_eq!((mob.pos_x, mob.pos_y), (120, 88));
    assert_eq!(mob.guild_id, 0, "sem guilda no banco ainda");
    drop(s);

    // Estado B: varia UM campo (level 41 → 42) e captura o mesmo frame.
    rt.block_on(backend.soft_delete(banco.id))
        .expect("remove A");
    let mut atualizado = personagem.clone();
    atualizado.level = 42;
    rt.block_on(backend.create_character(&atualizado))
        .expect("personagem B");
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta B");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login B");
    let cnf_b = ler_frame(&mut s);
    let (opcode_b, corpo_b) = abrir(&cnf_b);
    assert_eq!(opcode_b, Opcode::CNFAccountLogin);
    gravar_captura("selchar-b.json", &cnf_b);
    let (_, registros_b) = decode_character_body(&corpo_b).expect("selchar B");
    assert_eq!(registros_b[0].level, 42, "B reflete a variação no banco");
}
