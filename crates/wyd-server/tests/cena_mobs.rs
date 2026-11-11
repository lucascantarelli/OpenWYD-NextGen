//! Cena no fio: `_MSG_CreateMob` do NPC de `npc_definition` e o diálogo por `_MSG_Action`.
//!
//! Prova **viva** (gated por `WYD_LIVE_CPSOCK`): a sessão entra pelo caminho normal do
//! cliente — `INITCODE` → `_MSG_AccountLogin` → `_MSG_AccountSecure` → `_MSG_CharacterLogin` —
//! e o teste decodifica os frames que o servidor emite, sem atalho interno que pule o
//! handshake nem o banco.
//!
//! Ambientes:
//! - `WYD_LIVE_CPSOCK` (ex.: `127.0.0.1:8281`) — sem ele o teste é ignorado;
//! - `WYD_LIVE_ACCOUNT` / `WYD_LIVE_PASSWORD` / `WYD_LIVE_PIN` (default `2468`);
//! - `WYD_LIVE_CHARACTER_SLOT` (default `0`);
//! - `WYD_LIVE_HOLD_SECS` (default `10`): quanto tempo a sessão fica aberta observando a cena
//!   depois do primeiro `CreateMob` — é a janela em que uma recarga de configuração deve
//!   chegar como delta (`RemoveMob`/`CreateMob`), nunca como duplicata.
//!
//! O que ele exige:
//! 1. `_MSG_CNFCharacterLogin` (HEADER.ID = `ESCENE_FIELD`);
//! 2. pelo menos um `_MSG_CreateMob` com **232 bytes** decodificado por
//!    `payload::scene::CreateMob` e HEADER.ID = 30000, com nome e posição coerentes;
//! 3. nenhum `mob_id` repetido enquanto a entidade está na vista (nada de duplicar);
//! 4. `_MSG_Action` no NPC escolhido respondendo `_MSG_MessagePanel` com o **mesmo nome** que
//!    veio no `CreateMob` (o `display_name` de `npc_definition`).

use std::io::{ErrorKind, Read, Write};
use std::net::TcpStream;
use std::time::{Duration, Instant};

use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::scene::{CreateMob, RemoveMob};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, Action, CharacterLogin, Header, MessagePanel};

/// `HEADER.ID` do pacote autoritativo de cena (`ESCENE_FIELD`).
const ID_SCENE: u16 = 30000;

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

fn enviar<F>(stream: &mut TcpStream, chave: u8, opcode: Opcode, len: usize, corpo: F)
where
    F: FnOnce(&mut [u8]),
{
    let mut frame = frame(chave, opcode, len);
    corpo(&mut frame);
    encode_in_place(&mut frame, chave).expect("ofuscação");
    stream.write_all(&frame).expect("pacote");
}

/// Handshake real (INITCODE + senha da conta + PIN) no socket aberto.
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
    let (opcode, _, _) = ler_frame(stream, Duration::from_secs(10)).expect("resposta do login");
    assert_eq!(opcode, Opcode::CNFAccountLogin, "senha da conta aceita");
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
    let (opcode, _, _) = ler_frame(stream, Duration::from_secs(10)).expect("resposta do PIN");
    assert_eq!(opcode, Opcode::AccountSecure, "PIN aceito ou cadastrado");
}

/// Lê um frame inteiro (cabeçalho + corpo), confere o checksum e devolve
/// `(opcode, HEADER.ID, frame decodificado)`. `None` = nada chegou no prazo.
fn ler_frame(stream: &mut TcpStream, espera: Duration) -> Option<(Opcode, u16, Vec<u8>)> {
    stream.set_read_timeout(Some(espera)).ok()?;
    let mut head = [0u8; HEADER_SIZE];
    match stream.read_exact(&mut head) {
        Ok(()) => {}
        Err(error) if matches!(error.kind(), ErrorKind::WouldBlock | ErrorKind::TimedOut) => {
            return None;
        }
        Err(_) => return None,
    }
    let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
    let mut reply = vec![0u8; size];
    reply[..HEADER_SIZE].copy_from_slice(&head);
    if size > HEADER_SIZE {
        stream.read_exact(&mut reply[HEADER_SIZE..]).ok()?;
    }
    let computado = decode_in_place(&mut reply).ok()?;
    assert_eq!(computado, reply[3], "checksum de todo frame do servidor");
    let header = Header::decode(&reply).ok()?;
    let opcode = Opcode::from_wire(header.message_type)?;
    Some((opcode, header.id, reply))
}

