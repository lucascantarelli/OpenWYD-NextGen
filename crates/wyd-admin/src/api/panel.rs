//! Painel do jogador (FASE-07-2): conta, personagens, ranking e recuperação.
//!
//! Escopo de leitura: o BFF injeta `account_id` a partir do cookie de sessão assinado — o
//! browser nunca escolhe de quem é o dado. O backend confia no processo do portal (loopback,
//! token processo→processo) e audita autoria + alvo interno em toda leitura.
//!
//! Nada aqui escreve estado vivo: `request_recovery` registra uma **solicitação auditada**;
//! o reset de senha continua ato de staff (FASE-07-3), porque não há provedor de e-mail
//! aprovado nem tabela de token de recuperação.

use wyd_db::{Accounts, Characters, Role};

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
