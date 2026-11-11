//! Inventário/equipamento server-authoritative: o cliente manda **intenção**, o servidor decide.
//!
//! Mesmo padrão de `tests/autoridade.rs`: usa só a API pública e prova que a recusa é atômica
//! (inventário, equipamento e fila de eventos idênticos) e que o ciclo
//! equipar → trocar → desequipar → reequipar não duplica nem perde item.

use wyd_core::{
    equip, equip_into, move_item, unequip, ComponentStore, CoreError, DomainEvent, Entity,
    EntityKind, EquipmentRules, Inventory, Item, ItemRules, Position, Stats,
};

/// Catálogo de teste espelhando `item_catalog` + `item_kind`: espada → slot 0 (req 10),
/// elmo → slot 2 (req 40), armadura → slot 3 (req 1), minério não equipa, pilha até 10.
struct Catalogo;

impl ItemRules for Catalogo {
    fn equip_slot(&self, item_index: u32) -> Option<u8> {
        match item_index {
            100 => Some(0), // espada
            101 => Some(2), // elmo
            102 => Some(3), // armadura
            _ => None,      // minério, poção, quest…
        }
    }
    fn req_level(&self, item_index: u32) -> u16 {
        match item_index {
            100 => 10,
            101 => 40,
            102 => 1,
            _ => 0,
        }
    }
    fn max_stack(&self, _item_index: u32) -> u32 {
        10
    }
}

fn stats(level: u16) -> Stats {
    Stats {
        level,
        hp: 100,
        max_hp: 100,
        mp: 10,
        max_mp: 10,
        attack: 10,
        defense: 10,
    }
}

fn item(id: u32, qty: u32) -> Item {
    Item {
        id,
        qty,
        ..Item::default()
    }
}

fn spawn(store: &mut ComponentStore, level: u16, bag: &[(usize, u32, u32)]) -> Entity {
    let mut inv = Inventory::default();
    for (slot, id, qty) in bag {
        inv.bag[*slot] = item(*id, *qty);
    }
    store
        .spawn(EntityKind::Player, Position::default(), stats(level), inv)
        .expect("spawn válido")
}

/// Soma de todas as unidades do inventário (equipamento + mochila) — prova anti-dupe/perda.
fn total(inv: &Inventory, id: u32) -> u32 {
    inv.equip
        .iter()
        .chain(inv.bag.iter())
        .filter(|slot| slot.id == id)
        .map(|slot| slot.qty)
        .sum()
}

fn inventario(store: &ComponentStore, e: Entity) -> Inventory {
    *store.inventory(e).expect("entidade viva")
}

/// Item que o cliente não possui (slot vazio) não equipa: nada muda.
#[test]
fn equipar_item_nao_possuido_e_recusado() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 50, &[]);
    let antes = inventario(&store, e);

    assert_eq!(
        equip(&mut store, e, 0, &Catalogo, EquipmentRules::default()),
        Err(CoreError::EmptySlot { slot: 0 })
    );
    assert_eq!(inventario(&store, e), antes);
    assert!(store.events().is_empty());
}

/// Slot declarado pelo cliente divergente do catálogo é recusado (ele não escolhe o slot).
#[test]
fn equipar_em_slot_errado_e_recusado() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 50, &[(0, 100, 1)]);
    let antes = inventario(&store, e);

    assert_eq!(
        equip_into(&mut store, e, 0, 3, &Catalogo, EquipmentRules::default()),
        Err(CoreError::WrongEquipSlot {
            expected: 0,
            requested: 3
        })
    );
    assert_eq!(inventario(&store, e), antes, "item continua na mochila");
    assert!(store.events().is_empty());
}

/// Item acima do requisito de nível é recusado antes de qualquer escrita.
#[test]
fn item_acima_do_requisito_e_recusado() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 39, &[(0, 101, 1)]);
    let antes = inventario(&store, e);

    assert_eq!(
        equip(&mut store, e, 0, &Catalogo, EquipmentRules::default()),
        Err(CoreError::LevelTooLow {
            required: 40,
            current: 39
        })
    );
    assert_eq!(inventario(&store, e), antes);
    assert!(store.events().is_empty());
}

/// Item sem slot de equipamento (minério) é recusado, mesmo com slot pedido plausível.
#[test]
fn item_nao_equipavel_e_recusado() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 99, &[(0, 200, 5)]);

    assert_eq!(
        equip(&mut store, e, 0, &Catalogo, EquipmentRules::default()),
        Err(CoreError::NotEquippable { item_index: 200 })
    );
    assert_eq!(inventario(&store, e).bag[0].qty, 5);
    assert!(store.events().is_empty());
}

/// Política do servidor sem troca: slot ocupado recusa e o item novo fica onde estava.
#[test]
fn troca_recusada_por_regra_mantem_os_dois_itens() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 50, &[(0, 100, 1), (1, 100, 1)]);
    equip(&mut store, e, 0, &Catalogo, EquipmentRules::default()).expect("primeira espada");
    store.drain_events();
    let antes = inventario(&store, e);

    assert_eq!(
        equip(
            &mut store,
            e,
            1,
            &Catalogo,
            EquipmentRules { allow_swap: false }
        ),
        Err(CoreError::EquipSlotOccupied { slot: 0 })
    );
    assert_eq!(inventario(&store, e), antes);
    assert_eq!(total(&inventario(&store, e), 100), 2, "nenhuma cópia");
    assert!(store.events().is_empty());
}

