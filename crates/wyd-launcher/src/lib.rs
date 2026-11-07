//! Launcher/patcher (esqueleto).

/// Nome do binario do launcher.
pub const LAUNCHER_NAME: &str = "wyd-launcher";

#[must_use]
/// Mensagem de bootstrap exibida pelo binario minimo.
pub fn bootstrap_message() -> &'static str {
    "wyd-launcher: esqueleto OK (hash/patch virao nas issues FASE 8)"
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_message_mentions_launcher() {
        assert!(bootstrap_message().contains(LAUNCHER_NAME));
    }
}
