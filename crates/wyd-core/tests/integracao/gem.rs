//! Gema no inventário real da entidade — consumo, recusas tipadas, evento único e Anct × comum.
//!
//! Este teste usa **só a API pública** (`ComponentStore::apply_gem`): não existe caminho
//! alternativo para aplicar gema de fora do crate, o que é a prova de "caminho único" — se
//! alguém tentar aplicar numa cópia solta, não compila.
//!
//! Fonte da regra: `docs/wyd/sistemas/refino.md` §3 — a gema é aplicada **sobre o item**
//! (jogar/arrastar a gema no item), vale para Anct e para +10 ou superior, o efeito pode ser
//! trocado quantas vezes quiser e a escala é 40 por nível acima de +9 (drop 8% e EXP 2%
//! fixos). Decisões provisórias D-12/D-13 em `docs/arquitetura/matematica-sim.md` §4.

use wyd_core::{
    ComponentStore, CoreError, DomainEvent, Entity, EntityKind, Inventory, Item, Position, Stats,
    GEM_EFFECT_ABSORB, GEM_EFFECT_DROP, GEM_EFFECT_NONE, GEM_EFFECT_PIERCE,
};

const DIAMANTE: u32 = 3386;
const ESMERALDA: u32 = 3387;
const CORAL: u32 = 3388;
const GARNET: u32 = 3389;

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

/// Inventário com uma armadura no slot 3 e as gemas pedidas na mochila.
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

/// Aplicar consome a gema **da entidade** e o item guarda o efeito novo.
#[test]
fn aplicar_consome_a_gema_do_inventario_da_entidade() {
    let mut store = ComponentStore::default();
    let e = spawnar(&mut store, 12, false, &[(DIAMANTE, 2)]);

    let aplicado = store.apply_gem(e, 3, DIAMANTE).expect("gema válida");
    assert_eq!((aplicado.effect, aplicado.value), (GEM_EFFECT_DROP, 8));
    assert_eq!(item(&store, e).gem_effect, GEM_EFFECT_DROP);
    assert_eq!(
        store.inventory(e).expect("vivo").bag_qty(DIAMANTE),
        1,
        "uma unidade saiu da mochila"
    );

    store.apply_gem(e, 3, DIAMANTE).expect("segunda unidade");
    assert_eq!(store.inventory(e).expect("vivo").bag_qty(DIAMANTE), 0);

    // Estoque esgotado: recusa tipada e o efeito anterior continua valendo.
    assert_eq!(
        store.apply_gem(e, 3, DIAMANTE),
        Err(CoreError::GemNotInBag {
            item_index: DIAMANTE
        })
    );
    assert_eq!(item(&store, e).gem_value, 8);
}

/// Nenhuma recusa muda item, mochila ou fila de eventos.
#[test]
fn recusas_nao_mudam_item_mochila_nem_publicam_evento() {
    let mut store = ComponentStore::default();
    let e = spawnar(&mut store, 9, false, &[(DIAMANTE, 3), (GARNET, 3)]);

    // Item incompatível: comum abaixo de +10.
    assert_eq!(
        store.apply_gem(e, 3, DIAMANTE),
        Err(CoreError::GemRefineTooLow { refine: 9 })
    );
    // Gema que não existe no catálogo.
    assert_eq!(
        store.apply_gem(e, 3, 2441),
        Err(CoreError::UnknownGem { item_index: 2441 })
    );
    // Slot vazio / fora da faixa.
    assert_eq!(
        store.apply_gem(e, 5, DIAMANTE),
        Err(CoreError::EmptySlot { slot: 5 })
    );
    assert_eq!(
        store.apply_gem(e, 16, DIAMANTE),
        Err(CoreError::SlotOutOfRange { slot: 16, max: 15 })
    ); // Gema válida para o item, mas sem estoque na mochila (+12 aceita gema; a recusa é o
       // estoque, não a compatibilidade).
    let sem_estoque = spawnar(&mut store, 12, false, &[(DIAMANTE, 3)]);
    assert_eq!(
        store.apply_gem(sem_estoque, 3, CORAL),
        Err(CoreError::GemNotInBag { item_index: CORAL })
    );
    assert_eq!(item(&store, sem_estoque).gem_effect, GEM_EFFECT_NONE);

    assert_eq!(item(&store, e), inventario(9, false, &[]).equip[3]);
    assert_eq!(store.inventory(e).expect("vivo").bag_qty(DIAMANTE), 3);
    assert_eq!(store.inventory(e).expect("vivo").bag_qty(GARNET), 3);
    assert!(
        store.events().is_empty(),
        "recusa não publica evento de troca"
    );
}

