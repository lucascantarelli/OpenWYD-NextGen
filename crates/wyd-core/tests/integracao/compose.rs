//! Cadeia completa do refino alto (+9 → +15) pelos **handlers reais** de composição.
//!
//! O teste não usa fórmula isolada: chama [`aylin`], [`apply_powder`], [`odin`] e [`anct`],
//! que são o caminho de jogo. Os números de economia vêm do seed
//! `crates/wyd-db/migrations/0027_seed_combine.sql`, lido por `include_str!` — trocar uma
//! célula no dado muda o resultado e quebra o teste, o que é o ponto: nada de taxa mágica
//! dentro do handler.

use std::collections::HashMap;

use wyd_core::systems::compose::{EF_ITEMLEVEL, EF_MOBTYPE, EF_POS};
use wyd_core::systems::ANCT_MAX_SACRIFICES;
use wyd_core::{
    anct, apply_powder, aylin, downgrade_refine, odin, plus12_failure, powder_failure, roll_115,
    roll_199, ComponentStore, ComposeRules, CoreError, Entity, EntityKind, Inventory, Item,
    ItemFacts, Plus12Failure, Position, PowderFailure, Stats, GEM_EFFECT_DROP,
};

const SEED: &str = include_str!("../../../wyd-db/migrations/0027_seed_combine.sql");

/// Catálogo de teste: fatos/efeitos fixos, taxas lidas do seed.
struct Catalog {
    values: HashMap<String, HashMap<String, i32>>,
    facts: HashMap<u32, ItemFacts>,
    abilities: HashMap<(u32, u16), i32>,
}

impl Catalog {
    fn real() -> Self {
        let mut values: HashMap<String, HashMap<String, i32>> = HashMap::new();
        let mut rest = SEED;
        while let Some(start) = rest.find("('") {
            let tail = &rest[start + 2..];
            let Some(family_end) = tail.find("', '") else {
                break;
            };
            let family = &tail[..family_end];
            let after = &tail[family_end + 4..];
            let Some(key_end) = after.find("', ") else {
                break;
            };
            let key = &after[..key_end];
            let after_key = &after[key_end + 3..];
            let Some(comma) = after_key.find(',') else {
                break;
            };
            if let Ok(value) = after_key[..comma].trim().parse::<i32>() {
                values
                    .entry(family.to_owned())
                    .or_default()
                    .insert(key.to_owned(), value);
            }
            rest = &after_key[comma..];
        }

        let mut facts = HashMap::new();
        facts.insert(
            1900,
            ItemFacts {
                pos: 64,
                grade: 5,
                unique_id: 41,
                extra: 1910,
            },
        );
        facts.insert(
            769,
            ItemFacts {
                pos: 64,
                grade: 5,
                unique_id: 41,
                extra: 1920,
            },
        );
        for id in [2001u32, 2002, 2003] {
            facts.insert(
                id,
                ItemFacts {
                    pos: 4,
                    grade: 5,
                    unique_id: 0,
                    extra: 0,
                },
            );
        }

        let mut abilities = HashMap::new();
        abilities.insert((1900, EF_ITEMLEVEL), 30);
        abilities.insert((1900, EF_MOBTYPE), 0);
        for id in [2001u32, 2002, 2003] {
            abilities.insert((id, EF_ITEMLEVEL), 30);
            abilities.insert((id, EF_POS), 4);
        }

        Self {
            values,
            facts,
            abilities,
        }
    }
}

impl ComposeRules for Catalog {
    fn facts(&self, item_index: u32) -> Option<ItemFacts> {
        self.facts.get(&item_index).copied()
    }

    fn ability(&self, item_index: u32, effect: u16) -> i32 {
        self.abilities
            .get(&(item_index, effect))
            .copied()
            .unwrap_or(0)
    }

    fn value(&self, family: &str, key: &str) -> Option<i32> {
        self.values
            .get(family)
            .and_then(|keys| keys.get(key))
            .copied()
    }
}

fn item(id: u32, qty: u32, refine: u8) -> Item {
    Item {
        id,
        qty,
        refine,
        ..Item::default()
    }
}

