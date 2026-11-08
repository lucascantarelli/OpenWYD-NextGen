//! Skills e efeitos com duração — contagem em ticks, sem timer de sistema operacional.
//!
//! **Fato:** efeito tem `affect_type`, `value` e `time` no catálogo (`docs/wyd/skills.md` §4)
//! e durações publicadas são em segundos (64 s em *Flash*, *Proteção Absoluta*, *Clod Attack*;
//! `docs/wyd/classes.md` §2). Cooldown idem: `delay_ticks` — recarga no tick, nunca `sleep`.
//!
//! **Hipótese (D-07):** mesmo `affect_type` **não empilha**; nova aplicação troca a magnitude
//! e renova para a maior duração restante. É a política de menor complexidade e coincide com
//! o "usar qualquer outra skill desativa" descrito para *Proteção Absoluta*.
//!
//! O conjunto é `Copy`, tamanho fixo e ordem estável: sem `HashMap`, sem alocação por tick,
//! então dois replays com a mesma sequência de aplicações terminam no mesmo estado.

use crate::error::SimError;
use crate::units::Bps;

/// Teto de efeitos ativos simultâneos por entidade (fixo para não alocar no quente).
pub const MAX_ACTIVE_EFFECTS: usize = 16;

/// Efeito ativo: código do `affect_type`, magnitude e tempo restante.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TimedEffect {
    /// `affect_type` do catálogo (código estável; o nome é do conteúdo).
    pub affect: u16,
    /// Magnitude do efeito (pode ser negativa: debuff).
    pub magnitude: i32,
    /// Ticks restantes; `0` significa expirado e é removido pelo [`EffectSet::tick`].
    pub remaining_ticks: u32,
}

/// Conjunto de efeitos ativos, ordem de inserção preservada.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct EffectSet {
    slots: [Option<TimedEffect>; MAX_ACTIVE_EFFECTS],
    len: u8,
}

impl Default for EffectSet {
    fn default() -> Self {
        Self {
            slots: [None; MAX_ACTIVE_EFFECTS],
            len: 0,
        }
    }
}

impl EffectSet {
    /// Conjunto vazio.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Quantos efeitos ativos.
    #[must_use]
    pub fn len(&self) -> usize {
        usize::from(self.len)
    }

    /// Nenhum efeito ativo.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.len == 0
    }

    /// Aplica (ou renova) um efeito.
    ///
    /// Mesmo `affect`: magnitude substituída, duração vira a maior entre a atual e a nova.
    ///
    /// # Errors
    /// [`SimError::EffectSlotsFull`] quando não há slot livre; nada é alterado nesse caso.
    pub fn apply(&mut self, effect: TimedEffect) -> Result<(), SimError> {
        for ativo in self.slots.iter_mut().flatten() {
            if ativo.affect == effect.affect {
                ativo.magnitude = effect.magnitude;
                ativo.remaining_ticks = ativo.remaining_ticks.max(effect.remaining_ticks);
                return Ok(());
            }
        }
        if let Some(vazio) = self.slots.iter_mut().find(|s| s.is_none()) {
            *vazio = Some(effect);
            self.len = self.len.saturating_add(1);
            return Ok(());
        }
        Err(SimError::EffectSlotsFull {
            capacity: MAX_ACTIVE_EFFECTS,
        })
    }

    /// Efeito ativo com o código pedido.
    #[must_use]
    pub fn get(&self, affect: u16) -> Option<&TimedEffect> {
        self.slots
            .iter()
            .flatten()
            .find(|efeito| efeito.affect == affect)
    }

    /// Efeito ativo mutável (para consumo de carga, por exemplo).
    pub fn get_mut(&mut self, affect: u16) -> Option<&mut TimedEffect> {
        self.slots
            .iter_mut()
            .flatten()
            .find(|efeito| efeito.affect == affect)
    }

    /// Itera os efeitos ativos na ordem de inserção.
    pub fn iter(&self) -> impl Iterator<Item = &TimedEffect> {
        self.slots.iter().flatten()
    }

    /// Avança um tick: decrementa e remove os expirados. Devolve quantos saíram.
    pub fn tick(&mut self) -> usize {
        let mut removidos = 0usize;
        for slot in &mut self.slots {
            let expirou = match slot {
                Some(ativo) => {
                    ativo.remaining_ticks = ativo.remaining_ticks.saturating_sub(1);
                    ativo.remaining_ticks == 0
                }
                None => false,
            };
            if expirou {
                *slot = None;
                removidos += 1;
            }
        }
        self.len = self
            .len
            .saturating_sub(u8::try_from(removidos).unwrap_or(u8::MAX));
        removidos
    }
}