/// Desequipar slot vazio é recusa; o equipamento não é "limpo" por engano.
#[test]
fn desequipar_sem_estar_equipado_e_recusado() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 50, &[(0, 102, 1)]);
    let antes = inventario(&store, e);

    assert_eq!(
        unequip(&mut store, e, 5),
        Err(CoreError::EmptySlot { slot: 5 })
    );
    assert_eq!(inventario(&store, e), antes);
    assert!(store.events().is_empty());
}

/// Ciclo completo: equipar → trocar → desequipar → reequipar, sem duplicar nem perder unidade.
#[test]
fn ciclo_equipar_trocar_desequipar_reequipar_preserva_contagem() {
    let mut store = ComponentStore::default();
    let e = spawn(&mut store, 50, &[(0, 100, 1), (1, 101, 1)]);
    assert_eq!(
        total(&inventario(&store, e), 100) + total(&inventario(&store, e), 101),
        2
    );

    // 1) equipa a espada
    let primeira = equip(&mut store, e, 0, &Catalogo, EquipmentRules::default()).expect("equipa");
    assert_eq!((primeira.slot, primeira.item.id), (0, 100));
    assert_eq!(total(&inventario(&store, e), 100), 1);

    // 2) troca pela segunda arma que ocupa slot 0 no catálogo? Não: elmo vai para o slot 2,
    //    então a troca real é feita equipando outra espada do mesmo slot — usamos a armadura
    //    do slot 3 e uma espada reserva no slot 2 da mochila.
    let mut inv = inventario(&store, e);
    inv.bag[2] = item(100, 1);
    let inv_guardado = inv;
    let e = {
        // recria a entidade com o mesmo inventário para exercitar a troca no mesmo slot
        let mut novo = ComponentStore::default();
        let entidade = novo
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(50),
                inv_guardado,
            )
            .expect("spawn válido");
        store = novo;
        entidade
    };
    let troca = equip(&mut store, e, 2, &Catalogo, EquipmentRules::default()).expect("troca");
    assert_eq!((troca.slot, troca.item.id), (0, 100));
    assert_eq!(
        troca.swapped_out.map(|item| item.id),
        Some(100),
        "a espada anterior volta para a mochila"
    );
    assert_eq!(total(&inventario(&store, e), 100), 2, "troca não duplica");

    // 3) desequipa: item vai para o primeiro slot livre
    let saida = unequip(&mut store, e, 0).expect("desequipa");
    assert_eq!(saida.item.id, 100);
    assert_eq!(total(&inventario(&store, e), 100), 2);

    // 4) reequipa do slot devolvido
    let volta = equip(
        &mut store,
        e,
        usize::from(saida.bag_slot),
        &Catalogo,
        EquipmentRules::default(),
    )
    .expect("reequipa");
    assert_eq!((volta.slot, volta.item.id), (0, 100));
    assert_eq!(
        total(&inventario(&store, e), 100),
        2,
        "nada perdido no ciclo"
    );
    assert_eq!(inventario(&store, e).equip[0].id, 100);
}

/// Mover para slot ocupado por item diferente recusa; empilhar respeita o limite do catálogo.
#[test]
fn mover_item_respeita_ocupacao_e_limite_de_pilha() {
    let mut store = ComponentStore::default();
    let e = spawn(
        &mut store,
        50,
        &[(0, 100, 1), (1, 102, 1), (2, 200, 6), (3, 200, 7)],
    );
    let antes = inventario(&store, e);

    assert_eq!(
        move_item(&mut store, e, 0, 1, &Catalogo),
        Err(CoreError::BagSlotOccupied { slot: 1 })
    );
    assert_eq!(inventario(&store, e), antes);
    assert!(store.events().is_empty());

    let movidos = move_item(&mut store, e, 2, 3, &Catalogo).expect("empilha até 10");
    assert_eq!(movidos, 3);
    assert_eq!(inventario(&store, e).bag[2].qty, 3);
    assert_eq!(inventario(&store, e).bag[3].qty, 10);
    assert_eq!(
        store.drain_events(),
        vec![DomainEvent::ItemMoved {
            entity: e,
            from: 2,
            to: 3,
            item_index: 200,
            qty: 3
        }]
    );
}

/// Fila cheia recusa **antes** de mover item: inventário e equipamento intactos.
#[test]
fn fila_cheia_recusa_equipar_sem_mutar() {
    let mut store = ComponentStore::with_event_capacity(1);
    let e = spawn(&mut store, 50, &[(0, 100, 1), (1, 102, 1)]);
    equip(&mut store, e, 0, &Catalogo, EquipmentRules::default()).expect("primeira");
    let antes = inventario(&store, e);

    assert_eq!(
        equip(&mut store, e, 1, &Catalogo, EquipmentRules::default()),
        Err(CoreError::EventQueueFull { capacity: 1 })
    );
    assert_eq!(inventario(&store, e), antes);
    assert_eq!(store.events().len(), 1, "só o evento do primeiro equipar");
}
