//! `STRUCT_SELCHAR` e `STRUCT_MOB` saindo do banco: login real, frame decodificado com a
//! tabela de keyword/opcode do crate e comparação campo a campo com o estado persistido.
//!
//! O teste também grava dois artefatos de captura (`tmp/captura/selchar-a.json` e
//! `...-b.json`) — o estado A e o estado B depois de variar **um** campo (level) — para o
//! `tools/client/diff-captura.mjs` produzir o artefato de diff.

#[path = "../common/mod.rs"]
mod common;

use std::io::{Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::hash_password;
use wyd_db::{
    Accounts, Character, Characters, ItemRow, PgBackend, PoolConfig, Role, ServerChannel, Sessions,
    SHORT_SKILL_EMPTY, SHORT_SKILL_LEN,
};
use wyd_net::VersionGate;
use wyd_protocol::codec::keyword::decode_in_place;
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{
    AccountLogin, AccountSecure, ApplyBonus, CharacterLogin, CreateCharacter, Header,
};
use wyd_server::auth::Authenticator;
use wyd_server::channels::World;
use wyd_server::characters::SelectionStore;
use wyd_server::cpsock;
use wyd_server::personagens::{
    update_etc_body, update_score_body, Mob, Selchar, ACCOUNT_LOGIN_LEN, CHARACTER_LOGIN_LEN,
    SELCHAR_LEN, UPDATE_ETC_LEN, UPDATE_SCORE_LEN,
};
use wyd_server::session::{SessionError, SessionGate};

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

/// Login único por chamada: o relógio sozinho colide quando dois testes paralelos leem o
/// mesmo tick (~100 ns no Windows), então um contador do processo entra nos dígitos finais.
fn conta_unica() -> String {
    use std::sync::atomic::{AtomicU32, Ordering};
    static SEQ: AtomicU32 = AtomicU32::new(0);
    let seq = SEQ.fetch_add(1, Ordering::Relaxed) % 1000;
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    format!("sc{:010}{seq:03}", nanos % 10_000_000_000)
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
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
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

    assert_eq!(corpo.len(), ACCOUNT_LOGIN_LEN, "MSG_DBCNFAccountLogin");
    let sel = Selchar::decode(&corpo[20..20 + SELCHAR_LEN]).expect("sel@20");
    let banco = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster")
        .remove(0);
    // Campo a campo: o pacote tem de ser o estado persistido.
    let slot = usize::try_from(banco.slot).expect("slot");
    assert_eq!(sel.name[slot], nome_esperado);
    // `Level` do `STRUCT_SCORE` é o do banco menos um (o cliente soma 1 ao desenhar).
    assert_eq!(
        sel.score[slot].level,
        i32::try_from(banco.level - 1).expect("level")
    );
    assert_eq!(sel.exp[slot], banco.exp, "Exp@808");
    assert_eq!(
        sel.coin[slot],
        i32::try_from(banco.gold).expect("gold"),
        "Coin@792"
    );
    assert_eq!(
        sel.score[slot].hp,
        i32::try_from(banco.hp).expect("hp"),
        "Score.Hp"
    );
    assert_eq!(
        sel.score[slot].mp,
        i32::try_from(banco.mp).expect("mp"),
        "Score.Mp"
    );
    assert_eq!(
        sel.score[slot].max_hp,
        i32::try_from(banco.hp_max).expect("hp_max"),
        "Score.MaxHp"
    );
    assert_eq!(
        sel.score[slot].max_mp,
        i32::try_from(banco.mp_max).expect("mp_max"),
        "Score.MaxMp"
    );
    assert_eq!(sel.guild[slot], 0, "sem guilda: 0, não lixo");
    assert_eq!(
        sel.spx[slot],
        i16::try_from(banco.pos_x).expect("pos"),
        "SPX"
    );
    assert_eq!(
        sel.spy[slot],
        i16::try_from(banco.pos_y).expect("pos"),
        "SPY"
    );
    // Aparência: o cliente desenha a classe pelo rosto em Equip[slot][0] (DBGetSelChar).
    assert_eq!(sel.equip[slot][0].index, 11, "rosto Foema do BaseMob");
    assert!(
        sel.name[1 - slot].iter().all(|byte| *byte == 0),
        "slot vazio zerado"
    );

    // PIN (cadastro) e seleção: o CNFCharacterLogin carrega o STRUCT_MOB.
    s.write_all(&frame_pin(PIN)).expect("pin");
    let (opcode_pin, _) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_pin, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção");
    let (opcode_mob, corpo_mob) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_mob, Opcode::CNFCharacterLogin);
    assert_eq!(
        corpo_mob.len(),
        CHARACTER_LOGIN_LEN,
        "MSG_CNFClientCharacterLogin"
    );
    let mob = Mob::decode(&corpo_mob[4..]).expect("mob@4");
    assert_eq!(mob.name, nome_esperado);
    assert_eq!((mob.class, mob.guild), (1, 0));
    assert_eq!(
        mob.current_score.level, 40,
        "CurrentScore.Level (banco 41 → fio 40)"
    );
    assert_eq!(mob.exp, 123_456, "Exp@32");
    assert_eq!(
        (mob.current_score.hp, mob.current_score.mp),
        (900, 300),
        "Hp/Mp correntes"
    );
    assert_eq!(mob.current_score.max_hp, 1_100, "MaxHp");
    assert_eq!((mob.spx, mob.spy), (120, 88), "ponto de save SPX/SPY");
    assert_eq!(mob.equip[0].index, 11, "rosto Foema do BaseMob");
    assert_eq!(
        u16::from_le_bytes([corpo_mob[1028], corpo_mob[1029]]),
        0,
        "Slot@1028"
    );
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
    let sel_b = Selchar::decode(&corpo_b[20..20 + SELCHAR_LEN]).expect("selchar B");
    assert_eq!(sel_b.score[0].level, 41, "B reflete a variação no banco");
}

