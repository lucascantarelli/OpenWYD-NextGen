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
//! Depois do `CNF` a conexão **continua** na seleção de personagem (`FASE-08-2`): o laço
//! `serve_selection` consome cada frame inteiro (sem dessincronizar o `Size` seguinte), renova
//! a cerca e atende `_MSG_CreateCharacter`/`_MSG_DeleteCharacter`/`_MSG_CharacterLogin` por
//! `characters.rs`. Pacote que este processo ainda não serve (chat, `_MSG_Ping`, mundo) é
//! **tolerado** — antes, o primeiro pacote encerrava a sessão, que é exatamente o defeito
//! corrigido aqui. Só framing inválido e falha de I/O encerram a conexão.
//!
//! O que **não** existe ainda: autenticação de conta (`_MSG_AccountLogin` só tem a versão
//! validada) e pacotes de cena (`ESCENE_FIELD`, FASE-03/08-6).

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
use wyd_protocol::payload::ui::MessagePanel;
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
    /// Encarnado: entidade do mundo (FASE-03-8), quando o processo tem mundo, mais o
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
/// resíduo declarado na FASE-03-9.)
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

/// Resultado de um handshake aceito (o que ficou provado sobre a conexão).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct HandshakeOk {
    /// Versão declarada pelo cliente no login.
    pub client_version: i32,
    /// Chave de ofuscação usada na resposta (a mesma do frame de entrada).
    pub key_word: u8,
    /// Sessão aberta no banco (`None` quando o processo roda sem cerca, ex.: teste unitário).
    pub session_id: Option<i64>,
}

/// Falhas de handshake — todas viram log estruturado + fechamento da conexão.
#[derive(Debug)]
pub enum HandshakeError {
    /// Primeiros 4 bytes diferentes de [`INITCODE`].
    InitCode { received: u32 },
    /// `Size` fora de `[12, 8192]` ou frame incompleto.
    Frame(String),
    /// Checksum estrito divergiu.
    Checksum { computed: u8, declared: u8 },
    /// Opcode que este processo ainda não atende.
    OpcodeNotServed { opcode: Opcode },
    /// Versão de cliente recusada pelo gate (ADR-0020).
    Version { received: i32, expected: u16 },
    /// Cerca de sessão recusou (conta já em jogo) ou o banco não respondeu.
    Session(String),
    /// Senha de login recusada (ou banco indisponível): sem `CNF`, sem sessão.
    Auth(String),
    /// O cliente não enviou `_MSG_AccountSecure` no prazo: conexão fechada sem sessão.
    PinTimeout,
    /// PIN recusado: `_MSG_AccountSecureFail` enviado e conexão fechada sem sessão.
    Pin(String),
    /// I/O: cliente saiu no meio do handshake.
    Io(String),
}

impl core::fmt::Display for HandshakeError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::InitCode { received } => {
                write!(f, "INITCODE divergente: {received:#010x}")
            }
            Self::Frame(motivo) => write!(f, "frame inválido: {motivo}"),
            Self::Checksum { computed, declared } => {
                write!(
                    f,
                    "checksum divergiu: calculado {computed}, declarado {declared}"
                )
            }
            Self::OpcodeNotServed { opcode } => {
                write!(f, "opcode sem handler: {}", opcode.const_name())
            }
            Self::Version { received, expected } => {
                write!(
                    f,
                    "versão de cliente {received} fora do gate (esperado {expected})"
                )
            }
            Self::Session(motivo) => write!(f, "sessão: {motivo}"),
            Self::Auth(motivo) => write!(f, "autenticação: {motivo}"),
            Self::PinTimeout => write!(f, "cliente não enviou a senha numérica"),
            Self::Pin(motivo) => write!(f, "senha numérica: {motivo}"),
            Self::Io(motivo) => write!(f, "io: {motivo}"),
        }
    }
}

impl std::error::Error for HandshakeError {}

