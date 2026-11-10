//! Cerca de sessão do canal: uma sessão viva por conta, entre todos os canais do grupo.
//!
//! O canal é um **processo** (ADR-0022 §7) e o postulado do jogo é que cada canal é um mundo
//! separado — mas o personagem é o mesmo e mora no Postgres compartilhado. Sem cerca, o
//! mesmo personagem entra no canal 1 e no canal 2 e passa a existir em dois mundos.
//!
//! A garantia não pode ser memória local: quem decide é o índice parcial
//! `character_session_uma_por_conta` (`crates/wyd-db/migrations/0030_sessao_unica.sql`). Esta
//! camada só traduz o resultado para o vocabulário do handshake e mantém o sinal de vida
//! (`renewed_at`), que é o que impede um processo morto de prender o jogador para sempre.
//!
//! Transitar entre canais continua trivial: sair libera a sessão, e a entrada no outro canal
//! relê o personagem do banco — nada é copiado, nada é perdido.

use std::sync::Arc;
use std::time::Duration;

use wyd_db::{ClaimOutcome, PgBackend, Sessions};

/// TTL mínimo da sessão ociosa: abaixo disso a retomada por TTL viraria corrida.
const MIN_STALE: Duration = Duration::from_secs(30);

/// Falhas da cerca — ou o banco recusou, ou já existe sessão viva.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SessionError {
    /// A conta já tem sessão viva (este canal ou outro) — recusa explícita, nunca silêncio.
    Conflict {
        /// Canal onde a sessão viva está.
        channel: i16,
    },
    /// Persistência indisponível: sem banco não se abre sessão (fail-closed).
    Db(String),
}

impl core::fmt::Display for SessionError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Conflict { channel } => {
                write!(f, "conta já tem sessão viva no canal {channel}")
            }
            Self::Db(motivo) => write!(f, "sessão sem lastro no banco: {motivo}"),
        }
    }
}

impl std::error::Error for SessionError {}

/// Cerca de sessão de **um** canal (o grupo e o canal vêm da configuração do processo).
pub struct SessionGate {
    backend: Arc<PgBackend>,
    runtime: Arc<tokio::runtime::Runtime>,
    group_id: i16,
    channel: i16,
    stale_after: Duration,
    idle: Duration,
}

impl SessionGate {
    /// Monta a cerca do processo; `idle` é o tempo de ociosidade que renova sem fechar.
    #[must_use]
    pub fn new(
        backend: Arc<PgBackend>,
        runtime: Arc<tokio::runtime::Runtime>,
        group_id: i16,
        channel: i16,
        idle: Duration,
        stale_after: Duration,
    ) -> Self {
        Self {
            backend,
            runtime,
            group_id,
            channel,
            stale_after: stale_after.max(MIN_STALE),
            idle,
        }
    }

    /// Canal deste processo (vai para o log).
    #[must_use]
    pub fn channel(&self) -> i16 {
        self.channel
    }

    /// Grupo (servidor) deste processo.
    #[must_use]
    pub fn group_id(&self) -> i16 {
        self.group_id
    }

    /// Ociosidade tolerada antes de o servidor soltar a conexão.
    #[must_use]
    pub fn idle(&self) -> Duration {
        self.idle
    }

    /// Abre a sessão da conta **ou** recusa com o canal que já a tem.
    ///
    /// Conflito no **próprio** canal é retomada: o cliente reconectou (a conexão anterior já
    /// era), então a linha antiga é liberada e a nova abre. Conflito em outro canal continua
    /// recusa — ali há dois mundos com o mesmo personagem, que é o que a cerca protege.
    pub fn claim(&self, login: &str, client_version: i32) -> Result<SessionLease, SessionError> {
        match self.claim_once(login, client_version)? {
            ClaimOutcome::Acquired { id } => Ok(self.lease(id)),
            ClaimOutcome::Conflict { channel } if channel == self.channel => {
                // Retomada do mesmo canal: libera a linha abandonada e tenta de novo.
                let liberadas = self
                    .runtime
                    .block_on(self.backend.release_account_sessions(login))
                    .map_err(|error| SessionError::Db(error.to_string()))?;
                log(&format!(
                    "level=info event=session result=takeover account={login} group={} channel={} released={liberadas}",
                    self.group_id, self.channel
                ));
                match self.claim_once(login, client_version)? {
                    ClaimOutcome::Acquired { id } => Ok(self.lease(id)),
                    ClaimOutcome::Conflict { channel } => Err(SessionError::Conflict { channel }),
                }
            }
            ClaimOutcome::Conflict { channel } => Err(SessionError::Conflict { channel }),
        }
    }

    /// Uma tentativa de abertura (expira órfã e tenta o índice único).
    fn claim_once(&self, login: &str, client_version: i32) -> Result<ClaimOutcome, SessionError> {
        let stale = i64::try_from(self.stale_after.as_secs()).unwrap_or(i64::MAX);
        self.runtime
            .block_on(self.backend.claim_session(
                login,
                self.group_id,
                self.channel,
                client_version,
                stale,
            ))
            .map_err(|error| SessionError::Db(error.to_string()))
    }

    fn lease(&self, id: i64) -> SessionLease {
        SessionLease {
            backend: Arc::clone(&self.backend),
            runtime: Arc::clone(&self.runtime),
            id,
        }
    }
}

/// Log estruturado (`key=value`) no mesmo formato do canal.
fn log(linha: &str) {
    crate::logs::record(linha);
    println!("{linha}");
}

/// Sessão aberta por este processo; liberar é automático ao sair de escopo.
#[derive(Debug)]
pub struct SessionLease {
    backend: Arc<PgBackend>,
    runtime: Arc<tokio::runtime::Runtime>,
    id: i64,
}

impl SessionLease {
    /// Identificador da linha em `character_session`.
    #[must_use]
    pub fn id(&self) -> i64 {
        self.id
    }

    /// Sinal de vida: sem renovação a sessão expira pelo TTL e outro canal a retoma.
    pub fn renew(&self) {
        // Falha de renovação não derruba a partida: o TTL é a rede de segurança e o
        // próximo sinal tenta de novo (log fica no adapter, uma vez por operação).
        let _ = self.runtime.block_on(self.backend.renew_session(self.id));
    }

    /// Encerra a sessão (logout/queda) — o que libera a entrada em outro canal.
    pub fn release(&self) {
        let _ = self.runtime.block_on(self.backend.release_session(self.id));
    }
}

impl Drop for SessionLease {
    fn drop(&mut self) {
        self.release();
    }
}
