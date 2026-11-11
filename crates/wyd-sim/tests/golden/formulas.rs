//! Goldens da matemática: valor **exato**, preso à fonte citada.
//!
//! Um golden novo entra com a procedência no nome do teste. Número sem fonte é chute com
//! aparência de teste — e é pior que ausência de teste.

use wyd_sim::{
    apply_gem, damage_after_absorb, gem_effect_value, refine_success_bps, AppliedGem, GemEffect,
    GEM_CORAL, GEM_DIAMOND, GEM_EMERALD, GEM_GARNET,
};

/// WYD International, *Server Information* (`rate1`…`rate9`).
#[test]
fn refine_rates_match_wyd_international() {
    let esperado = [
        9_000, 8_500, 8_000, 7_500, 7_000, 6_500, 6_000, 5_500, 5_000,
    ];
    for (indice, bps) in esperado.iter().enumerate() {
        let step = u8::try_from(indice + 1).expect("passo cabe em u8");
        assert_eq!(refine_success_bps(step), Some(*bps), "passo {step}");
    }
}

/// Kersef / ABC do WYD: 40 por nível acima de +9, teto no +15.
#[test]
fn gem_scale_matches_kersef() {
    let tabela = [
        (10, 40),
        (11, 80),
        (12, 120),
        (13, 160),
        (14, 200),
        (15, 240),
    ];
    for (refine, valor) in tabela {
        assert_eq!(gem_effect_value(GemEffect::Pierce, refine), Some(valor));
        assert_eq!(gem_effect_value(GemEffect::Absorb, refine), Some(valor));
    }
    assert_eq!(
        gem_effect_value(GemEffect::Pierce, 16),
        None,
        "acima do +15"
    );
}

/// Drop e EXP são fixos; só perfuração/absorção escalam.
#[test]
fn flat_gem_bonuses_are_constant() {
    for refine in 10..=15 {
        assert_eq!(gem_effect_value(GemEffect::Drop, refine), Some(8));
        assert_eq!(gem_effect_value(GemEffect::Exp, refine), Some(2));
    }
}

/// As quatro gemas do `item_catalog` (3386..3389) apontam para efeitos distintos.
#[test]
fn gem_items_are_the_four_from_catalog() {
    assert_eq!(GemEffect::from_item(GEM_DIAMOND), Some(GemEffect::Drop));
    assert_eq!(GemEffect::from_item(GEM_EMERALD), Some(GemEffect::Pierce));
    assert_eq!(GemEffect::from_item(GEM_CORAL), Some(GemEffect::Exp));
    assert_eq!(GemEffect::from_item(GEM_GARNET), Some(GemEffect::Absorb));
    assert_eq!(GemEffect::from_item(2441), None, "Diamante cru não é gema");
}

/// Anct aceita a gema abaixo de +10 e recebe o valor base (40), nunca negativo.
#[test]
fn ancient_gets_base_value_below_ten() {
    assert_eq!(
        apply_gem(GEM_EMERALD, 9, true),
        Ok(AppliedGem {
            effect: GemEffect::Pierce,
            value: 40,
            refine: 10,
        })
    );
}

/// Absorção nunca produz dano negativo (invariante do combate).
#[test]
fn absorb_saturates_at_zero() {
    assert_eq!(damage_after_absorb(15, 40), 0);
    assert_eq!(damage_after_absorb(41, 40), 1);
}
