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
use std::time::Duration;

use wyd_net::{VersionGate, NET};
use wyd_protocol::codec::checksum::{self, ChecksumPolicy};
use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::{inspect, INITCODE};
use wyd_protocol::wire::header::{Header, HEADER_SIZE};
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, CharacterLogin, CreateCharacter, DeleteCharacter};

use crate::auth::{Authenticator, PinOutcome};
use crate::personagens::{character_body, StructMob};

use crate::channels::World;
use crate::characters::{SelectionError, SelectionStore};
use crate::session::{SessionError, SessionGate, SessionLease};

/// Timeout de leitura do handshake: conexão muda não segura thread.
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(10);

/// Quantos períodos ociosos a sessão sobrevive antes de o servidor soltar a conexão.
///
/// Sem esse teto, um cliente que desaparece sem FIN (queda de energia, cabo) manteria a
/// sessão renovada para sempre e o jogador ficaria preso fora do jogo.
const MAX_IDLE_ROUNDS: u32 = 10;

/// Estágio da conexão depois do login mantido só para o log (seleção × em jogo).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Stage {
    Selecting,
    InGame,
}

impl Stage {
    const fn as_str(self) -> &'static str {
        match self {
            Self::Selecting => "selecting",
            Self::InGame => "in_game",
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
    let citizen = sessions.map_or(0, |cerca| {
        u8::try_from(cerca.channel()).unwrap_or(0).saturating_add(1)
    });
    let resultado = (|| {
        let (mut ok, conta) = handshake(&mut stream, world, gate, auth, selection, citizen)?;
        // A cerca só abre **depois** do PIN: senha errada ou PIN errado não deixam sessão.
        let lease = authorize_pin(&mut stream, &conta, auth, sessions, ok.client_version)?;
        ok.session_id = lease.as_ref().map(SessionLease::id);
        // Segura a conexão: é a vida da sessão. Sem isso a sessão morreria no CNF e a cerca
        // não impediria nada — e sem o laço de seleção o jogador não passa do login.
        let idle = sessions.map_or(HANDSHAKE_TIMEOUT, SessionGate::idle);
        serve_selection(
            &mut stream,
            &conta,
            selection,
            lease.as_ref(),
            idle,
            citizen,
        );
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
    citizen: u8,
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

    // Corpo com os personagens da conta (`STRUCT_SELCHAR`): estado real do banco. Falha de
    // leitura devolve lista vazia — o cliente vê "sem personagem", nunca dado inventado.
    let roster = match selection.map(|store| store.roster(&conta)) {
        Some(Ok(roster)) => roster,
        Some(Err(error)) => {
            log(
                "warn",
                "login",
                &format!("result=empty_roster account={conta} reason={error}"),
            );
            Vec::new()
        }
        None => Vec::new(),
    };
    let corpo = character_body(&roster, citizen);
    reply_with_body(stream, header.key_word, Opcode::CNFAccountLogin, &corpo)
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
fn authorize_pin(
    stream: &mut TcpStream,
    account: &str,
    auth: Option<&Authenticator>,
    sessions: Option<&SessionGate>,
    version: i32,
) -> Result<Option<SessionLease>, HandshakeError> {
    let Some(auth) = auth else {
        return open_session(account, sessions, version);
    };
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
                        let _ = reply(stream, head[2], Opcode::AccountSecureFail);
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
                        match open_session(account, sessions, version) {
                            Ok(lease) => {
                                if !reply(stream, head[2], Opcode::AccountSecure) {
                                    return Err(HandshakeError::Io(
                                        "falha ao escrever _MSG_AccountSecure".to_string(),
                                    ));
                                }
                                return Ok(lease);
                            }
                            Err(error) => {
                                let _ = reply(stream, head[2], Opcode::AccountSecureFail);
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
                        let _ = reply(stream, head[2], Opcode::AccountSecureFail);
                        return Err(HandshakeError::Pin(error.to_string()));
                    }
                }
            }
            Opcode::CreateCharacter | Opcode::DeleteCharacter | Opcode::CharacterLogin => {
                log(
                    "warn",
                    "pin",
                    &format!("result=refused account={account} reason=missing_pin"),
                );
                return Err(HandshakeError::Pin(
                    "seleção antes da senha numérica".to_string(),
                ));
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
            NET.packet_rejected("session");
            let origem = match &error {
                SessionError::Conflict { channel } => *channel,
                SessionError::Db(_) => 0,
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
    citizen: u8,
) {
    let _ = stream.set_read_timeout(Some(idle));
    let mut stage = Stage::Selecting;
    let mut ocioso = 0u32;
    loop {
        let mut head = [0u8; HEADER_SIZE];
        match stream.read_exact(&mut head) {
            Ok(()) => ocioso = 0,
            Err(error) if is_timeout(&error) => {
                if let Some(lease) = lease {
                    lease.renew();
                }
                ocioso += 1;
                if ocioso >= MAX_IDLE_ROUNDS {
                    log(
                        "info",
                        "session",
                        &format!(
                            "result=idle account={account} stage={} rounds={ocioso}",
                            stage.as_str()
                        ),
                    );
                    return;
                }
                continue;
            }
            Err(_) => return,
        }
        let Some(frame) = read_frame(stream, &head, account) else {
            return;
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
            Sessao { account, citizen },
            selection,
            &mut stage,
        ) {
            return;
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
    let Sessao { account, citizen } = sessao;
    match opcode {
        Opcode::CreateCharacter => {
            let resultado = match selection {
                Some(store) => create_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            respond(
                stream,
                key_word,
                resultado,
                Opcode::CNFNewCharacter,
                Opcode::NewCharacterFail,
                account,
                "create",
            )
        }
        Opcode::DeleteCharacter => {
            let resultado = match selection {
                Some(store) => delete_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            respond(
                stream,
                key_word,
                resultado,
                Opcode::CNFDeleteCharacter,
                Opcode::DeleteCharacterFail,
                account,
                "delete",
            )
        }
        Opcode::CharacterLogin => {
            let resultado = match selection {
                Some(store) => select_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok((detalhe, personagem)) => {
                    *stage = Stage::InGame;
                    log(
                        "info",
                        "character",
                        &format!("result=ok action=select account={account} {detalhe}"),
                    );
                    // `STRUCT_MOB` do personagem escolhido, do estado persistido.
                    let corpo = StructMob::from_character(&personagem, citizen).encode();
                    reply_with_body(stream, key_word, Opcode::CNFCharacterLogin, &corpo)
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=select account={account} reason={error}"),
                    );
                    reply(stream, key_word, Opcode::CharacterLoginFail)
                }
            }
        }
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

fn create_character(
    store: &SelectionStore,
    frame: &[u8],
    account: &str,
) -> Result<String, SelectionError> {
    let packet = CreateCharacter::decode(frame)
        .map_err(|error| SelectionError::Layout(error.to_string()))?;
    let character = store.create(account, packet.slot, &packet.mob_name, packet.mob_class)?;
    Ok(format!(
        "slot={} name={} class={} id={}",
        character.slot, character.name, character.class, character.id
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
) -> Result<(String, wyd_db::Character), SelectionError> {
    let packet =
        CharacterLogin::decode(frame).map_err(|error| SelectionError::Layout(error.to_string()))?;
    let character = store.select(account, packet.slot)?;
    let detalhe = format!(
        "slot={} name={} class={} level={} id={} force={}",
        character.slot,
        character.name,
        character.class,
        character.level,
        character.id,
        packet.force
    );
    Ok((detalhe, character))
}

/// Responde o opcode de sucesso ou o de falha do mesmo domínio, com log do motivo.
fn respond(
    stream: &mut TcpStream,
    key_word: u8,
    resultado: Result<String, SelectionError>,
    ok_opcode: Opcode,
    fail_opcode: Opcode,
    account: &str,
    action: &str,
) -> bool {
    match resultado {
        Ok(detalhe) => {
            log(
                "info",
                "character",
                &format!("result=ok action={action} account={account} {detalhe}"),
            );
            reply(stream, key_word, ok_opcode)
        }
        Err(error) => {
            log(
                "warn",
                "character",
                &format!("result=refused action={action} account={account} reason={error}"),
            );
            reply(stream, key_word, fail_opcode)
        }
    }
}

/// Conta autenticada + canal de origem (o `citizen` do `STRUCT_SELCHAR` é canal + 1).
struct Sessao<'a> {
    account: &'a str,
    citizen: u8,
}

/// Frame com corpo (`Size = 12 + corpo`), ofuscado com a chave do frame de entrada.
fn reply_with_body(stream: &mut TcpStream, key_word: u8, opcode: Opcode, body: &[u8]) -> bool {
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
        id: 1,
        client_tick: 0,
    };
    if cabecalho.encode(&mut frame).is_err() || encode_in_place(&mut frame, key_word).is_err() {
        return false;
    }
    stream.write_all(&frame).is_ok() && stream.flush().is_ok()
}

/// Frame só de cabeçalho com o opcode pedido, ofuscado com a chave do frame de entrada.
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
