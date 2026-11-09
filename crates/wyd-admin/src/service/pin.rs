//! Troca de senha numérica pelo portal: cliente CPSock do servidor de jogo.
//!
//! O portal não reimplementa o protocolo nem escreve hash no banco: o gateway abre
//! uma conexão de jogo de verdade e faz **as duas etapas do `0x0FDE`** que o cliente
//! 7662 usa — passo 1 com o PIN atual (`change_numeric = 1`), passo 2 com o novo
//! (`change_numeric = 2`). Assim o bloqueio por tentativas e a auditoria continuam
//! acontecendo no servidor, no mesmo caminho do jogo.
//!
//! A senha da conta é necessária porque o `0x0FDE` só existe depois de `0x0FDB`
//! (`AccountLogin`) — o servidor nunca aceita PIN de sessão não autenticada.

use std::io::{Read, Write};
use std::net::TcpStream;
use std::time::Duration;

use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::frame::INITCODE;
use wyd_protocol::wire::header::HEADER_SIZE;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{AccountLogin, AccountSecure, Header};

/// Tempo máximo de cada leitura de quadro (o servidor responde na hora).
const READ_TIMEOUT: Duration = Duration::from_secs(5);
/// Chave de ofuscação usada pelo cliente (mesma nos testes de fio).
const KEY: u8 = 11;
/// Tamanho máximo aceito de um nome de conta no pacote.
const LOGIN_MAX: usize = 16;
/// Tamanho do campo de senha do `0x0FDB`.
const PASSWORD_MAX: usize = 12;

/// Falha da troca de PIN, já separada por causa (o portal vira mensagem com isso).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PinError {
    /// PIN fora de 4–6 dígitos: recusado antes de abrir conexão.
    InvalidPin,
    /// O servidor respondeu `0x0FDF`: PIN atual errado ou conta bloqueada.
    Rejected,
    /// Conexão recusada, timeout ou I/O.
    Unavailable,
    /// O servidor respondeu algo fora do contrato desta operação.
    Protocol,
}

impl std::fmt::Display for PinError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let texto = match self {
            Self::InvalidPin => "PIN deve ter de 4 a 6 dígitos",
            Self::Rejected => "PIN atual inválido ou conta bloqueada por tentativas",
            Self::Unavailable => "servidor de jogo indisponível",
            Self::Protocol => "resposta inesperada do servidor de jogo",
        };
        f.write_str(texto)
    }
}

impl std::error::Error for PinError {}

/// Regra única de PIN do projeto: 4 a 6 dígitos, só ASCII numérico.
#[must_use]
pub fn valid_pin(pin: &str) -> bool {
    (4..=6).contains(&pin.len()) && pin.bytes().all(|byte| byte.is_ascii_digit())
}

/// Troca o PIN de `login` no servidor de jogo em `addr` (`host:porta`).
///
/// Sequência no fio: `INITCODE` → `0x0FDB` (login com a senha da conta) → `0x0FDE`
/// com `change_numeric = 1` e o PIN **atual** → `0x0FDE` com `change_numeric = 2` e
/// o PIN **novo**. Sucesso só é devolvido depois do segundo `0x0FDE`: o servidor já
/// gravou o hash novo e abriu a sessão.
pub fn change_pin(
    addr: &str,
    login: &str,
    password: &str,
    current: &str,
    new: &str,
) -> Result<(), PinError> {
    if !valid_pin(current) || !valid_pin(new) {
        return Err(PinError::InvalidPin);
    }
    if login.is_empty() || login.len() > LOGIN_MAX || password.is_empty() {
        return Err(PinError::InvalidPin);
    }

    let mut stream = TcpStream::connect(addr).map_err(|_| PinError::Unavailable)?;
    stream
        .set_read_timeout(Some(READ_TIMEOUT))
        .map_err(|_| PinError::Unavailable)?;
    stream
        .set_write_timeout(Some(READ_TIMEOUT))
        .map_err(|_| PinError::Unavailable)?;

    write_all(&mut stream, &INITCODE.to_le_bytes())?;

    // 1) Login da conta: sem ele o `0x0FDE` nem é lido pelo servidor.
    let mut frame = frame(Opcode::AccountLogin, AccountLogin::WIRE_LEN)?;
    let mut nome = [0u8; LOGIN_MAX];
    nome[..login.len()].copy_from_slice(login.as_bytes());
    let mut senha = [0u8; PASSWORD_MAX];
    let limite = password.len().min(PASSWORD_MAX);
    senha[..limite].copy_from_slice(&password.as_bytes()[..limite]);
    AccountLogin {
        account_password: senha,
        account_name: nome,
        zero: [0u8; 52],
        client_version: 12_000,
        db_need_save: 1,
        adapter_name: [0; 4],
    }
    .encode_into(&mut frame)
    .map_err(|_| PinError::Protocol)?;
    send(&mut stream, frame)?;
    match read_opcode(&mut stream)? {
        Opcode::CNFAccountLogin => {}
        _ => return Err(PinError::Protocol),
    }

    // 2) Passo 1: o servidor confere o PIN atual e arma a troca (sem abrir sessão).
    match secure(&mut stream, current, 1)? {
        Opcode::AccountSecure => {}
        Opcode::AccountSecureFail => return Err(PinError::Rejected),
        _ => return Err(PinError::Protocol),
    }

    // 3) Passo 2: grava o PIN novo e abre a sessão.
    match secure(&mut stream, new, 2)? {
        Opcode::AccountSecure => Ok(()),
        Opcode::AccountSecureFail => Err(PinError::Rejected),
        _ => Err(PinError::Protocol),
    }
}