fn powder_stack(id: u32) -> Item {
    Item {
        id,
        qty: 10,
        ..Item::default()
    }
}

fn stats_level(level: u16) -> Stats {
    Stats {
        level,
        hp: 100,
        max_hp: 100,
        mp: 0,
        max_mp: 0,
        attack: 1,
        defense: 1,
    }
}

fn bag(store: &ComponentStore, entity: Entity, slot: usize) -> Item {
    store.inventory(entity).expect("entidade viva").bag[slot]
}

fn spawn(inv: Inventory) -> (ComponentStore, Entity) {
    let mut store = ComponentStore::default();
    let entity = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            stats_level(39),
            inv,
        )
        .expect("spawn válido");
    (store, entity)
}

/// Aylin: custo e insumos cobrados, resultado em +10 com a gema da jóia.
#[test]
fn aylin_cobra_custo_e_insumos_e_produz_mais_dez() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(1900, 1, 9);
    inv.bag[2] = item(1774, 1, 0);
    for slot in 3..7 {
        inv.bag[slot] = item(2441, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    assert!(store.set_coin(e, 60_000_000));

    let outcome =
        aylin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6], &catalog, 5).expect("receita válida");
    assert!(outcome.success);
    assert_eq!(outcome.rate_percent, 41, "1 + 4 × 10 do seed");
    assert_eq!(outcome.coin_spent, 50_000_000);
    assert_eq!(outcome.consumed, 7, "dois +9, pedra e quatro jóias");

    let base = bag(&store, e, 0);
    assert_eq!((base.id, base.refine), (1900, 10));
    assert_eq!(base.gem_effect, GEM_EFFECT_DROP, "Diamante vira drop");
    assert_eq!(bag(&store, e, 1).id, 0, "segundo +9 consumido no sucesso");
    assert_eq!(bag(&store, e, 2).id, 0, "pedra consumida");
    assert_eq!(store.coin(e), Some(10_000_000), "custo debitado do saldo");
}

/// Aylin recusa sem dinheiro e não gasta nada.
#[test]
fn aylin_sem_moedas_recusa_sem_mutar() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(1900, 1, 9);
    inv.bag[2] = item(1774, 1, 0);
    for slot in 3..7 {
        inv.bag[slot] = item(2441, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    assert!(store.set_coin(e, 49_999_999));
    assert_eq!(
        aylin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6], &catalog, 0).unwrap_err(),
        CoreError::NotEnoughCoin {
            needed: 50_000_000,
            available: 49_999_999
        }
    );
    assert_eq!(bag(&store, e, 0).refine, 9, "item intocado");
    assert_eq!(bag(&store, e, 2).id, 1774, "pedra intocada");
    assert_eq!(store.coin(e), Some(49_999_999));
}

