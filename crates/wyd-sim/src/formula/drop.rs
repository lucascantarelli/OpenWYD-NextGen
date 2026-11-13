//! Drop por item — um roll por linha, escala 1..1.000.000.
//!
//! **CONFIRMADO** (`docs/wyd/sistemas/drops.md` §1): cada item do monstro sorteia um número
//! de 1 a 1.000.000 e cai se `roll <= rate`. O erro clássico é fazer **um** roll e escolher
//! item ponderado — distribuição "mais justa" e errada. O módulo só implementa a regra certa:
//! é o chamador quem itera as linhas chamando [`drops`] uma vez por item.
//!
//! **Caixa no chão** (`docs/wyd/sistemas/drops.md` §8, regra de projeto do servidor): o drop
//! vai direto para o inventário; o que não coube vira caixa, reservada ao matador por
//! [`GROUND_EXCLUSIVE_MS`], aberta a qualquer jogador depois disso e removida em
//! [`GROUND_DECAY_MS`]. [`GroundLoot`] é a janela pura — sem timer, calculada na leitura.
//!
//! **Hipótese (D-08):** bônus de taxa (gema de diamante, +8%) entra como percentual sobre a
//! taxa e satura na escala do roll. A fonte diz que o bônus existe e é aplicado depois do
//! roll para bônus de canal, com teto — o teto aqui é `DROP_ROLL_MAX`, o topo da própria
//! escala, em vez de um número inventado.

use crate::error::SimError;
use crate::formula::refine::{GemEffect, GEM_DIAMOND};
use crate::units::Bps;

/// Topo da escala de drop do original (`1` = 0,0001%).
pub const DROP_ROLL_MAX: u32 = 1_000_000;

/// Janela em que a caixa no chão pertence a **quem matou** (`docs/wyd/sistemas/drops.md` §8).
///
/// É regra de projeto do servidor, não número do original: o drop moderno vai direto para o
/// inventário e só vira caixa quando não coube; a caixa nasce reservada por 30 s.
pub const GROUND_EXCLUSIVE_MS: u64 = 30_000;

/// Tempo de vida total da caixa no chão: depois disso ela some, com ou sem dono.
pub const GROUND_DECAY_MS: u64 = 180_000;

/// Estado da janela de posse de uma caixa no chão.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum LootAccess {
    /// Só o dono pode pegar (janela exclusiva).
    OwnerOnly,
    /// Qualquer jogador pode pegar.
    Open,
    /// Passou do decay: a caixa não existe mais.
    Gone,
}

/// Caixa no chão: dono e instante da queda (ms do processo).
///
/// `owner` é o id de personagem de quem matou; `dropped_at_ms` vem do relógio do processo —
/// **nunca** do cliente. A janela é calculada na leitura, então não há timer por caixa.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct GroundLoot {
    pub owner: u64,
    pub dropped_at_ms: u64,
}

impl GroundLoot {
    #[must_use]
    pub fn new(owner: u64, dropped_at_ms: u64) -> Self {
        Self {
            owner,
            dropped_at_ms,
        }
    }

    /// Quanto tempo a caixa está no chão. Relógio que anda para trás conta como instante zero
    /// (caixa recém-caída), nunca como caixa expirada.
    #[must_use]
    pub fn age_ms(&self, now_ms: u64) -> u64 {
        now_ms.saturating_sub(self.dropped_at_ms)
    }

    /// Instante em que a caixa some.
    #[must_use]
    pub fn expires_at_ms(&self) -> u64 {
        self.dropped_at_ms.saturating_add(GROUND_DECAY_MS)
    }

    /// Janela vigente, independente de quem pergunta.
    #[must_use]
    pub fn access(&self, now_ms: u64) -> LootAccess {
        let idade = self.age_ms(now_ms);
        if idade >= GROUND_DECAY_MS {
            LootAccess::Gone
        } else if idade < GROUND_EXCLUSIVE_MS {
            LootAccess::OwnerOnly
        } else {
            LootAccess::Open
        }
    }

