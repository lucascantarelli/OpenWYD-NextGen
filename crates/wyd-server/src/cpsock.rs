//! Listener CPSock (`:8281`): `INITCODE` → frame ofuscado → gate de versão → `CNF`.
//!
//! Ordem obrigatória (ADR-0022 §3, `docs/arquitetura/protocolo.md` §1):
//! `INITCODE` (4 B) → `HEADER` (12 B, `Size` em claro) → resto do frame → `decode_in_place`
//! → checksum `Strict` → `inspect` (opcode) → `AccountLogin::decode`.
//!
//! O que **não** existe ainda: autenticação de conta (`_MSG_AccountLogin` só tem a versão
//! validada aqui) e pacotes de mundo. Opcode fora do que este processo atende é erro tipado
//! explícito, nunca silêncio: a conexão fecha e o motivo vai para o log e para
//! `wyd_packets_rejected_total`.

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
use wyd_protocol::AccountLogin;

use crate::channels::World;

/// Timeout de leitura do handshake: conexão muda não segura thread.
const HANDSHAKE_TIMEOUT: Duration = Duration::from_secs(10);

/// Resultado de um handshake aceito (o que ficou provado sobre a conexão).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct HandshakeOk {
    /// Versão declarada pelo cliente no login.
    pub client_version: i32,
    /// Chave de ofuscação usada na resposta (a mesma do frame de entrada).
    pub key_word: u8,
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
pub fn serve_listener(listener: TcpListener, world: Arc<World>, gate: VersionGate) {
    for aceito in listener.incoming() {
        match aceito {
            Ok(stream) => {
                let world = Arc::clone(&world);
                std::thread::spawn(move || match handle_connection(stream, &world, gate) {
                    Ok(ok) => log(
                        "info",
                        "handshake",
                        &format!(
                            "result=accepted version={} channels={} kw={}",
                            ok.client_version,
                            world.game_channel_count(),
                            ok.key_word
                        ),
                    ),
                    Err(error) => log(
                        "warn",
                        "handshake",
                        &format!("result=rejected reason={error}"),
                    ),
                });
            }
            Err(error) => {
                log("error", "accept", &format!("reason={error}"));
                return;
            }
        }
    }
}

/// Executa o handshake de uma conexão já aceita e responde `_MSG_CNFAccountLogin`.
///
/// A resposta é um frame **só de cabeçalho** (`Size = 12`) com o opcode de confirmação: o
/// corpo do `_MSG_CNFAccountLogin` é UNVERIFIED (`docs/arquitetura/protocolo.md` §5) e
/// inventá-lo seria pior que confirmar o handshake. O layout entra com a captura da FASE-08-2.
pub fn handle_connection(
    mut stream: TcpStream,
    world: &World,
    gate: VersionGate,
) -> Result<HandshakeOk, HandshakeError> {
    let _ = stream.set_read_timeout(Some(HANDSHAKE_TIMEOUT));
    let _ = stream.set_write_timeout(Some(HANDSHAKE_TIMEOUT));
    NET.session_delta(1);
    let resultado = handshake(&mut stream, world, gate);
    NET.session_delta(-1);
    resultado
}

fn handshake(
    stream: &mut TcpStream,
    world: &World,
    gate: VersionGate,
) -> Result<HandshakeOk, HandshakeError> {
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

    NET.packet_accepted();
    NET.handshake_opened();
    log(
        "info",
        "login",
        &format!(
            "account={} version={} channels={}",
            String::from_utf8_lossy(login.account_name_bytes()),
            login.client_version,
            world.game_channel_count()
        ),
    );

    let mut reply = [0u8; HEADER_SIZE];
    Header {
        size: HEADER_SIZE as u16,
        key_word: header.key_word,
        checksum: 0,
        message_type: Opcode::CNFAccountLogin.wire(),
        id: 1,
        client_tick: 0,
    }
    .encode(&mut reply)
    .map_err(|error| HandshakeError::Frame(error.to_string()))?;
    encode_in_place(&mut reply, header.key_word)
        .map_err(|error| HandshakeError::Frame(error.to_string()))?;
    stream.write_all(&reply)?;
    stream.flush()?;

    Ok(HandshakeOk {
        client_version: login.client_version,
        key_word: header.key_word,
    })
}

/// Log estruturado (uma linha, `key=value`) — mesmo formato do gateway.
fn log(level: &str, event: &str, fields: &str) {
    println!("level={level} event={event} {fields}");
}
