//! `apply_gem` contra o **dado real** de `refine_gem_effect` (seed `0022_seed_refino.sql`).
//!
//! O teste anterior (`gem.rs`) prova a transação com números escritos à mão; este prova que
//! esses números são **os do banco**, lendo a migração que o Postgres aplica. Não há mock: o
//! seed é compilado para dentro do binário de teste (`include_str!`), então divergência entre
//! dado e política quebra o gate, não a produção.
//!
//! Fronteira respeitada: `wyd-core` **não** depende de `wyd-db` — a leitura do seed é do
//! compilador (teste), não I/O em runtime, e a direção `core → db` continua inexistente no DAG.
//!
//! Fonte da regra: `docs/wyd/sistemas/refino.md` §3; decisão em
//! `docs/arquitetura/adr/0012-gema-no-inventario.md`.

use wyd_core::{
    gem_effect_code, gem_value, ComponentStore, CoreError, DomainEvent, Entity, EntityKind,
    Inventory, Item, Position, Stats, GEM_EFFECT_ABSORB, GEM_EFFECT_DROP, GEM_EFFECT_EXP,
    GEM_EFFECT_NONE, GEM_EFFECT_PIERCE,
};

/// Migração aplicada ao banco: fonte do dado de gema, não uma cópia.
const SEED: &str = include_str!("../../../wyd-db/migrations/0022_seed_refino.sql");

/// Uma linha de `refine_gem_effect` como o banco a recebe.
#[derive(Debug, Clone, Copy)]
struct SeedGem {
    item_index: u32,
    effect: u8,
    flat: u16,
    per_level: u16,
    min_refine: u8,
    max_refine: u8,
    allows_ancient: bool,
}

/// Lê as quatro linhas do `INSERT INTO refine_gem_effect` do seed.
///
/// A leitura termina no `ON CONFLICT`: nenhuma linha do `refine_rate` nem dos comentários
/// entra. `splitn(10, ',')` isola os campos numéricos e deixa a nota (que tem vírgula) no
/// último pedaço, sem precisar de parser de SQL.
fn parse_seed() -> Vec<SeedGem> {
    let corpo = SEED
        .split("INSERT INTO refine_gem_effect")
        .nth(1)
        .expect("seed deve conter o INSERT de refine_gem_effect");
    corpo
        .lines()
        .filter_map(|linha| {
            let linha = linha.trim();
            let linha = linha.strip_prefix('(')?;
            let mut campos = linha.splitn(10, ',');
            let item_index: u32 = campos.next()?.trim().parse().ok()?;
            let codigo = campos.next()?.trim().trim_matches('\'');
            let flat: u16 = campos.next()?.trim().parse().ok()?;
            let per_level: u16 = campos.next()?.trim().parse().ok()?;
            let min_refine: u8 = campos.next()?.trim().parse().ok()?;
            let max_refine: u8 = campos.next()?.trim().parse().ok()?;
            let allows_ancient = match campos.next()?.trim() {
                "true" => true,
                "false" => false,
                outro => panic!("allows_ancient inesperado no seed: {outro}"),
            };
            let effect = match codigo {
                "drop" => GEM_EFFECT_DROP,
                "pierce" => GEM_EFFECT_PIERCE,
                "exp" => GEM_EFFECT_EXP,
                "absorb" => GEM_EFFECT_ABSORB,
                outro => panic!("effect_code inesperado no seed: {outro}"),
            };
            Some(SeedGem {
                item_index,
                effect,
                flat,
                per_level,
                min_refine,
                max_refine,
                allows_ancient,
            })
        })
        .collect()
}

/// Valor que o banco manda aplicar: fixo quando `per_level = 0`, `per_level × (refino − 9)`
/// quando escala. É a leitura literal das duas colunas do seed.
fn valor_do_seed(gema: &SeedGem, refine: u8) -> u16 {
    if gema.per_level == 0 {
        gema.flat
    } else {
        gema.per_level * u16::from(refine - (gema.min_refine - 1))
    }
}