    /// O jogador `viewer` pode pegar agora?
    #[must_use]
    pub fn can_take(&self, viewer: u64, now_ms: u64) -> bool {
        match self.access(now_ms) {
            LootAccess::OwnerOnly => viewer == self.owner,
            LootAccess::Open => true,
            LootAccess::Gone => false,
        }
    }
}

/// Bônus de drop da gema de diamante (`+8%` sobre a taxa — Kersef, *Refining +10*).
pub const GEM_DROP_BONUS: Bps = Bps::new(800);

/// Índice de catálogo da gema de diamante (reencontro entre catálogo e fórmula).
pub const DIAMOND_ITEM: u32 = GEM_DIAMOND;

/// Bônus da gema aplicável a uma taxa, se a gema for a de diamante.
#[must_use]
pub fn gem_bonus(effect: GemEffect) -> Bps {
    match effect {
        GemEffect::Drop => GEM_DROP_BONUS,
        GemEffect::Pierce | GemEffect::Exp | GemEffect::Absorb => Bps::ZERO,
    }
}

/// Teto do bônus de canal/cidadania: **50%**.
///
/// O legado descreve o bônus de cidadania (`Language.txt:423/432`) sem publicar a taxa; a
/// referência de produção dá o teto como +1% por 1.000.000.000 de imposto até 50%
/// (`docs/wyd/canais-e-cidadania.md` §2). Aqui ele é **teto de sanidade de parâmetro** — não
/// taxa aplicada: o valor efetivo vem do dado do canal (`game_setting.citizen_bonus_bps`).
pub const CITIZEN_BONUS_MAX: Bps = Bps::new(5_000);

/// Soma bônus de origens diferentes (gema, cidadania) sem estourar `u32`.
///
/// Os bônus são **composições aditivas**: cada um incide sobre a mesma taxa-base, nunca um
/// sobre o resultado do outro (`effective_rate` aplica a soma uma vez só).
#[must_use]
pub fn sum_bonus(bonuses: &[Bps]) -> Bps {
    Bps::new(
        bonuses
            .iter()
            .fold(0u32, |soma, bonus| soma.saturating_add(bonus.raw())),
    )
}

/// Taxa efetiva: `rate + rate × bônus`, limitada ao topo da escala.
#[must_use]
pub fn effective_rate(rate: u32, bonus: Bps) -> u32 {
    rate.saturating_add(bonus.apply(rate)).min(DROP_ROLL_MAX)
}

/// Taxa efetiva com bônus compostos (gema + canal), aplicados uma vez sobre a base.
#[must_use]
pub fn effective_rate_with(rate: u32, bonuses: &[Bps]) -> u32 {
    effective_rate(rate, sum_bonus(bonuses))
}

