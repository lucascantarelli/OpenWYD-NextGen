//! Listener CPSock (`:8281`): `INITCODE` → frame ofuscado → gate de versão → sessão → `CNF`.
//!
//! Ordem obrigatória (ADR-0022 §3, `docs/arquitetura/protocolo.md` §1):
//! `INITCODE` (4 B) → `HEADER` (12 B, `Size` em claro) → resto do frame → `decode_in_place`
//! → checksum `Strict` → `inspect` (opcode) → `AccountLogin::decode` → gate de versão →
//! **cerca de sessão** (`session.rs`) → `_MSG_CNFAccountLogin`.
//!
//! A cerca vem **depois** do gate: versão recusada é recusa de protocolo (não abre sessão), e
//! conta já em jogo é recusa de sessão (protocolo está ok). Sem banco a conexão é recusada em
//! vez de abrir sessão sem lastro.
//!
//! Depois do `CNF` a conexão **continua** na seleção de personagem (`CLIENT-02`): o laço
//! `serve_selection` consome cada frame inteiro (sem dessincronizar o `Size` seguinte), renova
//! a cerca e atende `_MSG_CreateCharacter`/`_MSG_DeleteCharacter`/`_MSG_CharacterLogin` por
//! `characters.rs`. Pacote que este processo ainda não serve (chat, `_MSG_Ping`, mundo) é
//! **tolerado** — antes, o primeiro pacote encerrava a sessão, que é exatamente o defeito
//! corrigido aqui. Só framing inválido e falha de I/O encerram a conexão.
//!
//! O que **não** existe ainda: autenticação de conta (`_MSG_AccountLogin` só tem a versão
//! validada) e pacotes de cena (`ESCENE_FIELD`, FASE-03/08-6).
//!
//! Árvore (ADR-0026): `handshake` (INITCODE/frame/versão), `login` (senha/PIN/sessão),
//! `selection` (seleção e personagens), `action` (dispatch de item/ataque/diálogo/cena) e
//! `limits` (vaga por IP); este `mod.rs` guarda o listener, o `Stage`/`Scene` e as respostas.

use std::io::{self, Read, Write};
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::{Duration, Instant};

use wyd_net::{VersionGate, NET};
use wyd_protocol::codec::checksum::{self, ChecksumPolicy};
use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::combat::Attack;
use wyd_protocol::payload::commerce::{ReqShopList, ShopList, MAX_SHOPLIST};
use wyd_protocol::payload::movement::Action;
use wyd_protocol::payload::quest::Quest;
use wyd_protocol::payload::scene::{CreateMob, RemoveMob};
use wyd_protocol::payload::ui::{MessageChat, MessagePanel};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::{inspect, INITCODE};
use wyd_protocol::wire::header::{Header, HEADER_SIZE};
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{
    AccountLogin, AccountSecure, ApplyBonus, Buy, CharacterLogin, CreateCharacter, DeleteCharacter,
    SendItem, UseItem,
};

use crate::auth::{Authenticator, PinOutcome};
use crate::personagens::{
    account_login_body, cargo_items, character_login_body, delete_character_body,
    new_character_body, Item, Mob, ID_DELETE_CHARACTER, ID_NEW_CHARACTER, ID_SCENE, ID_SELCHAR,
};

use crate::channels::World;
use crate::characters::{LoginSnapshot, Selected, SelectionError, SelectionStore};
use crate::gameplay::UseItemRolls;
use crate::session::{SessionError, SessionGate, SessionLease};
use crate::world::{plan_scene, SceneMob};
use wyd_sim::XorShift64;

mod action;
mod handshake;
mod limits;
mod login;
mod selection;

pub use action::send_progress;
use handshake::handshake;
pub use handshake::{HandshakeError, HandshakeOk};
pub use limits::ConnectionLimit;
use limits::IpSlot;
use login::authorize_pin;
use selection::serve_selection;

/// Timeout de leitura do handshake: conexão muda não segura thread.
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(10);

/// Quantos períodos ociosos a sessão sobrevive antes de o servidor soltar a conexão.
///
/// Sem esse teto, um cliente que desaparece sem FIN (queda de energia, cabo) manteria a
/// sessão renovada para sempre e o jogador ficaria preso fora do jogo.
const MAX_IDLE_ROUNDS: u32 = 10;

