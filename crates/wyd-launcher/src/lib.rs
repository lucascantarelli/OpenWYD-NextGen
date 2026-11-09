//! Launcher/patcher do WYD: manifesto de integridade, patch atômico e
//! bootstrap do jogo.
//!
//! Estratégia em `docs/arquitetura/adr/0003-estrategia-patch-cliente.md`;
//! especificação em `docs/cliente/customizacao.md`.

pub mod bootstrap;
pub mod branding;
pub mod distribuicao;
pub mod gui;
pub mod identidade;
pub mod manifest;
pub mod patch;
pub mod serverlist;
pub mod sha256;
pub mod snbin;
pub mod wyt;

/// Nome do binario do launcher.
pub const LAUNCHER_NAME: &str = "wyd-launcher";

/// Mensagem de bootstrap exibida pelo binario minimo.
#[must_use]
pub fn bootstrap_message() -> &'static str {
    "wyd-launcher: manifesto, patch e bootstrap (FASE-08-7)"
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_message_mentions_launcher() {
        assert!(bootstrap_message().contains(LAUNCHER_NAME));
    }
}