/// Frame só de cabeçalho (logout, ping): `Size = 12`, ofuscado a partir do offset 4.
fn frame_header(opcode: Opcode) -> Vec<u8> {
    let mut frame = vec![0u8; HEADER_SIZE];
    Header {
        size: HEADER_SIZE as u16,
        key_word: 11,
        checksum: 0,
        message_type: opcode.wire(),
        id: 1,
        client_tick: 6,
    }
    .encode(&mut frame)
    .expect("header");
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

/// `_MSG_CharacterLogout` (0x0215): o servidor responde `CNFCharacterLogout`, fecha a conexão e
/// libera a sessão **na hora** — fechar a janela do cliente não pode esperar o timeout.
#[test]
fn logout_explicito_fecha_a_conexao_e_libera_a_sessao() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("Q{:06}", nanos % 1_000_000);
    selection
        .create(&conta, 0, nome.as_bytes(), 0)
        .expect("criação");

    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFCharacterLogin);
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateScore,
        "score de entrada antes do logout"
    );
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateEtc,
        "etc de entrada antes do logout"
    );

    s.write_all(&frame_header(Opcode::CharacterLogout))
        .expect("logout");
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::CNFCharacterLogout,
        "o cliente recebe a confirmação de saída"
    );
    // A conexão fecha: ou EOF limpo, ou reset — nunca fica pendurada até o timeout.
    let mut resto = [0u8; 1];
    assert!(
        !matches!(s.read(&mut resto), Ok(n) if n > 0),
        "servidor encerra a conexão depois do logout"
    );

    // A sessão da conta cai na hora (o timeout de ociosidade é 120 s); outras contas do
    // banco de teste não contam.
    let saiu = common::esperar_ate(Duration::from_secs(5), || {
        rt.block_on(backend.live_session_of(&conta))
            .expect("sessão da conta")
            .is_none()
    });
    assert!(saiu, "sessão continuou viva depois do logout explícito");
}