impl From<io::Error> for HandshakeError {
    fn from(error: io::Error) -> Self {
        Self::Io(error.to_string())
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

/// Limite de conexões simultâneas por origem (`per_ip = 0` desliga).
#[derive(Debug, Clone, Copy)]
pub struct ConnectionLimit {
    pub per_ip: usize,
}

impl ConnectionLimit {
    #[must_use]
    pub fn new(per_ip: usize) -> Self {
        Self { per_ip }
    }

    /// `WYD_MAX_CONNECTIONS_PER_IP` (padrão 6; `0` desliga). Multiboxing custa 4–5.
    #[must_use]
    pub fn from_env() -> Self {
        let per_ip = std::env::var("WYD_MAX_CONNECTIONS_PER_IP")
            .ok()
            .and_then(|valor| valor.trim().parse().ok())
            .unwrap_or(6);
        Self { per_ip }
    }
}

/// Vaga de conexão de um IP: libera a contagem no `Drop`, inclusive em pânico da thread.
struct IpSlot {
    ativos: Option<Arc<std::sync::Mutex<std::collections::HashMap<std::net::IpAddr, usize>>>>,
    ip: std::net::IpAddr,
}

impl IpSlot {
    fn reservar(
        ativos: &Arc<std::sync::Mutex<std::collections::HashMap<std::net::IpAddr, usize>>>,
        ip: std::net::IpAddr,
        maximo: usize,
    ) -> Option<Self> {
        if maximo == 0 {
            return Some(Self { ativos: None, ip });
        }
        let mut mapa = ativos.lock().unwrap_or_else(|e| e.into_inner());
        let atual = mapa.entry(ip).or_insert(0);
        if *atual >= maximo {
            return None;
        }
        *atual += 1;
        Some(Self {
            ativos: Some(Arc::clone(ativos)),
            ip,
        })
    }
}

impl Drop for IpSlot {
    fn drop(&mut self) {
        let Some(ativos) = &self.ativos else {
            return;
        };
        let mut mapa = ativos.lock().unwrap_or_else(|e| e.into_inner());
        if let Some(atual) = mapa.get_mut(&self.ip) {
            *atual = atual.saturating_sub(1);
            if *atual == 0 {
                mapa.remove(&self.ip);
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

fn handshake(
    stream: &mut TcpStream,
    world: &World,
    gate: VersionGate,
    auth: Option<&Authenticator>,
    selection: Option<&SelectionStore>,
) -> Result<(HandshakeOk, String), HandshakeError> {
    let mut init = [0u8; 4];
    stream.read_exact(&mut init)?;
    let received = u32::from_le_bytes(init);
    if received != INITCODE {
        NET.packet_rejected("ip");
        return Err(HandshakeError::InitCode { received });
    }

    let mut head = [0u8; HEADER_SIZE];
    stream.read_exact(&mut head)?;
    let header = Header::decode(&head).map_err(|error| HandshakeError::Frame(error.to_string()))?;
    // Só o HEADER chegou até aqui: o tamanho vem do próprio cabeçalho (validado), e
    // `split_frame` exige o frame inteiro — por isso `frame_len`, não `split_frame`.
    let declared = header
        .frame_len()
        .map_err(|error| HandshakeError::Frame(error.to_string()))?;
    let mut frame = vec![0u8; declared];
    frame[..HEADER_SIZE].copy_from_slice(&head);
    stream.read_exact(&mut frame[HEADER_SIZE..])?;

    let computed =
        decode_in_place(&mut frame).map_err(|error| HandshakeError::Frame(error.to_string()))?;
    checksum::verify(computed, frame[3], ChecksumPolicy::Strict).map_err(|_| {
        HandshakeError::Checksum {
            computed,
            declared: frame[3],
        }
    })?;

    let valid = inspect(&frame).map_err(|error| HandshakeError::Frame(error.to_string()))?;
    if valid.opcode != Opcode::AccountLogin {
        NET.packet_rejected("timeout");
        return Err(HandshakeError::OpcodeNotServed {
            opcode: valid.opcode,
        });
    }

    let login =
        AccountLogin::decode(&frame).map_err(|error| HandshakeError::Frame(error.to_string()))?;
    gate.check(login.client_version, &NET)
        .map_err(|_| HandshakeError::Version {
            received: login.client_version,
            expected: gate.expected(),
        })?;
    let conta = String::from_utf8_lossy(login.account_name_bytes()).to_string();
    // Senha da conta **antes** de qualquer confirmação: senha errada não recebe `CNF` e não
    // chega perto da cerca. Sem porta de autenticação (teste de framing/gate) o passo é pulado.
    if let Some(auth) = auth {
        if let Err(error) = auth.verify_password(&conta, &login.account_password) {
            NET.packet_rejected("auth");
            log(
                "warn",
                "login",
                &format!("result=refused account={conta} reason={error}"),
            );
            // Senha/versão recusada: mesmo ponto único das demais recusas, com o código
            // próprio no trailer (`NewAccountFail` é o `*Fail` do login).
            let _ = reply_recusa(
                stream,
                header.key_word,
                Opcode::NewAccountFail,
                wyd_protocol::motivo::CodigoRecusa::Senha,
            );
            // O build 7662 **não** trata `_MSG_NewAccountFail` (0x0117): sem um pacote que ele
            // desenhe, senha errada não mostra nada na tela. `_MSG_MessagePanel` (0x0101) é o
            // único texto livre que o roteador do cliente reconhece; vai junto, com o layout
            // portado. Se o cliente ignorar fora do jogo, nada muda para ele — o `*Fail`
            // continua sendo o pacote de contrato.
            let _ = reply_panel(stream, header.key_word, "Senha ou conta incorreta");
            return Err(HandshakeError::Auth(error.to_string()));
        }
    }

    NET.packet_accepted();
    NET.handshake_opened();
    // Linha única e auto-descritiva do login aceito (a sessão abre só depois do PIN).
    log(
        "info",
        "login",
        &format!(
            "result=password_ok account={conta} version={} channels={}",
            login.client_version,
            world.game_channel_count()
        ),
    );

    // Corpo com os personagens da conta (`STRUCT_SELCHAR`), o cofre (`Cargo[128]`) e o
    // `Coin`: tudo do banco. Falha de leitura devolve lista vazia/zero — o cliente vê "sem
    // personagem", nunca dado inventado.
    let snapshot = match selection.map(|store| store.login_snapshot(&conta)) {
        Some(Ok(snapshot)) => snapshot,
        Some(Err(error)) => {
            log(
                "warn",
                "login",
                &format!("result=empty_roster account={conta} reason={error}"),
            );
            LoginSnapshot::default()
        }
        None => LoginSnapshot::default(),
    };
    let cargo = cargo_items(&snapshot.vault);
    let coin = i32::try_from(snapshot.coin).unwrap_or(i32::MAX);
    let corpo = account_login_body(&snapshot.roster, &cargo, coin, &conta);
    reply_with_body(
        stream,
        header.key_word,
        ID_SELCHAR,
        Opcode::CNFAccountLogin,
        &corpo,
    )
    .then_some(())
    .ok_or_else(|| HandshakeError::Io("falha ao escrever _MSG_CNFAccountLogin".to_string()))?;

    Ok((
        HandshakeOk {
            client_version: login.client_version,
            key_word: header.key_word,
            session_id: None,
        },
        conta,
    ))
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

fn authorize_pin(
    stream: &mut TcpStream,
    account: &str,
    auth: Option<&Authenticator>,
    sessions: Option<&SessionGate>,
    version: i32,
) -> Result<PinStep, HandshakeError> {
    let Some(auth) = auth else {
        return Ok((open_session(account, sessions, version)?, None));
    };
    // PIN é segunda senha, não cadastro obrigatório: conta sem PIN cadastrado vai direto
    // para a seleção, porque o cliente 7662 não manda `_MSG_AccountSecure` nesse caso.
    // Exigir a etapa aqui recusava todo personagem novo com "Falha ao conectar".
    match auth.pin_registered(account) {
        Ok(true) => {}
        Ok(false) => {
            log(
                "info",
                "pin",
                &format!("result=skip account={account} reason=not_registered"),
            );
            return Ok((open_session(account, sessions, version)?, None));
        }
        Err(error) => {
            log(
                "warn",
                "pin",
                &format!("result=refused account={account} reason={error}"),
            );
            return Err(HandshakeError::Pin(error.to_string()));
        }
    }
    let _ = stream.set_read_timeout(Some(HANDSHAKE_TIMEOUT));
    // Passo 1 da troca de PIN aceito nesta conexão; o 2 só vale com ele armado.
    let mut troca_armada = false;
    loop {
        let mut head = [0u8; HEADER_SIZE];
        stream
            .read_exact(&mut head)
            .map_err(|_| HandshakeError::PinTimeout)?;
        let Some(mut frame) = read_frame(stream, &head, account) else {
            return Err(HandshakeError::PinTimeout);
        };
        let computed = decode_in_place(&mut frame)
            .map_err(|error| HandshakeError::Frame(error.to_string()))?;
        checksum::verify(computed, frame[3], ChecksumPolicy::Strict).map_err(|_| {
            HandshakeError::Checksum {
                computed,
                declared: frame[3],
            }
        })?;
        let valid = inspect(&frame).map_err(|error| HandshakeError::Frame(error.to_string()))?;
        match valid.opcode {
            Opcode::AccountSecure => {
                let pacote = AccountSecure::decode(&frame)
                    .map_err(|error| HandshakeError::Pin(error.to_string()))?;
                // Troca de PIN em duas etapas no mesmo `0x0FDE` (decisão nossa, ver
                // `docs/wyd/senha-numerica.md`): `change_numeric = 1` confere o PIN atual e arma;
                // `2` grava o PIN novo. A sessão só abre depois do passo 2 (ou do login normal).
                let mut armou = false;
                let resultado = match pacote.change_numeric {
                    1 if !troca_armada => match auth.secure_pin(account, &pacote.numeric_token) {
                        Ok(PinOutcome::Registered) => Ok(PinOutcome::Registered),
                        Ok(outro) => {
                            troca_armada = true;
                            armou = true;
                            Ok(outro)
                        }
                        Err(error) => Err(error),
                    },
                    2 if troca_armada => {
                        troca_armada = false;
                        auth.change_pin(account, &pacote.numeric_token)
                    }
                    2 => {
                        log(
                            "warn",
                            "pin",
                            &format!("result=refused account={account} reason=change_not_armed"),
                        );
                        let _ = reply_recusa(
                            stream,
                            head[2],
                            Opcode::AccountSecureFail,
                            wyd_protocol::motivo::CodigoRecusa::PinIncorreto,
                        );
                        return Err(HandshakeError::Pin("troca sem PIN atual".to_string()));
                    }
                    _ => auth.secure_pin(account, &pacote.numeric_token),
                };
                if armou {
                    // Passo 1 aceito: responde e fica esperando o passo 2; nada de sessão ainda.
                    if !reply(stream, head[2], Opcode::AccountSecure) {
                        return Err(HandshakeError::Io(
                            "falha ao escrever _MSG_AccountSecure".to_string(),
                        ));
                    }
                    continue;
                }
                match resultado {
                    Ok(resultado) => {
                        // Log do resultado, nunca do PIN.
                        log(
                            "info",
                            "pin",
                            &format!(
                                "result=ok account={account} mode={resultado:?} change={}",
                                pacote.change_numeric
                            ),
                        );
                        // Sessão só abre com o PIN aceito; conflito de cerca também responde
                        // `0x0FDF` para o cliente não achar que entrou.
                        // Gancho de operação/teste: força `ContaEmUso` sem depender de uma
                        // segunda sessão real; mesmo `reply_recusa` do caminho normal.
                        if std::env::var("WYD_TEST_FORCE_CONTA_EM_USO").is_ok() {
                            let _ = reply_recusa(
                                stream,
                                head[2],
                                Opcode::AccountSecureFail,
                                wyd_protocol::motivo::CodigoRecusa::ContaEmUso,
                            );
                            return Err(HandshakeError::Session(
                                "conta em uso (forçado por WYD_TEST_FORCE_CONTA_EM_USO)"
                                    .to_string(),
                            ));
                        }
                        match open_session(account, sessions, version) {
                            Ok(lease) => {
                                if !reply(stream, head[2], Opcode::AccountSecure) {
                                    return Err(HandshakeError::Io(
                                        "falha ao escrever _MSG_AccountSecure".to_string(),
                                    ));
                                }
                                return Ok((lease, None));
                            }
                            Err(error) => {
                                let _ = reply_recusa(
                                    stream,
                                    head[2],
                                    Opcode::AccountSecureFail,
                                    wyd_protocol::motivo::CodigoRecusa::ContaEmUso,
                                );
                                return Err(error);
                            }
                        }
                    }
                    Err(error) => {
                        log(
                            "warn",
                            "pin",
                            &format!("result=refused account={account} reason={error}"),
                        );
                        let motivo = if error.to_string().to_lowercase().contains("bloque")
                            || error.to_string().to_lowercase().contains("aguarde")
                        {
                            wyd_protocol::motivo::CodigoRecusa::PinBloqueado
                        } else {
                            wyd_protocol::motivo::CodigoRecusa::PinIncorreto
                        };
                        let _ = reply_recusa(stream, head[2], Opcode::AccountSecureFail, motivo);
                        return Err(HandshakeError::Pin(error.to_string()));
                    }
                }
            }
            Opcode::CreateCharacter | Opcode::DeleteCharacter | Opcode::CharacterLogin => {
                // O cliente 7662 não repete o PIN ao entrar no canal: manda a seleção
                // direto. A senha já foi conferida no `_MSG_AccountLogin`, então a sessão
                // abre aqui e o pacote (já em claro) volta para o laço de seleção — o
                // jogador entra no mundo em vez de receber "Falha ao conectar".
                let lease = open_session(account, sessions, version)?;
                log(
                    "info",
                    "pin",
                    &format!("result=deferred account={account} reason=selection_first"),
                );
                return Ok((lease, Some((frame, head[2]))));
            }
            outro => log(
                "info",
                "packet",
                &format!(
                    "result=tolerated account={account} opcode={} wire={:#06x} stage=awaiting_pin",
                    outro.const_name(),
                    outro.wire()
                ),
            ),
        }
    }
}

/// Abre a sessão da conta (uma por grupo) ou recusa com o canal que já a tem.
fn open_session(
    account: &str,
    sessions: Option<&SessionGate>,
    version: i32,
) -> Result<Option<SessionLease>, HandshakeError> {
    let Some(cerca) = sessions else {
        return Ok(None);
    };
    match cerca.claim(account, version) {
        Ok(lease) => {
            log(
                "info",
                "session",
                &format!(
                    "result=open account={account} group={} channel={} id={}",
                    cerca.group_id(),
                    cerca.channel(),
                    lease.id()
                ),
            );
            NET.packet_accepted();
            Ok(Some(lease))
        }
        Err(error) => {
            // Lotação/manutenção (FASE-06-3) têm motivo próprio na métrica; a origem é
            // o próprio canal, que é quem recusa.
            let motivo = match &error {
                SessionError::Capacity { .. } => "canal_lotado",
                SessionError::Maintenance => "manutencao",
                _ => "session",
            };
            NET.packet_rejected(motivo);
            let origem = match &error {
                SessionError::Conflict { channel } => *channel,
                SessionError::Capacity { .. } | SessionError::Maintenance | SessionError::Db(_) => {
                    0
                }
            };
            log(
                "warn",
                "login",
                &format!(
                    "result=conflict account={account} group={} channel={} origin_channel={origem} reason={error}",
                    cerca.group_id(),
                    cerca.channel()
                ),
            );
            Err(HandshakeError::Session(error.to_string()))
        }
    }
}

/// Atende a conexão depois do `CNF` enquanto o cliente estiver do outro lado (FASE-08-2).
///
/// Cada iteração consome **o frame inteiro** antes de decidir: ler só o cabeçalho deixaria o
/// corpo no socket e o `Size` seguinte seria lido do meio do pacote anterior.
///
/// Ociosidade renova o sinal de vida até `MAX_IDLE_ROUNDS`; ociosidade além disso libera a
/// sessão. Qualquer falha de I/O (FIN, reset) também encerra — o `Drop` do `SessionLease`
/// libera a conta para outro canal.
fn serve_selection(
    stream: &mut TcpStream,
    account: &str,
    selection: Option<&SelectionStore>,
    lease: Option<&SessionLease>,
    idle: Duration,
    pendente: Option<Pendente>,
) -> Option<wyd_core::Entity> {
    let _ = stream.set_read_timeout(Some(SCENE_POLL));
    let mut stage = Stage::Selecting;
    let mut ultimo_pacote = Instant::now();
    let mut ultimo_renew = Instant::now();
    let mut chave = 0u8;
    // Pacote que o handshake já consumiu (seleção antes do PIN): processa agora, com o
    // frame em claro e o checksum já conferido, antes de esperar o próximo pacote.
    if let Some((frame, chave_pendente)) = pendente {
        chave = chave_pendente;
        let Ok(valid) = inspect(&frame) else {
            return stage.entity();
        };
        if let Some(lease) = lease {
            lease.renew();
        }
        if !dispatch(
            stream,
            &frame,
            chave,
            valid.opcode,
            Sessao { account },
            selection,
            &mut stage,
        ) {
            return stage.entity();
        }
    }
    loop {
        let mut head = [0u8; HEADER_SIZE];
        match stream.read_exact(&mut head) {
            Ok(()) => {
                ultimo_pacote = Instant::now();
                chave = head[2];
            }
            Err(error) if is_timeout(&error) => {
                if ultimo_renew.elapsed() >= RENEW_EVERY {
                    if let Some(lease) = lease {
                        lease.renew();
                    }
                    ultimo_renew = Instant::now();
                }
                // Recarga de configuração: o motor sobe a geração e a sessão emite o delta
                // (`CreateMob` do que nasceu, `RemoveMob` do que saiu) — sem reiniciar nada.
                if !refresh_scene(stream, chave, &mut stage, selection) {
                    return stage.entity();
                }
                if ultimo_pacote.elapsed() >= idle * MAX_IDLE_ROUNDS {
                    log(
                        "info",
                        "session",
                        &format!(
                            "result=idle account={account} stage={} rounds={MAX_IDLE_ROUNDS}",
                            stage.as_str()
                        ),
                    );
                    return stage.entity();
                }
                continue;
            }
            // Queda sem pacote de saída (FIN, reset): a sessão cai aqui também — o
            // `Drop` do `SessionLease` libera a conta e o `leave_world` do chamador
            // enfileira o save. O motivo `socket` distingue do `explicit` no log.
            Err(_) => {
                log(
                    "info",
                    "logout",
                    &format!(
                        "result=ok reason=socket account={account} stage={}",
                        stage.as_str()
                    ),
                );
                return stage.entity();
            }
        }
        let Some(frame) = read_frame(stream, &head, account) else {
            return stage.entity();
        };
        // O cliente está vivo: renovar antes de qualquer trabalho no banco.
        if let Some(lease) = lease {
            lease.renew();
        }
        let mut limpo = frame;
        let computed = match decode_in_place(&mut limpo) {
            Ok(computed) => computed,
            Err(error) => {
                // Frame já consumido por inteiro (sem dessincronizar): recusa o pacote e
                // mantém a sessão.
                NET.packet_rejected("checksum");
                log(
                    "warn",
                    "packet",
                    &format!("result=rejected account={account} reason={error}"),
                );
                continue;
            }
        };
        if checksum::verify(computed, limpo[3], ChecksumPolicy::Strict).is_err() {
            NET.packet_rejected("checksum");
            log(
                "warn",
                "packet",
                &format!("result=rejected account={account} reason=checksum"),
            );
            continue;
        }
        let valid = match inspect(&limpo) {
            Ok(valid) => valid,
            Err(error) => {
                NET.packet_rejected("frame");
                log(
                    "warn",
                    "packet",
                    &format!("result=rejected account={account} reason={error}"),
                );
                continue;
            }
        };
        if !dispatch(
            stream,
            &limpo,
            head[2],
            valid.opcode,
            Sessao { account },
            selection,
            &mut stage,
        ) {
            return stage.entity();
        }
    }
}

/// Lê o resto do frame declarado no cabeçalho. `None` = conexão encerrada ou `Size` inválido.
fn read_frame(stream: &mut TcpStream, head: &[u8; HEADER_SIZE], account: &str) -> Option<Vec<u8>> {
    let header = match Header::decode(head) {
        Ok(header) => header,
        Err(error) => {
            NET.packet_rejected("frame");
            log(
                "warn",
                "packet",
                &format!("result=rejected account={account} reason={error}"),
            );
            return None;
        }
    };
    let declared = match header.frame_len() {
        Ok(declared) => declared,
        Err(error) => {
            NET.packet_rejected("frame");
            log(
                "warn",
                "packet",
                &format!("result=rejected account={account} reason={error}"),
            );
            return None;
        }
    };
    let mut frame = vec![0u8; declared];
    frame[..HEADER_SIZE].copy_from_slice(head);
    stream.read_exact(&mut frame[HEADER_SIZE..]).ok()?;
    Some(frame)
}

/// Trata um pacote já desofuscado e íntegro. `false` = a escrita falhou, encerrar.
fn dispatch(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    opcode: Opcode,
    sessao: Sessao<'_>,
    selection: Option<&SelectionStore>,
    stage: &mut Stage,
) -> bool {
    let Sessao { account } = sessao;
    match opcode {
        // `0x0FDE` depois da seleção: conta sem PIN cadastrado pode receber o pacote
        // ainda assim (cliente que já conhece o fluxo). Confirma sem abrir sessão nova.
        Opcode::AccountSecure => {
            log(
                "info",
                "pin",
                &format!("result=ok account={account} mode=late"),
            );
            reply(stream, key_word, Opcode::AccountSecure)
        }
        Opcode::CreateCharacter => {
            let resultado = match selection {
                Some(store) => create_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok(detalhe) => {
                    log(
                        "info",
                        "character",
                        &format!("result=ok action=create account={account} {detalhe}"),
                    );
                    // O antigo responde `CNFNewCharacter` com a seleção inteira
                    // (`_MSG_DBCNFNewCharacter` → `MSG_CNFNewCharacter.sel`). O corpo é o
                    // `SELCHAR` provisório já enviado no login, agora com o personagem novo.
                    let roster = selection
                        .and_then(|store| store.roster_inventory(account).ok())
                        .unwrap_or_default();
                    let corpo = new_character_body(&roster);
                    reply_with_body(
                        stream,
                        key_word,
                        ID_NEW_CHARACTER,
                        Opcode::CNFNewCharacter,
                        &corpo,
                    )
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=create account={account} reason={error}"),
                    );
                    reply(stream, key_word, Opcode::NewCharacterFail)
                }
            }
        }
        Opcode::DeleteCharacter => {
            let resultado = match selection {
                Some(store) => delete_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok(detalhe) => {
                    log(
                        "info",
                        "character",
                        &format!("result=ok action=delete account={account} {detalhe}"),
                    );
                    // O legado devolve o roster inteiro (`MSG_CNFDeleteCharacter` carrega
                    // `STRUCT_SELCHAR`): é com ele que o cliente apaga o slot da tela.
                    let roster = selection
                        .and_then(|store| store.roster_inventory(account).ok())
                        .unwrap_or_default();
                    let corpo = delete_character_body(&roster);
                    reply_with_body(
                        stream,
                        key_word,
                        ID_DELETE_CHARACTER,
                        Opcode::CNFDeleteCharacter,
                        &corpo,
                    )
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=delete account={account} reason={error}"),
                    );
                    reply(stream, key_word, Opcode::DeleteCharacterFail)
                }
            }
        }
        Opcode::CharacterLogin => {
            // Gancho de operação/teste: força a recusa de zona sem mexer no caminho normal.
            // O código é escolhido aqui e sai pelo mesmo `reply_recusa` (ponto único).
            if std::env::var("WYD_TEST_FORCE_ZONA").is_ok() {
                return reply_recusa(
                    stream,
                    key_word,
                    Opcode::CharacterLoginFail,
                    wyd_protocol::motivo::CodigoRecusa::ZonaIndisponivel,
                );
            }
            let resultado = match selection {
                Some(store) => select_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok((detalhe, selected)) => {
                    let personagem = &selected.character;
                    // `STRUCT_MOB` do personagem escolhido: equipamento e mochila
                    // de `character_items`, nunca do template da classe. A montagem vem antes
                    // da entrada no mundo porque a mesma aparência vira `_MSG_CreateMob` na
                    // cena dos outros personagens do canal (FASE-03-9).
                    let mut mob = Mob::from_persisted(personagem, &selected.items);
                    // Atributos, pontos e `LearnedSkill` vêm do banco (`character_scores`
                    // + `character_learned_skill`), não do template da classe.
                    crate::characters::apply_progression(&mut mob, &selected.progression);
                    let skin = crate::world::ActorSkin {
                        name: mob.name,
                        equip: mob.equip.map(|item| u16::try_from(item.index).unwrap_or(0)),
                        guild: mob.guild,
                        attack_run: mob.current_score.attack_run,
                        direction: mob.current_score.direction,
                    };
                    // Encarna no mundo **antes** de confirmar: recusa do motor vira
                    // `CharacterLoginFail` e o personagem não fica pela metade. A cena de quem
                    // já está no canal recebe o `_MSG_CreateMob` na varredura seguinte.
                    let entrada = match selection {
                        Some(store) => store.enter_world(personagem, &skin),
                        None => Ok(None),
                    };
                    match entrada {
                        Ok(entrada) => {
                            let (entity, client_id) = match entrada {
                                Some(entrada) => (Some(entrada.entity), entrada.scene_id),
                                None => (None, 0),
                            };
                            *stage = Stage::InGame {
                                entity,
                                selected: Box::new(selected.clone()),
                                client_id,
                                scene: Scene::default(),
                            };
                            log(
                                "info",
                                "character",
                                &format!(
                                    "result=ok action=select account={account} {detalhe} entity={} client_id={client_id}",
                                    entity.map_or_else(
                                        || "none".to_string(),
                                        |e| e.index.to_string()
                                    )
                                ),
                            );
                            let corpo = character_login_body(
                                &mob,
                                u16::try_from(personagem.slot).unwrap_or(0),
                                client_id,
                                0,
                                &selected.short_skill,
                                account,
                            );
                            let enviado = reply_with_body(
                                stream,
                                key_word,
                                ID_SCENE,
                                Opcode::CNFCharacterLogin,
                                &corpo,
                            )
                                // O legado manda o `UpdateScore` do próprio personagem
                                // logo depois do `CNF` e **antes** da cena
                                // (`ProcessDBMessage` → `enterWorldView` → `sendScoreSelf`,
                                // porte Go `handler/character.go`): é o que pinta as barras
                                // de HP/MP/EXP na entrada. Sem ele o cliente fica com o
                                // score do blob de login.
                                && reply_with_body(
                                    stream,
                                    key_word,
                                    client_id,
                                    Opcode::UpdateScore,
                                    &crate::personagens::update_score_body(&mob),
                                )
                                // `_MSG_UpdateEtc` é quem carrega `Coin@28` (o `SendEtc` do
                                // legado, `SendFunc.cpp`): o `STRUCT_MOB` do `CNF` traz o gold
                                // no blob, mas a moeda que o cliente desenha é a do `UpdateEtc`.
                                // Sem esta metade do par o personagem entra com 5 000 000 no
                                // banco (Coin do `BaseMob`) e mostra 0 de ouro na tela.
                                && reply_with_body(
                                    stream,
                                    key_word,
                                    client_id,
                                    Opcode::UpdateEtc,
                                    &crate::personagens::update_etc_body(&mob),
                                );
                            // Cena logo depois do `CNF`: todo NPC materializado pela revisão
                            // vigente de `npc_definition` chega como `_MSG_CreateMob` com
                            // `HEADER.ID = ESCENE_FIELD` (o id vai em `MobID`).
                            enviado && refresh_scene(stream, key_word, stage, selection)
                        }
                        Err(error) => {
                            log(
                                "warn",
                                "character",
                                &format!(
                                    "result=refused action=enter account={account} reason={error}"
                                ),
                            );
                            reply_recusa(
                                stream,
                                key_word,
                                Opcode::CharacterLoginFail,
                                wyd_protocol::motivo::CodigoRecusa::ZonaIndisponivel,
                            )
                        }
                    }
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=select account={account} reason={error}"),
                    );
                    let motivo = if error.to_string().contains("payload") {
                        wyd_protocol::motivo::CodigoRecusa::SelecaoInvalida
                    } else {
                        wyd_protocol::motivo::CodigoRecusa::PersonagemIndisponivel
                    };
                    reply_recusa(stream, key_word, Opcode::CharacterLoginFail, motivo)
                }
            }
        }
        // `_MSG_CharacterLogout` (0x0215, C→J): o cliente pediu para sair (voltar à seleção ou
        // fechar a janela). Responder `CNFCharacterLogout` e encerrar aqui libera a sessão e
        // enfileira o save **na hora** (`leave_world` + `Drop` do `SessionLease`), sem esperar
        // o timeout de ociosidade — era o defeito de "cliente fechado continua online".
        Opcode::CharacterLogout => {
            log(
                "info",
                "logout",
                &format!(
                    "result=ok reason=explicit account={account} stage={}",
                    stage.as_str()
                ),
            );
            reply(stream, key_word, Opcode::CNFCharacterLogout);
            false
        }
        // `_MSG_MessageWhisper` (0x0334, C→J) é o canal do `/getout`, que no legado zera a
        // cidadania (`_MSG_MessageWhisper.cpp:57-64`). O **corpo** do sussurro não tem layout
        // provado na árvore (invariante 5): sem captura do cliente 7662 não se lê comando nem
        // nome, e tratar o pacote como `/getout` seria inventar layout. Tolerar e seguir deixa
        // a sessão viva; a cidadania, derivada do canal, não teria o que zerar hoje.
        Opcode::MessageWhisper => {
            log(
                "info",
                "citizen",
                &format!(
                    "result=tolerated action=getout account={account} reason=layout_unverified stage={}",
                    stage.as_str()
                ),
            );
            true
        }
        // `_MSG_UseItem` de refino: `SourType/SourPos` = pó, `DestType/DestPos` = alvo
        // (`Basedef.h:146-148`; `_MSG_UseItem.cpp:42-152`). O legado consome o pó, aplica
        // `BASE_SetItemSanc` e devolve `SendItem` (0x0182) com o alvo; aqui a taxa vem
        // do snapshot vivo, a piedade da tabela `refine_pity` e o inventário é gravado
        // antes de a sessão passar a ver o novo estado.
        Opcode::UseItem => {
            let (
                Stage::InGame {
                    selected,
                    client_id,
                    ..
                },
                Some(store),
            ) = (&mut *stage, selection)
            else {
                log(
                    "warn",
                    "use_item",
                    &format!("result=refused account={account} reason=not_in_game"),
                );
                return true;
            };
            let packet = match UseItem::decode(frame) {
                Ok(packet) => packet,
                Err(error) => {
                    log(
                        "warn",
                        "use_item",
                        &format!("result=refused account={account} reason=layout:{error}"),
                    );
                    return true;
                }
            };
            // Sorteio do processo (não do cliente): duas chamadas de 64 bits
            // independentes, como `rand()%100` e `rand()%4` no legado.
            let seed = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_nanos() as u64)
                .unwrap_or(1);
            let mut die = XorShift64::from_seed(seed);
            let rolls = UseItemRolls {
                roll: (die.next_u64() % 100) as u8,
                pity_roll: (die.next_u64() % 4) as u8,
            };
            let client_id = *client_id;
            match store.use_item(selected.as_mut(), &packet, rolls) {
                Ok(outcome) => {
                    log(
                        "info",
                        "use_item",
                        &format!(
                            "result=refined account={account} powder={:?} slot={} target={} from=+{} to=+{} rate_bps={} success={} destroyed={} powder_left={}",
                            outcome.powder,
                            outcome.source_slot,
                            outcome.target_slot,
                            outcome.resolution.from_level,
                            outcome.resolution.to_level,
                            outcome.resolution.rate_bps,
                            outcome.resolution.success,
                            outcome.destroyed,
                            outcome.powder_left
                        ),
                    );
                    // Alvo atualizado com o par EF_SANC no fio; destruído vira slot vazio
                    // (`sIndex` 0 limpa no cliente). Recusa não emite nada: o cliente já
                    // mostra o estado correto, pois o servidor nada mudou.
                    let quadro = selected
                        .items
                        .iter()
                        .find(|linha| linha.slot == outcome.target_slot)
                        .map(|linha| {
                            Item::from_row_with_pity(linha, outcome.resolution.next_pity).to_bytes()
                        })
                        .unwrap_or_default();
                    reply_send_item(
                        stream,
                        key_word,
                        client_id,
                        packet.dest_type,
                        packet.dest_pos,
                        quadro,
                    )
                }
                Err(error) => {
                    log(
                        "warn",
                        "use_item",
                        &format!(
                            "result=refused account={account} sour=({}, {}) dest=({}, {}) reason={error}",
                            packet.sour_type,
                            packet.sour_pos,
                            packet.dest_type,
                            packet.dest_pos
                        ),
                    );
                    true
                }
            }
        }
        // `_MSG_ApplyBonus`: distribuir ponto de atributo/skill. Responde `UpdateEtc` + `UpdateScore`
        // (o `SendEtc`/`SendScore` do legado) com o estado gravado no banco.
        Opcode::ApplyBonus => {
            let (client_id, aplicado) = match (&*stage, selection) {
                (
                    Stage::InGame {
                        selected,
                        client_id,
                        ..
                    },
                    Some(store),
                ) => match ApplyBonus::decode(frame) {
                    Ok(packet) => match store.apply_bonus(selected, &packet) {
                        Ok(aplicado) => {
                            log(
                                "info",
                                "bonus",
                                &format!(
                                    "result=ok account={account} bonus_type={} detail={} pontos={} skill_bonus={} learned={:#08x}",
                                    packet.bonus_type,
                                    packet.detail,
                                    aplicado.progression.score_bonus,
                                    aplicado.progression.skill_bonus,
                                    aplicado.progression.learned_skill
                                ),
                            );
                            (*client_id, aplicado)
                        }
                        Err(error) => {
                            log(
                                "warn",
                                "bonus",
                                &format!(
                                    "result=refused account={account} bonus_type={} detail={} reason={error}",
                                    packet.bonus_type, packet.detail
                                ),
                            );
                            return true;
                        }
                    },
                    Err(error) => {
                        log(
                            "warn",
                            "bonus",
                            &format!("result=refused account={account} reason=layout:{error}"),
                        );
                        return true;
                    }
                },
                _ => {
                    log(
                        "warn",
                        "bonus",
                        &format!("result=refused account={account} reason=not_in_game"),
                    );
                    return true;
                }
            };
            send_progress(stream, key_word, client_id, &aplicado.mob)
        }
        // `_MSG_Attack`/`_MSG_AttackOne`/`_MSG_AttackTwo`: mesmo corpo, aridade diferente.
        // O cliente aponta o alvo (x/y); o dano é do servidor e a EXP do template só entra
        // no abate, via `SelectionStore::attack` → `gain_exp` (multiplicador do snapshot).
        Opcode::AttackOne => handle_attack::<1>(stream, frame, key_word, account, stage, selection),
        Opcode::AttackTwo => handle_attack::<2>(stream, frame, key_word, account, stage, selection),
        Opcode::Attack => handle_attack::<13>(stream, frame, key_word, account, stage, selection),
        // `_MSG_Action` de NPC: o clique vira diálogo com o dado de `npc_definition`.
        Opcode::Action => handle_action(stream, frame, key_word, account, stage, selection),
        // `_MSG_Quest` (0x028B): treinadores do Campo de Treinamento (FASE-03-18).
        Opcode::Quest => handle_quest(stream, frame, key_word, account, stage, selection),
        // `_MSG_REQShopList` (0x027B): catálogo do mercador clicado (`npc_shop_item`).
        Opcode::REQShopList => handle_shop_list(stream, frame, key_word, account, stage, selection),
        // `_MSG_Buy` (0x0379): compra em loja de NPC (preço e gold são do servidor).
        Opcode::Buy => handle_buy(stream, frame, key_word, account, stage, selection),
        // Sem layout portado (UNVERIFIED) ou fora do escopo deste processo: tolerar, registrar
        // e seguir. Fechar a conexão aqui era o defeito que derrubava a sessão.
        outro => {
            log(
                "info",
                "packet",
                &format!(
                    "result=tolerated account={account} opcode={} wire={:#06x} stage={}",
                    outro.const_name(),
                    outro.wire(),
                    stage.as_str()
                ),
            );
            true
        }
    }
}

/// `_MSG_Attack*`: golpe corpo a corpo server-authoritative e EXP real no abate.
///
/// O `current_exp` que vem no pacote é ignorado (cliente não é autoridade); o valor enviado
/// de volta é o do `Mob` gravado. `UpdateEtc`+`UpdateScore` são o par que o cliente usa para
/// redesenhar a barra de EXP/HP.
fn handle_attack<const N: usize>(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            selected,
            entity: Some(entity),
            client_id,
            scene,
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "attack",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Attack::<N>::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "attack",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let alvo = packet.dam.iter().find(|alvo| alvo.target_id != 0);
    let (alvo_id, dano_cliente) = alvo.map_or((0, 0), |alvo| (alvo.target_id, alvo.damage)); // EXP do cliente: nunca creditada.
    let _ = &packet.current_exp;
    // Sorteio do processo (nunca do cliente): o loot do abate sai deste `die` e a seed vai
    // para o log, então o kill é reproduzível a partir da tabela do snapshot.
    let seed = loot_seed();
    let mut die = XorShift64::from_seed(seed);
    match store.attack(
        selected.as_ref(),
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
        now_ms(),
        &mut die,
    ) {
        Ok(Some(report)) => {
            log(
                "info",
                "attack",
                &format!(
                    "result=hit account={account} npc={} at=({}, {}) damage={} hp_left={} killed={} exp_base={} exp_granted={} levels={} client_damage={alvo_id}/{dano_cliente}",
                    report.target_npc,
                    packet.target_x,
                    packet.target_y,
                    report.damage,
                    report.remaining_hp,
                    report.killed,
                    report.base_exp,
                    report.granted_exp,
                    report.levels
                ),
            );
            let enviado = match &report.mob {
                Some(mob) => send_progress(stream, key_word, *client_id, mob),
                None => true,
            };
            if !enviado {
                return false;
            }
            if !report.killed {
                return true;
            }
            // Morte: o abate já saiu do motor (o mundo não guarda cadáver). O cliente recebe
            // `_MSG_RemoveMob` com `RemoveType = 1` (morte) e a vista local perde o id antes
            // do próximo delta, senão o poll de cena repetiria a remoção.
            scene.mobs.remove(&report.target_scene);
            log(
                "info",
                "death",
                &format!(
                    "result=ok account={account} npc={} template={} exp_base={} exp_granted={} levels={} at=({}, {}) seed={seed}",
                    report.target_npc,
                    report.template,
                    report.base_exp,
                    report.granted_exp,
                    report.levels,
                    packet.target_x,
                    packet.target_y
                ),
            );
            let mut envio_ok = true;
            match &report.loot {
                Some(Ok(loot)) => {
                    log(
                        "info",
                        "loot",
                        &format!(
                            "result=resolved account={account} npc={} template={} rolls={:?} drops={:?}",
                            report.target_npc, loot.template, loot.rolls, loot.drops
                        ),
                    );
                    // Entrega no inventário: empilha/ocupa slot, grava e avisa o cliente por
                    // `_MSG_SendItem` (o legado manda o slot entregue, não a mochila inteira).
                    match store.grant_loot(selected, &loot.drops) {
                        Ok(slots) => {
                            log(
                                "info",
                                "loot",
                                &format!(
                                    "result=inventory account={account} npc={} slots={slots:?}",
                                    report.target_npc
                                ),
                            );
                            for slot in slots {
                                let Some(linha) =
                                    selected.items.iter().find(|linha| linha.slot == slot)
                                else {
                                    continue;
                                };
                                let item = Item::from_row(linha).to_bytes();
                                if !reply_send_item(
                                    stream,
                                    key_word,
                                    *client_id,
                                    0,
                                    i32::try_from(slot).unwrap_or(0),
                                    item,
                                ) {
                                    envio_ok = false;
                                    break;
                                }
                            }
                        }
                        Err(error) => log(
                            "warn",
                            "loot",
                            &format!(
                                "result=refused account={account} npc={} reason=inventory:{error}",
                                report.target_npc
                            ),
                        ),
                    }
                }
                Some(Err(error)) => log(
                    "warn",
                    "loot",
                    &format!(
                        "result=refused account={account} npc={} template={} reason={error}",
                        report.target_npc, report.template
                    ),
                ),
                None => log(
                    "warn",
                    "loot",
                    &format!(
                        "result=skipped account={account} npc={} template={} reason=sem_resolvedor",
                        report.target_npc, report.template
                    ),
                ),
            }
            if !envio_ok {
                return false;
            }
            send_remove_mob(stream, key_word, report.target_scene, RemoveMob::DEATH)
        }
        Ok(None) => {
            log(
                "info",
                "attack",
                &format!(
                    "result=no_target account={account} at=({}, {}) npc={alvo_id}",
                    packet.target_x, packet.target_y
                ),
            );
            true
        }
        Err(error) => {
            log(
                "warn",
                "attack",
                &format!("result=refused account={account} reason={error}"),
            );
            true
        }
    }
}

/// `_MSG_Quest` (0x028B): etapa do Campo de Treinamento no treinador clicado.
///
/// `Parm1` é o `MobID` do NPC na cena; o **merchant** vem do mundo (`npc_definition`), não do
/// pacote, e o estágio/prova/recompensa vêm do motor puro (`gameplay::quest_engage`). O
/// sorteio do treinador 4 é do processo (`loot_seed`), nunca do cliente.
/// `_MSG_REQShopList` (0x027B): abre o catálogo do mercador clicado.
///
/// `Target` é o `MobID` da cena; o id de `npc_definition` sai do mundo vivo. A resposta é o
/// `_MSG_ShopList` (236 B) com `HEADER.ID = ESCENE_FIELD`, `ShopType = 0` e `Tax = 0`
/// (imposto de cidade não existe neste servidor — valor explícito, não estimado).
fn handle_shop_list(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (Stage::InGame { .. }, Some(store)) = (&mut *stage, selection) else {
        log(
            "warn",
            "shop",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let pedido = match ReqShopList::decode(frame) {
        Ok(pedido) => pedido,
        Err(error) => {
            log(
                "warn",
                "shop",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let Some((npc_id, merchant, nome)) = store.npc_shop(pedido.target) else {
        log(
            "warn",
            "shop",
            &format!(
                "result=refused account={account} npc={} reason=no_npc",
                pedido.target
            ),
        );
        return true;
    };
    if merchant <= 0 {
        log(
            "warn",
            "shop",
            &format!(
                "result=refused account={account} npc={npc_id} name={nome} reason=not_merchant"
            ),
        );
        return true;
    }
    let catalogo = store.shop_items(npc_id).unwrap_or_default();
    let mut lista = ShopList {
        shop_type: 0,
        items: [[0u8; 8]; MAX_SHOPLIST],
        tax: 0,
    };
    for regra in &catalogo {
        let Ok(posicao) = usize::try_from(regra.slot) else {
            continue;
        };
        if posicao >= MAX_SHOPLIST {
            continue;
        }
        let mut item = [0u8; 8];
        item[0..2].copy_from_slice(&i16::try_from(regra.item_index).unwrap_or(0).to_le_bytes());
        for (indice, par) in regra.effects.iter().enumerate() {
            item[2 + indice * 2] = par[0];
            item[3 + indice * 2] = par[1];
        }
        lista.items[posicao] = item;
    }
    let mut corpo = vec![0u8; ShopList::WIRE_LEN];
    if lista.encode_into(&mut corpo).is_err() {
        return false;
    }
    log(
        "info",
        "shop",
        &format!(
            "result=list account={account} npc={npc_id} name={nome} itens={} cheios={}",
            catalogo.len(),
            lista
                .items
                .iter()
                .filter(|item| item[0] != 0 || item[1] != 0)
                .count()
        ),
    );
    reply_with_body(
        stream,
        key_word,
        ID_SCENE,
        Opcode::ShopList,
        &corpo[HEADER_SIZE..],
    )
}

/// `_MSG_Buy` (0x0379): compra em loja de NPC.
///
/// Sucesso entrega `_MSG_SendItem` (mochila) + `UpdateEtc`/`UpdateScore` (gold novo).
/// Recusa não emite pacote de item: o log diz o motivo e o cliente continua com o estado
/// que o servidor tem.
fn handle_buy(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            selected,
            client_id,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "shop",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Buy::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "shop",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    match store.buy(
        selected.as_mut(),
        packet.target_id,
        packet.target_inven_pos,
        packet.coin,
    ) {
        Ok(compra) => {
            log(
                "info",
                "shop",
                &format!(
                    "result=bought account={account} npc={} pos={} coin={} slot={} item={}",
                    packet.target_id,
                    packet.target_inven_pos,
                    packet.coin,
                    compra.carry_pos,
                    i16::from_le_bytes([compra.item[0], compra.item[1]])
                ),
            );
            reply_send_item(
                stream,
                key_word,
                *client_id,
                1,
                i32::from(compra.carry_pos),
                compra.item,
            ) && send_progress(stream, key_word, *client_id, &compra.mob)
        }
        Err(error) => {
            log(
                "warn",
                "shop",
                &format!(
                    "result=refused account={account} npc={} pos={} coin={} reason={error}",
                    packet.target_id, packet.target_inven_pos, packet.coin
                ),
            );
            true
        }
    }
}

fn handle_quest(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            selected,
            client_id,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "quest",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Quest::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "quest",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let scene_id = u16::try_from(packet.npc_index).unwrap_or(0);
    let Some((merchant, nome)) = store.npc_merchant(scene_id) else {
        log(
            "warn",
            "quest",
            &format!("result=refused account={account} npc={scene_id} reason=no_npc"),
        );
        return true;
    };
    let roll = (loot_seed() % 3) as u8;
    match store.quest(selected.as_mut(), merchant, roll) {
        Ok(resultado) => {
            let recompensa = resultado.step.map_or_else(
                || "aceita".to_string(),
                |passo| format!("{:?}", passo.reward),
            );
            log(
                "info",
                "quest",
                &format!(
                    "result=ok account={account} npc={scene_id} name={nome} merchant={merchant} stage={} reward={recompensa} completed={} roll={roll} accepted={}",
                    resultado.state.stage,
                    resultado.state.completed,
                    resultado.step.is_none()
                ),
            );
            let texto = if resultado.step.is_none() {
                "Campo de Treinamento: etapa 1/4 (aceita)".to_string()
            } else {
                format!(
                    "Campo de Treinamento: etapa {}/4{}",
                    resultado.state.stage,
                    if resultado.state.completed {
                        " (concluido)"
                    } else {
                        ""
                    }
                )
            };
            let mut painel = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
            let bytes = texto.as_bytes();
            let limite = bytes.len().min(painel.len() - 1);
            painel[..limite].copy_from_slice(&bytes[..limite]);
            reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &painel)
                && send_progress(stream, key_word, *client_id, &resultado.mob)
        }
        Err(error) => {
            log(
                "warn",
                "quest",
                &format!(
                    "result=refused account={account} npc={scene_id} merchant={merchant} reason={error}"
                ),
            );
            let mut painel = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
            let texto = error.to_string();
            let bytes = texto.as_bytes();
            let limite = bytes.len().min(painel.len() - 1);
            painel[..limite].copy_from_slice(&bytes[..limite]);
            reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &painel)
        }
    }
}

/// `_MSG_Action`: clique em NPC → `_MSG_MessagePanel` com o nome real da definição.
///
/// Sem tabela de diálogo (`npc_dialogue` ainda é planejada), o texto é o `display_name` do
/// snapshot — não existe frase inventada. NPC fora do alcance não responde.
fn handle_action(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            entity: Some(entity),
            selected,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "action",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Action::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "action",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    // `_MSG_Action` também é o passo do personagem (`Route[24]` do legado): o destino é
    // validado no tick (orçamento por tempo + retângulo do mapa) antes de qualquer NPC.
    match store.move_to(
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
        now_ms(),
    ) {
        Ok((x, y)) => log(
            "info",
            "move",
            &format!("result=ok account={account} to=({x}, {y})"),
        ),
        Err(error) => log(
            "warn",
            "move",
            &format!(
                "result=refused account={account} at=({}, {}) reason={error}",
                packet.target_x, packet.target_y
            ),
        ),
    }
    // Portão no ponto clicado: chave por `EF_KEYID`, consumida no aceite, aberto por tempo.
    match store.use_key(
        selected,
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
        now_ms(),
    ) {
        Ok(Some(portao)) => {
            log(
                "info",
                "gate",
                &format!(
                    "result=open account={account} gate={} key_ef={} key_item={} closes_at_ms={}",
                    portao.gate, portao.key_ef, portao.key_item, portao.closes_at_ms
                ),
            );
            return reply_panel(stream, key_word, &format!("Portao {} aberto", portao.gate));
        }
        Ok(None) => {}
        Err(error) => {
            log(
                "warn",
                "gate",
                &format!(
                    "result=refused account={account} at=({}, {}) reason={error}",
                    packet.target_x, packet.target_y
                ),
            );
            return reply_panel(stream, key_word, &error.to_string());
        }
    }
    match store.talk(
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
    ) {
        Ok(Some(alvo)) => {
            log(
                "info",
                "action",
                &format!(
                    "result=ok account={account} npc={} name={} merchant={} at=({}, {})",
                    alvo.id, alvo.name, alvo.merchant, packet.target_x, packet.target_y
                ),
            );
            let mut text = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
            let bytes = alvo.name.as_bytes();
            let limite = bytes.len().min(text.len() - 1);
            text[..limite].copy_from_slice(&bytes[..limite]);
            reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &text)
        }
        Ok(None) => {
            log(
                "info",
                "action",
                &format!(
                    "result=no_npc account={account} at=({}, {})",
                    packet.target_x, packet.target_y
                ),
            );
            true
        }
        Err(error) => {
            log(
                "warn",
                "action",
                &format!("result=refused account={account} reason={error}"),
            );
            true
        }
    }
}

/// `_MSG_MessagePanel` com texto curto (resposta de portão/recusa ao cliente).
fn reply_panel(stream: &mut TcpStream, key_word: u8, texto: &str) -> bool {
    let mut painel = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
    let bytes = texto.as_bytes();
    let limite = bytes.len().min(painel.len() - 1);
    painel[..limite].copy_from_slice(&bytes[..limite]);
    reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &painel)
}

/// Emite o delta da cena de NPCs desde o último envio desta sessão.
///
/// Só busca a foto quando a **geração** do motor mudou (recarga de configuração mexeu em
/// `npc_definition`). Sai do snapshot do mundo, então a posição é a autoritativa: o cliente
/// não vê NPC onde o servidor não o tem.
fn refresh_scene(
    stream: &mut TcpStream,
    key_word: u8,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            scene,
            client_id,
            entity: Some(entity),
            ..
        },
        Some(store),
    ) = (stage, selection)
    else {
        return true;
    };
    // Entrega por **proximidade**: o mapa tem 6 110 NPCs e a rajada de 1,4 MB no login não
    // é desenhada pelo cliente 7662. A varredura é conferida a cada poll (a geração não muda
    // quando o personagem anda), e o delta tira quem saiu do raio e traz quem entrou.
    let Some(atual) = store.scene_around(*entity, scene_view_tiles()) else {
        return true;
    };
    let mut plano = plan_scene(&scene.mobs, &atual, scene_budget());
    // Perto primeiro: quem entra precisa ver a área onde está antes do resto do mapa. Sem
    // isso, um teto de varredura cortava justamente os NPCs criados por último (o Campo).
    if let Some(eu) = atual.mobs.iter().find(|mob| mob.id == *client_id) {
        let (px, py) = (eu.position.x, eu.position.y);
        plano.enter.sort_by_key(|mob| {
            (
                mob.position.x.abs_diff(px).max(mob.position.y.abs_diff(py)),
                mob.id,
            )
        });
    }
    let mut removidos = 0usize;
    for (id, remove_type) in &plano.leave {
        if !send_remove_mob(stream, key_word, *id, *remove_type) {
            return false;
        }
        scene.mobs.remove(id);
        removidos += 1;
    }
    let mut criados = 0usize;
    for mob in &plano.enter {
        // `CreateType` 2 ("acabou de entrar") só para o dono da sessão; os demais
        // entram como 0 (`SendFunc`/`GetCreateMob` do legado).
        let create_type = if mob.id == *client_id {
            CreateMob::CREATE_JUST_ENTERED
        } else {
            CreateMob::CREATE_NORMAL
        };
        if !send_create_mob(stream, key_word, mob, create_type) {
            return false;
        }
        scene.mobs.insert(mob.id, mob.actor);
        criados += 1;
    }
    if criados == 0 && removidos == 0 {
        // Nada mudou na vista: nenhum pacote, nenhuma linha de log por segundo.
        return true;
    }
    if plano.complete {
        scene.generation = atual.generation;
    } else {
        // Teto atingido: mantém a geração antiga para a próxima varredura continuar o delta.
        log(
            "warn",
            "scene",
            &format!(
                "result=budget limit={} created={criados} removed={removidos} pending=1",
                scene_budget()
            ),
        );
        return true;
    }
    log(
        "info",
        "scene",
        &format!(
            "result=applied generation={} mobs={} created={criados} removed={removidos}",
            scene.generation,
            scene.mobs.len()
        ),
    );
    true
}

/// Raio da cena visível (`WYD_SCENE_VIEW_TILES`), lido uma vez.
fn scene_view_tiles() -> u32 {
    use std::sync::OnceLock;
    static RAIO: OnceLock<u32> = OnceLock::new();
    *RAIO.get_or_init(crate::world::scene_view_tiles_from_env)
}

/// Teto de pacotes de cena por varredura (`WYD_SCENE_MAX_PER_REFRESH`), lido uma vez.
fn scene_budget() -> usize {
    use std::sync::OnceLock;
    static BUDGET: OnceLock<usize> = OnceLock::new();
    *BUDGET.get_or_init(|| match crate::world::scene_budget_from_env() {
        Ok(teto) => teto,
        Err(error) => {
            // O boot já recusa valor inválido; aqui é rede de segurança para o teste.
            log("error", "config", &format!("result=invalid {error}"));
            crate::world::DEFAULT_SCENE_MAX_PER_REFRESH
        }
    })
}

/// `_MSG_CreateMob`: uma entidade na vista (NPC ou personagem), `HEADER.ID = ESCENE_FIELD`.
///
/// O corpo é o de `Basedef.h`/porte Go (232 B, offsets em `payload/scene.rs`). `Affect`,
/// `AnctCode`, `Tab` e `Hold` vão zerados porque a origem não os tem — não se inventa
/// efeito nem aura de refino que o dado não sustenta. Do `STRUCT_MOB` do personagem saem
/// nome, `Equip[i].sIndex`, `Guild` e `CurrentScore`; `Merchant` de jogador é 0.
fn send_create_mob(stream: &mut TcpStream, key_word: u8, mob: &SceneMob, create_type: u16) -> bool {
    let packet = CreateMob {
        pos_x: u16::try_from(mob.position.x).unwrap_or(0),
        pos_y: u16::try_from(mob.position.y).unwrap_or(0),
        mob_id: mob.id,
        mob_name: mob.name,
        equip: mob.equip,
        affect: [0; 32],
        guild: mob.guild,
        guild_member_type: 0,
        unknow: [0; 3],
        level: i32::from(mob.level),
        ac: i32::try_from(mob.defense).unwrap_or(i32::MAX),
        damage: i32::try_from(mob.attack).unwrap_or(i32::MAX),
        merchant: mob.merchant,
        attack_run: mob.attack_run,
        direction: mob.direction,
        // CP neutro (`CP + 75`): é o byte que o cliente usa para a cor do nome. Zero cai no
        // ramo `< 10` (criminoso) e pinta o personagem novo de vermelho.
        chaos_rate: crate::personagens::CP_NEUTRO,
        max_hp: i32::try_from(mob.max_hp).unwrap_or(i32::MAX),
        max_mp: 0,
        hp: i32::try_from(mob.hp).unwrap_or(i32::MAX),
        mp: 0,
        strength: 0,
        intelligence: 0,
        dexterity: 0,
        constitution: 0,
        create_type,
        anct_code: [0; 16],
        tab: [0; 26],
        hold: 0,
    };
    let mut frame = vec![0u8; CreateMob::WIRE_LEN];
    if packet.encode_into(&mut frame).is_err() {
        return false;
    }
    reply_with_body(
        stream,
        key_word,
        ID_SCENE,
        Opcode::CreateMob,
        &frame[HEADER_SIZE..],
    )
}

/// `_MSG_RemoveMob`: `HEADER.ID` = id de cena, corpo `RemoveType` (0 = fora de vista,
/// 2 = logout — `ADR-0025`).
fn send_remove_mob(stream: &mut TcpStream, key_word: u8, id: u16, remove_type: i32) -> bool {
    let packet = RemoveMob { remove_type };
    let mut frame = vec![0u8; RemoveMob::WIRE_LEN];
    if packet.encode_into(&mut frame).is_err() {
        return false;
    }
    reply_with_body(
        stream,
        key_word,
        id,
        Opcode::RemoveMob,
        &frame[HEADER_SIZE..],
    )
}

/// Semente do sorteio de loot: relógio + contador do processo, nunca do cliente.
fn loot_seed() -> u64 {
    use std::sync::atomic::{AtomicU64, Ordering};
    static SEQ: AtomicU64 = AtomicU64::new(0);
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|duracao| u64::try_from(duracao.as_nanos()).unwrap_or(u64::MAX))
        .unwrap_or(1);
    nanos
        ^ SEQ
            .fetch_add(1, Ordering::Relaxed)
            .wrapping_mul(0x9E37_79B9_7F4A_7C15)
}

/// Relógio do processo em ms (cooldown de ataque); nunca no tick.
fn now_ms() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|duracao| u64::try_from(duracao.as_millis()).unwrap_or(u64::MAX))
        .unwrap_or(0)
}

