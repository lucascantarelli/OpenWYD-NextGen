//! Presença por canal — **somente leitura** para a área de monitoramento.
//!
//! O portal nunca escreve estado vivo (invariante 7): aqui só se lê quantas sessões
//! estão abertas em cada canal, da tabela `character_session` que o próprio servidor
//! mantém. É o número auditável de "quem está online onde", distinto da métrica do
//! processo (`wyd_sessions_open`), que morre com o container.
//!
//! A leitura é uma por canal (`group_id`, `channel`) — nunca um agregado do grupo:
//! é isso que prova o isolamento entre mundos (`docs/wyd/canais-e-cidadania.md` §5).

use wyd_db::Sessions;

use crate::api::accounts::staff_or_deny;
use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;

/// Uma linha de canal com a presença viva no momento da leitura.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChannelPresence {
    pub group_id: i64,
    /// `1..10`; a linha de status (`channel = 0`) não aparece aqui.
    pub channel: i64,
    /// Host que o cliente disca (`server_channel.host`), para casar com o launcher.
    pub host: String,
    pub online: u64,
}

/// Foto completa: canais publicados + total (soma das linhas, nunca do grupo inteiro).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PresenceSnapshot {
    pub channels: Vec<ChannelPresence>,
    pub total_online: u64,
}

/// Lê a presença de cada canal informado (`(group_id, channel, host)`).
///
/// Staff obrigatório; a leitura é auditada com o autor da sessão. Canal fora da faixa
/// `i16` ou linha de status (`channel <= 0`) é ignorada em vez de virar erro de rede.
pub async fn list_channel_presence(
    session: Option<Session>,
    sessions: &impl Sessions,
    canais: &[(i64, i64, String)],
    sink: &impl AuditSink,
) -> Result<PresenceSnapshot, AdminError> {
    let staff = staff_or_deny(session, sink, "server.presence")?;
    let mut channels = Vec::with_capacity(canais.len());
    let mut total_online = 0_u64;

    for (group_id, channel, host) in canais {
        let (Ok(grupo), Ok(canal)) = (i16::try_from(*group_id), i16::try_from(*channel)) else {
            continue;
        };
        if canal <= 0 {
            continue;
        }
        let online = sessions.live_sessions(grupo, canal).await?;
        total_online += online;
        channels.push(ChannelPresence {
            group_id: *group_id,
            channel: *channel,
            host: host.clone(),
            online,
        });
    }

    log_action(
        sink,
        &staff,
        "server.presence",
        "channels",
        &format!("canais={} online={total_online}", channels.len()),
    );
    Ok(PresenceSnapshot {
        channels,
        total_online,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    use wyd_db::{ClaimOutcome, DbError, Role};

    /// Backend de teste: presença por canal declarada, sem banco.
    struct FakeSessions {
        online: HashMap<(i16, i16), u64>,
    }

    impl Sessions for FakeSessions {
        async fn claim_session(
            &self,
            _login: &str,
            _group_id: i16,
            _channel: i16,
            _client_version: i32,
            _stale_after_secs: i64,
        ) -> Result<ClaimOutcome, DbError> {
            Err(DbError::Transport { op: "test.fake" })
        }

        async fn renew_session(&self, _id: i64) -> Result<(), DbError> {
            Ok(())
        }

        async fn release_session(&self, _id: i64) -> Result<(), DbError> {
            Ok(())
        }

        async fn release_account_sessions(&self, _login: &str) -> Result<u64, DbError> {
            Ok(0)
        }

        async fn live_session_of(&self, _login: &str) -> Result<Option<i16>, DbError> {
            Ok(None)
        }

        async fn live_sessions(&self, group_id: i16, channel: i16) -> Result<u64, DbError> {
            Ok(self.online.get(&(group_id, channel)).copied().unwrap_or(0))
        }
    }

    fn canais() -> Vec<(i64, i64, String)> {
        vec![
            (1, 0, "192.168.100.3".to_string()),
            (1, 1, "192.168.100.3".to_string()),
            (1, 2, "192.168.100.4".to_string()),
        ]
    }

    #[test]
    fn presenca_de_um_canal_nao_aparece_no_outro() {
        let sessions = FakeSessions {
            online: HashMap::from([((1, 2), 3_u64)]),
        };
        let sink = crate::audit::VecSink::default();
        let snapshot = crate::block_on(list_channel_presence(
            Some(Session {
                account_id: 7,
                role: Role::Moderator,
            }),
            &sessions,
            &canais(),
            &sink,
        ))
        .expect("staff lê presença");

        // Linha de status (channel 0) fica de fora; o total é a soma das linhas.
        assert_eq!(snapshot.channels.len(), 2);
        assert_eq!(snapshot.channels[0].channel, 1);
        assert_eq!(snapshot.channels[0].online, 0);
        assert_eq!(snapshot.channels[1].channel, 2);
        assert_eq!(snapshot.channels[1].online, 3);
        assert_eq!(snapshot.total_online, 3);
        assert_eq!(sink.entries()[0].action, "server.presence");
    }

    #[test]
    fn jogador_nao_le_presenca() {
        let sessions = FakeSessions {
            online: HashMap::new(),
        };
        let sink = crate::audit::VecSink::default();
        let recusa = crate::block_on(list_channel_presence(
            Some(Session {
                account_id: 9,
                role: Role::Player,
            }),
            &sessions,
            &canais(),
            &sink,
        ));
        assert!(recusa.is_err());
    }
}