/// Período de leitura em jogo: a recarga de configuração chega ao cliente em ~1 s, e não no
/// timeout de ociosidade da sessão (que continua sendo `idle × MAX_IDLE_ROUNDS`).
const SCENE_POLL: Duration = Duration::from_secs(1);

/// Renovação do sinal de vida da sessão. Um `UPDATE` por segundo no banco seria desperdício;
/// o TTL da cerca é de minutos.
const RENEW_EVERY: Duration = Duration::from_secs(30);

/// Estágio da conexão depois do login mantido só para o log (seleção × em jogo).
#[derive(Debug, Clone)]
enum Stage {
    Selecting,
    /// Encarnado: entidade do mundo (GAME-08), quando o processo tem mundo, mais o
    /// personagem escolhido — `_MSG_ApplyBonus` precisa dele para gravar a progressão.
    InGame {
        entity: Option<wyd_core::Entity>,
        selected: Box<Selected>,
        /// `HEADER.ID` que o cliente conhece para o próprio mob (`ClientID` do login).
        client_id: u16,
        /// Entidades que esta sessão já recebeu (`_MSG_CreateMob`) e a geração que as produziu.
        scene: Scene,
    },
}

/// Cena já emitida ao cliente nesta sessão.
///
/// Guardar o conjunto (e não só a geração) é o que permite emitir **delta** na recarga e na
/// entrada/saída de jogador: quem saiu vira `_MSG_RemoveMob`, quem entrou vira
/// `_MSG_CreateMob`, e nada é reenviado sem necessidade — nem fica mob órfão no cliente.
/// (Entidade que só mudou de posição não é reenviada: o pacote de movimento não está provado,
/// resíduo declarado na GAME-09.)
#[derive(Debug, Clone, Default)]
struct Scene {
    generation: u64,
    /// Id de cena → é personagem? O `RemoveType` da saída depende disso (logout × fora de vista).
    mobs: std::collections::BTreeMap<u16, bool>,
}

impl Stage {
    fn as_str(&self) -> &'static str {
        match self {
            Self::Selecting => "selecting",
            Self::InGame { .. } => "in_game",
        }
    }

    /// Entidade viva desta conexão (para o save de saída).
    fn entity(&self) -> Option<wyd_core::Entity> {
        match self {
            Self::Selecting => None,
            Self::InGame { entity, .. } => *entity,
        }
    }
}

/// Aceita conexões para sempre, uma thread por conexão (handshake é curto e raro).
pub fn serve_listener(
    listener: TcpListener,
    world: Arc<World>,
    gate: VersionGate,
    auth: Option<Arc<Authenticator>>,
    sessions: Option<Arc<SessionGate>>,
    selection: Option<Arc<SelectionStore>>,
) {
    serve_listener_with_limit(
        listener,
        world,
        gate,
        auth,
        sessions,
        selection,
        ConnectionLimit::from_env(),
    );
}

