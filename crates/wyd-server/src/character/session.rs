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

use std::collections::{BTreeSet, HashMap, VecDeque};
use std::sync::mpsc::{Receiver, Sender};
use std::sync::{Arc, Mutex};
use std::time::Duration;

use wyd_db::{ChannelCapacity, ClaimOutcome, PgBackend, Sessions};

/// TTL mínimo da sessão ociosa: abaixo disso a retomada por TTL viraria corrida.
const MIN_STALE: Duration = Duration::from_secs(30);

/// Uma linha de `_MSG_MessageChat` pronta para entrega.
///
/// `speaker` é o id de cena (`ClientID`/`MobID`) de quem falou: o cliente resolve esse id
/// no mob da vista e monta o nome sozinho (receptor `0x48241f`: busca `MobID` e formata
/// `nome : texto`). Não existe campo de nome no pacote.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChatLine {
    /// Id de cena do falante (`HEADER.ID` do `_MSG_MessageChat`).
    pub speaker: u16,
    /// Texto já recortado em `text[128]`, como o corpo do pacote.
    pub text: [u8; 128],
}

/// Registro de sessões **deste processo** (canal) para o chat social.
///
/// Cada conexão tem uma fila própria: quem fala não escreve no socket de ninguém, apenas
/// entrega a linha para as filas de quem está na vista. O loop da conexão drena a fila e
/// emite com o **próprio** `key_word` — é o que permite um processo atender várias sessões
/// sem compartilhar socket nem escrever de fora da thread dona da conexão.
/// Anti-flood do chat: no máximo [`CHAT_BURST`] falas a cada [`CHAT_WINDOW_MS`].
///
/// Vale só para mensagem social; `/cp`, `/mute` e `/unmute` não consomem cota.
pub const CHAT_BURST: usize = 3;
/// Janela do anti-flood (ms).
pub const CHAT_WINDOW_MS: u64 = 5_000;

/// Uma sessão inscrita: fila, nome do personagem, histórico do anti-flood e ignorados.
#[derive(Debug)]
struct ChatSession {
    tx: Sender<ChatLine>,
    name: [u8; 16],
    recent: VecDeque<u64>,
    muted: BTreeSet<[u8; 16]>,
}

/// Recusa ao silenciar (a resposta vai no `_MSG_MessagePanel`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MuteRefusal {
    /// Tentou silenciar o próprio personagem.
    SelfPlayer,
    /// Nome não está em jogo neste canal.
    NotFound,
}

/// `MobName[16]` normalizado em maiúsculas ASCII (o cliente já usa nomes em caixa alta).
fn nome_16(nome: &str) -> [u8; 16] {
    let mut out = [0u8; 16];
    let bytes = nome.as_bytes();
    let limite = bytes.len().min(out.len());
    for (destino, byte) in out.iter_mut().zip(&bytes[..limite]) {
        *destino = byte.to_ascii_uppercase();
    }
    out
}

fn nome_str(nome: &[u8; 16]) -> String {
    let fim = nome
        .iter()
        .position(|byte| *byte == 0)
        .unwrap_or(nome.len());
    String::from_utf8_lossy(&nome[..fim]).to_string()
}

#[derive(Debug, Default)]
pub struct ChatHub {
    sessions: Mutex<HashMap<u16, ChatSession>>,
}

