//! Painel do jogador (WEB-02): conta, personagens, ranking e recuperação.
//!
//! Escopo de leitura: o BFF injeta `account_id` a partir do cookie de sessão assinado — o
//! browser nunca escolhe de quem é o dado. O backend confia no processo do portal (loopback,
//! token processo→processo) e audita autoria + alvo interno em toda leitura.
//!
//! Nada aqui escreve estado vivo: `request_recovery` registra uma **solicitação auditada**;
//! o reset de senha continua ato de staff (WEB-03), porque não há provedor de e-mail
//! aprovado nem tabela de token de recuperação.

use wyd_db::{Accounts, AuditTrail, Characters, Role, SessionTrail};

use crate::audit::{log_panel, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;

/// Conta vista pelo dono (sem hash, sem e-mail, sem segredo).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PanelAccount {
    pub id: i64,
    pub login: String,
    pub role: Role,
    /// A conta já tem segunda senha gravada (`pin_hash` preenchido).
    ///
    /// A conta nasce sem PIN: o jogador cria o primeiro no cliente. O portal só
    /// troca um PIN existente, então precisa saber qual dos dois fluxos oferecer.
    pub pin_defined: bool,
}

/// Personagem na lista do painel.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PanelCharacter {
    pub id: i64,
    pub slot: i64,
    pub name: String,
    pub class: i64,
    pub level: i64,
}

/// Linha do ranking (dado público: sem `account_id`, sem posição no mundo).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RankingEntry {
    pub rank: usize,
    pub name: String,
    pub class: i64,
    pub level: i64,
}

/// Limite máximo de linhas do ranking (o BFF pede 10; a rota nunca devolve mais que 100).
pub const MAX_RANKING: usize = 100;

/// Sessão do jogador vista por ele mesmo (sem login, sem id de conta).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PanelSession {
    pub group_id: i16,
    pub channel: i16,
    pub client_version: i32,
    pub character_id: Option<i64>,
    pub opened_ms: u64,
    pub renewed_ms: u64,
    /// `None` = sessão ainda viva.
    pub released_ms: Option<u64>,
}

/// Evento da trilha que pertence ao jogador (`auth.login`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PanelEvent {
    pub at_ms: u64,
    pub action: String,
    /// `ok`, `conta bloqueada` ou `credencial recusada` — nunca a senha.
    pub detail: String,
}

/// Histórico do jogador: últimas sessões e tentativas de login da própria conta.
///
/// `login` sai do banco pelo `account_id` da sessão assinada, nunca de parâmetro
/// do browser. A trilha é filtrada pelo id do ator, não por texto de login.
pub async fn own_history(
    session: Option<Session>,
    account_id: i64,
    accounts: &impl Accounts,
    sessions: &dyn SessionTrail,
    audit: &dyn AuditTrail,
    sink: &impl AuditSink,
) -> Result<(Vec<PanelSession>, Vec<PanelEvent>), AdminError> {
    let session = require_session(session)?;
    let account_id = check_account(account_id)?;
    let account = accounts.find_by_id(account_id).await?;
    let linhas = sessions.recent_for_login(&account.login, 10).await?;
    let eventos = audit.by_actor(account_id, Some("auth.login"), 20).await?;
    log_panel(
        sink,
        session,
        "panel.history",
        &format!("account:{account_id}"),
        &format!("{} sessão(ões), {} evento(s)", linhas.len(), eventos.len()),
    );
    Ok((
        linhas
            .into_iter()
            .map(|row| PanelSession {
                group_id: row.group_id,
                channel: row.channel,
                client_version: row.client_version,
                character_id: row.character_id,
                opened_ms: row.opened_ms,
                renewed_ms: row.renewed_ms,
                released_ms: row.released_ms,
            })
            .collect(),
        eventos
            .into_iter()
            .map(|row| PanelEvent {
                at_ms: row.at_ms,
                action: row.action,
                detail: row.detail,
            })
            .collect(),
    ))
}

