//! Fórmulas puras do jogo — fato separado de hipótese.
//!
//! Um arquivo por sistema, cada um com entrada e saída numéricas: sem I/O, sem `rand`, sem
//! estado. Fórmula nova vira módulo novo (OCP), nunca um `if` a mais em `damage.rs`.
//!
//! Mapa completo (entradas, saídas, unidade, ordem no tick, arredondamento e estado):
//! `docs/arquitetura/matematica-sim.md`.

pub mod attributes;
pub mod damage;
pub mod drop;
pub mod effect;
pub mod hit;
pub mod progression;
pub mod refine;
pub mod regen;
pub mod resist;
pub mod threat;

pub use attributes::*;
pub use damage::*;
pub use drop::*;
pub use effect::*;
pub use hit::*;
pub use progression::*;
pub use refine::*;
pub use regen::*;
pub use resist::*;
pub use threat::*;

#[cfg(test)]
mod tests {
    use super::*;
    use crate::units::Bps;

    /// Liga os sistemas numa única conta: derivado → dano → drop. Se este teste quebrar,
    /// alguma unidade mudou de escala entre módulos (bps virou inteiro, por exemplo).
    #[test]
    fn systems_compose_with_the_same_units() {
        let perfil = GrowthProfile {
            hp_per_level: 10,
            mp_per_level: 5,
            attack_per_level: 1,
            defense_per_level: 1,
            weight: AttrWeight::ZERO,
        };
        let derivado = attributes::derive(10, AttrPoints::ZERO, GearBonus::default(), perfil);
        assert_eq!(derivado.attack, 9);

        let golpe = DamageInput {
            attribute: derivado.attack,
            skill_bps: Bps::from_percent(150),
            ..DamageInput::new(100)
        };
        assert_eq!(damage_chain(&golpe), 163);

        let gema = apply_gem(GEM_DIAMOND, 10, false).expect("gema válida");
        let taxa = drop::effective_rate(10_000, drop::gem_bonus(gema.effect));
        assert_eq!(taxa, 10_800);
        assert_eq!(drop::drops(taxa, 10_799), Ok(true));
        assert_eq!(drop::drops(taxa, 10_801), Ok(false));
    }
}
