//! Transporte do Cold Path (ADR-0008): HTTP/1.1 + JSON em loopback.
//!
//! Roteia `POST /admin/v1/{servico}/{metodo}` para os endpoints de `api/*`,
//! autentica pelo token de staff, aplica o envelope `AdminResult` e audita toda
//! chamada autenticada. Nunca é importado pelo Hot Path (`wyd-core`/`wyd-net`).
//!
//! Enquanto a ADR-0006 mantém o runtime async diferido, os handlers usam futuros
//! prontos no primeiro poll (`poll_once`); um futuro pendente responde `500` em
//! vez de travar a conexão.

pub mod http;
pub mod json;

use std::future::Future;
use std::io::BufReader;
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::task::{Context, Poll, Waker};

use wyd_db::{
    AccountModeration, Accounts, AuditRow, AuditTrail, CatalogSnapshot, Characters, Economy,
    EconomySummary, Role,
};

use crate::api::{
    create_account, economy_summary, find_account, moderation, own_account, own_characters,
    own_history, recent_audit, request_grant, request_recovery, set_blocked, set_role,
    top_rankings, AccountSummary, ChannelPresence, ConfigStore, GrantKind, GrantRequest,
    PanelAccount, PanelCharacter, PanelEvent, PanelSession, RankingEntry,
};
use crate::audit::{log_action, log_denied, log_panel, AuditSink};
use crate::auth::{Session, StaffSession};
use crate::error::AdminError;
use crate::service::{GrantService, GrantTrail};

/// Porta do contrato no loopback (ADR-0008).
pub const ADMIN_PORT: u16 = 7600;

/// Superfície declarada pelo portal em `wyd-portal/src/server/admin-client.ts`.
///
/// Cada par é `servico` → métodos já convertidos para o formato da rota
/// (`camelCase` do TS → `snake_case`, como o `toSnake` do cliente). A lista existe
/// para que **nenhuma** chamada do portal caia em `404` por rota ausente: método
/// declarado sem handler responde `501` e é auditado (`admin.pending`), enquanto
/// serviço/método fora daqui continua `404` (nome inválido).
///
/// Paridade é testada, não confiada: `tests/contract.rs` extrai a mesma lista do
/// TypeScript e falha se o portal passar a declarar um método que o roteador não
/// conhece.
pub const SERVICOS: &[(&str, &[&str])] = &[
    (
        "account_admin",
        &[
            "list_accounts",
            "get_account",
            "get_moderation",
            "set_role",
            "set_blocked",
            "create_account",
        ],
    ),
    // Trilha de auditoria (somente leitura, append-only) e economia auditada.
    ("audit_admin", &["recent"]),
    ("economy_admin", &["summary"]),
    ("account_web", &["verify_credentials", "change_numeric_pin"]),
    (
        "attribute_map_admin",
        &["get_attribute_map_info", "transform_attribute_map"],
    ),
    (
        "character_admin",
        &[
            "search_characters",
            "get_character",
            "grant_item",
            "list_character_grants",
        ],
    ),
    (
        "content_rates",
        &["get_rates", "upsert_rate", "reset_rate", "get_server_flags"],
    ),
    (
        "daily_reward_admin",
        &[
            "list_reward_items",
            "upsert_reward_item",
            "set_reward_item_enabled",
            "delete_reward_item",
        ],
    ),
    (
        "donate_admin",
        &[
            "list_shop_items",
            "upsert_shop_item",
            "set_shop_item_enabled",
            "delete_shop_item",
            "credit_donate_balance",
        ],
    ),
    (
        "donate_revenue_admin",
        &[
            "get_revenue_summary",
            "list_topup_orders",
            "list_top_buyers",
            "list_donate_spend",
            "search_accounts",
        ],
    ),
    (
        "drop_admin",
        &[
            "list_drop_rates",
            "upsert_drop_rate",
            "reset_drop_rate",
            "list_mob_carry",
            "set_mob_carry",
            "reset_mob_carry",
        ],
    ),
    ("item_catalog", &["list_items"]),
    (
        "item_effect_admin",
        &["list_effects", "upsert_effect", "delete_effect"],
    ),
    (
        "mob_admin",
        &[
            "list_mob_templates",
            "get_mob_template_stat",
            "upsert_mob_template_stat",
            "set_mob_template_equip",
            "delete_mob_template_stat",
        ],
    ),
    (
        "npc_admin",
        &[
            "list_npcs",
            "get_npc",
            "upsert_npc",
            "set_npc_visibility",
            "set_npc_shop",
            "delete_npc",
            "list_merchant_templates",
            "list_map_zones",
            "list_item_prices",
            "set_item_price",
            "list_drop_items",
            "list_mob_drops",
        ],
    ),
    (
        "teleport_admin",
        &[
            "list_routes",
            "upsert_route",
            "set_route_enabled",
            "delete_route",
            "simulate_route",
        ],
    ),
    (
        "world_event_admin",
        &["get_world_event_config", "set_world_event_config"],
    ),
    (
        "world_state_admin",
        &[
            "get_world_state",
            "set_city_tax",
            "set_kefra_defeated",
            "set_guild_master",
            "remove_guild_member",
            "set_guild_relation",
            "set_zone_owner",
            "set_tower_owner",
        ],
    ),
    // Leitura auditada da área de monitoramento: presença viva por canal. Nunca
    // escrita — o portal não toca estado de personagem (AGENTS.md invariante 7).
    ("server_admin", &["list_channel_presence"]),
];

/// `true` quando o contrato do portal declara `servico/metodo`.
#[must_use]
pub fn conhecido(service: &str, method: &str) -> bool {
    SERVICOS
        .iter()
        .any(|(nome, metodos)| *nome == service && metodos.contains(&method))
}

/// `AdminResult::OK` do portal (`admin-result.ts`).
const RESULT_OK: u8 = 1;
/// `AdminResult::FORBIDDEN`.
const RESULT_FORBIDDEN: u8 = 2;
/// `AdminResult::INVALID`.
const RESULT_INVALID: u8 = 3;
/// `AdminResult::NOT_FOUND`.
const RESULT_NOT_FOUND: u8 = 4;
/// `AdminResult::CONTENT_OWNED`.
const RESULT_CONTENT_OWNED: u8 = 5;

/// Verificação do token de sessão de staff (ADR-0008 §2).
pub trait Authenticator {
    /// Devolve a sessão para um token válido; `None` é `401`.
    fn authenticate(&self, token: &str) -> Option<Session>;
}

/// Registro explícito token → sessão (dev/testes).
///
/// A verificação HMAC do JWT emitido pelo portal entra na fiação do BFF; o
/// transporte não decide sozinho quem é staff.
#[derive(Debug, Default)]
pub struct StaticTokens {
    entries: Vec<(String, Session)>,
}

impl StaticTokens {
    /// Registra um token válido para a sessão indicada.
    pub fn insert(&mut self, token: impl Into<String>, session: Session) {
        self.entries.push((token.into(), session));
    }
}

impl Authenticator for StaticTokens {
    fn authenticate(&self, token: &str) -> Option<Session> {
        self.entries
            .iter()
            .find(|(known, _)| known == token)
            .map(|(_, session)| *session)
    }
}

/// Resposta pronta para o socket.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Reply {
    pub status: u16,
    pub body: String,
    pub trace_id: String,
}

impl Reply {
    fn ok(trace: &str, extras: &str) -> Self {
        let quoted = json::quote(trace);
        Self {
            status: 200,
            body: format!("{{\"result\":{RESULT_OK}{extras},\"trace_id\":{quoted}}}"),
            trace_id: trace.to_string(),
        }
    }

    fn business(trace: &str, result: u8, message: &str) -> Self {
        let quoted = json::quote(trace);
        let message = json::quote(message);
        Self {
            status: 200,
            body: format!("{{\"result\":{result},\"message\":{message},\"trace_id\":{quoted}}}"),
            trace_id: trace.to_string(),
        }
    }

    fn transport(status: u16, trace: &str, message: &str) -> Self {
        let quoted = json::quote(trace);
        let message = json::quote(message);
        Self {
            status,
            body: format!("{{\"message\":{message},\"trace_id\":{quoted}}}"),
            trace_id: trace.to_string(),
        }
    }

    fn from_error(trace: &str, error: &AdminError) -> Self {
        if matches!(error, AdminError::Unauthorized) {
            return Self::transport(401, trace, "sessão de staff ausente ou inválida");
        }
        Self::business(trace, admin_result(error), &error.to_string())
    }
}

const fn admin_result(error: &AdminError) -> u8 {
    match error {
        AdminError::Forbidden { .. } => RESULT_FORBIDDEN,
        AdminError::InvalidInput { .. } => RESULT_INVALID,
        AdminError::NotFound { .. } => RESULT_NOT_FOUND,
        AdminError::Conflict { .. } => RESULT_CONTENT_OWNED,
        AdminError::Unauthorized => 0,
    }
}