fn require_session(session: Option<Session>) -> Result<Session, AdminError> {
    session.ok_or(AdminError::Unauthorized)
}

fn check_account(account_id: i64) -> Result<i64, AdminError> {
    if account_id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "conta inválida",
        });
    }
    Ok(account_id)
}

/// Conta do próprio jogador (`panel.self`).
pub async fn own_account(
    session: Option<Session>,
    account_id: i64,
    repo: &impl Accounts,
    sink: &impl AuditSink,
) -> Result<PanelAccount, AdminError> {
    let session = require_session(session)?;
    let account_id = check_account(account_id)?;
    let account = repo.find_by_id(account_id).await?;
    let pin_defined = repo.pin_hash(&account.login).await?.is_some();
    log_panel(
        sink,
        session,
        "panel.account",
        &format!("account:{account_id}"),
        "leitura",
    );
    Ok(PanelAccount {
        id: account.id,
        login: account.login,
        role: account.role,
        pin_defined,
    })
}

/// Personagens da conta do jogador (`panel.characters`).
pub async fn own_characters(
    session: Option<Session>,
    account_id: i64,
    repo: &impl Characters,
    sink: &impl AuditSink,
) -> Result<Vec<PanelCharacter>, AdminError> {
    let session = require_session(session)?;
    let account_id = check_account(account_id)?;
    let characters = repo.list_by_account(account_id).await?;
    log_panel(
        sink,
        session,
        "panel.characters",
        &format!("account:{account_id}"),
        &format!("{} personagem(ns)", characters.len()),
    );
    Ok(characters
        .into_iter()
        .map(|c| PanelCharacter {
            id: c.id,
            slot: c.slot,
            name: c.name,
            class: c.class,
            level: c.level,
        })
        .collect())
}

/// Ranking por nível (`panel.rankings`) — público, sem identidade de conta.
pub async fn top_rankings(
    session: Option<Session>,
    limit: usize,
    repo: &impl Characters,
    sink: &impl AuditSink,
) -> Result<Vec<RankingEntry>, AdminError> {
    let session = require_session(session)?;
    if limit == 0 || limit > MAX_RANKING {
        return Err(AdminError::InvalidInput {
            reason: "limite do ranking fora da faixa",
        });
    }
    let characters = repo.top_by_level(limit).await?;
    log_panel(
        sink,
        session,
        "panel.rankings",
        "ranking",
        &format!("top {limit}"),
    );
    Ok(characters
        .into_iter()
        .enumerate()
        .map(|(index, c)| RankingEntry {
            rank: index + 1,
            name: c.name,
            class: c.class,
            level: c.level,
        })
        .collect())
}