/// Cadeia inteira: +9 → Aylin (+10) → Lac (+11) → Odin (+12..+15), com consumo cobrado.
#[test]
fn cadeia_de_mais_nove_ate_mais_quinze_pelos_handlers() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(1900, 1, 9);
    inv.bag[2] = item(1774, 1, 0);
    for slot in 3..7 {
        inv.bag[slot] = item(2441, 1, 0);
    }
    // Oito pilhas de 10 Lac: duas por passo do +12+.
    for slot in 8..16 {
        inv.bag[slot] = powder_stack(413);
    }
    // Uma pilha de pó separada para o passo +10 → +11 (não pode comer as do +12+).
    inv.bag[32] = powder_stack(413);
    // Quatro conjuntos de runas (5334..5337), um por passo.
    let runas = [5334u32, 5335, 5336, 5337];
    for step in 0..4usize {
        for (index, runa) in runas.into_iter().enumerate() {
            inv.bag[16 + step * 4 + index] = item(runa, 1, 0);
        }
    }
    let (mut store, e) = spawn(inv);
    assert!(store.set_coin(e, 60_000_000));

    let step10 = aylin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6], &catalog, 5).expect("Aylin válido");
    assert!(step10.success);
    assert_eq!(bag(&store, e, 0).refine, 10);

    let step11 = apply_powder(&mut store, e, 0, 32, &catalog, 10).expect("pó válido");
    assert!(step11.success);
    assert_eq!(step11.rate_percent, 15, "Lac do seed");
    assert_eq!(bag(&store, e, 0).refine, 11);
    assert_eq!(bag(&store, e, 32).qty, 9, "uma unidade de pó por tentativa");

    let mut target: u8 = 11;
    for (step, slots) in [
        [8usize, 9, 0, 16, 17, 18, 19, 63],
        [10, 11, 0, 20, 21, 22, 23, 63],
        [12, 13, 0, 24, 25, 26, 27, 63],
        [14, 15, 0, 28, 29, 30, 31, 63],
    ]
    .into_iter()
    .enumerate()
    {
        let outcome = odin(&mut store, e, &slots, &catalog, 114, 0).expect("ramo +12+");
        assert!(outcome.success, "passo {step} do +12+");
        assert_eq!(outcome.recipe, 2, "receita Plus12");
        assert_eq!(
            outcome.consumed, 25,
            "2 pilhas de 10 + 4 runas + o alvo retirado e regravado ({step})"
        );
        target += 1;
        assert_eq!(bag(&store, e, 0).refine, target, "passo {step}");
    }
    assert_eq!(target, 15);
    assert_eq!(store.coin(e), Some(10_000_000), "só o Aylin cobra moeda");
    for slot in 8..16 {
        assert_eq!(bag(&store, e, slot).id, 0, "pilha de pó gasta: {slot}");
    }
    for slot in 16..32 {
        assert_eq!(bag(&store, e, slot).id, 0, "runa gasta: {slot}");
    }
}

/// Os dois sorteios são diferentes e aplicados no lugar certo.
///
/// Se o Odin usasse `rand()%115`, o Pista (taxa 40) falharia com raw 114; se o Aylin usasse
/// `rand()%199`, ele passaria. Cada asserção prende um lado.
#[test]
fn sorteio_de_115_nao_serve_para_o_odin_e_vice_versa() {
    assert_eq!(roll_115(114), Ok(99), "115 achata ≥100 em −15");
    assert_eq!(roll_199(114), Ok(15), "199 achata >100 em −99");

    let catalog = Catalog::real();

    // Aylin com raw 114: 99 > 41, falha — prova que o caminho usa o sorteio de 115.
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(1900, 1, 9);
    inv.bag[2] = item(1774, 1, 0);
    for slot in 3..7 {
        inv.bag[slot] = item(2441, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    assert!(store.set_coin(e, 50_000_000));
    let falha =
        aylin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6], &catalog, 114).expect("receita válida");
    assert!(!falha.success, "99 > 41");
    assert_eq!((falha.rate_percent, falha.roll), (41, 99));
    assert_eq!(bag(&store, e, 0).refine, 9, "falha preserva os +9");

    // Odin Pista com raw 114: 15 ≤ 40, passa — prova que o caminho usa o sorteio de 199.
    let mut inv = Inventory::default();
    for slot in 0..7 {
        inv.bag[slot] = powder_stack(413);
    }
    let (mut store, e) = spawn(inv);
    let ok =
        odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 114, 0).expect("Pista de Runas");
    assert!(ok.success, "15 ≤ 40");
    assert_eq!((ok.rate_percent, ok.roll), (40, 15));
    assert_eq!(ok.produced.expect("resultado").id, 5134);
}

