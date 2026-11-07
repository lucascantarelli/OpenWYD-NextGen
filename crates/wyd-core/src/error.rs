//! Erros do Hot Path — nenhum caminho do tick causa panic.
//!
//! Toda fronteira (config, spawn, grade, relógio) falha explícita com contexto;
//! default silencioso é violação (ver FASE-03 passo 4).

use core::fmt;

/// Erro de domínio da engine (fronteira que o produziu no nome da variante).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CoreError {
    /// Frequência fora da faixa contratada (20–50 Hz).
    BadTickHz { hz: u32 },
    /// Componente com invariante estrutural violado no spawn.
    InvalidStats { field: &'static str, value: u32 },
    /// Posição fora dos limites do mundo (nunca clamp silencioso).
    OutOfBounds { x: i32, y: i32 },
    /// Dimensão da grade espacial inválida no boot.
    InvalidGrid { field: &'static str, value: i32 },
    /// Relógio voltou no tempo além do tolerável pelo acumulador.
    TimeWentBackwards { before_ms: u64, now_ms: u64 },
}

impl fmt::Display for CoreError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::BadTickHz { hz } => {
                write!(f, "frequência de tick inválida: {hz} Hz (contrato: 20–50)")
            }
            Self::InvalidStats { field, value } => {
                write!(f, "stats inválido no spawn: {field} = {value}")
            }
            Self::OutOfBounds { x, y } => {
                write!(f, "posição fora do mundo: ({x}, {y})")
            }
            Self::InvalidGrid { field, value } => {
                write!(f, "grade espacial inválida: {field} = {value}")
            }
            Self::TimeWentBackwards { before_ms, now_ms } => {
                write!(
                    f,
                    "relógio voltou no tempo: antes {before_ms} ms, agora {now_ms} ms"
                )
            }
        }
    }
}

impl std::error::Error for CoreError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_variant_displays_in_portuguese() {
        assert_eq!(
            CoreError::BadTickHz { hz: 10 }.to_string(),
            "frequência de tick inválida: 10 Hz (contrato: 20–50)"
        );
        assert_eq!(
            CoreError::InvalidStats {
                field: "max_hp",
                value: 0
            }
            .to_string(),
            "stats inválido no spawn: max_hp = 0"
        );
        assert_eq!(
            CoreError::OutOfBounds { x: -1, y: 5000 }.to_string(),
            "posição fora do mundo: (-1, 5000)"
        );
        assert_eq!(
            CoreError::InvalidGrid {
                field: "cell",
                value: 0
            }
            .to_string(),
            "grade espacial inválida: cell = 0"
        );
        assert!(CoreError::TimeWentBackwards {
            before_ms: 100,
            now_ms: 50
        }
        .to_string()
        .contains("voltou no tempo"));
    }
}
