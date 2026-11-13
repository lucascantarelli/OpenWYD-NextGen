//! Erro da fronteira de rede (`NetError`): tamanho → opcode → payload → sessão.
//!
//! O crate não inventa variante para o que ainda não implementa; a ordem de validação é a do
//! legado (`docs/arquitetura/protocolo.md` §2) e o erro nunca carrega bytes crus.

use core::fmt;

/// Falha tipada de rede. Entrada hostil vira `Err`, nunca `panic` (skill `wyd-rust-net`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NetError {
    /// Versão de cliente recusada no handshake (`docs/arquitetura/adr/0020-versao-de-cliente-no-login.md`).
    ///
    /// `expected` é o valor de `PROTOCOL_VERSION` (medido no cliente oficial de `client/`);
    /// `received` é o que veio no campo `client_version` do `MSG_AccountLogin`.
    ClientVersionUnsupported { received: i32, expected: u16 },
}

impl fmt::Display for NetError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::ClientVersionUnsupported { received, expected } => write!(
                f,
                "versão de cliente {received} não suportada (esperado {expected})"
            ),
        }
    }
}

impl std::error::Error for NetError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn display_carrega_os_dois_valores() {
        let erro = NetError::ClientVersionUnsupported {
            received: 7640,
            expected: 12_000,
        };
        let texto = erro.to_string();
        assert!(texto.contains("7640"), "{texto}");
        assert!(texto.contains("12000"), "{texto}");
    }
}