impl ChatHub {
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Inscreve a sessão pelo id de cena + nome do personagem e devolve a fila e a licença.
    pub fn join(self: &Arc<Self>, id: u16, nome: &str) -> (ChatLease, Receiver<ChatLine>) {
        let (tx, rx) = std::sync::mpsc::channel();
        self.sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner())
            .insert(
                id,
                ChatSession {
                    tx,
                    name: nome_16(nome),
                    recent: VecDeque::new(),
                    muted: BTreeSet::new(),
                },
            );
        (
            ChatLease {
                hub: Arc::clone(self),
                id,
            },
            rx,
        )
    }

    /// Remove a sessão (idempotente).
    pub fn leave(&self, id: u16) {
        self.sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner())
            .remove(&id);
    }

    /// Anti-flood: `None` libera; `Some(ms)` é quanto falta para a janela abrir.
    pub fn allow(&self, id: u16, now_ms: u64) -> Option<u64> {
        let mut sessoes = self
            .sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner());
        let sessao = sessoes.get_mut(&id)?;
        while let Some(&mais_antigo) = sessao.recent.front() {
            if now_ms.saturating_sub(mais_antigo) >= CHAT_WINDOW_MS {
                sessao.recent.pop_front();
            } else {
                break;
            }
        }
        if sessao.recent.len() >= CHAT_BURST {
            let mais_antigo = *sessao.recent.front().unwrap_or(&now_ms);
            return Some((mais_antigo + CHAT_WINDOW_MS).saturating_sub(now_ms).max(1));
        }
        sessao.recent.push_back(now_ms);
        None
    }

    /// Entrega para os ids indicados; devolve quantas filas receberam a linha.
    ///
    /// Id sem sessão (saiu entre a foto da vista e o envio) é ignorado, nunca erro. Quem
    /// silenciou o falante também não recebe — o filtro é do destinatário, não do servidor.
    pub fn deliver(&self, ids: &[u16], line: &ChatLine) -> usize {
        let sessoes = self
            .sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner());
        let Some(falante) = sessoes.get(&line.speaker) else {
            return 0;
        };
        let nome = falante.name;
        let mut entregues = 0;
        for id in ids {
            let Some(destino) = sessoes.get(id) else {
                continue;
            };
            if destino.muted.contains(&nome) {
                continue;
            }
            if destino.tx.send(line.clone()).is_ok() {
                entregues += 1;
            }
        }
        entregues
    }

    /// Silencia o personagem `nome` para a sessão `id`.
    pub fn mute(&self, id: u16, nome: &str) -> Result<(), MuteRefusal> {
        let alvo = nome_16(nome);
        let mut sessoes = self
            .sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner());
        let Some(atual) = sessoes.get(&id) else {
            return Err(MuteRefusal::NotFound);
        };
        if atual.name == alvo {
            return Err(MuteRefusal::SelfPlayer);
        }
        if !sessoes.values().any(|sessao| sessao.name == alvo) {
            return Err(MuteRefusal::NotFound);
        }
        if let Some(atual) = sessoes.get_mut(&id) {
            atual.muted.insert(alvo);
        }
        Ok(())
    }

    /// Volta a ouvir `nome`; `false` = não estava na lista.
    pub fn unmute(&self, id: u16, nome: &str) -> bool {
        let alvo = nome_16(nome);
        self.sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner())
            .get_mut(&id)
            .is_some_and(|sessao| sessao.muted.remove(&alvo))
    }

    /// Lista de silenciados da sessão, em ordem alfabética.
    #[must_use]
    pub fn muted(&self, id: u16) -> Vec<String> {
        self.sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner())
            .get(&id)
            .map(|sessao| sessao.muted.iter().map(nome_str).collect())
            .unwrap_or_default()
    }

    /// Sessões inscritas agora (observabilidade/teste).
    #[must_use]
    pub fn online(&self) -> usize {
        self.sessions
            .lock()
            .unwrap_or_else(|erro| erro.into_inner())
            .len()
    }
}

/// Licença de chat: sai do registro quando a sessão termina, inclusive em erro/panico.
#[derive(Debug)]
pub struct ChatLease {
    hub: Arc<ChatHub>,
    id: u16,
}

impl ChatLease {
    #[must_use]
    pub fn id(&self) -> u16 {
        self.id
    }
}

impl Drop for ChatLease {
    fn drop(&mut self) {
        self.hub.leave(self.id);
    }
}

#[cfg(test)]
mod chat_hub_tests {
    use super::*;

    fn linha(speaker: u16, texto: &[u8]) -> ChatLine {
        let mut body = [0u8; 128];
        body[..texto.len()].copy_from_slice(texto);
        ChatLine {
            speaker,
            text: body,
        }
    }

    #[test]
    fn anti_flood_libera_tres_por_janela_e_devolve_a_espera() {
        let hub = Arc::new(ChatHub::new());
        let (_licenca, _rx) = hub.join(1, "Kae");
        assert_eq!(hub.allow(1, 1_000), None);
        assert_eq!(hub.allow(1, 1_100), None);
        assert_eq!(hub.allow(1, 1_200), None);
        assert_eq!(
            hub.allow(1, 1_300),
            Some(4_700),
            "espera até a primeira fala sair da janela"
        );
        assert_eq!(hub.allow(1, 6_000), None, "janela rolou e liberou de novo");
    }