/// Item 769 fica em +9 em todos os caminhos novos.
#[test]
fn item_769_para_no_mais_nove_em_todos_os_caminhos() {
    let catalog = Catalog::real();
    let capped = CoreError::RefineCappedItem { item_index: 769 };

    let mut inv = Inventory::default();
    inv.bag[0] = item(769, 1, 9);
    inv.bag[1] = item(769, 1, 9);
    inv.bag[2] = item(1774, 1, 0);
    for slot in 3..7 {
        inv.bag[slot] = item(2441, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    assert!(store.set_coin(e, 50_000_000));
    assert_eq!(
        aylin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6], &catalog, 0).unwrap_err(),
        capped
    );

    let mut inv = Inventory::default();
    inv.bag[0] = item(769, 1, 10);
    inv.bag[1] = item(413, 1, 0);
    let (mut store, e) = spawn(inv);
    assert_eq!(
        apply_powder(&mut store, e, 0, 1, &catalog, 0).unwrap_err(),
        capped
    );

    let mut inv = Inventory::default();
    inv.bag[0] = powder_stack(413);
    inv.bag[1] = powder_stack(413);
    inv.bag[2] = item(769, 1, 11);
    for (slot, runa) in [5334u32, 5335, 5336, 5337].into_iter().enumerate() {
        inv.bag[3 + slot] = item(runa, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    assert_eq!(
        odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 0, 0).unwrap_err(),
        capped
    );
}

/// Anct: gates, taxa somada dos sacrifícios e teto da régua.
#[test]
fn anct_aplica_gates_e_teto_de_sacrificios() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(2441, 1, 0);
    inv.bag[2] = item(2001, 1, 7);
    inv.bag[3] = item(2002, 1, 8);
    inv.bag[4] = item(2003, 1, 9);
    let (mut store, e) = spawn(inv);

    let outcome = anct(&mut store, e, &[0, 1, 2, 3, 4], &catalog, 5).expect("Anct válido");
    assert!(outcome.success);
    assert_eq!(outcome.rate_percent, 17, "1 + 2 + 4 + 10 do seed");
    let resultado = bag(&store, e, 0);
    assert_eq!((resultado.id, resultado.refine), (1910, 7));
    assert!(resultado.ancient);
    assert_eq!(resultado.gem_effect, GEM_EFFECT_DROP);
    assert_eq!(bag(&store, e, 1).id, 0, "jóia consumida");
    assert_eq!(bag(&store, e, 4).id, 0, "sacrifício consumido");

    // Sete sacrifícios (9 posições) passam do teto de 6 e são recusados sem mutação.
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(2441, 1, 0);
    for slot in 2..9 {
        inv.bag[slot] = item(2001, 1, 7);
    }
    let (mut store, e) = spawn(inv);
    assert_eq!(
        anct(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 7, 8], &catalog, 0).unwrap_err(),
        CoreError::ComposeSacrificeLimit {
            count: 7,
            max: ANCT_MAX_SACRIFICES
        }
    );
    assert_eq!(bag(&store, e, 0).refine, 9);
    assert_eq!(bag(&store, e, 8).id, 2001, "nada consumido na recusa");
}

/// Sacrifício fora de +7..+9 (ou nível de item menor) não entra na receita.
#[test]
fn anct_recusa_sacrificio_fora_da_faixa() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 9);
    inv.bag[1] = item(2441, 1, 0);
    inv.bag[2] = item(2001, 1, 6);
    let (mut store, e) = spawn(inv);
    assert_eq!(
        anct(&mut store, e, &[0, 1, 2], &catalog, 0).unwrap_err(),
        CoreError::ComposeRecipeMismatch { family: "Anct" }
    );
}

/// Falha do pó preserva o item; o Emblema rebaixa em vez de destruir.
#[test]
fn regras_de_falha_seguem_valendo_nos_caminhos_novos() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(1900, 1, 10);
    inv.bag[1] = Item {
        id: 413,
        qty: 2,
        ..Item::default()
    };
    let (mut store, e) = spawn(inv);
    let falha = apply_powder(&mut store, e, 0, 1, &catalog, 99).expect("pó válido");
    assert!(!falha.success, "99 > 15");
    assert_eq!(bag(&store, e, 0).refine, 10, "pó não destrói item comum");
    assert_eq!(bag(&store, e, 1).qty, 1, "o pó é gasto mesmo na falha");

    assert_eq!(powder_failure(false, 8), PowderFailure::Kept);
    assert_eq!(powder_failure(true, 8), PowderFailure::Destroyed);
    assert_eq!(powder_failure(true, 6), PowderFailure::Kept);
    assert_eq!(powder_failure(true, 10), PowderFailure::Kept);
    assert_eq!(plus12_failure(true), Plus12Failure::Downgraded);
    assert_eq!(plus12_failure(false), Plus12Failure::Destroyed);
    assert_eq!(downgrade_refine(11), 10);
    assert_eq!(downgrade_refine(14), 13);
    assert_eq!(downgrade_refine(7), 7, "nunca abaixo de +7");
}