/// `/getout` chega como sussurro (`_MSG_MessageWhisper` 0x0334) e o corpo dele **não** tem
/// layout provado: o servidor tolera, loga `action=getout` e mantém a sessão. Tratar o pacote
/// como comando seria inventar layout (invariante 5); derrubar a conexão seria pior.
#[test]
fn sussurro_de_getout_e_tolerado_e_nao_derruba_a_sessao() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("G{:06}", nanos % 1_000_000);
    selection
        .create(&conta, 0, nome.as_bytes(), 0)
        .expect("criação");

    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFCharacterLogin);
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateScore,
        "score de entrada antes do sussurro"
    );
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateEtc,
        "etc de entrada antes do sussurro"
    );

    // Sussurro (`/getout` do legado): só cabeçalho, corpo não provado — o handler não lê o
    // corpo, então o frame mínimo é o teste honesto da tolerância.
    s.write_all(&frame_header(Opcode::MessageWhisper))
        .expect("sussurro");

    // A sessão continua viva: o próximo pacote de controle é atendido normalmente.
    s.write_all(&frame_header(Opcode::CharacterLogout))
        .expect("logout");
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::CNFCharacterLogout,
        "sussurro tolerado não encerrou a conexão"
    );
    let viva = rt
        .block_on(backend.live_session_of(&conta))
        .expect("sessão");
    assert_eq!(viva, None, "o logout seguinte liberou a sessão");
}

/// Reconexão no **mesmo** canal retoma a sessão (o cliente reconectou); outro canal continua
/// recusado — a cerca de 0030 segue sendo a autoridade entre canais.
#[test]
fn reconexao_no_mesmo_canal_retoma_a_sessao() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let conta = conta_unica();
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let canal1 = SessionGate::new(
        Arc::clone(&backend),
        Arc::clone(&rt),
        1,
        1,
        Duration::from_secs(120),
        Duration::from_secs(180),
    );
    let primeira = canal1.claim(&conta, 12_000).expect("primeira sessão");
    // Sem liberar a primeira: a reconexão do mesmo processo toma o lugar da antiga.
    let segunda = canal1
        .claim(&conta, 12_000)
        .expect("reconexão no mesmo canal");
    assert_ne!(primeira.id(), segunda.id(), "linha nova, não a mesma");
    let canal2 = SessionGate::new(
        Arc::clone(&backend),
        Arc::clone(&rt),
        1,
        2,
        Duration::from_secs(120),
        Duration::from_secs(180),
    );
    assert!(
        matches!(
            canal2.claim(&conta, 12_000),
            Err(SessionError::Conflict { channel: 1 })
        ),
        "outro canal continua recusado enquanto a sessão do canal 1 está viva"
    );
}