/// Igual a [`serve_listener`], com limite de conexões por IP explícito (testes/configuração).
pub fn serve_listener_with_limit(
    listener: TcpListener,
    world: Arc<World>,
    gate: VersionGate,
    auth: Option<Arc<Authenticator>>,
    sessions: Option<Arc<SessionGate>>,
    selection: Option<Arc<SelectionStore>>,
    limite: ConnectionLimit,
) {
    let ativos: Arc<std::sync::Mutex<std::collections::HashMap<std::net::IpAddr, usize>>> =
        Arc::new(std::sync::Mutex::new(std::collections::HashMap::new()));
    for aceito in listener.incoming() {
        match aceito {
            Ok(stream) => {
                let ip = stream
                    .peer_addr()
                    .map_or(std::net::IpAddr::V4(std::net::Ipv4Addr::UNSPECIFIED), |e| {
                        e.ip()
                    });
                let Some(vaga) = IpSlot::reservar(&ativos, ip, limite.per_ip) else {
                    let total = crate::metrics::ip_limit_refused();
                    log(
                        "warn",
                        "accept",
                        &format!("result=refused reason=ip_limit ip={ip} total={total}"),
                    );
                    // Agregado periódico: um evento por recusa vira ruído; a cada 25 sai o resumo.
                    if total.is_multiple_of(25) {
                        log(
                            "warn",
                            "accept",
                            &format!("result=ip_limit_summary total={total} last_ip={ip}"),
                        );
                    }
                    continue;
                };
                let world = Arc::clone(&world);
                let auth = auth.clone();
                let sessions = sessions.clone();
                let selection = selection.clone();
                std::thread::spawn(move || {
                    let _vaga = vaga;
                    match handle_connection(
                        stream,
                        &world,
                        gate,
                        auth.as_deref(),
                        sessions.as_deref(),
                        selection.as_deref(),
                    ) {
                        Ok(ok) => log(
                            "info",
                            "handshake",
                            &format!(
                                "result=accepted version={} channels={} kw={} session={}",
                                ok.client_version,
                                world.game_channel_count(),
                                ok.key_word,
                                ok.session_id
                                    .map(|id| id.to_string())
                                    .unwrap_or_else(|| "-".to_string())
                            ),
                        ),
                        Err(error) => log(
                            "warn",
                            "handshake",
                            &format!("result=rejected reason={error}"),
                        ),
                    }
                });
            }
            Err(error) => {
                log("error", "accept", &format!("reason={error}"));
                return;
            }
        }
    }
}

/// Executa o handshake de uma conexão já aceita, responde `_MSG_CNFAccountLogin` e atende a
/// seleção de personagem enquanto o cliente estiver conectado.
///
/// A resposta do login é um frame **só de cabeçalho** (`Size = 12`) com o opcode de
/// confirmação: o corpo do `_MSG_CNFAccountLogin` é UNVERIFIED (`docs/arquitetura/protocolo.md`
/// §6) e inventá-lo seria pior que confirmar o handshake. O mesmo vale para as confirmações
/// de personagem — o **resultado** é decidido no banco, o corpo entra com a captura (08-6).
pub fn handle_connection(
    mut stream: TcpStream,
    world: &World,
    gate: VersionGate,
    auth: Option<&Authenticator>,
    sessions: Option<&SessionGate>,
    selection: Option<&SelectionStore>,
) -> Result<HandshakeOk, HandshakeError> {
    let _ = stream.set_read_timeout(Some(HANDSHAKE_TIMEOUT));
    let _ = stream.set_write_timeout(Some(HANDSHAKE_TIMEOUT));
    NET.session_delta(1);
    let resultado = (|| {
        let (mut ok, conta) = handshake(&mut stream, world, gate, auth, selection)?;
        // A cerca só abre **depois** da senha: senha errada ou PIN errado não deixam sessão.
        // O `pendente` é o pacote de seleção que o cliente mandou no lugar do PIN.
        let (lease, pendente) =
            authorize_pin(&mut stream, &conta, auth, sessions, ok.client_version)?;
        ok.session_id = lease.as_ref().map(SessionLease::id);
        // Segura a conexão: é a vida da sessão. Sem isso a sessão morreria no CNF e a cerca
        // não impediria nada — e sem o laço de seleção o jogador não passa do login.
        let idle = sessions.map_or(HANDSHAKE_TIMEOUT, SessionGate::idle);
        let entity = serve_selection(
            &mut stream,
            &conta,
            selection,
            lease.as_ref(),
            idle,
            pendente,
        );
        // Saída (logout, timeout ou queda): tira do mundo e enfileira o save do estado.
        if let Some(selection) = selection {
            selection.leave_world(entity);
        }
        Ok(ok)
    })();
    NET.session_delta(-1);
    resultado
}

/// Exige `_MSG_AccountSecure` (PIN) e **só então** abre a cerca de sessão.
///
/// Pacote de personagem antes do PIN é recusa: sem isso o cliente pularia a segunda senha
/// usando `_MSG_CharacterLogin` direto. PIN recusado responde `_MSG_AccountSecureFail` e
/// encerra a conexão — nenhum caminho deixa sessão pela metade.
/// Pacote que o handshake consumiu antes do PIN: frame já em claro + key word.
type Pendente = (Vec<u8>, u8);

