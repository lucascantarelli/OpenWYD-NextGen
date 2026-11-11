//! `tmserver` — raiz de composição do monólito de jogo (ADR-0022).
//!
//! Nesta entrega o processo serve o que o cliente precisa **antes** do mundo:
//!
//! - `cpsock`: listener `:8281` que aceita `INITCODE`, desofusca o frame, aplica o gate de
//!   versão (ADR-0020), abre a **sessão única** da conta (`session`, índice parcial no
//!   `character_session`), responde `_MSG_CNFAccountLogin` e atende a **seleção de
//!   personagem** (`_MSG_CreateCharacter`/`_MSG_DeleteCharacter`/`_MSG_CharacterLogin`) sem
//!   derrubar a conexão; `characters` é a porta que traduz esses pacotes no Postgres;
//! - `status`: `GET /serv00.htm` no `:80` com os números dos canais lidos do Postgres e a
//!   sonda UDP do `:8281` — é o que popula e valida o seletor de canais do cliente;
//! - `metrics`: `GET /metrics` no `:9898` (contrato da FASE-06-1) com o fragmento de rede
//!   (`wyd_packets_*`, `wyd_handshakes_total`, `wyd_sessions_open`, versão de cliente), o
//!   registro **real** do motor (`wyd_ticks_total`, `wyd_tick_duration_microseconds`,
//!   `wyd_entities_active`) e os gauges do serviço;
//! - `world`: o `Engine` do `wyd-core` deste processo — tick fixo a `WYD_TICK_HZ`, mundo em
//!   memória e parada limpa da thread (FASE-03-7).
//!
//! O que ainda **não** existe aqui: personagem como entidade do mundo, save na saída
//! (03-8) e pacotes de cena (`ESCENE_FIELD`, 03-9). Cada um entra pela issue dona; nenhum
//! deles é simulado.

pub mod auth;
pub mod boot;
pub mod channels;
pub mod character;
pub mod config;
pub mod cpsock;
pub mod gameplay;
pub mod logs;
pub mod metrics;
pub mod status;
pub mod world;

// Caminhos históricos preservados durante a migração da ADR-0026.
pub use character as characters;
pub use character::session;
pub use character::snapshot as personagens;

pub use channels::{render_status_body, status_url, World};
pub use channels::{ChannelRules, ROLE_ENV};
pub use characters::{SelectionError, SelectionStore};
pub use config::{ConfigSource, GameConfig, LiveConfig, PgConfigSource, ReloadOutcome};
pub use cpsock::{serve_listener, HandshakeError, HandshakeOk};
pub use metrics::ServiceMetrics;
pub use session::{ChannelPolicy, PolicyOverrides, SessionError, SessionGate, SessionLease};
pub use world::{Engine, EngineThread};