/// Solicitação de recuperação de senha (`accounts.recovery`).
///
/// Responde o mesmo resultado exista ou não o login: a trilha guarda a tentativa, nunca a
/// existência da conta, e o alvo auditado é fixo (`login:*`) para não gravar login em log.
pub fn request_recovery(
    session: Option<Session>,
    login: &str,
    sink: &impl AuditSink,
) -> Result<(), AdminError> {
    let session = require_session(session)?;
    let login = login.trim();
    if login.is_empty() {
        return Err(AdminError::InvalidInput {
            reason: "login obrigatório",
        });
    }
    if login.len() > 16 {
        return Err(AdminError::InvalidInput {
            reason: "login inválido",
        });
    }
    log_panel(
        sink,
        session,
        "accounts.recovery",
        "login:*",
        "solicitação de recuperação",
    );
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::audit::VecSink;
    use wyd_db::{AuditRow, DbError, SessionHistoryRow};

    /// Sessões falsas: prova que o histórico usa o login do **banco**, nunca um
    /// login vindo de parâmetro, e que a janela é a pedida pela rota.
    struct FakeTrail {
        linhas: Vec<SessionHistoryRow>,
        login_pedido: std::cell::RefCell<String>,
    }

    impl wyd_db::SessionTrail for FakeTrail {
        fn recent_for_login(
            &self,
            login: &str,
            limit: usize,
        ) -> std::pin::Pin<
            Box<
                dyn std::future::Future<Output = Result<Vec<SessionHistoryRow>, DbError>>
                    + Send
                    + '_,
            >,
        > {
            *self.login_pedido.borrow_mut() = login.to_string();
            let linhas = self.linhas.iter().take(limit).cloned().collect();
            Box::pin(async move { Ok(linhas) })
        }
    }

    struct FakeAudit {
        linhas: Vec<AuditRow>,
        ator_pedido: std::cell::RefCell<Option<i64>>,
    }

    impl wyd_db::AuditTrail for FakeAudit {
        fn record<'a>(
            &'a self,
            _entry: &'a wyd_db::NewAuditEntry,
        ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), DbError>> + Send + 'a>>
        {
            Box::pin(async { Ok(()) })
        }

        fn recent(
            &self,
            _limit: usize,
        ) -> std::pin::Pin<
            Box<dyn std::future::Future<Output = Result<Vec<AuditRow>, DbError>> + Send + '_>,
        > {
            Box::pin(async { Ok(Vec::new()) })
        }

        fn by_actor(
            &self,
            actor_id: i64,
            _action: Option<&str>,
            limit: usize,
        ) -> std::pin::Pin<
            Box<dyn std::future::Future<Output = Result<Vec<AuditRow>, DbError>> + Send + '_>,
        > {
            *self.ator_pedido.borrow_mut() = Some(actor_id);
            let linhas = self.linhas.iter().take(limit).cloned().collect();
            Box::pin(async move { Ok(linhas) })
        }
    }

    fn conta_de(id: i64) -> PanelAccount {
        PanelAccount {
            id,
            login: "jogador".to_string(),
            role: Role::Player,
            pin_defined: false,
        }
    }

    struct ContaUnica(i64);

    impl Accounts for ContaUnica {
        async fn find_by_login(&self, _login: &str) -> Result<wyd_db::Account, DbError> {
            Err(DbError::NotFoundKey { entity: "account" })
        }

        async fn find_by_id(&self, id: i64) -> Result<wyd_db::Account, DbError> {
            let login = if id == self.0 { "jogador" } else { "outro" };
            Ok(wyd_db::Account {
                id,
                login: login.to_string(),
                role: Role::Player,
            })
        }

        async fn create_account(
            &self,
            login: &str,
            role: Role,
        ) -> Result<wyd_db::Account, DbError> {
            Ok(wyd_db::Account {
                id: 1,
                login: login.to_string(),
                role,
            })
        }

        async fn pin_hash(&self, _login: &str) -> Result<Option<String>, DbError> {
            Ok(None)
        }
    }

    #[test]
    fn historico_usa_o_login_do_banco_e_o_ator_da_conta() {
        let sink = VecSink::default();
        let sessoes = FakeTrail {
            linhas: vec![SessionHistoryRow {
                opened_ms: 1_700_000_000_000,
                renewed_ms: 1_700_000_060_000,
                released_ms: None,
                group_id: 1,
                channel: 2,
                client_version: 76_620,
                character_id: Some(9),
            }],
            login_pedido: std::cell::RefCell::new(String::new()),
        };
        let trilha = FakeAudit {
            linhas: vec![AuditRow {
                at_ms: 1_700_000_000_000,
                actor_id: 7,
                actor_role: "player".to_string(),
                action: "auth.login".to_string(),
                target: "jogador".to_string(),
                detail: "ok".to_string(),
            }],
            ator_pedido: std::cell::RefCell::new(None),
        };

        let (sessoes_lidas, eventos) = crate::block_on(own_history(
            session(),
            7,
            &ContaUnica(7),
            &sessoes,
            &trilha,
            &sink,
        ))
        .expect("histórico");

        assert_eq!(sessoes_lidas.len(), 1);
        assert_eq!(sessoes_lidas[0].channel, 2);
        assert_eq!(
            *sessoes.login_pedido.borrow(),
            "jogador",
            "login vem do banco"
        );
        assert_eq!(
            *trilha.ator_pedido.borrow(),
            Some(7),
            "trilha filtra pelo ator"
        );
        assert_eq!(eventos.len(), 1);
        assert_eq!(sink.entries()[0].action, "panel.history");
    }

    #[test]
    fn historico_sem_sessao_e_recusado_antes_de_ler() {
        let sink = VecSink::default();
        let sessoes = FakeTrail {
            linhas: Vec::new(),
            login_pedido: std::cell::RefCell::new(String::new()),
        };
        let trilha = FakeAudit {
            linhas: Vec::new(),
            ator_pedido: std::cell::RefCell::new(None),
        };
        let erro = crate::block_on(own_history(
            None,
            7,
            &ContaUnica(7),
            &sessoes,
            &trilha,
            &sink,
        ));
        assert_eq!(erro, Err(AdminError::Unauthorized));
        assert!(sessoes.login_pedido.borrow().is_empty());
        let _ = conta_de(7);
    }

    fn session() -> Option<Session> {
        Some(Session {
            account_id: 7,
            role: Role::Player,
        })
    }

    #[test]
    fn recovery_never_reveals_whether_login_exists() {
        let sink = VecSink::default();
        assert_eq!(request_recovery(session(), "existe", &sink), Ok(()));
        assert_eq!(request_recovery(session(), "nao_existe", &sink), Ok(()));
        let entries = sink.entries();
        assert_eq!(entries.len(), 2);
        assert_eq!(entries[0].action, "accounts.recovery");
        assert_eq!(entries[0].target, "login:*", "login não vai para a trilha");
        assert!(!entries[0].detail.contains("existe\""));
    }

    #[test]
    fn recovery_rejects_empty_and_long_login() {
        let sink = VecSink::default();
        assert_eq!(
            request_recovery(session(), "   ", &sink),
            Err(AdminError::InvalidInput {
                reason: "login obrigatório"
            })
        );
        assert_eq!(
            request_recovery(session(), &"x".repeat(17), &sink),
            Err(AdminError::InvalidInput {
                reason: "login inválido"
            })
        );
        assert!(
            sink.entries().is_empty(),
            "validação falha antes de auditar"
        );
    }

    #[test]
    fn without_session_is_unauthorized() {
        let sink = VecSink::default();
        assert_eq!(
            request_recovery(None, "alvo", &sink),
            Err(AdminError::Unauthorized)
        );
        assert!(sink.entries().is_empty());
    }

    #[test]
    fn ranking_limit_is_bounded() {
        let sink = VecSink::default();
        let repo = tests_support::EmptyCharacters;
        let zero = crate::block_on(top_rankings(session(), 0, &repo, &sink));
        assert!(zero.is_err());
        let too_many = crate::block_on(top_rankings(session(), 101, &repo, &sink));
        assert!(too_many.is_err());
    }

    pub(crate) mod tests_support {
        use wyd_db::{Character, Characters, DbError};

        pub struct EmptyCharacters;

        impl Characters for EmptyCharacters {
            async fn list_by_account(&self, _account_id: i64) -> Result<Vec<Character>, DbError> {
                Ok(Vec::new())
            }

            async fn top_by_level(&self, _limit: usize) -> Result<Vec<Character>, DbError> {
                Ok(Vec::new())
            }

            async fn search_by_name(
                &self,
                _term: &str,
                _limit: usize,
            ) -> Result<Vec<Character>, DbError> {
                Ok(Vec::new())
            }

            async fn create_character(&self, _character: &Character) -> Result<Character, DbError> {
                Err(DbError::Transport {
                    op: "fake-não-cria",
                })
            }

            async fn soft_delete(&self, _id: i64) -> Result<(), DbError> {
                Ok(())
            }
        }
    }
}