/// Resultado do passo de PIN: sessão aberta (quando há cerca) e pacote pendente.
type PinStep = (Option<SessionLease>, Option<Pendente>);

/// Conta autenticada nesta conexão (`account` decide o roster e o dono da criação).
struct Sessao<'a> {
    account: &'a str,
}

/// Frame com corpo (`Size = 12 + corpo`), ofuscado com a chave do frame de entrada.
///
/// `id` é o `HEADER.ID` do pacote: o legado envia `ESCENE_FIELD` (30000) no snapshot do
/// mundo e os ids de seletor nos pacotes de seleção (`Basedef.h`, `IDSelChar`/`IDNewCharacter`).
fn reply_with_body(
    stream: &mut TcpStream,
    key_word: u8,
    id: u16,
    opcode: Opcode,
    body: &[u8],
) -> bool {
    let mut frame = vec![0u8; HEADER_SIZE + body.len()];
    frame[HEADER_SIZE..].copy_from_slice(body);
    let size = match u16::try_from(frame.len()) {
        Ok(size) => size,
        Err(_) => return false,
    };
    let cabecalho = Header {
        size,
        key_word,
        checksum: 0,
        message_type: opcode.wire(),
        id,
        client_tick: 0,
    };
    if cabecalho.encode(&mut frame).is_err() || encode_in_place(&mut frame, key_word).is_err() {
        return false;
    }
    stream.write_all(&frame).is_ok() && stream.flush().is_ok()
}

/// Frame só de cabeçalho com o opcode pedido, ofuscado com a chave do frame de entrada.
/// Ponto único do pacote de recusa: cabeçalho + trailer opcional de 1 byte com o motivo.
///
/// Cliente antigo lê só o `HEADER` (12 B) e ignora o byte extra; cliente novo lê o motivo.
/// O caminho de sucesso não passa por aqui.
fn reply_recusa(
    stream: &mut TcpStream,
    key_word: u8,
    opcode: Opcode,
    motivo: wyd_protocol::motivo::CodigoRecusa,
) -> bool {
    let mut frame = [0u8; HEADER_SIZE + 1];
    let cabecalho = Header {
        size: (HEADER_SIZE + 1) as u16,
        key_word,
        checksum: 0,
        message_type: opcode.wire(),
        id: 1,
        client_tick: 0,
    };
    if cabecalho.encode(&mut frame).is_err() {
        return false;
    }
    // O trailer entra antes da ofuscação: o cliente decodifica o frame inteiro.
    frame[HEADER_SIZE] = motivo.byte();
    if encode_in_place(&mut frame, key_word).is_err() {
        return false;
    }
    stream.write_all(&frame).is_ok() && stream.flush().is_ok()
}

fn reply(stream: &mut TcpStream, key_word: u8, opcode: Opcode) -> bool {
    let mut frame = [0u8; HEADER_SIZE];
    let cabecalho = Header {
        size: HEADER_SIZE as u16,
        key_word,
        checksum: 0,
        message_type: opcode.wire(),
        id: 1,
        client_tick: 0,
    };
    if cabecalho.encode(&mut frame).is_err() || encode_in_place(&mut frame, key_word).is_err() {
        return false;
    }
    stream.write_all(&frame).is_ok() && stream.flush().is_ok()
}

fn is_timeout(error: &io::Error) -> bool {
    matches!(
        error.kind(),
        io::ErrorKind::WouldBlock | io::ErrorKind::TimedOut
    )
}

/// Log estruturado (uma linha, `key=value`) — mesmo formato do gateway.
fn log(level: &str, event: &str, fields: &str) {
    let linha = format!("level={level} event={event} {fields}");
    crate::logs::record(&linha);
    println!("{linha}");
}

#[cfg(test)]
mod tests {
    use super::action::{cp_panel, drain_chat, send_create_mob, send_remove_mob};
    use super::*;
    use std::net::{TcpListener, TcpStream};
    use wyd_protocol::payload::scene::{CreateMob, RemoveMob};

