//! Cadência e orçamento da IA — 5 Hz, com fase por entidade (D-IA01).
//!
//! A 20 Hz, decidir todo tick seria caro e concentraria todos os NPCs no mesmo instante.
//! A regra é `tick % 4 == entity_index % 4`: cada criatura decide uma vez a cada 4 ticks,
//! espalhadas pelo ciclo. Movimento e combate continuam no tick cheio, validados pelos
//! sistemas de autoridade.

/// Ticks entre duas decisões da mesma criatura (`20 Hz / 4 = 5 Hz`).
pub const DECISION_EVERY_TICKS: u32 = 4;

/// Esta criatura decide neste tick?
#[must_use]
pub fn should_decide(tick: u32, entity_index: u32) -> bool {
    tick % DECISION_EVERY_TICKS == entity_index % DECISION_EVERY_TICKS
}

/// Teto de decisões por tick: ao estourar, a criatura espera o próximo ciclo em vez de
/// atrasar o tick (meta ≤ 10% do orçamento — `UNVERIFIED` até o bench existir).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DecisionBudget {
    /// Máximo de decisões neste tick.
    pub max_per_tick: u16,
    /// Já consumidas neste tick.
    pub used: u16,
}

impl DecisionBudget {
    /// Orçamento do tick. `max_per_tick = 0` desliga a IA (nenhuma decisão).
    #[must_use]
    pub fn new(max_per_tick: u16) -> Self {
        Self {
            max_per_tick,
            used: 0,
        }
    }

    /// Consome uma decisão; `false` quando o teto do tick já foi atingido.
    pub fn try_consume(&mut self) -> bool {
        if self.used >= self.max_per_tick {
            return false;
        }
        self.used += 1;
        true
    }

    /// Reinicia no começo de cada tick.
    pub fn reset(&mut self) {
        self.used = 0;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn cada_entidade_decide_uma_vez_a_cada_quatro_ticks() {
        for entity in 0..4u32 {
            let decididos: Vec<u32> = (0..8u32)
                .filter(|tick| should_decide(*tick, entity))
                .collect();
            assert_eq!(decididos, vec![entity, entity + 4], "fase = entity % 4");
        }
    }

    #[test]
    fn orcamento_recusa_acima_do_teto_e_reseta() {
        let mut orcamento = DecisionBudget::new(2);
        assert!(orcamento.try_consume());
        assert!(orcamento.try_consume());
        assert!(!orcamento.try_consume());
        orcamento.reset();
        assert!(orcamento.try_consume());
    }
}
