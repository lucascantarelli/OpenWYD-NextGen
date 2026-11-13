//! Submódulo de `cpsock` (ADR-0026, etapa 2).

use super::action::{dispatch, drain_chat, refresh_scene};
use super::*;

/// Atende a conexão depois do `CNF` enquanto o cliente estiver do outro lado (CLIENT-02).
///
/// Cada iteração consome **o frame inteiro** antes de decidir: ler só o cabeçalho deixaria o
/// corpo no socket e o `Size` seguinte seria lido do meio do pacote anterior.
///
/// Ociosidade renova o sinal de vida até `MAX_IDLE_ROUNDS`; ociosidade além disso libera a
/// sessão. Qualquer falha de I/O (FIN, reset) também encerra — o `Drop` do `SessionLease`
/// libera a conta para outro canal.
pub(super) fn serve_selection(
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
    // Chat social: a inscrição nasce no login do personagem e o `ChatLease` a remove no fim.
    let mut chat: Option<(
        crate::session::ChatLease,
        std::sync::mpsc::Receiver<crate::session::ChatLine>,
    )> = None;
    loop {
        // Inscrição no chat nasce assim que o personagem entra em jogo (o `dispatch` mudou o
        // estágio); o `ChatLease` remove no fim, inclusive em erro.
        if chat.is_none() {
            if let Stage::InGame {
                client_id,
                selected,
                ..
            } = &stage
            {
                if *client_id != 0 {
                    if let Some(store) = selection {
                        chat = Some(store.chat().join(*client_id, &selected.character.name));
                    }
                }
            }
        }
        // Drena antes de bloquear no read: uma linha de outro jogador não espera pacote.
        if let Some((_, rx)) = &chat {
            if !drain_chat(stream, chave, rx) {
                return stage.entity();
            }
        }
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
pub(super) fn read_frame(
    stream: &mut TcpStream,
    head: &[u8; HEADER_SIZE],
    account: &str,
) -> Option<Vec<u8>> {
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

pub(super) fn create_character(
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

pub(super) fn delete_character(
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

pub(super) fn select_character(
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