/// O item cai? `roll` precisa estar em `1..=DROP_ROLL_MAX`.
///
/// # Errors
/// [`SimError::RollOutOfRange`] fora da escala — roll zero silencioso esconderia bug de
/// gerador, então é erro tipado.
pub fn drops(rate: u32, roll: u32) -> Result<bool, SimError> {
    if !(1..=DROP_ROLL_MAX).contains(&roll) {
        return Err(SimError::RollOutOfRange {
            roll,
            max: DROP_ROLL_MAX,
        });
    }
    Ok(roll <= rate)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn roll_equal_to_rate_drops_and_one_above_does_not() {
        assert_eq!(drops(1_000, 1_000), Ok(true));
        assert_eq!(drops(1_000, 1_001), Ok(false));
        assert_eq!(drops(0, 1), Ok(false));
        assert_eq!(drops(DROP_ROLL_MAX, DROP_ROLL_MAX), Ok(true));
    }

    #[test]
    fn roll_outside_scale_is_an_error() {
        assert_eq!(
            drops(10, 0),
            Err(SimError::RollOutOfRange {
                roll: 0,
                max: DROP_ROLL_MAX
            })
        );
        assert_eq!(
            drops(10, DROP_ROLL_MAX + 1),
            Err(SimError::RollOutOfRange {
                roll: DROP_ROLL_MAX + 1,
                max: DROP_ROLL_MAX
            })
        );
    }

    #[test]
    fn diamond_gem_raises_rate_by_eight_percent_and_caps() {
        assert_eq!(effective_rate(10_000, GEM_DROP_BONUS), 10_800);
        assert_eq!(effective_rate(1, GEM_DROP_BONUS), 1, "8% de 1 trunca em 0");
        assert_eq!(effective_rate(DROP_ROLL_MAX, GEM_DROP_BONUS), DROP_ROLL_MAX);
    }

    #[test]
    fn bonus_de_cidadania_soma_com_a_gema_e_respeita_o_teto() {
        // 50% de cidadania + 8% da gema sobre 10.000 = 15.800 (soma, não composição).
        assert_eq!(
            effective_rate_with(10_000, &[GEM_DROP_BONUS, CITIZEN_BONUS_MAX]),
            15_800
        );
        // Sem bônus, a taxa é a base (default do servidor é 0).
        assert_eq!(effective_rate_with(10_000, &[Bps::ZERO]), 10_000);
        // Nenhum bônus vira teto da escala de sorteio.
        assert_eq!(
            effective_rate_with(DROP_ROLL_MAX, &[CITIZEN_BONUS_MAX]),
            DROP_ROLL_MAX
        );
        // Soma satura em `u32`, nunca estoura.
        assert_eq!(
            sum_bonus(&[Bps::new(u32::MAX), Bps::new(10)]).raw(),
            u32::MAX
        );
        assert_eq!(CITIZEN_BONUS_MAX.raw(), 5_000);
    }

    #[test]
    fn only_diamond_carries_the_drop_bonus() {
        assert_eq!(gem_bonus(GemEffect::Drop).raw(), 800);
        assert_eq!(gem_bonus(GemEffect::Pierce), Bps::ZERO);
        assert_eq!(gem_bonus(GemEffect::Exp), Bps::ZERO);
        assert_eq!(gem_bonus(GemEffect::Absorb), Bps::ZERO);
        assert_eq!(DIAMOND_ITEM, 3386);
    }

    #[test]
    fn caixa_fica_exclusiva_por_trinta_segundos() {
        let caixa = GroundLoot::new(7, 1_000);
        assert_eq!(caixa.access(1_000), LootAccess::OwnerOnly);
        assert_eq!(caixa.access(30_999), LootAccess::OwnerOnly);
        assert!(caixa.can_take(7, 30_999), "dono pega na janela");
        assert!(
            !caixa.can_take(8, 30_999),
            "outro jogador não pega na janela"
        );
    }

    #[test]
    fn caixa_abre_para_todos_depois_da_janela_e_some_no_decay() {
        let caixa = GroundLoot::new(7, 1_000);
        assert_eq!(caixa.access(31_000), LootAccess::Open);
        assert!(caixa.can_take(8, 31_000), "liberada para qualquer jogador");
        assert_eq!(caixa.access(180_999), LootAccess::Open);
        assert_eq!(caixa.access(181_000), LootAccess::Gone);
        assert!(
            !caixa.can_take(7, 181_000),
            "dono também não pega depois do decay"
        );
        assert_eq!(caixa.expires_at_ms(), 181_000);
    }

    #[test]
    fn relogio_que_anda_para_tras_nao_expera_a_caixa() {
        let caixa = GroundLoot::new(7, 50_000);
        assert_eq!(caixa.age_ms(49_000), 0);
        assert_eq!(caixa.access(49_000), LootAccess::OwnerOnly);
        assert!(caixa.can_take(7, 49_000));
    }
}