/// Verificação de login **síncrona** já resolvida pelo binário.
///
/// O roteador não tem runtime (ADR-0006): o gateway passa aqui uma função que
/// consulta o banco com `block_on` e verifica argon2id, devolvendo
/// `(account_id, login, role)` só quando a senha confere.
pub type LoginCheck = std::sync::Arc<dyn Fn(&str, &str) -> Option<LoginOk> + Send + Sync>;

/// Conta autenticada devolvida pelo [`LoginCheck`] (sem hash).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LoginOk {
    pub account_id: i64,
    pub login: String,
    pub role: Role,
    /// Senha correta, mas moderação bloqueou a conta.
    ///
    /// Só quem acertou a senha vê essa distinção: senha errada e conta
    /// inexistente continuam na mesma recusa genérica.
    pub blocked: bool,
}

/// Header com o ator autenticado pelo BFF (`id:role`).
///
/// O token de staff autentica o **processo** (portal); o ator é a conta de
/// staff que fez a chamada, derivada do cookie assinado no portal. Sem o header
/// vale a identidade do próprio token (serviço). Header malformado é `400`:
/// cair para o serviço mascararia bug de fiação e ampliaria privilégio.
const ACTOR_HEADER: &str = "x-wyd-actor";

/// Resolve o ator declarado pelo BFF (ou `None` quando o header está ausente).
fn actor_from(request: &http::Request) -> Result<Option<Session>, ()> {
    let Some(raw) = request.header(ACTOR_HEADER) else {
        return Ok(None);
    };
    let Some((id, role)) = raw.split_once(':') else {
        return Err(());
    };
    let Ok(account_id) = id.trim().parse::<i64>() else {
        return Err(());
    };
    let Some(role) = role_from_str(role.trim()) else {
        return Err(());
    };
    if account_id <= 0 {
        return Err(());
    }
    Ok(Some(Session { account_id, role }))
}

/// Presença por canal já resolvida pelo **binário**.
///
/// O roteador não tem runtime (ADR-0006): o gateway passa um fecho que lê
/// `character_session` com `block_on` e devolve uma linha por canal. O roteador só
/// autentica (staff), audita e serializa — nada de SQL no Cold Path do transporte.
pub type PresenceCheck =
    std::sync::Arc<dyn Fn() -> Result<Vec<ChannelPresence>, String> + Send + Sync>;

/// Roteador do contrato: `servico/metodo` → `api/*`.
///
/// Sem `derive(Debug)` de propósito: `login` é `Arc<dyn Fn>` e imprimir fecho
/// não ajuda em diagnóstico nenhum.
pub struct Router<A, C, S> {
    accounts: A,
    characters: C,
    audit: S,
    grants: GrantService,
    tokens: StaticTokens,
    catalog: Option<CatalogSnapshot>,
    login: Option<LoginCheck>,
    presence: Option<PresenceCheck>,
    /// Trilha de auditoria do banco (leitura do painel de staff).
    audit_trail: Option<Arc<dyn AuditTrail + Send + Sync>>,
    /// Fontes reais da economia (`city_tax_ledger`, `delivery_queue`).
    economy: Option<Arc<dyn Economy + Send + Sync>>,
    /// Leitura da fila de entregas para a tela de staff (somente leitura).
    grant_trail: Option<Arc<dyn GrantTrail>>,
    /// `host:porta` do CPSock do servidor de jogo (troca de PIN pelo portal).
    game: Option<String>,
    /// Configuração editável (taxas, drops, NPC, teleporte) — nunca estado vivo.
    config: Option<Arc<dyn ConfigStore>>,
    /// Histórico de sessões do jogador (`character_session`, leitura pura).
    session_trail: Option<Arc<dyn wyd_db::SessionTrail + Send + Sync>>,
    max_body: usize,
}

impl<A: Accounts, C: Characters, S: AuditSink> Router<A, C, S> {
    /// Monta o roteador com as portas de persistência e auditoria.
    #[must_use]
    pub fn new(accounts: A, characters: C, audit: S, tokens: StaticTokens) -> Self {
        Self {
            accounts,
            characters,
            audit,
            grants: GrantService::new(),
            tokens,
            catalog: None,
            login: None,
            presence: None,
            audit_trail: None,
            economy: None,
            grant_trail: None,
            game: None,
            config: None,
            session_trail: None,
            max_body: http::MAX_BODY,
        }
    }

    /// Liga a leitura da fila de entregas (tela de staff; nunca escreve).
    #[must_use]
    pub fn with_grant_trail(mut self, trail: Arc<dyn GrantTrail>) -> Self {
        self.grant_trail = Some(trail);
        self
    }

    /// Liga a verificação de login (gateway monta com o pool; testes com fake).
    #[must_use]
    pub fn with_login_check(mut self, check: LoginCheck) -> Self {
        self.login = Some(check);
        self
    }

    /// Liga a leitura de presença por canal (fecho com o pool, como o login).
    #[must_use]
    pub fn with_presence_check(mut self, check: PresenceCheck) -> Self {
        self.presence = Some(check);
        self
    }

    /// Liga a trilha de auditoria (`audit_log`) para leitura no painel.
    ///
    /// Sem ela a rota responde `503`: o painel nunca mostra trilha inventada.
    #[must_use]
    pub fn with_audit_trail(mut self, trail: Arc<dyn AuditTrail + Send + Sync>) -> Self {
        self.audit_trail = Some(trail);
        self
    }

    /// Liga as fontes reais da economia (livro-caixa e fila de entrega).
    /// Liga a leitura do histórico de sessões (área do jogador).
    ///
    /// Sem ela `panel/history` responde `503`: a tela não inventa sessão.
    #[must_use]
    pub fn with_session_trail(
        mut self,
        trail: Arc<dyn wyd_db::SessionTrail + Send + Sync>,
    ) -> Self {
        self.session_trail = Some(trail);
        self
    }

    #[must_use]
    pub fn with_economy(mut self, economy: Arc<dyn Economy + Send + Sync>) -> Self {
        self.economy = Some(economy);
        self
    }

    /// Endereço CPSock do servidor de jogo usado pela troca de PIN do portal.
    ///
    /// Sem ele `account_web/change_pin` responde `503`: o gateway não inventa um
    /// caminho alternativo para gravar hash sem passar pelo servidor.
    #[must_use]
    pub fn with_game_address(mut self, address: impl Into<String>) -> Self {
        self.game = Some(address.into());
        self
    }

    /// Foto do catálogo lida do banco no boot do gateway (ADR-0022).
    ///
    /// Sem ela `item_catalog/list_items` responde `503` — nunca lista vazia inventada.
    #[must_use]
    pub fn with_catalog(mut self, catalog: CatalogSnapshot) -> Self {
        self.catalog = Some(catalog);
        self
    }

    /// Liga a configuração editável (WEB-06).
    ///
    /// Sem ela, as rotas de taxa/drop/NPC/teleporte respondem `503`: o gateway não
    /// inventa leitura nem escrita de config sem a porta montada.
    #[must_use]
    pub fn with_config(mut self, config: Arc<dyn ConfigStore>) -> Self {
        self.config = Some(config);
        self
    }

    /// Limite de corpo (o teste usa valor pequeno para provar o `413`).
    #[must_use]
    pub fn with_max_body(mut self, max_body: usize) -> Self {
        self.max_body = max_body;
        self
    }

    /// Trilha de auditoria (testes leem/verificam o que foi registrado).
    #[must_use]
    pub fn audit(&self) -> &S {
        &self.audit
    }

    /// Conecta a fila durável dos grants (uma vez, na configuração).
    #[must_use]
    pub fn with_grant_queue(self, queue: std::sync::Arc<dyn crate::service::GrantQueue>) -> Self {
        self.grants.attach_queue(queue);
        self
    }

    #[must_use]
    pub fn max_body(&self) -> usize {
        self.max_body
    }