/// O +12+ do legado vivo é incondicional: taxa é calculada, mas não barra o passo.
#[test]
fn mais_doze_e_incondicional_como_no_codigo_vivo() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = powder_stack(413);
    inv.bag[1] = powder_stack(413);
    inv.bag[2] = item(1900, 1, 14);
    for (index, runa) in [5334u32, 5335, 5336, 5337].into_iter().enumerate() {
        inv.bag[3 + index] = item(runa, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    // Mesmo com o pior sorteio possível (99) o passo sobe.
    let outcome =
        odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 198, 0).expect("ramo +12+");
    assert!(outcome.success);
    assert_eq!(outcome.roll, 99);
    assert_eq!(outcome.rate_percent, 100, "100 + 4 runas − 4 do alvo +14");
    assert_eq!(bag(&store, e, 2).refine, 15, "o alvo fica no slot que veio");
}

/// Emblema da Proteção entra no lugar das pilhas de Lac e é cobrado no +12+.
#[test]
fn emblema_da_protecao_e_cobrado_no_mais_doze() {
    let catalog = Catalog::real();
    let mut inv = Inventory::default();
    inv.bag[0] = item(4043, 1, 0);
    inv.bag[1] = item(4043, 1, 0);
    inv.bag[2] = item(1900, 1, 12);
    for (index, runa) in [5334u32, 5335, 5336, 5337].into_iter().enumerate() {
        inv.bag[3 + index] = item(runa, 1, 0);
    }
    let (mut store, e) = spawn(inv);
    let outcome = odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 198, 0)
        .expect("ramo +12+ com Emblema");
    assert!(outcome.success);
    assert_eq!(outcome.protect_items, 2);
    assert_eq!(outcome.consumed, 7, "2 emblemas + 4 runas + o alvo");
    assert_eq!(bag(&store, e, 0).id, 0);
    assert_eq!(bag(&store, e, 1).id, 0);
    assert_eq!(bag(&store, e, 2).refine, 13);
}

/// Destrave Lv40: exige nível 39, classe celestial e não repetir o destrave.
#[test]
fn destrave_lv40_exige_nivel_e_classe() {
    let catalog = Catalog::real();
    let receita = [4127u32, 4127, 5135, 5113, 5129, 5112, 5110];
    let mut inv = Inventory::default();
    for (slot, id) in receita.iter().enumerate() {
        inv.bag[slot] = item(*id, 1, 0);
    }
    let mut store = ComponentStore::default();
    let e = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            stats_level(40),
            inv,
        )
        .expect("spawn válido");
    assert!(store.set_celestial(e, true));
    assert!(odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 0, 0).is_err());

    let mut store = ComponentStore::default();
    let e = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            stats_level(39),
            inv,
        )
        .expect("spawn válido");
    assert!(
        odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 0, 0).is_err(),
        "sem classe celestial não destrava"
    );

    let mut store = ComponentStore::default();
    let e = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            stats_level(39),
            inv,
        )
        .expect("spawn válido");
    assert!(store.set_celestial(e, true));
    let outcome =
        odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 114, 0).expect("destrave");
    assert!(outcome.success);
    assert_eq!(outcome.recipe, 4);
    assert_eq!(outcome.rate_percent, 100);
    assert!(outcome.produced.is_none(), "destrave não gera item");
    assert_eq!(store.celestial_lv40(e), Some(true));
    assert_eq!(outcome.consumed, 7);

    // Repetir o destrave é recusado (gate do legado).
    let mut inv = Inventory::default();
    for (slot, id) in receita.iter().enumerate() {
        inv.bag[slot] = item(*id, 1, 0);
    }
    let mut store = ComponentStore::default();
    let e = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            stats_level(39),
            inv,
        )
        .expect("spawn válido");
    assert!(store.set_celestial(e, true));
    assert!(store.set_celestial_lv40(e, true));
    assert!(odin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6, 63], &catalog, 0, 0).is_err());
}