fn secure(stream: &mut TcpStream, pin: &str, change_numeric: i32) -> Result<Opcode, PinError> {
    let mut frame = frame(Opcode::AccountSecure, AccountSecure::WIRE_LEN)?;
    let mut token = [0u8; 6];
    token[..pin.len()].copy_from_slice(pin.as_bytes());
    AccountSecure {
        numeric_token: token,
        unknown: [0u8; 10],
        change_numeric,
    }
    .encode_into(&mut frame)
    .map_err(|_| PinError::Protocol)?;
    send(stream, frame)?;
    read_opcode(stream)
}

fn frame(opcode: Opcode, len: usize) -> Result<Vec<u8>, PinError> {
    let mut frame = vec![0u8; len];
    Header {
        size: len as u16,
        key_word: KEY,
        checksum: 0,
        message_type: opcode.wire(),
        id: 1,
        client_tick: 1,
    }
    .encode(&mut frame)
    .map_err(|_| PinError::Protocol)?;
    Ok(frame)
}

fn send(stream: &mut TcpStream, mut frame: Vec<u8>) -> Result<(), PinError> {
    encode_in_place(&mut frame, KEY).map_err(|_| PinError::Protocol)?;
    write_all(stream, &frame)
}

fn write_all(stream: &mut TcpStream, bytes: &[u8]) -> Result<(), PinError> {
    stream.write_all(bytes).map_err(|_| PinError::Unavailable)
}

/// Lê um quadro completo e devolve o opcode (checksum conferido).
fn read_opcode(stream: &mut TcpStream) -> Result<Opcode, PinError> {
    let mut head = [0u8; HEADER_SIZE];
    stream
        .read_exact(&mut head)
        .map_err(|_| PinError::Unavailable)?;
    let size = usize::from(u16::from_le_bytes([head[0], head[1]]));
    if !(HEADER_SIZE..=8192).contains(&size) {
        return Err(PinError::Protocol);
    }
    let mut frame = vec![0u8; size];
    frame[..HEADER_SIZE].copy_from_slice(&head);
    if size > HEADER_SIZE {
        stream
            .read_exact(&mut frame[HEADER_SIZE..])
            .map_err(|_| PinError::Unavailable)?;
    }
    let computed = decode_in_place(&mut frame).map_err(|_| PinError::Protocol)?;
    if computed != frame[3] {
        return Err(PinError::Protocol);
    }
    let header = Header::decode(&frame[..HEADER_SIZE]).map_err(|_| PinError::Protocol)?;
    Opcode::from_wire(header.message_type).ok_or(PinError::Protocol)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pin_valido_e_de_quatro_a_seis_digitos() {
        assert!(valid_pin("1234"));
        assert!(valid_pin("123456"));
        assert!(!valid_pin("123"));
        assert!(!valid_pin("1234567"));
        assert!(!valid_pin("12a4"));
        assert!(!valid_pin(""));
    }

    #[test]
    fn pin_invalido_nao_abre_conexao() {
        // Endereço que não existe: se a validação rodasse depois do connect, viria
        // `Unavailable` em vez de `InvalidPin`.
        assert_eq!(
            change_pin("127.0.0.1:1", "conta", "SenhaForte1", "12", "9999"),
            Err(PinError::InvalidPin)
        );
    }
}