fn create_character(
    store: &SelectionStore,
    frame: &[u8],
    account: &str,
) -> Result<String, SelectionError> {
    let packet = CreateCharacter::decode(frame)
        .map_err(|error| SelectionError::Layout(error.to_string()))?;
    let character = store.create(account, packet.slot, &packet.mob_name, packet.mob_class)?;
    // `pedido` é o slot que o cliente desenhou; `slot` é o escolhido pelo servidor (o primeiro
    // livre quando o pedido estava ocupado). A diferença aparece no log, não é silenciosa.
    Ok(format!(
        "slot={} pedido={} name={} class={} id={}",
        character.slot, packet.slot, character.name, character.class, character.id
    ))
}

fn delete_character(
    store: &SelectionStore,
    frame: &[u8],
    account: &str,
) -> Result<String, SelectionError> {
    let packet = DeleteCharacter::decode(frame)
        .map_err(|error| SelectionError::Layout(error.to_string()))?;
    let character = store.delete(account, packet.slot, &packet.mob_name)?;
    // `password_checked=false`: a senha do pacote pertence à autenticação de conta, que ainda
    // não existe; a remoção é do dono da sessão. Auditoria não pode fingir que houve conferência.
    Ok(format!(
        "slot={} name={} id={} password_checked=false",
        character.slot, character.name, character.id
    ))
}