    /// Par de sockets loopback: a sessão escreve num lado, o teste lê do outro.
    fn par() -> (TcpStream, TcpStream) {
        let listener = TcpListener::bind("127.0.0.1:0").expect("porta efêmera");
        let endereco = listener.local_addr().expect("endereço");
        let cliente = TcpStream::connect(endereco).expect("cliente conecta");
        let (servidor, _) = listener.accept().expect("servidor aceita");
        (servidor, cliente)
    }

    /// Lê um frame do lado do cliente, desofusca e devolve `(cabeçalho, frame claro)`.
    fn ler_frame(cliente: &mut TcpStream) -> (Header, Vec<u8>) {
        use std::io::Read;
        let mut head = [0u8; HEADER_SIZE];
        cliente.read_exact(&mut head).expect("cabeçalho");
        let tamanho = usize::from(u16::from_le_bytes([head[0], head[1]]));
        let mut frame = vec![0u8; tamanho];
        frame[..HEADER_SIZE].copy_from_slice(&head);
        cliente
            .read_exact(&mut frame[HEADER_SIZE..])
            .expect("corpo");
        decode_in_place(&mut frame).expect("desofusca");
        let cabecalho = Header::decode(&frame[..HEADER_SIZE]).expect("cabeçalho claro");
        (cabecalho, frame)
    }

    #[test]
    fn chat_social_chega_ao_cliente_com_o_id_do_falante() {
        let (mut servidor, mut cliente) = par();
        let hub = Arc::new(crate::session::ChatHub::new());
        let (licenca, rx) = hub.join(7, "Kae");
        let mut texto = [0u8; 128];
        texto[..2].copy_from_slice(b"oi");
        assert_eq!(
            hub.deliver(
                &[7, 8],
                &crate::session::ChatLine {
                    speaker: 7,
                    text: texto,
                }
            ),
            1,
            "id sem sessão (8) não recebe"
        );
        assert!(drain_chat(&mut servidor, 0, &rx));
        let (cabecalho, frame) = ler_frame(&mut cliente);
        assert_eq!(cabecalho.message_type, Opcode::MessageChat.wire());
        assert_eq!(cabecalho.id, 7, "HEADER.ID é o id de cena do falante");
        assert_eq!(&frame[HEADER_SIZE..HEADER_SIZE + 2], b"oi");
        drop(licenca);
        assert_eq!(hub.online(), 0);
    }

    #[test]
    fn cp_panel_cabe_no_painel_e_descreve_o_efeito() {
        let texto = cp_panel(0);
        assert!(texto.contains("neutro"), "{texto}");
        assert!(
            texto.len() < MessagePanel::WIRE_LEN - HEADER_SIZE,
            "{texto}"
        );
        assert!(cp_panel(-75).contains("PK bloqueado"));
        assert!(cp_panel(30).contains("positivo"));
        assert!(cp_panel(-30).contains("listado"));
    }

    fn mob(id: u16, name: &[u8], actor: bool) -> SceneMob {
        let mut nome = [0u8; 16];
        nome[..name.len()].copy_from_slice(name);
        let mut equip = [0u16; 16];
        equip[0] = 831;
        SceneMob {
            id,
            actor,
            npc_id: 0,
            name: nome,
            position: wyd_core::Position { x: 300, y: 400 },
            level: 42,
            hp: 90,
            max_hp: 90,
            attack: 12,
            defense: 5,
            merchant: 0,
            chaos_rate: crate::personagens::CP_NEUTRO,
            attack_run: 82,
            direction: 3,
            equip,
            guild: 4,
        }
    }