    /// Resolve uma requisição já lida (auth → rota → handler → envelope).
    #[must_use]
    pub fn dispatch(&self, request: &http::Request) -> Reply {
        let trace = request.trace_id();
        if request.method != "POST" {
            return Reply::transport(405, trace, "método não permitido");
        }
        let Some(token) = bearer(request.header("authorization")) else {
            return Reply::transport(401, trace, "sessão de staff ausente");
        };
        let Some(session) = self.tokens.authenticate(token) else {
            return Reply::transport(401, trace, "sessão de staff inválida");
        };
        let session = match actor_from(request) {
            Ok(Some(actor)) => actor,
            Ok(None) => session,
            Err(()) => return Reply::transport(400, trace, "x-wyd-actor inválido"),
        };
        let Ok(body) = std::str::from_utf8(&request.body) else {
            return Reply::transport(400, trace, "corpo não é UTF-8");
        };
        let Ok(params) = json::Json::parse(body) else {
            return Reply::transport(400, trace, "JSON inválido");
        };
        let Some((service, method)) = route(&request.path) else {
            return Reply::transport(404, trace, "método desconhecido");
        };
        match (service, method) {
            ("accounts", "find") => self.accounts_find(session, &params, trace),
            ("accounts", "create") => self.accounts_create(session, &params, trace),
            ("accounts", "recovery") => self.accounts_recovery(session, &params, trace),
            ("accounts", "get_moderation") => self.accounts_moderation(session, &params, trace),
            ("accounts", "set_role") => self.accounts_set_role(session, &params, trace),
            ("accounts", "set_blocked") => self.accounts_set_blocked(session, &params, trace),
            ("audit_admin", "recent") => self.audit_recent(session, &params, trace),
            ("economy_admin", "summary") => self.economy_summary(session, trace),
            ("grants", "request") => self.grants_request(session, &params, trace),
            ("character_admin", "search_characters") => {
                self.character_search(session, &params, trace)
            }
            ("character_admin", "list_character_grants") => {
                self.character_grants(session, &params, trace)
            }
            ("panel", "self") => self.panel_self(session, &params, trace),
            ("panel", "characters") => self.panel_characters(session, &params, trace),
            ("panel", "rankings") => self.panel_rankings(session, &params, trace),
            ("panel", "history") => self.panel_history(session, &params, trace),
            ("item_catalog", "list_items") => self.catalog_list_items(session, trace),
            ("account_web", "verify_credentials") => self.verify_credentials(&params, trace),
            ("account_web", "change_numeric_pin") => self.change_pin(&params, trace),
            ("server_admin", "list_channel_presence") => self.presence_list(session, trace),
            ("content_rates", "get_rates") => self.config_rates(session, trace),
            ("content_rates", "get_server_flags") => self.config_flags(session, trace),
            ("content_rates", "upsert_rate") => self.config_upsert_rate(session, &params, trace),
            ("content_rates", "reset_rate") => self.config_reset_rate(session, &params, trace),
            ("drop_admin", "list_drop_rates") => self.config_drops(session, &params, trace),
            ("drop_admin", "upsert_drop_rate") => self.config_set_drop(session, &params, trace),
            ("drop_admin", "reset_drop_rate") => self.config_reset_drop(session, &params, trace),
            ("npc_admin", "get_npc") => self.config_npc(session, &params, trace),
            ("npc_admin", "list_npcs") => self.config_npc_page(session, &params, trace),
            ("npc_admin", "upsert_npc") => self.config_set_npc(session, &params, trace),
            ("npc_admin", "list_item_prices") => self.config_item_prices(session, &params, trace),
            ("npc_admin", "set_item_price") => self.config_set_item_price(session, &params, trace),
            ("mob_admin", "list_mob_templates") => {
                self.config_mob_templates(session, &params, trace)
            }
            ("item_effect_admin", "list_effects") => self.config_effects(session, &params, trace),
            ("daily_reward_admin", "list_reward_items") => {
                self.config_daily_list(session, &params, trace)
            }
            ("daily_reward_admin", "upsert_reward_item") => {
                self.config_daily_upsert(session, &params, trace)
            }
            ("daily_reward_admin", "delete_reward_item") => {
                self.config_daily_delete(session, &params, trace)
            }
            ("donate_admin", "list_shop_items") => self.config_donate_list(session, &params, trace),
            ("donate_admin", "upsert_shop_item") => {
                self.config_donate_upsert(session, &params, trace)
            }
            ("donate_admin", "set_shop_item_enabled") => {
                self.config_donate_enabled(session, &params, trace)
            }
            ("donate_admin", "delete_shop_item") => {
                self.config_donate_delete(session, &params, trace)
            }
            ("donate_admin", "credit_donate_balance") => {
                self.config_donate_credit(session, &params, trace)
            }
            ("donate_admin", "debit_donate_balance") => {
                self.config_donate_debit(session, &params, trace)
            }
            ("donate_revenue_admin", "get_revenue_summary") => {
                self.config_donate_revenue(session, trace)
            }
            ("donate_revenue_admin", "list_topup_orders") => {
                self.config_donate_topups(session, &params, trace)
            }
            ("donate_revenue_admin", "list_donate_spend") => {
                self.config_donate_spend(session, &params, trace)
            }
            ("donate_revenue_admin", "list_top_buyers") => {
                self.config_donate_buyers(session, &params, trace)
            }
            ("donate_revenue_admin", "search_accounts") => {
                self.config_donate_search(session, &params, trace)
            }
            ("world_event_admin", "get_world_event_config") => {
                self.config_event_get(session, trace)
            }
            ("world_event_admin", "set_world_event_config") => {
                self.config_event_set(session, &params, trace)
            }
            ("account_admin", "list_accounts") => self.config_account_list(session, &params, trace),
            ("account_admin", "get_account") => self.config_account_get(session, &params, trace),
            ("character_admin", "grant_item") => self.character_grant_item(session, &params, trace),
            ("teleport_admin", "list_routes") => self.config_routes(session, &params, trace),
            ("teleport_admin", "upsert_route") => self.config_set_route(session, &params, trace),
            ("teleport_admin", "set_route_enabled") => {
                self.config_set_route(session, &params, trace)
            }
            _ if conhecido(service, method) => self.pending(session, service, method, trace),
            _ => Reply::transport(404, trace, "método desconhecido"),
        }
    }

    /// `server_admin/list_channel_presence` — leitura de presença por canal.
    ///
    /// Staff obrigatório; a leitura entra na trilha de auditoria. É **somente leitura**:
    /// a origem é `character_session`, mantida pelo servidor de jogo, não por aqui.
    fn presence_list(&self, session: Session, trace: &str) -> Reply {
        let staff = match StaffSession::require(session) {
            Ok(staff) => staff,
            Err(error) => {
                log_denied(
                    &self.audit,
                    session.account_id,
                    session.role,
                    "server.presence",
                    "sem papel de staff",
                );
                return Reply::from_error(trace, &error);
            }
        };
        let Some(check) = &self.presence else {
            return Reply::transport(503, trace, "presença por canal não montada no gateway");
        };
        match check() {
            Ok(channels) => {
                log_action(
                    &self.audit,
                    &staff,
                    "server.presence",
                    "channels",
                    "leitura",
                );
                Reply::ok(
                    trace,
                    &format!(",\"channels\":{}", presence_json(&channels)),
                )
            }
            Err(_) => Reply::transport(503, trace, "banco indisponível para presença"),
        }
    }

    /// `account_web/change_pin` — troca de PIN do portal pelo fluxo do jogo.
    ///
    /// O gateway abre uma conexão CPSock real e faz `0x0FDE` em duas etapas
    /// (`change_numeric` 1 com o PIN atual, 2 com o novo), como o cliente 7662. A
    /// senha da conta é exigida porque o `0x0FDE` só existe depois do `AccountLogin`;
    /// não há atalho que grave hash sem passar pelo servidor de jogo.
    fn change_pin(&self, params: &json::Json, trace: &str) -> Reply {
        let Some(address) = &self.game else {
            return Reply::transport(503, trace, "servidor de jogo não configurado no gateway");
        };
        let campo = |nome: &str| params.get(nome).and_then(json::Json::as_str);
        let (Some(login), Some(senha), Some(atual), Some(novo)) = (
            campo("login"),
            campo("password"),
            campo("current_pin"),
            campo("new_pin"),
        ) else {
            return Reply::transport(
                400,
                trace,
                "login, password, current_pin e new_pin são obrigatórios",
            );
        };
        use crate::service::pin::PinError;
        match crate::service::pin::change_pin(address, login, senha, atual, novo) {
            Ok(()) => Reply::ok(trace, ",\"changed\":true"),
            Err(PinError::InvalidPin) => {
                Reply::transport(422, trace, "PIN deve ter de 4 a 6 dígitos")
            }
            Err(PinError::Rejected) => Reply::transport(
                400,
                trace,
                "PIN atual inválido ou conta bloqueada por tentativas",
            ),
            Err(PinError::Unavailable) => {
                Reply::transport(503, trace, "servidor de jogo indisponível")
            }
            Err(PinError::Protocol) => {
                Reply::transport(502, trace, "resposta inesperada do servidor de jogo")
            }
        }
    }