fn stats() -> Stats {
    Stats {
        level: 60,
        hp: 1_000,
        max_hp: 1_000,
        mp: 100,
        max_mp: 100,
        attack: 120,
        defense: 80,
    }
}

/// Armadura no slot 3 + as gemas pedidas na mochila.
fn inventario(refine: u8, ancient: bool, gemas: &[(u32, u32)]) -> Inventory {
    let mut inv = Inventory::default();
    inv.equip[3] = Item {
        id: 1668,
        qty: 1,
        refine,
        ancient,
        gem_effect: GEM_EFFECT_NONE,
        gem_value: 0,
    };
    for (slot, (gema, qty)) in gemas.iter().enumerate() {
        inv.bag[slot] = Item {
            id: *gema,
            qty: *qty,
            ..Item::default()
        };
    }
    inv
}

fn spawnar(store: &mut ComponentStore, refine: u8, ancient: bool, gemas: &[(u32, u32)]) -> Entity {
    store
        .spawn(
            EntityKind::Player,
            Position::default(),
            stats(),
            inventario(refine, ancient, gemas),
        )
        .expect("spawn válido")
}

fn item(store: &ComponentStore, entity: Entity) -> Item {
    store.inventory(entity).expect("entidade viva").equip[3]
}

/// O seed publicado é a fonte: quatro gemas, faixa 10..=15, e a política do core concorda em
/// **todos** os refinos — não só no valor citado na documentação.
#[test]
fn politica_do_core_concorda_com_o_seed_em_toda_a_faixa() {
    let gemas = parse_seed();
    assert_eq!(
        gemas.len(),
        4,
        "seed deve publicar as quatro gemas de troca de efeito\n{gemas:#?}"
    );

    for gema in &gemas {
        assert_eq!(
            gem_effect_code(gema.item_index),
            Some(gema.effect),
            "item {} do seed não é reconhecido pela política do core",
            gema.item_index
        );
        assert_eq!(
            (gema.min_refine, gema.max_refine),
            (10, 15),
            "faixa publicada para a gema {}",
            gema.item_index
        );
        assert!(
            gema.allows_ancient,
            "seed autoriza Anct para a gema {}",
            gema.item_index
        );

        for refine in gema.min_refine..=gema.max_refine {
            assert_eq!(
                gem_value(gema.effect, refine),
                Some(valor_do_seed(gema, refine)),
                "gema {} no refino +{refine} diverge do seed",
                gema.item_index
            );
        }
        // Abaixo do mínimo publicado não há valor: o core não inventa faixa.
        assert_eq!(
            gem_value(gema.effect, gema.min_refine - 1),
            None,
            "abaixo de +{} a gema {} não tem valor",
            gema.min_refine,
            gema.item_index
        );
    }
}

/// No mesmo refino (um abaixo do mínimo do seed), o Anct recebe o efeito publicado e o comum
/// é recusado — a diferença vem do dado, não de um literal do teste.
#[test]
fn anct_recebe_o_efeito_do_seed_e_o_comum_no_mesmo_refino_nao() {
    let perfuracao = parse_seed()
        .into_iter()
        .find(|gema| gema.effect == GEM_EFFECT_PIERCE)
        .expect("seed publica a gema de perfuração");
    let refino_comum = perfuracao.min_refine - 1;

    let mut store = ComponentStore::default();
    let anct = spawnar(
        &mut store,
        refino_comum,
        true,
        &[(perfuracao.item_index, 1)],
    );
    let comum = spawnar(
        &mut store,
        refino_comum,
        false,
        &[(perfuracao.item_index, 1)],
    );

    let no_anct = store
        .apply_gem(anct, 3, perfuracao.item_index)
        .expect("Anct aceita gema abaixo do mínimo comum");
    assert_eq!(
        (no_anct.effect, no_anct.refine),
        (GEM_EFFECT_PIERCE, perfuracao.min_refine),
        "Anct em +{refino_comum} conta como +{}",
        perfuracao.min_refine
    );
    assert_eq!(
        no_anct.value,
        valor_do_seed(&perfuracao, perfuracao.min_refine),
        "valor do Anct é o do seed no refino mínimo"
    );

    let no_comum = store.apply_gem(comum, 3, perfuracao.item_index);
    assert_eq!(
        no_comum,
        Err(CoreError::GemRefineTooLow {
            refine: refino_comum
        }),
        "comum em +{refino_comum} não recebe gema"
    );

    let efeito_anct = item(&store, anct);
    let efeito_comum = item(&store, comum);
    assert_eq!(
        (efeito_anct.gem_effect, efeito_anct.gem_value),
        (
            GEM_EFFECT_PIERCE,
            valor_do_seed(&perfuracao, perfuracao.min_refine)
        )
    );
    assert_eq!(
        (efeito_comum.gem_effect, efeito_comum.gem_value),
        (GEM_EFFECT_NONE, 0)
    );

    // Consumo segue a regra: o Anct gastou a gema, o recusado não.
    assert_eq!(
        store
            .inventory(anct)
            .expect("vivo")
            .bag_qty(perfuracao.item_index),
        0
    );
    assert_eq!(
        store
            .inventory(comum)
            .expect("vivo")
            .bag_qty(perfuracao.item_index),
        1
    );
}

