//! Porta de sessão — garante **uma** sessão viva por conta entre os canais do grupo.
//!
//! O canal é processo separado, mas o Postgres é o mesmo (`docs/wyd/canais-e-cidadania.md`
//! §5): a exclusividade não pode viver em memória de um deles. A porta não conhece SQL; o
//! adapter `pg` é quem usa o índice parcial `character_session_uma_por_conta`.
//!
//! Duas operações e uma consequência:
//!
//! - `claim_session` abre a sessão **ou** devolve o canal que já a tem;
//! - `renew_session` é o sinal de vida do processo dono (sessão órfã expira pelo TTL e é
//!   retomada — sem isso um crash deixaria o jogador travado fora do jogo);
//! - `release_session` encerra (logout, queda da conexão), e é o que permite **transitar**
//!   entre canais mantendo os dados: nada é copiado, o personagem só volta a ser lido do
//!   banco no próximo canal.

use std::future::Future;

use crate::error::DbError;
use crate::repo::content::{ChannelCapacity, ChannelRole};

/// Resultado de uma tentativa de abrir sessão.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ClaimOutcome {
    /// Sessão aberta por este processo (`id` é a chave para renovar/liberar).
    Acquired { id: i64 },
    /// Outro processo (ou outra conexão) já tem a sessão viva desta conta.
    Conflict {
        /// Canal onde a sessão viva está — vai para o log e para a recusa do cliente.
        channel: i16,
    },
}

/// Operações sobre `character_session` (uma por fronteira, erros tipados).
pub trait Sessions {
    /// Abre a sessão da conta neste canal, ou reporta o conflito.
    ///
    /// Sessão viva cujo dono parou de renovar há mais de `stale_after_secs` é **retomada**:
    /// crash de um canal não pode prender o jogador (`renewed_at` é a evidência).
    fn claim_session(
        &self,
        login: &str,
        group_id: i16,
        channel: i16,
        client_version: i32,
        stale_after_secs: i64,
    ) -> impl Future<Output = Result<ClaimOutcome, DbError>> + Send;

    /// Renova o sinal de vida da sessão (ociosidade do cliente não expira a sessão).
    fn renew_session(&self, id: i64) -> impl Future<Output = Result<(), DbError>> + Send;

    /// Encerra a sessão; idempotente (sessão já liberada não é erro).
    fn release_session(&self, id: i64) -> impl Future<Output = Result<(), DbError>> + Send;

    /// Libera **todas** as sessões vivas da conta e devolve quantas fechou.
    ///
    /// Usado só na retomada do mesmo canal: se o cliente reconectou, a conexão anterior já
    /// era e a linha antiga não pode impedir a nova. Sessão de **outro** canal não passa por
    /// aqui — trocar de canal continua exigindo sair do primeiro.
    fn release_account_sessions(
        &self,
        login: &str,
    ) -> impl Future<Output = Result<u64, DbError>> + Send;

    /// Canal da sessão viva da conta, se houver (diagnóstico e teste de logout por conta).
    fn live_session_of(
        &self,
        login: &str,
    ) -> impl Future<Output = Result<Option<i16>, DbError>> + Send;

    /// Quantas sessões vivas existem hoje no canal (diagnóstico e `serv00.htm` futuro).
    fn live_sessions(
        &self,
        group_id: i16,
        channel: i16,
    ) -> impl Future<Output = Result<u64, DbError>> + Send;

    /// Reaper: libera linhas de `character_session` com `renewed_at` vencido (desconexão
    /// abrupta sem `Drop`/`leave`). Devolve quantas linhas foram liberadas.
    fn release_stale_sessions(
        &self,
        ttl_secs: i64,
    ) -> impl Future<Output = Result<u64, DbError>> + Send;

    /// Lotação e manutenção do canal (FASE-06-3); linha ausente = [`ChannelCapacity::permissive`].
    fn channel_capacity(
        &self,
        group_id: i16,
        channel: i16,
    ) -> impl Future<Output = Result<ChannelCapacity, DbError>> + Send;

    /// Papel do canal (FASE-03-10); linha ausente = `None` (o chamador assume `normal`).
    fn channel_role(
        &self,
        group_id: i16,
        channel: i16,
    ) -> impl Future<Output = Result<Option<ChannelRole>, DbError>> + Send;
}