    /// `account_web/verify_credentials` — login do portal (contrato `VerifyCredentialsResponse`).
    ///
    /// A senha só existe aqui: compara argon2id com o hash do banco e devolve
    /// `ok`/`role`, nunca o hash. Recusa de senha **e** de conta inexistente usam a
    /// mesma resposta (`ok:false`), para não vazar existência de login.
    fn verify_credentials(&self, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("name").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo name obrigatório");
        };
        let Some(password) = params.get("password").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo password obrigatório");
        };
        let recusa = || {
            Reply::ok(
                trace,
                ",\"ok\":false,\"account_id\":0,\"blocked\":false,\"role\":\"\"",
            )
        };
        let Some(check) = &self.login else {
            return Reply::transport(503, trace, "verificação de login não montada no gateway");
        };
        let Some(account) = check(login, password) else {
            // Sem senha no log: só o login tentado e o resultado.
            log_panel(
                &self.audit,
                Session {
                    account_id: 0,
                    role: Role::Player,
                },
                "auth.login",
                login,
                "credencial recusada",
            );
            return recusa();
        };
        if account.blocked {
            log_panel(
                &self.audit,
                Session {
                    account_id: account.account_id,
                    role: account.role,
                },
                "auth.login",
                &account.login,
                "conta bloqueada",
            );
            // Senha certa + bloqueio: responde `blocked` para o portal explicar
            // sem transformar a recusa em vazamento para quem não tem a senha.
            return Reply::ok(
                trace,
                ",\"ok\":false,\"account_id\":0,\"blocked\":true,\"role\":\"\"",
            );
        }
        log_panel(
            &self.audit,
            Session {
                account_id: account.account_id,
                role: account.role,
            },
            "auth.login",
            &account.login,
            "ok",
        );
        let role = json::quote(account.role.as_db());
        Reply::ok(
            trace,
            &format!(
                ",\"ok\":true,\"account_id\":{},\"blocked\":false,\"role\":{role}",
                account.account_id
            ),
        )
    }

    /// `item_catalog/list_items` —a rota que o `/api/health` do portal consome (ADR-0022),
    /// servida da foto real do Postgres carregada no boot.
    fn catalog_list_items(&self, session: Session, trace: &str) -> Reply {
        let Some(snapshot) = &self.catalog else {
            return Reply::transport(503, trace, "catálogo ainda não carregado do banco");
        };
        log_panel(
            &self.audit,
            session,
            "catalog.list_items",
            "content_version",
            &format!("{} itens", snapshot.items.len()),
        );
        let items = crate::api::items_json(&snapshot.items);
        let version = json::quote(&snapshot.version);
        Reply::ok(
            trace,
            &format!(",\"items\":{items},\"catalog_version\":{version},\"icon_pack_version\":\"\""),
        )
    }

    /// `content_rates/get_rates` — foto completa das taxas e do refino de acessório.
    fn config_rates(&self, session: Session, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        match crate::api::rates::get_rates(Some(session), store.as_ref(), &self.audit) {
            Ok(snapshot) => Reply::ok(
                trace,
                &format!(",{}", crate::api::rates::snapshot_json(&snapshot)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `content_rates/get_server_flags` — configurações inteiras do servidor.
    fn config_flags(&self, session: Session, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        match crate::api::rates::get_rates(Some(session), store.as_ref(), &self.audit) {
            Ok(snapshot) => {
                let flags: Vec<String> = crate::api::rates::flags_from(&snapshot)
                    .iter()
                    .map(|(key, value)| format!("{}:{value}", json::quote(key)))
                    .collect();
                Reply::ok(
                    trace,
                    &format!(
                        ",\"flags\":{{{}}},\"revision\":{}",
                        flags.join(","),
                        snapshot.revision
                    ),
                )
            }
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `content_rates/upsert_rate` — grava taxa com fonte obrigatória.
    fn config_upsert_rate(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let campo = |nome: &str| params.get(nome).and_then(json::Json::as_str);
        let (Some(domain), Some(key), Some(value)) = (
            campo("domain"),
            campo("key"),
            params.get("value").and_then(json::Json::as_i64),
        ) else {
            return Reply::business(trace, RESULT_INVALID, "domain, key e value obrigatórios");
        };
        let source = campo("source").unwrap_or_default();
        match crate::api::rates::upsert_rate(
            Some(session),
            domain,
            key,
            value,
            source,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `content_rates/reset_rate` — remove o override.
    fn config_reset_rate(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(domain), Some(key)) = (
            params.get("domain").and_then(json::Json::as_str),
            params.get("key").and_then(json::Json::as_str),
        ) else {
            return Reply::business(trace, RESULT_INVALID, "domain e key obrigatórios");
        };
        match crate::api::rates::reset_rate(Some(session), domain, key, store.as_ref(), &self.audit)
        {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `drop_admin/list_drop_rates` — pesos por template de monstro.
    fn config_drops(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let template = params.get("template_name").and_then(json::Json::as_str);
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        match crate::api::drop::list_drops(
            Some(session),
            template,
            limit,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"drops\":{}", crate::api::drop::drops_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `drop_admin/upsert_drop_rate` — grava o peso de (template, item).
    fn config_set_drop(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(template), Some(item), Some(weight)) = (
            params.get("template_name").and_then(json::Json::as_str),
            params.get("item_index").and_then(json::Json::as_i64),
            params.get("weight").and_then(json::Json::as_i64),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "template_name, item_index e weight obrigatórios",
            );
        };
        let Ok(item) = i32::try_from(item) else {
            return Reply::business(trace, RESULT_INVALID, "item_index fora da faixa");
        };
        let min_qty = params
            .get("min_qty")
            .and_then(json::Json::as_i64)
            .unwrap_or(1);
        let max_qty = params
            .get("max_qty")
            .and_then(json::Json::as_i64)
            .unwrap_or(min_qty);
        let (Ok(min_qty), Ok(max_qty)) = (i16::try_from(min_qty), i16::try_from(max_qty)) else {
            return Reply::business(trace, RESULT_INVALID, "quantidade fora da faixa");
        };
        let source = params
            .get("source")
            .and_then(json::Json::as_str)
            .unwrap_or_default();
        let edit = crate::api::DropEdit {
            template_name: template.to_string(),
            item_index: item,
            weight,
            min_qty,
            max_qty,
            source: source.to_string(),
        };
        match crate::api::drop::upsert_drop(Some(session), &edit, store.as_ref(), &self.audit) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `drop_admin/reset_drop_rate` — remove o peso editado.
    fn config_reset_drop(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(template), Some(item)) = (
            params.get("template_name").and_then(json::Json::as_str),
            params.get("item_index").and_then(json::Json::as_i64),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "template_name e item_index obrigatórios",
            );
        };
        let Ok(item) = i32::try_from(item) else {
            return Reply::business(trace, RESULT_INVALID, "item_index fora da faixa");
        };
        match crate::api::drop::reset_drop(
            Some(session),
            template,
            item,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `npc_admin/get_npc` — posição e visibilidade de um NPC.
    fn config_npc(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let Some(id) = params.get("id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo id obrigatório");
        };
        let Ok(id) = i32::try_from(id) else {
            return Reply::business(trace, RESULT_INVALID, "id fora da faixa");
        };
        match crate::api::npc::get_npc(Some(session), id, store.as_ref(), &self.audit) {
            Ok(Some(npc)) => Reply::ok(
                trace,
                &format!(",\"npc\":{}", crate::api::npc::npc_json(&npc)),
            ),
            Ok(None) => Reply::business(trace, RESULT_NOT_FOUND, "NPC não encontrado"),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `npc_admin/list_npcs` — NPCs em páginas por id (só leitura).
    fn config_npc_page(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let after_id = params
            .get("after_id")
            .and_then(json::Json::as_i64)
            .and_then(|id| i32::try_from(id).ok());
        match crate::api::npc::list_npcs(
            Some(session),
            limit,
            after_id,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"npcs\":{}", crate::api::npcs_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `npc_admin/list_item_prices` — preços de NPC (`item_price`), filtrados por item.
    ///
    /// Tabela vazia é resposta v‌álida: significa "nenhum preço definido pelo dono do
    /// servidor", e o portal mostra ausência em vez de zero.
    fn config_item_prices(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let item_index = params
            .get("item_index")
            .and_then(json::Json::as_i64)
            .and_then(|item| i32::try_from(item).ok());
        match crate::api::list_item_prices(
            Some(session),
            item_index,
            limit,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"prices\":{}", crate::api::item_prices_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `npc_admin/set_item_price` — grava o preço de (item, NPC).
    ///
    /// A forma é validada antes de tocar o store: corpo sem item/NPC responde `INVALID`
    /// mesmo sem configuração montada, e nunca chega ao banco.
    fn config_set_item_price(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let (Some(item), Some(npc), Some(buy), Some(sell)) = (
            params.get("item_index").and_then(json::Json::as_i64),
            params.get("npc_id").and_then(json::Json::as_i64),
            params.get("buy_price").and_then(json::Json::as_i64),
            params.get("sell_price").and_then(json::Json::as_i64),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "item_index, npc_id, buy_price e sell_price obrigatórios",
            );
        };
        let (Ok(item), Ok(npc)) = (i32::try_from(item), i32::try_from(npc)) else {
            return Reply::business(trace, RESULT_INVALID, "item ou NPC fora da faixa");
        };
        let edit = crate::api::ItemPriceEdit {
            item_index: item,
            npc_id: npc,
            buy_price: buy,
            sell_price: sell,
            source: params
                .get("source")
                .and_then(json::Json::as_str)
                .unwrap_or_default()
                .to_string(),
        };
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        match crate::api::set_item_price(Some(session), &edit, store.as_ref(), &self.audit) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `mob_admin/list_mob_templates` — templates de mob em páginas por nome (só leitura).
    fn config_mob_templates(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let after = params.get("after").and_then(json::Json::as_str);
        match crate::api::mob::list_mob_templates(
            Some(session),
            limit,
            after,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(
                    ",\"mob_templates\":{}",
                    crate::api::mob_templates_json(&rows)
                ),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `item_effect_admin/list_effects` — efeitos de item em páginas por nome (só leitura).
    fn config_effects(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let after = params.get("after").and_then(json::Json::as_str);
        match crate::api::item_effect::list_effects(
            Some(session),
            limit,
            after,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"effects\":{}", crate::api::effects_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `daily_reward_admin/list_reward_items` — recompensas em páginas por id.
    fn config_daily_list(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let after_id = params
            .get("after_id")
            .and_then(json::Json::as_i64)
            .and_then(|id| i32::try_from(id).ok());
        match crate::api::daily::list_rewards(
            Some(session),
            limit,
            after_id,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"rewards\":{}", crate::api::daily::rewards_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `daily_reward_admin/upsert_reward_item` — cria ou substitui recompensa.
    fn config_daily_upsert(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let edit = crate::api::DailyEdit {
            id: params
                .get("id")
                .and_then(json::Json::as_i64)
                .and_then(|id| i32::try_from(id).ok()),
            day: params
                .get("day")
                .and_then(json::Json::as_i64)
                .and_then(|day| i16::try_from(day).ok()),
            item_index: params
                .get("item_index")
                .and_then(json::Json::as_i64)
                .and_then(|item| i32::try_from(item).ok()),
            gold: params.get("gold").and_then(json::Json::as_i64).unwrap_or(0),
            qty: params
                .get("qty")
                .and_then(json::Json::as_i64)
                .and_then(|qty| i16::try_from(qty).ok())
                .unwrap_or(1),
            account_wide: params
                .get("account_wide")
                .and_then(json::Json::as_bool)
                .unwrap_or(true),
        };
        match crate::api::daily::upsert_reward(Some(session), &edit, store.as_ref(), &self.audit) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `daily_reward_admin/delete_reward_item` — remove recompensa.
    fn config_daily_delete(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let Some(id) = params
            .get("id")
            .and_then(json::Json::as_i64)
            .and_then(|id| i32::try_from(id).ok())
        else {
            return Reply::business(trace, RESULT_INVALID, "campo id obrigatório");
        };
        match crate::api::daily::delete_reward(Some(session), id, store.as_ref(), &self.audit) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_admin/list_shop_items` — vitrine em páginas por id.
    fn config_donate_list(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let after_id = params
            .get("after_id")
            .and_then(json::Json::as_i64)
            .and_then(|id| i32::try_from(id).ok());
        match crate::api::donate::list_shop_items(
            Some(session),
            limit,
            after_id,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(
                    ",\"shop_items\":{}",
                    crate::api::donate::shop_items_json(&rows)
                ),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_admin/upsert_shop_item` — cria ou substitui item da vitrine.
    fn config_donate_upsert(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(item_index), Some(price_credits)) = (
            params
                .get("item_index")
                .and_then(json::Json::as_i64)
                .and_then(|item| i32::try_from(item).ok()),
            params
                .get("price_credits")
                .and_then(json::Json::as_i64)
                .and_then(|preco| i32::try_from(preco).ok()),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "item_index e price_credits obrigatórios",
            );
        };
        let edit = crate::api::DonateEdit {
            id: params
                .get("id")
                .and_then(json::Json::as_i64)
                .and_then(|id| i32::try_from(id).ok()),
            item_index,
            price_credits,
            duration_days: params
                .get("duration_days")
                .and_then(json::Json::as_i64)
                .and_then(|d| i16::try_from(d).ok()),
            enabled: params
                .get("enabled")
                .and_then(json::Json::as_bool)
                .unwrap_or(true),
        };
        match crate::api::donate::upsert_shop_item(
            Some(session),
            &edit,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_admin/set_shop_item_enabled` — liga/desliga item da vitrine.
    fn config_donate_enabled(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(id), Some(enabled)) = (
            params
                .get("id")
                .and_then(json::Json::as_i64)
                .and_then(|id| i32::try_from(id).ok()),
            params.get("enabled").and_then(json::Json::as_bool),
        ) else {
            return Reply::business(trace, RESULT_INVALID, "id e enabled obrigatórios");
        };
        match crate::api::donate::set_shop_item_enabled(
            Some(session),
            id,
            enabled,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_admin/delete_shop_item` — remove item da vitrine.
    fn config_donate_delete(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let Some(id) = params
            .get("id")
            .and_then(json::Json::as_i64)
            .and_then(|id| i32::try_from(id).ok())
        else {
            return Reply::business(trace, RESULT_INVALID, "campo id obrigatório");
        };
        match crate::api::donate::delete_shop_item(Some(session), id, store.as_ref(), &self.audit) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_admin/credit_donate_balance` — credita saldo na conta, idempotente.
    fn config_donate_credit(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(account_id), Some(credits), Some(key)) = (
            params.get("account_id").and_then(json::Json::as_i64),
            params.get("credits").and_then(json::Json::as_i64),
            params.get("idempotency_key").and_then(json::Json::as_str),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "account_id, credits e idempotency_key obrigatórios",
            );
        };
        match crate::api::donate::credit_balance(
            Some(session),
            account_id,
            credits,
            key,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(receipt) => {
                let key = json::quote(&receipt.idempotency_key);
                Reply::ok(
                    trace,
                    &format!(
                        ",\"idempotency_key\":{key},\"credit_id\":{},\"replay\":{}",
                        receipt.credit_id, receipt.replay
                    ),
                )
            }
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_revenue_admin/get_revenue_summary` — receita somada (staff).
    fn config_donate_revenue(&self, session: Session, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        match crate::api::donate::revenue_summary(Some(session), store.as_ref(), &self.audit) {
            Ok(row) => Reply::ok(
                trace,
                &format!(",\"revenue\":{}", crate::api::donate::revenue_json(&row)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_revenue_admin/list_topup_orders` — extrato de recargas.
    fn config_donate_topups(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(20);
        match crate::api::donate::topup_orders(Some(session), limit, store.as_ref(), &self.audit) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"orders\":{}", crate::api::donate::ledger_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_revenue_admin/list_donate_spend` — extrato de débitos.
    fn config_donate_spend(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(20);
        match crate::api::donate::spend_orders(Some(session), limit, store.as_ref(), &self.audit) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"spend\":{}", crate::api::donate::ledger_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_revenue_admin/list_top_buyers` — contas que mais compraram.
    fn config_donate_buyers(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(10);
        match crate::api::donate::top_buyers(Some(session), limit, store.as_ref(), &self.audit) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"buyers\":{}", crate::api::donate::buyers_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_revenue_admin/search_accounts` — busca conta por login.
    fn config_donate_search(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let term = params
            .get("login")
            .and_then(json::Json::as_str)
            .unwrap_or("");
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(10);
        match crate::api::donate::search_accounts(
            Some(session),
            term,
            limit,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"accounts\":{}", crate::api::donate::lookup_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `donate_admin/debit_donate_balance` — debita saldo da conta, idempotente.
    fn config_donate_debit(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(account_id), Some(credits), Some(key)) = (
            params.get("account_id").and_then(json::Json::as_i64),
            params.get("credits").and_then(json::Json::as_i64),
            params.get("idempotency_key").and_then(json::Json::as_str),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "account_id, credits e idempotency_key obrigatórios",
            );
        };
        match crate::api::donate::debit_balance(
            Some(session),
            account_id,
            credits,
            key,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(receipt) => {
                let key = json::quote(&receipt.idempotency_key);
                Reply::ok(
                    trace,
                    &format!(
                        ",\"idempotency_key\":{key},\"debit_id\":{},\"replay\":{}",
                        receipt.debit_id, receipt.replay
                    ),
                )
            }
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `world_event_admin/get_world_event_config` — eventos com horários.
    fn config_event_get(&self, session: Session, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        match crate::api::event::get_config(Some(session), store.as_ref(), &self.audit) {
            Ok(full) => Reply::ok(
                trace,
                &format!(",\"events\":{}", crate::api::event_config_json(&full)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `world_event_admin/set_world_event_config` — liga/desliga um evento.
    fn config_event_set(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(code), Some(enabled)) = (
            params.get("code").and_then(json::Json::as_str),
            params.get("enabled").and_then(json::Json::as_bool),
        ) else {
            return Reply::business(trace, RESULT_INVALID, "code e enabled obrigatórios");
        };
        match crate::api::event::set_enabled(
            Some(session),
            code,
            enabled,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `account_admin/list_accounts` — contas em páginas por id, sem segredo.
    fn config_account_list(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let after_id = params.get("after_id").and_then(json::Json::as_i64);
        match crate::api::accounts::list_accounts(
            Some(session),
            limit,
            after_id,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(
                    ",\"accounts\":{}",
                    crate::api::accounts::accounts_json(&rows)
                ),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `account_admin/get_account` — uma conta por id, sem segredo.
    fn config_account_get(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let Some(id) = params.get("id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo id obrigatório");
        };
        match crate::api::accounts::get_account(Some(session), id, store.as_ref(), &self.audit) {
            Ok(Some(account)) => Reply::ok(
                trace,
                &format!(
                    ",\"account\":{}",
                    crate::api::accounts::account_json(&account)
                ),
            ),
            Ok(None) => Reply::business(trace, RESULT_NOT_FOUND, "conta não encontrada"),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `character_admin/grant_item` — enfileira item na `delivery_queue` (fila, não mutação).
    ///
    /// O `kind` é sempre item: ouro/nível continuam em `grants/request`. A entrega acontece
    /// no dreno do servidor (`claim_item` → slot livre de `Carry`); mochila cheia deixa a
    /// linha pendente, nunca duplica nem perde.
    fn character_grant_item(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let (Some(key), Some(character_id), Some(item_id), Some(qty)) = (
            params.get("idempotency_key").and_then(json::Json::as_str),
            params.get("character_id").and_then(json::Json::as_i64),
            params.get("item_id").and_then(json::Json::as_i64),
            params.get("qty").and_then(json::Json::as_i64),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "idempotency_key, character_id, item_id e qty obrigatórios",
            );
        };
        let request = crate::api::GrantRequest {
            idempotency_key: key.to_string(),
            character_id,
            kind: crate::api::GrantKind::Item { item_id, qty },
        };
        match crate::api::request_grant(Some(session), request, &self.grants, &self.audit) {
            Ok(receipt) => {
                let key = json::quote(&receipt.idempotency_key);
                let grant_id = receipt.grant_id;
                let replay = receipt.replay;
                Reply::ok(
                    trace,
                    &format!(
                        ",\"idempotency_key\":{key},\"grant_id\":{grant_id},\"replay\":{replay}"
                    ),
                )
            }
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `npc_admin/upsert_npc` — move/ativa NPC.
    fn config_set_npc(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let (Some(id), Some(map_id), Some(x), Some(y), Some(enabled)) = (
            params.get("id").and_then(json::Json::as_i64),
            params.get("map_id").and_then(json::Json::as_i64),
            params.get("pos_x").and_then(json::Json::as_i64),
            params.get("pos_y").and_then(json::Json::as_i64),
            params.get("enabled").and_then(json::Json::as_bool),
        ) else {
            return Reply::business(
                trace,
                RESULT_INVALID,
                "id, map_id, pos_x, pos_y e enabled obrigatórios",
            );
        };
        let (Ok(id), Ok(map_id), Ok(x), Ok(y)) = (
            i32::try_from(id),
            i16::try_from(map_id),
            i16::try_from(x),
            i16::try_from(y),
        ) else {
            return Reply::business(trace, RESULT_INVALID, "coordenada fora da faixa");
        };
        match crate::api::npc::set_npc(
            Some(session),
            id,
            (map_id, x, y),
            enabled,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `teleport_admin/list_routes` — rotas com custo e requisito.
    fn config_routes(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(100);
        match crate::api::teleport::list_routes(Some(session), limit, store.as_ref(), &self.audit) {
            Ok(rows) => Reply::ok(
                trace,
                &format!(",\"routes\":{}", crate::api::teleport::routes_json(&rows)),
            ),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `teleport_admin/upsert_route` e `set_route_enabled` — política da rota.
    fn config_set_route(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(store) = &self.config else {
            return Reply::transport(503, trace, "configuração não montada no gateway");
        };
        let Some(id) = params.get("id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo id obrigatório");
        };
        let Ok(id) = i32::try_from(id) else {
            return Reply::business(trace, RESULT_INVALID, "id fora da faixa");
        };
        let cost = match params.get("cost_gold").and_then(json::Json::as_i64) {
            Some(value) => match i32::try_from(value) {
                Ok(value) => Some(value),
                Err(_) => return Reply::business(trace, RESULT_INVALID, "custo fora da faixa"),
            },
            None => None,
        };
        let level = match params.get("req_level").and_then(json::Json::as_i64) {
            Some(value) => match i16::try_from(value) {
                Ok(value) => Some(value),
                Err(_) => return Reply::business(trace, RESULT_INVALID, "nível fora da faixa"),
            },
            None => None,
        };
        let enabled = params.get("enabled").and_then(json::Json::as_bool);
        match crate::api::teleport::set_route(
            Some(session),
            id,
            cost,
            level,
            enabled,
            store.as_ref(),
            &self.audit,
        ) {
            Ok(revision) => Reply::ok(trace, &format!(",\"revision\":{revision}")),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// Rota do contrato ainda sem handler de domínio: `501` explícito, com autor auditado.
    ///
    /// Não é `404`: `404` significa "nome inválido" (contrato violado). Aqui o portal
    /// pediu algo que **existe** no contrato e o control plane ainda não implementa —
    /// a diferença é diagnóstico, não cosmética.
    fn pending(&self, session: Session, service: &str, method: &str, trace: &str) -> Reply {
        let alvo = format!("{service}.{method}");
        log_panel(&self.audit, session, "admin.pending", &alvo, "sem handler");
        Reply::transport(
            501,
            trace,
            &format!("contrato {alvo} existe no portal e ainda não tem handler no control plane"),
        )
    }

    fn accounts_recovery(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        match request_recovery(Some(session), login, &self.audit) {
            Ok(()) => Reply::ok(trace, ",\"requested\":true"),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    fn panel_self(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(account_id) = params.get("account_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo account_id obrigatório");
        };
        match poll_once(own_account(
            Some(session),
            account_id,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(account)) => Reply::ok(
                trace,
                &format!(",\"account\":{}", panel_account_json(&account)),
            ),
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn panel_characters(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(account_id) = params.get("account_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo account_id obrigatório");
        };
        match poll_once(own_characters(
            Some(session),
            account_id,
            &self.characters,
            &self.audit,
        )) {
            Some(Ok(characters)) => {
                let lista = characters_json(&characters);
                Reply::ok(trace, &format!(",\"characters\":{lista}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    /// `panel/history` — sessões da conta e tentativas de login do próprio jogador.
    ///
    /// `account_id` vem da sessão assinada no BFF; o login é lido do banco no
    /// gateway. A trilha filtra pelo id do ator, nunca por texto de login.
    fn panel_history(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(account_id) = params.get("account_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo account_id obrigatório");
        };
        let (Some(sessoes), Some(trilha)) = (&self.session_trail, &self.audit_trail) else {
            return Reply::transport(503, trace, "histórico do jogador não montado no gateway");
        };
        match poll_once(own_history(
            Some(session),
            account_id,
            &self.accounts,
            sessoes.as_ref(),
            trilha.as_ref(),
            &self.audit,
        )) {
            Some(Ok((sessions, events))) => {
                let sessions =
                    json::Json::Arr(sessions.iter().map(session_json).collect()).render();
                let events = json::Json::Arr(events.iter().map(event_json).collect()).render();
                Reply::ok(
                    trace,
                    &format!(",\"sessions\":{sessions},\"events\":{events}"),
                )
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn panel_rankings(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(10);
        let limit = usize::try_from(limit).unwrap_or(0);
        match poll_once(top_rankings(
            Some(session),
            limit,
            &self.characters,
            &self.audit,
        )) {
            Some(Ok(entries)) => {
                let lista = rankings_json(&entries);
                Reply::ok(trace, &format!(",\"ranking\":{lista}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn accounts_find(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        match poll_once(find_account(
            Some(session),
            login,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(account)) => {
                let account = account_json(&account);
                Reply::ok(trace, &format!(",\"account\":{account}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn accounts_moderation(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        match poll_once(moderation(
            Some(session),
            login,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(estado)) => Reply::ok(
                trace,
                &format!(",\"moderation\":{}", moderation_json(&estado)),
            ),
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    /// `accounts/set_role` — só admin; o papel novo vem do corpo, o autor da sessão.
    fn accounts_set_role(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        let Some(role) = params.get("role").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo role obrigatório");
        };
        let Some(role) = role_from_str(role) else {
            return Reply::business(trace, RESULT_INVALID, "role inválido");
        };
        match poll_once(set_role(
            Some(session),
            login,
            role,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(estado)) => Reply::ok(
                trace,
                &format!(",\"moderation\":{}", moderation_json(&estado)),
            ),
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    /// `accounts/set_blocked` — bloqueio com motivo; liberação sem motivo.
    fn accounts_set_blocked(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        let Some(blocked) = params.get("blocked").and_then(json::Json::as_bool) else {
            return Reply::business(trace, RESULT_INVALID, "campo blocked obrigatório");
        };
        let reason = params
            .get("reason")
            .and_then(json::Json::as_str)
            .unwrap_or("");
        match poll_once(set_blocked(
            Some(session),
            login,
            blocked,
            reason,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(estado)) => Reply::ok(
                trace,
                &format!(",\"moderation\":{}", moderation_json(&estado)),
            ),
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    /// `audit_admin/recent` — últimas linhas da trilha (admin).
    fn audit_recent(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(trail) = &self.audit_trail else {
            return Reply::transport(503, trace, "trilha de auditoria não montada no gateway");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(50);
        let limit = usize::try_from(limit).unwrap_or(0);
        match poll_once(recent_audit(
            Some(session),
            limit,
            trail.as_ref(),
            &self.audit,
        )) {
            Some(Ok(linhas)) => Reply::ok(trace, &format!(",\"entries\":{}", audit_json(&linhas))),
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    /// `economy_admin/summary` — imposto por cidade e concessões por tipo.
    fn economy_summary(&self, session: Session, trace: &str) -> Reply {
        let Some(economy) = &self.economy else {
            return Reply::transport(503, trace, "economia não montada no gateway");
        };
        match poll_once(economy_summary(
            Some(session),
            economy.as_ref(),
            &self.audit,
        )) {
            Some(Ok(resumo)) => {
                Reply::ok(trace, &format!(",\"economy\":{}", economy_json(&resumo)))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn accounts_create(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        let Some(role) = params.get("role").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo role obrigatório");
        };
        let Some(role) = role_from_str(role) else {
            return Reply::business(trace, RESULT_INVALID, "role inválido");
        };
        match poll_once(create_account(
            Some(session),
            login,
            role,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(account)) => {
                let account = account_json(&account);
                Reply::ok(trace, &format!(",\"account\":{account}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn grants_request(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(idempotency_key) = params.get("idempotency_key").and_then(json::Json::as_str)
        else {
            return Reply::business(trace, RESULT_INVALID, "campo idempotency_key obrigatório");
        };
        let Some(character_id) = params.get("character_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo character_id obrigatório");
        };
        let Some(kind) = params.get("kind").and_then(grant_kind) else {
            return Reply::business(trace, RESULT_INVALID, "campo kind inválido");
        };
        let request = GrantRequest {
            idempotency_key: idempotency_key.to_string(),
            character_id,
            kind,
        };
        match request_grant(Some(session), request, &self.grants, &self.audit) {
            Ok(receipt) => {
                let key = json::quote(&receipt.idempotency_key);
                let grant_id = receipt.grant_id;
                let replay = receipt.replay;
                let extras = format!(
                    ",\"idempotency_key\":{key},\"grant_id\":{grant_id},\"replay\":{replay}"
                );
                Reply::ok(trace, &extras)
            }
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    /// `character_admin/search_characters` — staff busca personagens por nome.
    ///
    /// Devolve o destino com `gold` atual: a tela de entregas confere antes de
    /// enfileirar e depois relê para provar o efeito no jogo.
    fn character_search(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let staff = match StaffSession::require(session) {
            Ok(staff) => staff,
            Err(error) => return Reply::from_error(trace, &error),
        };
        let term = params
            .get("term")
            .and_then(json::Json::as_str)
            .unwrap_or("");
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(20)
            .clamp(1, 100);
        let limit = usize::try_from(limit).unwrap_or(20);
        match poll_once(self.characters.search_by_name(term, limit)) {
            Some(Ok(rows)) => {
                log_action(
                    &self.audit,
                    &staff,
                    "character.search",
                    if term.is_empty() { "*" } else { term },
                    &format!("{} personagem(ns)", rows.len()),
                );
                Reply::ok(
                    trace,
                    &format!(",\"characters\":{}", characters_search_json(&rows)),
                )
            }
            Some(Err(error)) => Reply::from_error(trace, &AdminError::from(error)),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    /// `character_admin/list_character_grants` — fila do personagem (somente leitura).
    fn character_grants(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let staff = match StaffSession::require(session) {
            Ok(staff) => staff,
            Err(error) => return Reply::from_error(trace, &error),
        };
        let Some(character_id) = params.get("character_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo character_id obrigatório");
        };
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(20)
            .clamp(1, 50);
        let limit = usize::try_from(limit).unwrap_or(20);
        let Some(trail) = &self.grant_trail else {
            return Reply::transport(503, trace, "fila de entrega não montada no gateway");
        };
        match trail.list_for_character(character_id, limit) {
            Ok(rows) => {
                log_action(
                    &self.audit,
                    &staff,
                    "grants.list",
                    &format!("char:{character_id}"),
                    &format!("{} entrega(s)", rows.len()),
                );
                Reply::ok(trace, &format!(",\"grants\":{}", grants_rows_json(&rows)))
            }
            Err(error) => Reply::from_error(trace, &AdminError::from(error)),
        }
    }
}

/// Fila de conexões do Cold Path (sequencial: um cliente local, tráfego de admin).
pub fn serve<A: Accounts, C: Characters, S: AuditSink>(
    listener: &TcpListener,
    router: &Router<A, C, S>,
) -> std::io::Result<()> {
    loop {
        match listener.accept() {
            Ok((stream, _)) => serve_conn(stream, router),
            Err(error) => return Err(error),
        }
    }
}

/// Atende uma conexão: lê uma requisição, responde e fecha (`Connection: close`).
pub fn serve_conn<A: Accounts, C: Characters, S: AuditSink>(
    stream: TcpStream,
    router: &Router<A, C, S>,
) {
    let Ok(peer) = stream.peer_addr() else {
        return;
    };
    if !peer.ip().is_loopback() {
        return;
    }
    let Ok(read_half) = stream.try_clone() else {
        return;
    };
    let mut reader = BufReader::new(read_half);
    let mut writer = stream;
    let reply = match http::read_request(&mut reader, router.max_body()) {
        Ok(request) => router.dispatch(&request),
        Err(error) => Reply::transport(error.status(), "", "falha de transporte"),
    };
    let _ = http::write_response(&mut writer, reply.status, &reply.trace_id, &reply.body);
}

/// Executa um futuro que deve estar pronto no primeiro poll (ADR-0006).
fn poll_once<F: Future>(future: F) -> Option<F::Output> {
    let waker: Waker = Waker::noop().clone();
    let mut context = Context::from_waker(&waker);
    let mut pinned = Box::pin(future);
    match pinned.as_mut().poll(&mut context) {
        Poll::Ready(value) => Some(value),
        Poll::Pending => None,
    }
}

fn route(path: &str) -> Option<(&str, &str)> {
    let rest = path.strip_prefix("/admin/v1/")?;
    let (service, method) = rest.split_once('/')?;
    let valid = |segment: &str| {
        !segment.is_empty()
            && segment
                .bytes()
                .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'_')
    };
    if valid(service) && valid(method) {
        Some((service, method))
    } else {
        None
    }
}

fn bearer(header: Option<&str>) -> Option<&str> {
    let (scheme, token) = header?.split_once(' ')?;
    if !scheme.eq_ignore_ascii_case("bearer") {
        return None;
    }
    let token = token.trim();
    if token.is_empty() {
        None
    } else {
        Some(token)
    }
}

fn role_from_str(role: &str) -> Option<Role> {
    match role {
        "player" => Some(Role::Player),
        "moderator" => Some(Role::Moderator),
        "admin" => Some(Role::Admin),
        _ => None,
    }
}

fn grant_kind(value: &json::Json) -> Option<GrantKind> {
    match value.get("type").and_then(json::Json::as_str)? {
        "item" => Some(GrantKind::Item {
            item_id: value.get("item_id").and_then(json::Json::as_i64)?,
            qty: value.get("qty").and_then(json::Json::as_i64)?,
        }),
        "gold" => Some(GrantKind::Gold {
            amount: value.get("amount").and_then(json::Json::as_i64)?,
        }),
        "level" => Some(GrantKind::Level {
            delta: value.get("delta").and_then(json::Json::as_i64)?,
        }),
        _ => None,
    }
}

fn panel_account_json(account: &PanelAccount) -> String {
    json::Json::Obj(vec![
        ("id".to_string(), json::Json::Num(account.id as f64)),
        ("login".to_string(), json::Json::Str(account.login.clone())),
        (
            "role".to_string(),
            json::Json::Str(account.role.as_db().to_string()),
        ),
        (
            "pin_defined".to_string(),
            json::Json::Bool(account.pin_defined),
        ),
    ])
    .render()
}

fn characters_json(characters: &[PanelCharacter]) -> String {
    let items = characters
        .iter()
        .map(|c| {
            json::Json::Obj(vec![
                ("id".to_string(), json::Json::Num(c.id as f64)),
                ("slot".to_string(), json::Json::Num(c.slot as f64)),
                ("name".to_string(), json::Json::Str(c.name.clone())),
                ("class".to_string(), json::Json::Num(c.class as f64)),
                ("level".to_string(), json::Json::Num(c.level as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn characters_search_json(characters: &[wyd_db::Character]) -> String {
    let items = characters
        .iter()
        .map(|c| {
            json::Json::Obj(vec![
                ("id".to_string(), json::Json::Num(c.id as f64)),
                (
                    "account_id".to_string(),
                    json::Json::Num(c.account_id as f64),
                ),
                ("name".to_string(), json::Json::Str(c.name.clone())),
                ("level".to_string(), json::Json::Num(c.level as f64)),
                ("gold".to_string(), json::Json::Num(c.gold as f64)),
                ("class".to_string(), json::Json::Num(c.class as f64)),
                ("slot".to_string(), json::Json::Num(c.slot as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn grants_rows_json(rows: &[wyd_db::QueuedGrantRow]) -> String {
    let items = rows
        .iter()
        .map(|g| {
            json::Json::Obj(vec![
                ("id".to_string(), json::Json::Num(g.id as f64)),
                ("kind".to_string(), json::Json::Str(g.kind.clone())),
                ("gold".to_string(), json::Json::Num(g.gold as f64)),
                ("status".to_string(), json::Json::Str(g.status.clone())),
                (
                    "idempotency_key".to_string(),
                    g.idempotency_key
                        .clone()
                        .map_or(json::Json::Null, json::Json::Str),
                ),
                (
                    "created_by".to_string(),
                    g.created_by
                        .map_or(json::Json::Null, |v| json::Json::Num(v as f64)),
                ),
                ("at_ms".to_string(), json::Json::Num(g.at_ms as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn session_json(row: &PanelSession) -> json::Json {
    json::Json::Obj(vec![
        (
            "group_id".to_string(),
            json::Json::Num(f64::from(row.group_id)),
        ),
        (
            "channel".to_string(),
            json::Json::Num(f64::from(row.channel)),
        ),
        (
            "client_version".to_string(),
            json::Json::Num(f64::from(row.client_version)),
        ),
        (
            "character_id".to_string(),
            row.character_id
                .map_or(json::Json::Null, |id| json::Json::Num(id as f64)),
        ),
        (
            "opened_ms".to_string(),
            json::Json::Num(row.opened_ms as f64),
        ),
        (
            "renewed_ms".to_string(),
            json::Json::Num(row.renewed_ms as f64),
        ),
        (
            "released_ms".to_string(),
            row.released_ms
                .map_or(json::Json::Null, |ms| json::Json::Num(ms as f64)),
        ),
    ])
}

fn event_json(row: &PanelEvent) -> json::Json {
    json::Json::Obj(vec![
        ("at_ms".to_string(), json::Json::Num(row.at_ms as f64)),
        ("action".to_string(), json::Json::Str(row.action.clone())),
        ("detail".to_string(), json::Json::Str(row.detail.clone())),
    ])
}

fn rankings_json(entries: &[RankingEntry]) -> String {
    let items = entries
        .iter()
        .map(|e| {
            json::Json::Obj(vec![
                ("rank".to_string(), json::Json::Num(e.rank as f64)),
                ("name".to_string(), json::Json::Str(e.name.clone())),
                ("class".to_string(), json::Json::Num(e.class as f64)),
                ("level".to_string(), json::Json::Num(e.level as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn presence_json(channels: &[ChannelPresence]) -> String {
    let items = channels
        .iter()
        .map(|c| {
            json::Json::Obj(vec![
                ("group_id".to_string(), json::Json::Num(c.group_id as f64)),
                ("channel".to_string(), json::Json::Num(c.channel as f64)),
                ("host".to_string(), json::Json::Str(c.host.clone())),
                ("online".to_string(), json::Json::Num(c.online as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn moderation_json(estado: &AccountModeration) -> String {
    let conta = json::Json::Obj(vec![
        ("id".to_string(), json::Json::Num(estado.account.id as f64)),
        (
            "login".to_string(),
            json::Json::Str(estado.account.login.clone()),
        ),
        (
            "role".to_string(),
            json::Json::Str(estado.account.role.as_db().to_string()),
        ),
    ]);
    let blocked_at = match estado.blocked_at_ms {
        Some(ms) => json::Json::Num(ms as f64),
        None => json::Json::Null,
    };
    json::Json::Obj(vec![
        ("account".to_string(), conta),
        ("blocked".to_string(), json::Json::Bool(estado.blocked)),
        ("reason".to_string(), json::Json::Str(estado.reason.clone())),
        ("blocked_at_ms".to_string(), blocked_at),
    ])
    .render()
}

fn audit_json(linhas: &[AuditRow]) -> String {
    let items = linhas
        .iter()
        .map(|linha| {
            json::Json::Obj(vec![
                ("at_ms".to_string(), json::Json::Num(linha.at_ms as f64)),
                (
                    "actor_id".to_string(),
                    json::Json::Num(linha.actor_id as f64),
                ),
                (
                    "actor_role".to_string(),
                    json::Json::Str(linha.actor_role.clone()),
                ),
                ("action".to_string(), json::Json::Str(linha.action.clone())),
                ("target".to_string(), json::Json::Str(linha.target.clone())),
                ("detail".to_string(), json::Json::Str(linha.detail.clone())),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn economy_json(resumo: &EconomySummary) -> String {
    let cidades = resumo
        .tax_by_city
        .iter()
        .map(|cidade| {
            json::Json::Obj(vec![
                (
                    "city_code".to_string(),
                    json::Json::Str(cidade.city_code.clone()),
                ),
                ("total".to_string(), json::Json::Num(cidade.total as f64)),
            ])
        })
        .collect();
    let grants = resumo
        .grants
        .iter()
        .map(|grant| {
            json::Json::Obj(vec![
                ("kind".to_string(), json::Json::Str(grant.kind.clone())),
                ("count".to_string(), json::Json::Num(grant.count as f64)),
                ("gold".to_string(), json::Json::Num(grant.gold as f64)),
            ])
        })
        .collect();
    json::Json::Obj(vec![
        ("tax_by_city".to_string(), json::Json::Arr(cidades)),
        ("grants".to_string(), json::Json::Arr(grants)),
    ])
    .render()
}

fn account_json(account: &AccountSummary) -> String {
    json::Json::Obj(vec![
        ("id".to_string(), json::Json::Num(account.id as f64)),
        ("login".to_string(), json::Json::Str(account.login.clone())),
        (
            "role".to_string(),
            json::Json::Str(account.role.as_db().to_string()),
        ),
    ])
    .render()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn route_accepts_only_contract_shape() {
        assert_eq!(route("/admin/v1/accounts/find"), Some(("accounts", "find")));
        assert_eq!(route("/admin/v1/accounts"), None);
        assert_eq!(route("/admin/v1/accounts/find/extra"), None);
        assert_eq!(route("/admin/v1/Accounts/find"), None);
        assert_eq!(route("/outro/accounts/find"), None);
    }

    #[test]
    fn bearer_is_case_insensitive_and_rejects_empty() {
        assert_eq!(bearer(Some("Bearer abc")), Some("abc"));
        assert_eq!(bearer(Some("bearer abc")), Some("abc"));
        assert_eq!(bearer(Some("Bearer   ")), None);
        assert_eq!(bearer(Some("Basic abc")), None);
        assert_eq!(bearer(None), None);
    }

    #[test]
    fn static_tokens_map_token_to_session() {
        let mut tokens = StaticTokens::default();
        tokens.insert(
            "t",
            Session {
                account_id: 7,
                role: Role::Admin,
            },
        );
        assert_eq!(
            tokens.authenticate("t"),
            Some(Session {
                account_id: 7,
                role: Role::Admin
            })
        );
        assert_eq!(tokens.authenticate("outro"), None);
    }
}
