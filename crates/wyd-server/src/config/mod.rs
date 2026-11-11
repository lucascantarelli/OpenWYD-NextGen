//! Configuração viva do processo de jogo: snapshot imutável, revisão e recarga.
//!
//! ADR-0026 (etapa 5): a fachada só declara o domínio; a leitura do banco e a troca
//! atômica do snapshot ficam em [`service`].

mod service;

pub use service::*;
