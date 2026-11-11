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

/// Texto da tela de conexão para um código de recusa do servidor.
///
/// Não há texto aqui: `wyd_bots::tela::render` é a fonte única, a mesma chamada pela CLI
/// headless. Código 0/desconhecido cai na frase genérica acionável do renderizador.
#[must_use]
pub fn mensagem_de_recusa(codigo: u8) -> String {
    wyd_bots::tela::render(&wyd_bots::tela::por_codigo(codigo))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_message_mentions_launcher() {
        assert!(bootstrap_message().contains(LAUNCHER_NAME));
    }

    #[test]
    fn tela_de_recusa_usa_o_mesmo_render_da_cli() {
        // Prova de render compartilhado: a saída do launcher é idêntica à da função pura.
        for codigo in 0u8..=7 {
            assert_eq!(
                mensagem_de_recusa(codigo),
                wyd_bots::tela::render(&wyd_bots::tela::por_codigo(codigo))
            );
            assert!(!mensagem_de_recusa(codigo).trim().is_empty());
        }
    }
}