/// Ganho de nível de ponta a ponta: EXP da curva do legado, +5 atributo / +3 skill /
/// +2 maestria por nível, incrementos de HP/MP da classe, vitais cheios, gravação transacional
/// e os bodies de `MSG_UpdateEtc`/`MSG_UpdateScore` que o cliente recebe.
#[test]
fn ganho_de_nivel_persiste_e_devolve_os_pacotes() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let selection = Arc::new(SelectionStore::new(Arc::clone(&backend), Arc::clone(&rt)));
    let conta = conta_unica();
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("L{:06}", nanos % 1_000_000);
    let criado = selection
        .create(&conta, 0, nome.as_bytes(), 0)
        .expect("criação");
    assert_eq!((criado.level, criado.exp), (1, 0), "nível inicial MORTAL");
    let inicial = selection.select_snapshot(&conta, 0).expect("seleção");
    assert_eq!(
        (inicial.character.hp_max, inicial.character.mp_max),
        (83, 46),
        "TransKnight no nível 1"
    );

    // 500 = g_pNextLevel[1]: o limiar do nível 2.
    let subiu = selection.gain_exp(&inicial, 500).expect("EXP do nível 2");
    assert_eq!(subiu.levels, 1, "um nível ganho");
    assert_eq!(
        (subiu.progression.level, subiu.progression.exp),
        (2, 500),
        "nível e EXP acumulada"
    );
    assert_eq!(
        (
            subiu.progression.score_bonus,
            subiu.progression.skill_bonus,
            subiu.progression.special_bonus
        ),
        (5, 3, 2),
        "+5 atributo, +3 skill, +2 maestria sobre os 0 do nível 1"
    );
    assert_eq!(
        (subiu.progression.hp_max, subiu.progression.mp_max),
        (86, 47),
        "g_pIncrementHp/Mp do TransKnight (+3/+1)"
    );
    assert_eq!(
        (subiu.progression.hp, subiu.progression.mp),
        (86, 47),
        "vitais cheios no level-up"
    );
    // Fio: `Level` é o do banco menos um; os dois pacotes levam o estado novo.
    assert_eq!(subiu.mob.current_score.level, 1, "fio 0-based");
    let etc = update_etc_body(&subiu.mob);
    assert_eq!(
        i64::from_le_bytes(etc[4..12].try_into().expect("exp")),
        500,
        "MSG_UpdateEtc.Exp@4"
    );
    assert_eq!(u16::from_le_bytes([etc[20], etc[21]]), 5, "ScoreBonus@20");
    assert_eq!(u16::from_le_bytes([etc[22], etc[23]]), 2, "SpecialBonus@22");
    assert_eq!(u16::from_le_bytes([etc[24], etc[25]]), 3, "SkillBonus@24");
    let score = update_score_body(&subiu.mob);
    assert_eq!(
        i32::from_le_bytes(score[0..4].try_into().expect("level")),
        1,
        "MSG_UpdateScore.Score.Level@0"
    );
    assert_eq!(
        i32::from_le_bytes(score[16..20].try_into().expect("maxhp")),
        86,
        "Score.MaxHp@16"
    );

    // O banco é a fonte: a progressão relida é a mesma que o pacote levou.
    let salvo = rt
        .block_on(backend.progression(criado.id))
        .expect("progressão");
    assert_eq!((salvo.level, salvo.exp), (2, 500));
    assert_eq!(
        (salvo.score_bonus, salvo.skill_bonus, salvo.special_bonus),
        (5, 3, 2)
    );

    // Vários níveis de uma vez: 624 fecha o limiar do 3 (1124) e 3 000 além levam ao 6
    // (limiar 3480, excedente 644 retido em vez de descartado).
    let relido = selection.select_snapshot(&conta, 0).expect("relê");
    assert_eq!(relido.character.level, 2, "relogin lê o nível gravado");
    let salto = selection
        .gain_exp(&relido, 624 + 3_000)
        .expect("EXP de vários níveis");
    assert_eq!(salto.levels, 4, "nível 2 → 6");
    assert_eq!(
        (salto.progression.level, salto.progression.exp),
        (6, 4_124),
        "nível 6 com 644 de excedente (3 480 + 644)"
    );
    assert_eq!(
        (
            salto.progression.hp_max,
            salto.progression.mp_max,
            salto.progression.score_bonus,
            salto.progression.skill_bonus,
            salto.progression.special_bonus
        ),
        (98, 51, 25, 15, 10),
        "quatro níveis acumulados"
    );
    assert_eq!(
        salto.mob.current_score.level, 5,
        "nível 6 no banco, 5 no fio"
    );
    assert_eq!(salto.mob.exp, 4_124, "Exp@32 do STRUCT_MOB");

    // EXP não positiva é recusa explícita, não nível inventado.
    assert!(
        selection.gain_exp(&relido, 0).is_err(),
        "EXP zero não pode subir nível"
    );
}

