//! Resistência elemental — redução percentual por elemento, trancada em 100%.
//!
//! **Fato:** existe resistência por elemento (escudo de gelo, proteção de fogo,
//! `docs/wyd/classes.md` §2) e existe `affect_resist` por skill (`docs/wyd/skills.md` §4).
//! A tabela completa de elementos do build 7662 não está publicada: aqui ficam os três que
//! as fontes nomeiam, e `Element` cresce quando o catálogo (`item_effect`) provar os demais —
//! enum exaustivo faz o compilador cobrar o caso novo.
//!
//! Sem teto inventado: a redução satura em 100% porque "reduzir mais que tudo" é identidade
//! aritmética, não regra de jogo.

use crate::units::{mul_bps, Bps};

/// Elemento de um golpe.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum Element {
    /// Dano físico (arma): resistência física é defesa, não este eixo.
    #[default]
    Physical,
    /// Fogo.
    Fire,
    /// Gelo (congelamento, *Enchant Frost*).
    Frost,
}

impl Element {
    /// Todos os elementos conhecidos — iteração estável para relatório.
    pub const ALL: [Self; 3] = [Self::Physical, Self::Fire, Self::Frost];
}

/// Resistência por elemento, em bps (`1_000` = 10%).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ResistProfile {
    /// Resistência a fogo.
    pub fire_bps: Bps,
    /// Resistência a gelo.
    pub frost_bps: Bps,
}

// `Default` escrito à mão: `Bps` não implementa `Default` de propósito, porque `Bps::ZERO`
// e `Bps::FULL` são intenções diferentes e um `0` implícito em buff percentual zeraria dano.
impl Default for ResistProfile {
    fn default() -> Self {
        Self {
            fire_bps: Bps::ZERO,
            frost_bps: Bps::ZERO,
        }
    }
}

impl ResistProfile {
    /// Resistência ao elemento pedido; físico é sempre zero neste eixo.
    #[must_use]
    pub const fn of(self, element: Element) -> Bps {
        match element {
            Element::Physical => Bps::ZERO,
            Element::Fire => self.fire_bps,
            Element::Frost => self.frost_bps,
        }
    }
}

/// Aplica a resistência: `dano − dano × resistência`, saturando em zero.
#[must_use]
pub fn mitigate(damage: u32, resist: Bps) -> u32 {
    let efetiva = resist.clamp(Bps::ZERO, Bps::FULL);
    damage.saturating_sub(mul_bps(damage, efetiva.raw()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn no_resist_keeps_damage() {
        assert_eq!(mitigate(100, Bps::ZERO), 100);
    }

    #[test]
    fn resist_is_percentage_with_floor() {
        assert_eq!(mitigate(100, Bps::from_percent(30)), 70);
        // 33% de 99 = 32,67 → piso 32; sobra 67.
        assert_eq!(mitigate(99, Bps::from_percent(33)), 67);
    }

    #[test]
    fn full_resist_zeroes_damage() {
        assert_eq!(mitigate(100, Bps::FULL), 0);
        // Acima de 100% não vira cura.
        assert_eq!(mitigate(100, Bps::from_percent(250)), 0);
    }

    #[test]
    fn profile_maps_elements() {
        let perfil = ResistProfile {
            fire_bps: Bps::from_percent(40),
            frost_bps: Bps::from_percent(10),
        };
        assert_eq!(perfil.of(Element::Fire).raw(), 4_000);
        assert_eq!(perfil.of(Element::Frost).raw(), 1_000);
        assert_eq!(perfil.of(Element::Physical), Bps::ZERO);
        assert_eq!(Element::ALL.len(), 3);
    }
}
