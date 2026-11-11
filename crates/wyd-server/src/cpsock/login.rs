//! Submódulo de `cpsock` (ADR-0026, etapa 2).

use super::selection::read_frame;
use super::*;

pub(super) fn authorize_pin(
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
                return Ok((lease, Some((frame.to_vec(), head[2]))));
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
pub(super) fn open_session(
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