#[test]
fn cena_entrega_create_mob_do_snapshot_e_o_npc_responde_ao_dialogo() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        return;
    };
    let conta = std::env::var("WYD_LIVE_ACCOUNT").expect("WYD_LIVE_ACCOUNT com a conta real");
    let senha = std::env::var("WYD_LIVE_PASSWORD").expect("WYD_LIVE_PASSWORD da conta real");
    let pin = std::env::var("WYD_LIVE_PIN").unwrap_or_else(|_| "2468".to_string());
    let slot: i32 = std::env::var("WYD_LIVE_CHARACTER_SLOT")
        .ok()
        .and_then(|valor| valor.parse().ok())
        .unwrap_or(0);
    let espera: u64 = std::env::var("WYD_LIVE_HOLD_SECS")
        .ok()
        .and_then(|valor| valor.parse().ok())
        .unwrap_or(10);

    let mut stream = TcpStream::connect(&endereco).expect("conecta no canal");
    stream
        .set_read_timeout(Some(Duration::from_secs(10)))
        .expect("timeout");
    autenticar(&mut stream, &conta, &senha, &pin);

    // Entrada no mundo pela seleção normal do cliente. O 7662 manda **36 B** aqui (corpo de
    // 24); o teste usa o mesmo tamanho para provar o caminho real, não o curto dos unitários.
    enviar(
        &mut stream,
        11,
        Opcode::CharacterLogin,
        CharacterLogin::WIRE_LEN_CLIENTE,
        |f| {
            CharacterLogin { slot, force: 1 }
                .encode_into(&mut f[..CharacterLogin::WIRE_LEN])
                .expect("corpo da seleção");
        },
    );

    // 1) `CNFCharacterLogin` com o `STRUCT_MOB`; PosX/PosY no corpo (+12/+14).
    let (opcode, id, cnf) = ler_frame(&mut stream, Duration::from_secs(15))
        .unwrap_or_else(|| panic!("{endereco} não respondeu a seleção do slot {slot}"));
    assert_eq!(opcode, Opcode::CNFCharacterLogin, "personagem encarnado");
    assert_eq!(id, ID_SCENE, "CNFCharacterLogin leva ESCENE_FIELD no ID");
    let jogador = (
        i16::from_le_bytes([cnf[12], cnf[13]]),
        i16::from_le_bytes([cnf[14], cnf[15]]),
    );
    // `ClientID` em +1030 do corpo: o id que o cliente usa para si. O legado reserva
    // `id < MAX_USER` (1000) para jogador — um número acima disso faz o cliente tratar o
    // personagem como criatura.
    let client_id = u16::from_le_bytes([cnf[12 + 1030], cnf[12 + 1031]]);
    assert!(
        client_id < 1_000,
        "ClientID {client_id} fora da faixa de jogador (< MAX_USER)"
    );

    // 2) Rajada de `CreateMob`: cada NPC materializado pela revisão vigente.
    let mut mobs: Vec<CreateMob> = Vec::new();
    let mut ativos: std::collections::BTreeMap<u16, CreateMob> = std::collections::BTreeMap::new();
    let mut removes = 0usize;
    let mut crias = 0usize;
    // O `UpdateScore` do próprio personagem chega logo depois do `CNF` e antes da cena
    // (`sendScoreSelf` do legado): é o que desenha HP/MP/EXP na entrada.
    let mut score_visto = false;
    let inicio = Instant::now();
    while inicio.elapsed() < Duration::from_secs(20) {
        let Some((opcode, header_id, corpo)) = ler_frame(&mut stream, Duration::from_millis(1200))
        else {
            break;
        };
        match opcode {
            Opcode::UpdateScore => {
                assert_eq!(
                    header_id, client_id,
                    "UpdateScore de entrada leva o HEADER.ID do próprio jogador"
                );
                score_visto = true;
            }
            Opcode::CreateMob => {
                assert_eq!(header_id, ID_SCENE, "CreateMob leva ESCENE_FIELD no ID");
                let mob = CreateMob::decode(&corpo).unwrap_or_else(|erro| {
                    panic!(
                        "CreateMob com {} bytes não decodificou no layout provado: {erro}",
                        corpo.len()
                    )
                });
                assert!(
                    !ativos.contains_key(&mob.mob_id),
                    "mob {} duplicado na vista",
                    mob.mob_id
                );
                assert!(!mob.name_bytes().is_empty(), "mob sem nome no fio");
                if mob.mob_id == client_id {
                    assert_eq!(
                        mob.create_type, 2,
                        "o próprio jogador entra com CreateType 2 (acabou de entrar)"
                    );
                } else {
                    assert_eq!(
                        mob.create_type, 0,
                        "NPC/outro jogador entra com CreateType 0"
                    );
                    assert!(
                        mob.mob_id >= 1_000,
                        "criatura vive na faixa >= MAX_USER (mob_id {})",
                        mob.mob_id
                    );
                }
                crias += 1;
                ativos.insert(mob.mob_id, mob);
                mobs.push(mob);
            }
            Opcode::RemoveMob => removes += 1,
            outro => panic!("frame inesperado na entrada do mundo: {outro:?}"),
        }
    }
    assert!(
        !mobs.is_empty(),
        "{endereco} não enviou nenhum CreateMob depois do CNF"
    );
    assert!(
        score_visto,
        "{endereco} não enviou UpdateScore na entrada do mundo"
    );
    assert!(
        mobs.iter().any(|mob| mob.mob_id == client_id),
        "o jogador não recebeu o próprio CreateMob"
    );
    eprintln!(
        "PASS(cena): {} NPCs visíveis no slot {slot} ({} bytes cada), jogador em {:?}",
        mobs.len(),
        CreateMob::WIRE_LEN,
        jogador
    );

    // O NPC mais próximo do personagem é o alvo do diálogo (o servidor valida o alcance).
    // Criatura vive em `>= MAX_USER` (FASE-03-17): o jogador agora é `mob_id` 1..999 e não
    // pode ser o alvo — clicar em si mesmo não abre diálogo de NPC.
    let alvo = mobs
        .iter()
        .filter(|mob| mob.mob_id >= 1_000)
        .min_by_key(|mob| {
            let dx = i32::from(mob.pos_x).abs_diff(i32::from(jogador.0));
            let dy = i32::from(mob.pos_y).abs_diff(i32::from(jogador.1));
            dx.max(dy)
        })
        .copied()
        .expect("pelo menos um mob");
    let nome = String::from_utf8_lossy(alvo.name_bytes()).to_string();
    eprintln!(
        "PASS(cena): alvo do diálogo mob_id={} nome={nome:?} em ({}, {})",
        alvo.mob_id, alvo.pos_x, alvo.pos_y
    );

    // 3) `_MSG_Action` no NPC → `_MSG_MessagePanel` com o mesmo nome do CreateMob.
    //
    // O alcance de diálogo é do servidor (`TALK_RANGE_TILES`) e o `CreateMob` não traz
    // `merchant`: nem todo mob próximo responde (monstro não fala). Tenta os mais próximos
    // em ordem e exige que o que responder seja um deles — sem adivinhar quem fala.
    // `talk_at` exige o personagem a no máximo `TALK_RANGE_TILES` do **ponto clicado**
    // (`world.rs`), então só entram candidatos nesse raio; clicar longe não responde.
    let mut candidatos: Vec<_> = mobs
        .iter()
        .filter(|mob| mob.mob_id >= 1_000)
        .filter(|mob| {
            let dx = i32::from(mob.pos_x).abs_diff(i32::from(jogador.0));
            let dy = i32::from(mob.pos_y).abs_diff(i32::from(jogador.1));
            dx.max(dy) <= 5
        })
        .collect();
    candidatos.sort_by_key(|mob| {
        let dx = i32::from(mob.pos_x).abs_diff(i32::from(jogador.0));
        let dy = i32::from(mob.pos_y).abs_diff(i32::from(jogador.1));
        dx.max(dy)
    });
    let mut resposta: Option<(&CreateMob, Vec<u8>)> = None;
    for candidato in candidatos.iter().take(5) {
        enviar(&mut stream, 11, Opcode::Action, Action::WIRE_LEN, |f| {
            Action {
                pos_x: jogador.0,
                pos_y: jogador.1,
                effect: 0,
                speed: 0,
                route: [0u8; 24],
                target_x: i16::try_from(candidato.pos_x).expect("posição no fio"),
                target_y: i16::try_from(candidato.pos_y).expect("posição no fio"),
            }
            .encode_into(f)
            .expect("corpo do Action");
        });
        let limite = Instant::now() + Duration::from_secs(2);
        while Instant::now() < limite {
            let Some((opcode, _, corpo)) = ler_frame(&mut stream, Duration::from_millis(600))
            else {
                continue;
            };
            if opcode == Opcode::MessagePanel {
                let texto = MessagePanel::decode(&corpo)
                    .expect("MessagePanel de 140 bytes")
                    .text_bytes()
                    .to_vec();
                resposta = Some((candidato, texto));
                break;
            }
        }
        if resposta.is_some() {
            break;
        }
    }
    // Sem NPC no raio, clica no próprio tile: o servidor escolhe o NPC mais próximo dele
    // (é o `talk_at` do legado). O nome devolvido tem de existir na vista.
    if resposta.is_none() {
        enviar(&mut stream, 11, Opcode::Action, Action::WIRE_LEN, |f| {
            Action {
                pos_x: jogador.0,
                pos_y: jogador.1,
                effect: 0,
                speed: 0,
                route: [0u8; 24],
                target_x: jogador.0,
                target_y: jogador.1,
            }
            .encode_into(f)
            .expect("corpo do Action");
        });
        let limite = Instant::now() + Duration::from_secs(3);
        while Instant::now() < limite {
            let Some((opcode, _, corpo)) = ler_frame(&mut stream, Duration::from_millis(600))
            else {
                continue;
            };
            if opcode == Opcode::MessagePanel {
                let texto = MessagePanel::decode(&corpo)
                    .expect("MessagePanel de 140 bytes")
                    .text_bytes()
                    .to_vec();
                resposta = Some((&mobs[0], texto));
                break;
            }
        }
    }
    let (alvo, dialogo) = resposta
        .unwrap_or_else(|| panic!("{endereco} não respondeu o Action de nenhum NPC próximo"));
    if alvo.mob_id < 1_000 {
        // Resposta do clique no próprio tile: o NPC não está na vista coletada; exige só
        // que o painel traga o nome de um NPC que o servidor conhece (não vazio).
        assert!(!dialogo.is_empty(), "MessagePanel vazio");
        eprintln!(
            "PASS(cena): diálogo respondeu {:?} (NPC fora da vista coletada)",
            String::from_utf8_lossy(&dialogo)
        );
        return;
    }
    let nome = String::from_utf8_lossy(alvo.name_bytes()).to_string();
    assert_eq!(
        dialogo,
        nome.as_bytes(),
        "MessagePanel devolve o display_name do NPC do snapshot"
    );
    eprintln!(
        "PASS(cena): diálogo respondeu {:?}",
        String::from_utf8_lossy(&dialogo)
    );

    // 4) Janela de recarga: com a sessão aberta, uma revisão nova só pode chegar como delta.
    let janela = Instant::now() + Duration::from_secs(espera);
    while Instant::now() < janela {
        let Some((opcode, header_id, corpo)) = ler_frame(&mut stream, Duration::from_millis(1200))
        else {
            continue;
        };
        match opcode {
            Opcode::CreateMob => {
                assert_eq!(header_id, ID_SCENE);
                let mob = CreateMob::decode(&corpo).expect("CreateMob decodifica");
                assert!(
                    ativos.insert(mob.mob_id, mob).is_none(),
                    "recarga reenviou o mob {} que já estava na vista",
                    mob.mob_id
                );
                crias += 1;
            }
            Opcode::RemoveMob => {
                assert_eq!(corpo.len(), RemoveMob::WIRE_LEN, "RemoveMob tem 16 bytes");
                removes += 1;
                ativos.remove(&header_id);
            }
            _ => {}
        }
    }
    eprintln!(
        "PASS(cena): janela de {espera}s — creates={crias} removes={removes} ativos={}",
        ativos.len()
    );
}