    #[test]
    fn create_mob_de_personagem_traz_escene_field_e_a_aparencia_do_login() {
        let (mut servidor, mut cliente) = par();
        assert!(send_create_mob(
            &mut servidor,
            7,
            &mob(9, b"Buffy", true),
            CreateMob::CREATE_JUST_ENTERED
        ));

        let (cabecalho, frame) = ler_frame(&mut cliente);
        // Invariante 4: pacote autoritativo de cena vai com `ESCENE_FIELD`, nunca com o id
        // da conexão (`ClientID`), senão a barra de EXP/HP do cliente não atualiza.
        assert_eq!(cabecalho.id, ID_SCENE);
        assert_eq!(cabecalho.id, wyd_protocol::ESCENE_FIELD);
        assert_eq!(cabecalho.message_type, Opcode::CreateMob.wire());
        assert_eq!(usize::from(cabecalho.size), CreateMob::WIRE_LEN);
        assert_eq!(frame.len() - HEADER_SIZE, CreateMob::WIRE_LEN - HEADER_SIZE);

        let pacote = CreateMob::decode(&frame).expect("corpo do CreateMob");
        assert_eq!(
            pacote.mob_id, 9,
            "o id da criatura vai em MobID, não no HEADER"
        );
        assert_eq!(pacote.name_bytes(), b"Buffy");
        assert_eq!(pacote.equip[0], 831, "aparece pelo s_Index do equipamento");
        assert_eq!(pacote.guild, 4);
        assert_eq!((pacote.level, pacote.max_hp, pacote.hp), (42, 90, 90));
        assert_eq!(pacote.merchant, 0, "jogador não é NPC de loja");
        assert_eq!(
            pacote.create_type,
            CreateMob::CREATE_JUST_ENTERED,
            "o dono da sessão entra como `CreateType` 2"
        );
    }

    /// BUG-01: ponto exato do pacote de recusa (sem mudar o fio).
    ///
    /// Congela o comportamento atual — cabeçalho (12 B) + trailer de 1 B com o
    /// motivo — para que qualquer mudança de `Size` (ex.: 13 → 12) quebre aqui
    /// primeiro e exija prova RE/captura 08-6 antes de ir ao cliente 7662.
    #[test]
    fn recusa_leva_size_13_com_o_motivo_no_trailer() {
        use wyd_protocol::motivo::CodigoRecusa;
        let casos = [
            (Opcode::NewAccountFail, CodigoRecusa::Senha),
            (Opcode::AccountSecureFail, CodigoRecusa::PinIncorreto),
            (Opcode::CharacterLoginFail, CodigoRecusa::SelecaoInvalida),
        ];
        for (opcode, motivo) in casos {
            let (mut servidor, mut cliente) = par();
            assert!(reply_recusa(&mut servidor, 11, opcode, motivo));
            let (cabecalho, frame) = ler_frame(&mut cliente);
            assert_eq!(
                usize::from(cabecalho.size),
                HEADER_SIZE + 1,
                "recusa {opcode:?}: Size sai 13 até prova RE em contrário"
            );
            assert_eq!(cabecalho.message_type, opcode.wire());
            assert_eq!(frame.len(), HEADER_SIZE + 1);
            assert_eq!(
                frame[HEADER_SIZE],
                motivo.byte(),
                "recusa {opcode:?}: trailer carrega {motivo:?}"
            );
            assert_eq!(CodigoRecusa::from_byte(Some(frame[HEADER_SIZE])), motivo);
        }
    }

    #[test]
    fn remove_mob_leva_o_id_da_cena_no_header_e_o_tipo_no_corpo() {
        let (mut servidor, mut cliente) = par();
        assert!(send_remove_mob(&mut servidor, 3, 9, RemoveMob::LOGOUT));

        let (cabecalho, frame) = ler_frame(&mut cliente);
        assert_eq!(cabecalho.id, 9, "RemoveMob usa o id da entidade no HEADER");
        assert_eq!(cabecalho.message_type, Opcode::RemoveMob.wire());
        let pacote = RemoveMob::decode(&frame).expect("corpo do RemoveMob");
        assert_eq!(pacote.remove_type, RemoveMob::LOGOUT);
    }

    #[test]
    fn create_mob_de_npc_vai_como_entrada_normal() {
        let (mut servidor, mut cliente) = par();
        assert!(send_create_mob(
            &mut servidor,
            3,
            &mob(1_007, b"Bardes", false),
            CreateMob::CREATE_NORMAL
        ));
        let (_, frame) = ler_frame(&mut cliente);
        let pacote = CreateMob::decode(&frame).expect("corpo do CreateMob");
        assert_eq!(pacote.mob_id, 1_007);
        assert_eq!(pacote.create_type, CreateMob::CREATE_NORMAL);
    }
}