/// Trocar o efeito de um item já gemedo usa os valores do seed e publica **um** evento por
/// troca, com o antes e o depois.
#[test]
fn troca_de_efeito_em_item_ja_gemedo_usa_os_valores_do_seed() {
    let gemas = parse_seed();
    let drop = gemas
        .iter()
        .find(|gema| gema.effect == GEM_EFFECT_DROP)
        .expect("seed publica a gema de drop");
    let absorcao = gemas
        .iter()
        .find(|gema| gema.effect == GEM_EFFECT_ABSORB)
        .expect("seed publica a gema de absorção");
    let refine = absorcao.min_refine + 2; // +12: escala de três passos

    let mut store = ComponentStore::default();
    let e = spawnar(
        &mut store,
        refine,
        false,
        &[(drop.item_index, 1), (absorcao.item_index, 1)],
    );

    let primeira = store
        .apply_gem(e, 3, drop.item_index)
        .expect("primeira gema");
    assert_eq!(
        (primeira.effect, primeira.value),
        (GEM_EFFECT_DROP, drop.flat),
        "drop é fixo no seed"
    );

    let segunda = store
        .apply_gem(e, 3, absorcao.item_index)
        .expect("troca de efeito");
    assert_eq!(
        (segunda.effect, segunda.value),
        (GEM_EFFECT_ABSORB, valor_do_seed(absorcao, refine)),
        "absorção escala por nível no seed"
    );

    let eventos = store.drain_events();
    assert_eq!(eventos.len(), 2, "uma troca, um evento");
    match eventos[1] {
        DomainEvent::GemEffectChanged {
            entity,
            slot,
            gem_item,
            from,
            to,
            refine: usado,
        } => {
            assert_eq!(
                (entity, slot, gem_item, usado),
                (e, 3, absorcao.item_index, refine)
            );
            assert_eq!((from.effect, from.value), (GEM_EFFECT_DROP, drop.flat));
            assert_eq!(
                (to.effect, to.value),
                (GEM_EFFECT_ABSORB, valor_do_seed(absorcao, refine))
            );
        }
        outro => panic!("evento inesperado: {outro:?}"),
    }
    assert_eq!(item(&store, e).gem_value, valor_do_seed(absorcao, refine));
}

/// A gema de EXP do seed é fixa e não escala: o teste fixa isso porque um `per_level` trocado
/// por engano no seed passaria despercebido nos demais casos.
#[test]
fn gema_de_exp_publicada_e_fixa() {
    let exp = parse_seed()
        .into_iter()
        .find(|gema| gema.effect == GEM_EFFECT_EXP)
        .expect("seed publica a gema de EXP");
    assert_eq!((exp.flat, exp.per_level), (2, 0));
    assert_eq!(gem_value(GEM_EFFECT_EXP, 15), Some(exp.flat));
}
