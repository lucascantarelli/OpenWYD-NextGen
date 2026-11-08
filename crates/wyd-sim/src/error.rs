//! Erro do laboratório — um por fronteira do crate.
//!
//! `formula::refine` mantém o próprio `GemError` (fronteira de política de gema, já
//! consumida por `wyd-core` com os mesmos números); o resto do crate usa [`SimError`].
//! Nenhuma função pública faz `panic!` por entrada fora de faixa: devolve erro tipado.

use std::fmt;

/// Entrada inválida para uma fórmula ou estrutura de simulação.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SimError {
    /// Roll fora da escala: drop é `1..=1_000_000`, chance é `0..10_000`.
    RollOutOfRange {
        /// Roll recebido.
        roll: u32,
        /// Teto da escala usada.
        max: u32,
    },
    /// Cadência de tick zero — divisor indefinido.
    ZeroTickRate,
    /// Tabela de efeitos ativos cheia; nada foi aplicado.
    EffectSlotsFull {
        /// Capacidade fixa do conjunto.
        capacity: usize,
    },
    /// Tabela de ameaça cheia; nada foi registrado.
    ThreatTableFull {
        /// Capacidade fixa da tabela.
        capacity: usize,
    },
    /// Curva de EXP vazia (não há nível 1).
    EmptyExpTable,
    /// Curva de EXP não é estritamente crescente na posição indicada.
    NonMonotonicExp {
        /// Nível cujo limiar está fora de ordem (1-based).
        level: u16,
    },
}

impl fmt::Display for SimError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::RollOutOfRange { roll, max } => {
                write!(f, "roll {roll} fora da escala 1..={max}")
            }
            Self::ZeroTickRate => write!(f, "cadência de tick zero"),
            Self::EffectSlotsFull { capacity } => {
                write!(f, "tabela de efeitos cheia ({capacity} slots)")
            }
            Self::ThreatTableFull { capacity } => {
                write!(f, "tabela de ameaça cheia ({capacity} entradas)")
            }
            Self::EmptyExpTable => write!(f, "curva de EXP vazia"),
            Self::NonMonotonicExp { level } => {
                write!(f, "curva de EXP não cresce no nível {level}")
            }
        }
    }
}

impl std::error::Error for SimError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn display_names_the_offending_value() {
        let texto = SimError::RollOutOfRange {
            roll: 0,
            max: 1_000_000,
        }
        .to_string();
        assert!(texto.contains('0'), "texto: {texto}");
        assert!(texto.contains("1000000"), "texto: {texto}");
    }
}