/// Frame de `_MSG_CreateCharacter` (slot + nome 16 B + classe).
fn frame_criacao(slot: i32, nome: &str, classe: i32) -> Vec<u8> {
    let mut campo = [0u8; 16];
    campo[..nome.len()].copy_from_slice(nome.as_bytes());
    let mut frame = vec![0u8; CreateCharacter::WIRE_LEN];
    Header {
        size: CreateCharacter::WIRE_LEN as u16,
        key_word: 11,
        checksum: 0,
        message_type: Opcode::CreateCharacter.wire(),
        id: 1,
        client_tick: 4,
    }
    .encode(&mut frame)
    .expect("header");
    CreateCharacter {
        slot,
        mob_name: campo,
        mob_class: classe,
    }
    .encode_into(&mut frame)
    .expect("corpo");
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

/// Frame de `_MSG_ApplyBonus` (tipo/detalhe 2×i16 + alvo u16).
fn frame_bonus(tipo: i16, detalhe: i16, alvo: u16) -> Vec<u8> {
    let mut frame = vec![0u8; ApplyBonus::WIRE_LEN];
    Header {
        size: ApplyBonus::WIRE_LEN as u16,
        key_word: 11,
        checksum: 0,
        message_type: Opcode::ApplyBonus.wire(),
        id: 1,
        client_tick: 5,
    }
    .encode(&mut frame)
    .expect("header");
    ApplyBonus {
        bonus_type: tipo,
        detail: detalhe,
        target_id: alvo,
    }
    .encode_into(&mut frame)
    .expect("corpo");
    wyd_protocol::codec::keyword::encode_in_place(&mut frame, 11).expect("ofuscação");
    frame
}

/// Cofre e `Coin` da conta vêm de `vault_items`/`account_wallet` no `CNFAccountLogin`.
#[test]
fn cofre_e_moeda_da_conta_chegam_no_login() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, _selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
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
    // Cofre e moeda reais no banco — é o que o login tem de levar, sem personagem no meio.
    rt.block_on(backend.store_vault_items(
        conta_id,
        &[
            ItemRow {
                slot: 3,
                item_id: 1234,
                qty: 1,
                ..ItemRow::default()
            },
            ItemRow {
                slot: 127,
                item_id: 547,
                qty: 1,
                ..ItemRow::default()
            },
        ],
    ))
    .expect("cofre");
    rt.block_on(backend.set_account_coin(conta_id, 7777))
        .expect("moeda");

    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    let (opcode, corpo) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode, Opcode::CNFAccountLogin);
    assert_eq!(corpo.len(), ACCOUNT_LOGIN_LEN, "MSG_DBCNFAccountLogin");

    let cargo = |slot: usize| 860 + slot * 8;
    let item = |slot: usize| i16::from_le_bytes([corpo[cargo(slot)], corpo[cargo(slot) + 1]]);
    assert_eq!(item(0), 0, "Cargo[0] vazio: nenhum item sintético");
    assert_eq!(item(3), 1234, "Cargo[3]@{} do banco", cargo(3));
    assert_eq!(item(127), 547, "Cargo[127]@{} do banco", cargo(127));
    assert_eq!(
        i32::from_le_bytes(corpo[1884..1888].try_into().expect("coin")),
        7777,
        "Coin@1884 de account_wallet"
    );
}

/// `_MSG_ApplyBonus`: ponto de `Con` e skill aprendida chegam no `UpdateEtc`/`UpdateScore`,
/// ficam gravados no banco e voltam no `STRUCT_MOB` do relogin.
#[test]
fn bonus_de_atributo_e_skill_chegam_ao_cliente_e_ao_banco() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, _selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("B{:06}", nanos % 1_000_000);
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    let conta_id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;

    // 1) Login + PIN + criação + seleção pelo protocolo real. Recém-criado não tem
    //    pontos; o teste injeta 5/3 como fixture para exercitar o débito do pacote.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_criacao(0, &nome, 0)).expect("create");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFNewCharacter);
    let personagem = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster")
        .remove(0);
    let mut progressao = rt
        .block_on(backend.progression(personagem.id))
        .expect("progressão");
    progressao.score_bonus = 5;
    progressao.skill_bonus = 3;
    rt.block_on(backend.store_progression(personagem.id, &progressao))
        .expect("pontos de fixture");
    s.write_all(&frame_selecao(0)).expect("seleção");
    let (opcode_mob, corpo_mob) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_mob, Opcode::CNFCharacterLogin);
    let mob = Mob::decode(&corpo_mob[4..]).expect("mob@4");
    assert_eq!(
        (
            mob.score_bonus,
            mob.skill_bonus,
            mob.current_score.constitution
        ),
        (5, 3, 6),
        "TransKnight com os pontos de fixture"
    );
    assert_eq!(mob.current_score.max_hp, 83, "MaxHp inicial 80 + 3");
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateScore,
        "score de entrada"
    );
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateEtc,
        "etc de entrada"
    );

    // 2) Um ponto de `Con`: 5 → 4 pontos, Con 6 → 7, MaxHp +2 (legado soma 2 por ponto).
    s.write_all(&frame_bonus(0, 3, 0)).expect("apply con");
    let (opcode_etc, etc) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_etc, Opcode::UpdateEtc);
    assert_eq!(etc.len(), UPDATE_ETC_LEN, "MSG_UpdateEtc − HEADER");
    assert_eq!(
        u16::from_le_bytes([etc[20], etc[21]]),
        4,
        "ScoreBonus@20 decrementado"
    );
    let (opcode_score, score) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_score, Opcode::UpdateScore);
    assert_eq!(score.len(), UPDATE_SCORE_LEN, "MSG_UpdateScore − HEADER");
    assert_eq!(
        i16::from_le_bytes([score[38], score[39]]),
        7,
        "Score.Con@38"
    );
    assert_eq!(
        i32::from_le_bytes(score[16..20].try_into().expect("maxhp")),
        85,
        "Score.MaxHp@16 recalculado"
    );

    let personagem = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster")
        .remove(0);
    let progressao = rt
        .block_on(backend.progression(personagem.id))
        .expect("progressão");
    assert_eq!(
        (
            progressao.constitution,
            progressao.hp_max,
            progressao.score_bonus
        ),
        (7, 85, 4),
        "progressão gravada no banco"
    );

    // 3) Skill do catálogo: pontos concedidos no banco (fixture) e aprendida pelo pacote.
    let mut com_pontos = progressao;
    com_pontos.skill_bonus = 100;
    rt.block_on(backend.store_progression(personagem.id, &com_pontos))
        .expect("pontos");
    s.write_all(&frame_bonus(2, 5000, 1200))
        .expect("apply skill");
    let (opcode_etc, etc) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_etc, Opcode::UpdateEtc);
    assert_eq!(
        i64::from_le_bytes(etc[12..20].try_into().expect("learn")),
        1,
        "Learn@12 com a skill 0 do TransKnight"
    );
    assert_eq!(
        u16::from_le_bytes([etc[24], etc[25]]),
        76,
        "SkillBonus@24 = 100 − 24"
    );
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::UpdateScore);
    let progressao = rt
        .block_on(backend.progression(personagem.id))
        .expect("progressão");
    assert_eq!(
        (progressao.learned_skill, progressao.skill_bonus),
        (1, 76),
        "skill e pontos no banco"
    );
    drop(s);

    // 4) Relogin: `STRUCT_MOB` traz atributo, `LearnedSkill` e os pontos do banco.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta B");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login B");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin B");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção B");
    let (opcode_mob, corpo_mob) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_mob, Opcode::CNFCharacterLogin);
    let mob = Mob::decode(&corpo_mob[4..]).expect("mob@4");
    assert_eq!(mob.learned_skill, 1, "LearnedSkill@780");
    assert_eq!(mob.base_score.constitution, 7, "BaseScore.Con@126");
    assert_eq!(mob.current_score.max_hp, 85, "CurrentScore.MaxHp@108");
    assert_eq!((mob.score_bonus, mob.skill_bonus), (4, 76));
}

