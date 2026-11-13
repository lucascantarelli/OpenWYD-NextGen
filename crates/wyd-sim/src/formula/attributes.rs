//! Atributos e derivados — composição, não constante mágica.
//!
//! O que é **fato**: existem quatro atributos (FOR/INT/DES/CON — `pre-release-rust.md`
//! §10.1) e o Arch recebe **+8 pontos de atributo por nível** do personagem normal
//! (`docs/wyd/classes.md` §4, `docs/wyd/progressao.md` §2).
//!
//! O que **não** é fato: quanto HP vale 1 de CON, quanto dano vale 1 de FOR. Nenhuma fonte
//! pública publica isso para o build 7662 (`docs/wyd/progressao.md` §7). Por isso os pesos
//! são **entrada** ([`AttrWeight`]), carregados do banco — o módulo fixa a *composição* e a
//! ordem da soma, nunca o número. Decisão provisória D-01 em
//! `docs/arquitetura/matematica-sim.md` §4, a confirmar por trace
//! (`docs/cliente/depuracao.md`).

/// Pontos de atributo do personagem (os quatro do `pre-release-rust.md` §10.1).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct AttrPoints {
    /// Força.
    pub strength: u32,
    /// Inteligência.
    pub intelligence: u32,
    /// Destreza.
    pub dexterity: u32,
    /// Constituição.
    pub constitution: u32,
}

impl AttrPoints {
    /// Todos os atributos em zero.
    pub const ZERO: Self = Self {
        strength: 0,
        intelligence: 0,
        dexterity: 0,
        constitution: 0,
    };
}

/// Quanto cada ponto de atributo entrega em cada derivado (dado de conteúdo, não regra).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct AttrWeight {
    /// HP por ponto de constituição.
    pub hp_per_constitution: u32,
    /// MP por ponto de inteligência.
    pub mp_per_intelligence: u32,
    /// Ataque físico por ponto de força.
    pub attack_per_strength: u32,
    /// Ataque por ponto de destreza (arma de destreza soma aqui).
    pub attack_per_dexterity: u32,
    /// Defesa por ponto de constituição.
    pub defense_per_constitution: u32,
}

impl AttrWeight {
    /// Todos os pesos em zero (derivado = só nível + equipamento).
    pub const ZERO: Self = Self {
        hp_per_constitution: 0,
        mp_per_intelligence: 0,
        attack_per_strength: 0,
        attack_per_dexterity: 0,
        defense_per_constitution: 0,
    };
}

/// Soma de equipamento + montaria (unidades absolutas, já resolvidas pelo chamador).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct GearBonus {
    /// Ataque somado do equipamento.
    pub attack: u32,
    /// Defesa somada do equipamento.
    pub defense: u32,
    /// HP máximo somado.
    pub max_hp: u32,
    /// MP máximo somado.
    pub max_mp: u32,
}

/// Crescimento por nível (dado de conteúdo; hoje viria de `mob_template_stat`/classe).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct GrowthProfile {
    /// HP por nível acima do 1.
    pub hp_per_level: u32,
    /// MP por nível acima do 1.
    pub mp_per_level: u32,
    /// Ataque por nível acima do 1.
    pub attack_per_level: u32,
    /// Defesa por nível acima do 1.
    pub defense_per_level: u32,
    /// Conversão de atributo em derivado.
    pub weight: AttrWeight,
}

/// Atributos derivados prontos para o combate (inteiros absolutos).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Derived {
    /// HP máximo.
    pub max_hp: u32,
    /// MP máximo.
    pub max_mp: u32,
    /// Ataque físico.
    pub attack: u32,
    /// Defesa.
    pub defense: u32,
}

/// Pontos de atributo que o Arch recebe por nível do personagem normal.
pub const ARCH_ATTR_PER_LEVEL: u32 = 8;

/// Bônus de atributo do Arch para um Mortal de `mortal_level`.
///
/// Fonte: `docs/wyd/classes.md` §4 ("+8 pontos de atributo por nível evoluído") e
/// `docs/wyd/progressao.md` §2 ("Arch | +8 atributos por nível").
#[must_use]
pub fn arch_attribute_points(mortal_level: u16) -> u32 {
    u32::from(mortal_level).saturating_mul(ARCH_ATTR_PER_LEVEL)
}

