//! `tmserver` — raiz de composição do monólito de jogo (ADR-0022).
//!
//! Nesta entrega o processo serve o que o cliente precisa **antes** do mundo:
//!
//! - `cpsock`: listener `:8281` que aceita `INITCODE`, desofusca o frame, aplica o gate de
//!   versão (ADR-0020) e responde `_MSG_CNFAccountLogin`;
//! - `status`: `GET /serv00.htm` no `:80` com o CSV de canais lido do Postgres — é o que
//!   popula o seletor de canais do cliente;
//! - `metrics`: `GET /metrics` no `:9898` (contrato da FASE-06-1) com o fragmento de rede
//!   (`wyd_packets_*`, `wyd_handshakes_total`, `wyd_sessions_open`, versão de cliente) e os
//!   gauges do serviço.
//!
//! O que ainda **não** existe aqui: tick do `wyd-core`, autenticação de conta e pacotes de
//! cena (`ESCENE_FIELD`). Cada um entra pela issue dona; nenhum deles é simulado.

pub mod channels;
pub mod cpsock;
pub mod metrics;
pub mod status;

pub use channels::{render_status_csv, status_url, World};
pub use cpsock::{serve_listener, HandshakeError, HandshakeOk};
pub use metrics::ServiceMetrics;