    #[test]
    fn mute_esconde_a_linha_ate_o_unmute() {
        let hub = Arc::new(ChatHub::new());
        let (_a, _rx_a) = hub.join(1, "Kae");
        let (_b, _rx_b) = hub.join(2, "Pkzinho");
        let fala = linha(2, b"oi");
        assert_eq!(hub.deliver(&[1, 2], &fala), 2);
        assert_eq!(hub.mute(1, "pkzinho"), Ok(()));
        assert_eq!(hub.deliver(&[1], &fala), 0, "silenciado não entrega");
        assert_eq!(hub.muted(1), vec!["PKZINHO".to_string()]);
        assert!(hub.unmute(1, "Pkzinho"));
        assert_eq!(hub.deliver(&[1], &fala), 1);
        assert_eq!(hub.mute(1, "Kae"), Err(MuteRefusal::SelfPlayer));
        assert_eq!(hub.mute(1, "Ninguem"), Err(MuteRefusal::NotFound));
    }
}

/// Lotação de um canal (FASE-06-3): fonte = `channel_capacity` no banco, com
/// sobrescrita por processo via env (`WYD_SERVER_MAX_PLAYERS` e
/// `WYD_SERVER_MAINTENANCE`, lidos no boot em `channels::policy_overrides_from_env`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ChannelPolicy {
    /// Teto de sessões vivas; `1000` é o `MAX_USER` do legado.
    pub max_players: u32,
    /// Canal em manutenção: recusa entrada nova, preserva sessão viva.
    pub maintenance: bool,
}

impl ChannelPolicy {
    /// Política operacional: 1000 sessões, sem manutenção.
    #[must_use]
    pub const fn open() -> Self {
        Self {
            max_players: 1000,
            maintenance: false,
        }
    }
}

impl From<ChannelCapacity> for ChannelPolicy {
    fn from(linha: ChannelCapacity) -> Self {
        Self {
            max_players: linha.max_players,
            maintenance: linha.maintenance,
        }
    }
}

/// Sobrescritas de política por processo (env); `None` = vale o banco.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct PolicyOverrides {
    /// Teto de sessões vivas deste processo, se fixado por env.
    pub max_players: Option<u32>,
    /// Manutenção deste processo, se fixada por env.
    pub maintenance: Option<bool>,
}

/// Falhas da cerca — ou o banco recusou, ou já existe sessão viva, ou o canal
/// está lotado/em manutenção.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SessionError {
    /// A conta já tem sessão viva (este canal ou outro) — recusa explícita, nunca silêncio.
    Conflict {
        /// Canal onde a sessão viva está.
        channel: i16,
    },
    /// Canal lotado: sessões vivas atingiram `max_players` — recusa tipada (FASE-06-3).
    Capacity {
        /// Teto vigente quando a entrada foi recusada.
        limit: u32,
    },
    /// Canal em manutenção: entrada nova recusada, sessão viva preservada.
    Maintenance,
    /// Persistência indisponível: sem banco não se abre sessão (fail-closed).
    Db(String),
}

impl core::fmt::Display for SessionError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Conflict { channel } => {
                write!(f, "conta já tem sessão viva no canal {channel}")
            }
            Self::Capacity { limit } => {
                write!(f, "canal lotado: {limit} sessões vivas")
            }
            Self::Maintenance => write!(f, "canal em manutenção"),
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
    overrides: PolicyOverrides,
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
            overrides: PolicyOverrides::default(),
        }
    }

    /// Fixa sobrescritas de política por processo (env lido no boot; o padrão deixa
    /// tudo valendo o banco).
    #[must_use]
    pub fn with_overrides(mut self, overrides: PolicyOverrides) -> Self {
        self.overrides = overrides;
        self
    }

    /// Política efetiva do canal: banco como fonte, env como sobrescrita.
    fn policy(&self) -> Result<ChannelPolicy, SessionError> {
        let linha = self
            .runtime
            .block_on(self.backend.channel_capacity(self.group_id, self.channel))
            .map_err(|error| SessionError::Db(error.to_string()))?;
        let mut politica = ChannelPolicy::from(linha);
        if let Some(teto) = self.overrides.max_players {
            politica.max_players = teto;
        }
        if let Some(manutencao) = self.overrides.maintenance {
            politica.maintenance = manutencao;
        }
        Ok(politica)
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
        let politica = self.policy()?;
        if politica.maintenance {
            log(&format!(
                "level=warn event=session result=maintenance account={login} group={} channel={}",
                self.group_id, self.channel
            ));
            return Err(SessionError::Maintenance);
        }
        let vivas = self
            .runtime
            .block_on(self.backend.live_sessions(self.group_id, self.channel))
            .map_err(|error| SessionError::Db(error.to_string()))?;
        if u64::from(politica.max_players) <= vivas {
            log(&format!(
                "level=warn event=session result=full account={login} group={} channel={} limit={} live={vivas}",
                self.group_id,
                self.channel,
                politica.max_players
            ));
            return Err(SessionError::Capacity {
                limit: politica.max_players,
            });
        }
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