/// `_MSG_ApplyBonus` tipo 1 (maestria): ponto debitado, `Special` gravado em
/// `character_special`, relogin com o mesmo `Special[4]` e pedido no teto recusado sem gravar.
#[test]
fn maestria_do_bonus_tipo_1_vai_ao_banco_e_volta_no_relogin() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, _selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("M{:06}", nanos % 1_000_000);
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    let conta_id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;

    // Login + PIN + criação (TransKnight nível 1) + seleção, pelo protocolo real.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_criacao(0, &nome, 0)).expect("create");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFNewCharacter);
    s.write_all(&frame_selecao(0)).expect("seleção");
    let (opcode_mob, corpo_mob) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_mob, Opcode::CNFCharacterLogin);
    let mob = Mob::decode(&corpo_mob[4..]).expect("mob@4");
    assert_eq!(
        (mob.current_score.special, mob.special_bonus),
        ([0; 4], 0),
        "BaseMob nasce com Special[4] e SpecialBonus zerados"
    );
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateScore,
        "score de entrada"
    );
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::UpdateEtc,
        "etc de entrada"
    );

    // Pontos de maestria: o ganho por nível (`CMob.cpp:1128/1142`) é outra entrega; aqui eles
    // entram como fixture explícita **no banco**, como o `skill_bonus` da prova anterior.
    let personagem = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster")
        .remove(0);
    let mut progressao = rt
        .block_on(backend.progression(personagem.id))
        .expect("progressão");
    progressao.special_bonus = 10;
    rt.block_on(backend.store_progression(personagem.id, &progressao))
        .expect("pontos de maestria");

    // 1) Um ponto em `Special[0]`: `SpecialBonus` 10 → 9 no `UpdateEtc`; o `Special[4]` do
    //    `UpdateScore` continua a sentinela `0xCC` do `SendScore`.
    s.write_all(&frame_bonus(1, 0, 0)).expect("maestria 0");
    let (opcode_etc, etc) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_etc, Opcode::UpdateEtc);
    assert_eq!(
        u16::from_le_bytes([etc[22], etc[23]]),
        9,
        "SpecialBonus@22 debitado"
    );
    let (opcode_score, score) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_score, Opcode::UpdateScore);
    assert_eq!(
        score[136..140],
        [0xCC; 4],
        "Special[4] do SendScore é sentinela 0xCC, não a maestria"
    );

    // 2) Mais dois pontos no mesmo atributo: teto do nível 1 = 3 × (1 + 1) ÷ 2 = 3.
    for esperado in [8u16, 7] {
        s.write_all(&frame_bonus(1, 0, 0)).expect("maestria 0");
        let (opcode_etc, etc) = abrir(&ler_frame(&mut s));
        assert_eq!(opcode_etc, Opcode::UpdateEtc);
        assert_eq!(
            u16::from_le_bytes([etc[22], etc[23]]),
            esperado,
            "SpecialBonus@22"
        );
        assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::UpdateScore);
    }

    // 3) Quarto ponto no mesmo atributo é recusado (teto do nível) e não gera frame; o pedido
    //    seguinte, em outro atributo, é aceito — a ordem do socket garante que a recusa já
    //    foi processada quando este par de frames chega.
    s.write_all(&frame_bonus(1, 0, 0))
        .expect("maestria no teto");
    s.write_all(&frame_bonus(1, 2, 0)).expect("maestria 2");
    let (opcode_etc, etc) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_etc, Opcode::UpdateEtc);
    assert_eq!(
        u16::from_le_bytes([etc[22], etc[23]]),
        6,
        "SpecialBonus@22: 7 − 1, a recusa não debitou"
    );
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::UpdateScore);

    // 4) Banco: `Special[0] = 3` (o quarto ponto não entrou), `Special[2] = 1`, 6 pontos.
    let progressao = rt
        .block_on(backend.progression(personagem.id))
        .expect("progressão");
    assert_eq!(
        (progressao.special, progressao.special_bonus),
        ([3, 0, 1, 0], 6),
        "maestria e pontos em character_special"
    );
    drop(s);

    // 5) Relogin: `STRUCT_MOB` volta com a maestria do banco nas duas cópias de `Special`.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta B");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login B");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin B");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção B");
    let (opcode_mob, corpo_mob) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode_mob, Opcode::CNFCharacterLogin);
    let mob = Mob::decode(&corpo_mob[4..]).expect("mob@4");
    assert_eq!(mob.base_score.special, [3, 0, 1, 0], "BaseScore.Special@84");
    assert_eq!(
        mob.current_score.special,
        [3, 0, 1, 0],
        "CurrentScore.Special@132"
    );
    assert_eq!(mob.special_bonus, 6, "SpecialBonus@790");
}