/// Um evento por aplicação bem-sucedida, na ordem, com o antes e o depois.
#[test]
fn evento_e_publicado_uma_vez_por_aplicacao() {
    let mut store = ComponentStore::default();
    let e = spawnar(&mut store, 12, false, &[(DIAMANTE, 1), (GARNET, 1)]);

    store.apply_gem(e, 3, DIAMANTE).expect("primeira");
    let eventos = store.drain_events();
    assert_eq!(eventos.len(), 1);
    match eventos[0] {
        DomainEvent::GemEffectChanged {
            entity,
            slot,
            gem_item,
            from,
            to,
            refine,
        } => {
            assert_eq!((entity, slot, gem_item, refine), (e, 3, DIAMANTE, 12));
            assert_eq!(from.effect, GEM_EFFECT_NONE);
            assert_eq!((to.effect, to.value), (GEM_EFFECT_DROP, 8));
        }
        outro => panic!("evento inesperado: {outro:?}"),
    }

    // Troca de efeito: o evento carrega o delta (drop → absorção 120 no +12).
    store.apply_gem(e, 3, GARNET).expect("troca");
    let eventos = store.drain_events();
    assert_eq!(eventos.len(), 1, "uma aplicação, um evento");
    match eventos[0] {
        DomainEvent::GemEffectChanged { from, to, .. } => {
            assert_eq!((from.effect, from.value), (GEM_EFFECT_DROP, 8));
            assert_eq!((to.effect, to.value), (GEM_EFFECT_ABSORB, 120));
        }
        outro => panic!("evento inesperado: {outro:?}"),
    }

    // Recusa depois do sucesso não gera evento.
    assert!(store.apply_gem(e, 3, DIAMANTE).is_err());
    assert!(store.events().is_empty());
}

/// Requisito de jogo: no mesmo refino, o Anct recebe o efeito e o comum não — a gema afeta
/// perfuração, então o Anct fica estritamente melhor no atributo que ela controla.
#[test]
fn anct_e_estritamente_melhor_que_comum_no_efeito_da_gema() {
    let mut store = ComponentStore::default();
    let anct = spawnar(&mut store, 9, true, &[(ESMERALDA, 1)]);
    let comum = spawnar(&mut store, 9, false, &[(ESMERALDA, 1)]);

    let no_anct = store
        .apply_gem(anct, 3, ESMERALDA)
        .expect("Anct aceita gema abaixo de +10");
    let no_comum = store.apply_gem(comum, 3, ESMERALDA);

    assert_eq!(
        no_comum,
        Err(CoreError::GemRefineTooLow { refine: 9 }),
        "comum +9 não recebe gema"
    );
    assert_eq!(
        (no_anct.effect, no_anct.value, no_anct.refine),
        (GEM_EFFECT_PIERCE, 40, 10),
        "Anct +9 conta como +10 e começa em 40 de perfuração"
    );
    let perfuracao_anct = item(&store, anct).gem_value;
    let perfuracao_comum = item(&store, comum).gem_value;
    assert!(
        perfuracao_anct > perfuracao_comum,
        "Anct {perfuracao_anct} deve ser estritamente melhor que comum {perfuracao_comum}"
    );
    assert_eq!((perfuracao_anct, perfuracao_comum), (40, 0));

    // O consumo segue a regra: o Anct gastou a gema, o comum recusado não.
    assert_eq!(store.inventory(anct).expect("vivo").bag_qty(ESMERALDA), 0);
    assert_eq!(store.inventory(comum).expect("vivo").bag_qty(ESMERALDA), 1);
}

/// Fila cheia é recusa de transação: nada é consumido e o item fica como estava.
#[test]
fn fila_cheia_recusa_sem_consumir_a_gema() {
    let mut store = ComponentStore::with_event_capacity(1);
    let e = spawnar(&mut store, 12, false, &[(DIAMANTE, 1), (GARNET, 1)]);

    store.apply_gem(e, 3, DIAMANTE).expect("cabe na fila");
    assert_eq!(
        store.apply_gem(e, 3, GARNET),
        Err(CoreError::EventQueueFull { capacity: 1 })
    );
    assert_eq!(
        store.inventory(e).expect("vivo").bag_qty(GARNET),
        1,
        "recusa por fila não gasta estoque"
    );
    assert_eq!(item(&store, e).gem_effect, GEM_EFFECT_DROP);
}

/// Entidade morta não tem inventário para alterar.
#[test]
fn entidade_morta_nao_aplica_gema() {
    let mut store = ComponentStore::default();
    let e = spawnar(&mut store, 12, false, &[(DIAMANTE, 1)]);
    assert!(store.despawn(e));
    assert_eq!(
        store.apply_gem(e, 3, DIAMANTE),
        Err(CoreError::EntityNotAlive {
            index: e.index,
            generation: e.generation
        })
    );
    assert!(store.events().is_empty());
    assert!(store.inventory(e).is_none());
}