fn select_character(
    store: &SelectionStore,
    frame: &[u8],
    account: &str,
) -> Result<(String, Selected), SelectionError> {
    // O cliente 7662 manda 36 B aqui (log do canal: `esperado 20, recebido 36`); o
    // decodificador aceita os dois tamanhos provados sem inventar os 16 B extras.
    let packet = CharacterLogin::decode_client(frame)
        .map_err(|error| SelectionError::Layout(error.to_string()))?;
    let selected = store.select_snapshot(account, packet.slot)?;
    let character = &selected.character;
    let detalhe = format!(
        "slot={} name={} class={} level={} id={} force={}",
        character.slot,
        character.name,
        character.class,
        character.level,
        character.id,
        packet.force
    );
    Ok((detalhe, selected))
}

/// `_MSG_SendItem` (0x0182): atualiza um slot do inventário no cliente.
///
/// É a resposta do refino: o alvo com `BASE_SetItemSanc` aplicado no fio e `HEADER.ID`
/// da conexão do dono. `inv_type` segue `ITEM_PLACE_*` e `slot` é relativo ao lugar.
fn reply_send_item(
    stream: &mut TcpStream,
    key_word: u8,
    client_id: u16,
    inv_type: i32,
    slot: i32,
    item: [u8; 8],
) -> bool {
    let packet = SendItem {
        inv_type: inv_type as u16,
        slot: slot as u16,
        item,
    };
    let mut frame = vec![0u8; SendItem::WIRE_LEN];
    if packet.encode_into(&mut frame).is_err() {
        return false;
    }
    reply_with_body(
        stream,
        key_word,
        client_id,
        Opcode::SendItem,
        &frame[HEADER_SIZE..],
    )
}

/// Envia o estado do personagem ao cliente: `MSG_UpdateEtc` e `MSG_UpdateScore`.
///
/// É o par `SendEtc`/`SendScore` do legado, usado quando a progressão muda (bônus e nível).
/// Os dois levam o `HEADER.ID` do próprio dono (`ClientID` do login).
pub fn send_progress(stream: &mut TcpStream, key_word: u8, client_id: u16, mob: &Mob) -> bool {
    let etc = crate::personagens::update_etc_body(mob);
    let score = crate::personagens::update_score_body(mob);
    reply_with_body(stream, key_word, client_id, Opcode::UpdateEtc, &etc)
        && reply_with_body(stream, key_word, client_id, Opcode::UpdateScore, &score)
}

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