/// Prova do inventário persistido: criação pelo protocolo grava `character_items`, um item
/// acrescentado **no banco** aparece no `STRUCT_MOB` do segundo `_MSG_CharacterLogin`, e o
/// corpo inteiro é comparado byte a byte com o esperado montado das linhas lidas do Postgres.
#[test]
fn inventario_persistido_vira_struct_mob_no_relogin() {
    let Some(dsn) = common::dsn_ou_skip("selchar") else {
        return;
    };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );
    let (porta, _selection) = servidor(Arc::clone(&backend), Arc::clone(&rt));
    let conta = conta_unica();
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("I{:06}", nanos % 1_000_000);
    rt.block_on(backend.create_account(&conta, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&conta, &hash, Role::Player))
        .expect("credencial");
    let conta_id = rt
        .block_on(backend.find_by_login(&conta))
        .expect("conta")
        .id;

    // 1) Login + PIN + `_MSG_CreateCharacter` pelo protocolo real.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_criacao(0, &nome, 0)).expect("create");
    assert_eq!(
        abrir(&ler_frame(&mut s)).0,
        Opcode::CNFNewCharacter,
        "criação aceita"
    );
    drop(s);

    // 2) O inventário inicial está persistido, não só no pacote.
    let personagem = rt
        .block_on(backend.list_by_account(conta_id))
        .expect("roster")
        .remove(0);
    let linhas = rt.block_on(backend.items(personagem.id)).expect("itens");
    eprintln!(
        "Q character_items personagem={} linhas={}",
        personagem.id,
        linhas.len()
    );
    for linha in &linhas {
        eprintln!(
            "Q slot={} item={} qty={} refine={} ancient={} gem={}/{}",
            linha.slot,
            linha.item_id,
            linha.qty,
            linha.refine,
            linha.ancient,
            linha.gem_effect,
            linha.gem_value
        );
    }
    let por_slot = |slot: i64| linhas.iter().find(|linha| linha.slot == slot).copied();
    assert_eq!(
        por_slot(0).map(|linha| linha.item_id),
        Some(1),
        "Equip[0] = rosto TransKnight do BaseMob"
    );
    assert_eq!(
        por_slot(1).map(|linha| linha.item_id),
        Some(1103),
        "Equip[1] = arma inicial"
    );
    let pilha = por_slot(16).expect("Carry[0]");
    assert_eq!(
        (pilha.item_id, pilha.qty),
        (401, 120),
        "EF_AMOUNT do template vira qty"
    );
    assert_eq!(
        por_slot(79).map(|linha| linha.item_id),
        Some(547),
        "kill mark em Carry[63]"
    );
    let short = rt
        .block_on(backend.short_skill(personagem.id))
        .expect("short_skill")
        .expect("linha de ShortSkill gravada na criação");
    assert_eq!(short, [SHORT_SKILL_EMPTY; SHORT_SKILL_LEN]);

    // 3) Equipa no banco um item que NÃO está no template: o login tem de refleti-lo.
    rt.block_on(backend.insert_items(
        personagem.id,
        &[ItemRow {
            slot: 7,
            item_id: 1234,
            qty: 1,
            ..ItemRow::default()
        }],
    ))
    .expect("item equipado no banco");
    let linhas = rt.block_on(backend.items(personagem.id)).expect("itens");
    assert!(
        linhas
            .iter()
            .any(|linha| linha.slot == 7 && linha.item_id == 1234),
        "item persistido antes do relogin"
    );

    // 4) Relogin: `STRUCT_MOB` montado do banco e comparado byte a byte.
    let mut s = TcpStream::connect(("127.0.0.1", porta)).expect("conecta relogin");
    s.set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    s.write_all(&INITCODE.to_le_bytes()).expect("init");
    s.write_all(&frame_login(&conta)).expect("login");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::CNFAccountLogin);
    s.write_all(&frame_pin(PIN)).expect("pin");
    assert_eq!(abrir(&ler_frame(&mut s)).0, Opcode::AccountSecure);
    s.write_all(&frame_selecao(0)).expect("seleção");
    let (opcode, corpo) = abrir(&ler_frame(&mut s));
    assert_eq!(opcode, Opcode::CNFCharacterLogin);
    assert_eq!(corpo.len(), CHARACTER_LOGIN_LEN);
    let mob = Mob::decode(&corpo[4..]).expect("mob@4");
    assert_eq!(
        mob.equip[7].index, 1234,
        "equipamento do banco no segundo login"
    );
    assert_eq!(mob.equip[0].index, 1, "rosto TransKnight do banco");
    assert_eq!(mob.carry[63].index, 547, "kill mark persistido");
    assert_eq!(
        mob.carry[63].effects,
        [[75, 0], [76, 0], [77, 0]],
        "EF_CURKILL/EF_LTOTKILL/EF_HTOTKILL"
    );
    assert_eq!(
        &corpo[1034..1034 + SHORT_SKILL_LEN],
        &[SHORT_SKILL_EMPTY; SHORT_SKILL_LEN],
        "ShortSkill@1034 do banco"
    );
    let esperado = Mob::from_persisted(&personagem, &linhas).encode();
    assert_eq!(
        corpo[4..4 + esperado.len()],
        esperado[..],
        "STRUCT_MOB byte a byte contra as linhas do Postgres"
    );
    for (rotulo, base) in [("Equip", 4 + 140), ("Carry", 4 + 268)] {
        let slots = if rotulo == "Equip" { 16 } else { 64 };
        for indice in 0..slots {
            let item = &corpo[base + indice * 8..base + indice * 8 + 8];
            if item.iter().all(|byte| *byte == 0) {
                continue;
            }
            let hex: String = item.iter().map(|byte| format!("{byte:02x}")).collect();
            eprintln!("PROVA {rotulo}[{indice}] {hex}");
        }
    }
    drop(s);
}
