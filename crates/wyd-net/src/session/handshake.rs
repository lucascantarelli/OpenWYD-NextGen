//! Gate de versão do login (`MSG_AccountLogin.client_version`).
//!
//! Decisão em `docs/arquitetura/adr/0020-versao-de-cliente-no-login.md`: `APP_VERSION = 7640` é a
//! constante **histórica** do build de referência, quem manda é `PROTOCOL_VERSION` — a versão
//! medida no cliente oficial de `client/` (12000). Recusar com erro tipado e **contar** o valor
//! recebido é o que transforma "não conecta" em diagnóstico.

use crate::error::NetError;
use crate::metrics::NetMetrics;
use wyd_protocol::PROTOCOL_VERSION;

/// Como o gate trata uma versão diferente da esperada.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum VersionPolicy {
    /// Produção: só a versão esperada entra (default).
    Strict,
    /// Captura/replay/diagnóstico: aceita e registra, para não perder a sessão a observar.
    Tolerant,
}

/// Compara a versão declarada no login com a esperada pelo servidor.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct VersionGate {
    expected: u16,
    policy: VersionPolicy,
}

impl VersionGate {
    /// Gate estrito para `expected`.
    #[must_use]
    pub const fn new(expected: u16) -> Self {
        Self {
            expected,
            policy: VersionPolicy::Strict,
        }
    }

    /// Gate estrito com o valor medido no cliente oficial (`PROTOCOL_VERSION`).
    #[must_use]
    pub const fn from_protocol_version() -> Self {
        Self::new(PROTOCOL_VERSION)
    }

    /// Gate que aceita qualquer versão e só a registra (captura).
    #[must_use]
    pub const fn tolerant(expected: u16) -> Self {
        Self {
            expected,
            policy: VersionPolicy::Tolerant,
        }
    }

    /// Versão esperada, como o boot/configuração a informou.
    #[must_use]
    pub const fn expected(&self) -> u16 {
        self.expected
    }

    /// Política em vigor.
    #[must_use]
    pub const fn policy(&self) -> VersionPolicy {
        self.policy
    }

    /// Valida a versão do login e conta o valor recebido (`wyd_client_version_total{version=…}`).
    ///
    /// # Errors
    ///
    /// [`NetError::ClientVersionUnsupported`] quando a política é `Strict` e a versão difere.
    pub fn check(&self, received: i32, metrics: &NetMetrics) -> Result<(), NetError> {
        metrics.client_version(received);
        if received == i32::from(self.expected) || self.policy == VersionPolicy::Tolerant {
            return Ok(());
        }
        metrics.client_version_rejected();
        Err(NetError::ClientVersionUnsupported {
            received,
            expected: self.expected,
        })
    }
}

impl Default for VersionGate {
    fn default() -> Self {
        Self::from_protocol_version()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn aceita_a_versao_do_cliente_oficial() {
        let gate = VersionGate::from_protocol_version();
        assert_eq!(gate.expected(), 12_000);
        assert_eq!(gate.check(12_000, &NetMetrics::new()), Ok(()));
    }

    #[test]
    fn recusa_a_constante_historica_7640() {
        let gate = VersionGate::default();
        let erro = gate
            .check(7640, &NetMetrics::new())
            .expect_err("7640 é a constante histórica, não o cliente atendido");
        assert_eq!(
            erro,
            NetError::ClientVersionUnsupported {
                received: 7640,
                expected: 12_000,
            }
        );
    }

    #[test]
    fn tolerante_aceita_mas_nao_muda_o_esperado() {
        let gate = VersionGate::tolerant(12_000);
        assert_eq!(gate.policy(), VersionPolicy::Tolerant);
        assert_eq!(gate.check(7640, &NetMetrics::new()), Ok(()));
    }

    #[test]
    fn gate_e_configuravel_no_boot() {
        let gate = VersionGate::new(13_001);
        assert_eq!(gate.check(13_001, &NetMetrics::new()), Ok(()));
        assert!(gate.check(12_000, &NetMetrics::new()).is_err());
    }
}