/// Converte duração publicada em segundos para ticks do servidor.
///
/// # Errors
/// [`SimError::ZeroTickRate`] quando a cadência é zero (divisão indefinida).
pub fn ticks_for_seconds(seconds: u32, tick_hz: u32) -> Result<u32, SimError> {
    if tick_hz == 0 {
        return Err(SimError::ZeroTickRate);
    }
    Ok(seconds.saturating_mul(tick_hz))
}

/// Duração que cresce com a maestria da linhagem e por modificador percentual.
///
/// Modela o que está confirmado para *Anti-Magia* ("duração cresce com os pontos na
/// linhagem") e *Holy Power* ("dobra a duração do debuff"): os coeficientes são entrada de
/// conteúdo, não constante do módulo.
#[must_use]
pub fn scaled_ticks(
    base_ticks: u32,
    mastery: u16,
    ticks_per_mastery_point: u32,
    duration_mod: Bps,
) -> u32 {
    let com_maestria =
        base_ticks.saturating_add(u32::from(mastery).saturating_mul(ticks_per_mastery_point));
    duration_mod.apply(com_maestria)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn efeito(affect: u16, magnitude: i32, remaining_ticks: u32) -> TimedEffect {
        TimedEffect {
            affect,
            magnitude,
            remaining_ticks,
        }
    }

    #[test]
    fn sixty_four_seconds_at_twenty_hz_is_1280_ticks() {
        assert_eq!(ticks_for_seconds(64, 20), Ok(1_280));
        assert_eq!(ticks_for_seconds(64, 0), Err(SimError::ZeroTickRate));
    }

    #[test]
    fn same_affect_replaces_magnitude_and_keeps_the_longest_duration() {
        let mut set = EffectSet::new();
        set.apply(efeito(11, 10, 100)).expect("cabe");
        set.apply(efeito(11, 25, 40)).expect("renova");
        let ativo = set.get(11).expect("ativo");
        assert_eq!(ativo.magnitude, 25);
        assert_eq!(ativo.remaining_ticks, 100);
        assert_eq!(set.len(), 1, "não empilha");
    }

    #[test]
    fn tick_expires_exactly_at_zero_and_reports() {
        let mut set = EffectSet::new();
        set.apply(efeito(1, 1, 2)).expect("cabe");
        assert_eq!(set.tick(), 0);
        assert_eq!(set.get(1).map(|e| e.remaining_ticks), Some(1));
        assert_eq!(set.tick(), 1);
        assert!(set.is_empty());
        assert_eq!(set.get(1), None);
    }

    #[test]
    fn full_set_refuses_without_mutating() {
        let mut set = EffectSet::new();
        for affect in 0..MAX_ACTIVE_EFFECTS {
            set.apply(efeito(u16::try_from(affect).expect("cabe"), 1, 10))
                .expect("cabe");
        }
        assert_eq!(
            set.apply(efeito(999, 1, 10)),
            Err(SimError::EffectSlotsFull {
                capacity: MAX_ACTIVE_EFFECTS
            })
        );
        assert_eq!(set.len(), MAX_ACTIVE_EFFECTS);
        assert_eq!(set.get(999), None);
    }

    #[test]
    fn scaled_duration_matches_mastery_and_modifier() {
        assert_eq!(scaled_ticks(100, 50, 2, Bps::FULL), 200);
        assert_eq!(scaled_ticks(100, 50, 2, Bps::from_percent(200)), 400);
    }

    #[test]
    fn iteration_order_is_insertion_order() {
        let mut set = EffectSet::new();
        set.apply(efeito(9, 1, 5)).expect("cabe");
        set.apply(efeito(3, 1, 5)).expect("cabe");
        set.apply(efeito(7, 1, 5)).expect("cabe");
        let ordem: Vec<u16> = set.iter().map(|e| e.affect).collect();
        assert_eq!(ordem, vec![9, 3, 7]);
    }
}