/// Deriva HP/MP/ataque/defesa na ordem fixa: nível → atributo → equipamento.
///
/// Nível `0` é tratado como `1` (sem crescimento) em vez de `panic!`: o chamador que valida
/// nível é o `Stats::validate` do core, não a fórmula. Toda soma satura.
#[must_use]
pub fn derive(level: u16, attrs: AttrPoints, gear: GearBonus, profile: GrowthProfile) -> Derived {
    let niveis = u32::from(level).saturating_sub(1);
    let peso = profile.weight;

    let hp_atributo = attrs.constitution.saturating_mul(peso.hp_per_constitution);
    let mp_atributo = attrs.intelligence.saturating_mul(peso.mp_per_intelligence);
    let ataque_atributo = attrs
        .strength
        .saturating_mul(peso.attack_per_strength)
        .saturating_add(attrs.dexterity.saturating_mul(peso.attack_per_dexterity));
    let defesa_atributo = attrs
        .constitution
        .saturating_mul(peso.defense_per_constitution);

    Derived {
        max_hp: niveis
            .saturating_mul(profile.hp_per_level)
            .saturating_add(hp_atributo)
            .saturating_add(gear.max_hp),
        max_mp: niveis
            .saturating_mul(profile.mp_per_level)
            .saturating_add(mp_atributo)
            .saturating_add(gear.max_mp),
        attack: niveis
            .saturating_mul(profile.attack_per_level)
            .saturating_add(ataque_atributo)
            .saturating_add(gear.attack),
        defense: niveis
            .saturating_mul(profile.defense_per_level)
            .saturating_add(defesa_atributo)
            .saturating_add(gear.defense),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const PERFIL: GrowthProfile = GrowthProfile {
        hp_per_level: 10,
        mp_per_level: 4,
        attack_per_level: 1,
        defense_per_level: 1,
        weight: AttrWeight {
            hp_per_constitution: 3,
            mp_per_intelligence: 2,
            attack_per_strength: 2,
            attack_per_dexterity: 1,
            defense_per_constitution: 1,
        },
    };

    #[test]
    fn level_one_has_no_growth() {
        let d = derive(1, AttrPoints::ZERO, GearBonus::default(), PERFIL);
        assert_eq!((d.max_hp, d.max_mp, d.attack, d.defense), (0, 0, 0, 0));
    }

    #[test]
    fn order_is_level_then_attribute_then_gear() {
        let attrs = AttrPoints {
            strength: 5,
            intelligence: 4,
            dexterity: 3,
            constitution: 2,
        };
        let gear = GearBonus {
            attack: 7,
            defense: 11,
            max_hp: 100,
            max_mp: 50,
        };
        let d = derive(11, attrs, gear, PERFIL);
        // 10 níveis de crescimento + atributo + equipamento (contas já resolvidas).
        assert_eq!(d.max_hp, 100 + 6 + 100);
        assert_eq!(d.max_mp, 40 + 8 + 50);
        assert_eq!(d.attack, 10 + 13 + 7);
        assert_eq!(d.defense, 10 + 2 + 11);
    }

    #[test]
    fn zero_level_is_treated_as_one() {
        let d = derive(0, AttrPoints::ZERO, GearBonus::default(), PERFIL);
        assert_eq!(d.max_hp, 0);
    }

    #[test]
    fn arch_bonus_matches_published_eight_per_level() {
        assert_eq!(arch_attribute_points(1), 8);
        assert_eq!(arch_attribute_points(370), 2_960);
    }

    #[test]
    fn saturation_instead_of_wrap() {
        let perfil = GrowthProfile {
            hp_per_level: u32::MAX,
            ..PERFIL
        };
        let d = derive(5, AttrPoints::ZERO, GearBonus::default(), perfil);
        assert_eq!(d.max_hp, u32::MAX);
    }
}
