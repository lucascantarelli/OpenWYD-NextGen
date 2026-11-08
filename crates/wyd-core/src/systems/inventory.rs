//! Inventário e equipamento server-authoritative (FASE-03-5, eixo inventário).
//!
//! A intenção aceita é mínima: **qual** item da mochila e, quando o protocolo pedir slot, qual
//! slot o cliente *diz* querer. O servidor decide o resto — slot de equipamento derivado do
//! catálogo, requisito de nível, política de troca, destino do item desequipado e limite de
//! pilha. Nenhuma assinatura recebe bônus, item resultante ou "slot final confirmado".
//!
//! Regras vêm de [`ItemRules`], implementado pelo servidor a partir de `item_catalog` +
//! `item_kind` (ver `docs/wyd/itens.md` §5): o core não fixa requisito nem mapa de slot.
//!
//! Toda operação segue a ordem dos demais sistemas: entidade viva → pré-condições completas →
//! espaço na fila → evento → mutação. Recusa não muda item, equipamento, mochila nem fila.

use crate::ecs::components::{ComponentStore, Entity, Item, BAG_SLOTS, EQUIP_SLOTS};
use crate::ecs::resources::DomainEvent;
use crate::error::CoreError;

/// Regras de item do servidor (ponte para o catálogo do banco).
///
/// `equip_slot` devolve `None` para item não equipável (material, consumível, quest…), o que
/// mantém o core ignorante de `kind`: o mapeamento `kind → slot` é dado, não regra da engine.
pub trait ItemRules {
    /// Slot de equipamento do item (`None` = não equipável).
    fn equip_slot(&self, item_index: u32) -> Option<u8>;
    /// Nível mínimo para equipar.
    fn req_level(&self, item_index: u32) -> u16;
    /// Pilha máxima permitida (`1` = não empilha).
    fn max_stack(&self, item_index: u32) -> u32;
}

/// Política de troca quando o slot de equipamento já está ocupado.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct EquipmentRules {
    /// Permite trocar: o item que estava no slot volta para a mochila, no lugar do novo.
    pub allow_swap: bool,
}

impl Default for EquipmentRules {
    fn default() -> Self {
        Self { allow_swap: true }
    }
}

/// Resultado de um equipamento aceito.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Equipped {
    /// Slot de equipamento que passou a valer (derivado do catálogo).
    pub slot: u8,
    /// Slot da mochila de onde o item saiu.
    pub bag_slot: u8,
    /// Item equipado (com refino/Anct/gema que já tinha).
    pub item: Item,
    /// Item que voltou para a mochila, quando houve troca.
    pub swapped_out: Option<Item>,
}

/// Resultado de um desequipamento aceito.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Unequipped {
    /// Slot de equipamento que ficou vazio.
    pub slot: u8,
    /// Slot da mochila que recebeu o item (escolhido pelo servidor).
    pub bag_slot: u8,
    /// Item devolvido.
    pub item: Item,
}

fn item_do_slot(
    store: &ComponentStore,
    entity: Entity,
    bag_slot: usize,
) -> Result<Item, CoreError> {
    if bag_slot >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: bag_slot as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    let inv = store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let item = inv.bag[bag_slot];
    if item.id == 0 {
        return Err(CoreError::EmptySlot {
            slot: bag_slot as u16,
        });
    }
    Ok(item)
}

/// Equipa o item `bag_slot` no slot que o **catálogo** manda.
///
/// Caminho de jogo: o cliente manda o item, nunca o slot. Use [`equip_into`] só quando o
/// protocolo trouxer um slot declarado — e ele será conferido contra o catálogo, não obedecido.
pub fn equip<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    bag_slot: usize,
    rules: &R,
    policy: EquipmentRules,
) -> Result<Equipped, CoreError> {
    let item = item_do_slot(store, entity, bag_slot)?;
    let slot = rules.equip_slot(item.id).ok_or(CoreError::NotEquippable {
        item_index: item.id,
    })?;
    equip_into(store, entity, bag_slot, slot, rules, policy)
}

/// Equipa conferindo o slot **pedido** contra o slot do catálogo.
///
/// Slot divergente é recusa tipada (`WrongEquipSlot`): o cliente não escolhe onde o item entra.
pub fn equip_into<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    bag_slot: usize,
    requested_slot: u8,
    rules: &R,
    policy: EquipmentRules,
) -> Result<Equipped, CoreError> {
    let item = item_do_slot(store, entity, bag_slot)?;
    let expected = rules.equip_slot(item.id).ok_or(CoreError::NotEquippable {
        item_index: item.id,
    })?;
    if usize::from(requested_slot) >= EQUIP_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: u16::from(requested_slot),
            max: (EQUIP_SLOTS - 1) as u16,
        });
    }
    if requested_slot != expected {
        return Err(CoreError::WrongEquipSlot {
            expected,
            requested: requested_slot,
        });
    }
    let stats = store.stats_of(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let required = rules.req_level(item.id);
    if stats.level < required {
        return Err(CoreError::LevelTooLow {
            required,
            current: stats.level,
        });
    }
    let inv = store
        .inventory(entity)
        .expect("entidade viva tem inventário");
    let ocupado = inv.equip[usize::from(expected)];
    if ocupado.id != 0 && !policy.allow_swap {
        return Err(CoreError::EquipSlotOccupied { slot: expected });
    }
    let swapped_out = (ocupado.id != 0).then_some(ocupado);

    store.publish(DomainEvent::ItemEquipped {
        entity,
        bag_slot: bag_slot as u8,
        equip_slot: expected,
        item_index: item.id,
    })?;
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    inv.equip[usize::from(expected)] = item;
    inv.bag[bag_slot] = swapped_out.unwrap_or_default();
    Ok(Equipped {
        slot: expected,
        bag_slot: bag_slot as u8,
        item,
        swapped_out,
    })
}

/// Desequipa e devolve o item ao **primeiro** slot livre da mochila (escolha do servidor).
///
/// Mochila cheia é recusa (`BagFull`): nada é destruído nem sobrescrito para "abrir espaço".
pub fn unequip(
    store: &mut ComponentStore,
    entity: Entity,
    slot: u8,
) -> Result<Unequipped, CoreError> {
    if usize::from(slot) >= EQUIP_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: u16::from(slot),
            max: (EQUIP_SLOTS - 1) as u16,
        });
    }
    let inv = store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let item = inv.equip[usize::from(slot)];
    if item.id == 0 {
        return Err(CoreError::EmptySlot {
            slot: u16::from(slot),
        });
    }
    let bag_slot = inv
        .bag
        .iter()
        .position(|bag| bag.id == 0)
        .ok_or(CoreError::BagFull {
            slots: BAG_SLOTS as u16,
        })?;

    store.publish(DomainEvent::ItemUnequipped {
        entity,
        equip_slot: slot,
        bag_slot: bag_slot as u8,
        item_index: item.id,
    })?;
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    inv.equip[usize::from(slot)] = Item::default();
    inv.bag[bag_slot] = item;
    Ok(Unequipped {
        slot,
        bag_slot: bag_slot as u8,
        item,
    })
}

/// Move (ou empilha) item entre slots da mochila.
///
/// Destino vazio recebe o item inteiro; destino com o **mesmo** item empilhável recebe até o
/// limite do catálogo (`max_stack`), e o que não couber fica na origem. Item diferente no
/// destino é recusa (`BagSlotOccupied`) — nada de troca silenciosa entre slots.
pub fn move_item<R: ItemRules>(
    store: &mut ComponentStore,
    entity: Entity,
    from: usize,
    to: usize,
    rules: &R,
) -> Result<u32, CoreError> {
    if from >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: from as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    if to >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: to as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    if from == to {
        return Err(CoreError::SameSlot { slot: from as u16 });
    }
    let origem = item_do_slot(store, entity, from)?;
    let inv = store
        .inventory(entity)
        .expect("entidade viva tem inventário");
    let destino = inv.bag[to];
    let movidos = if destino.id == 0 {
        origem.qty
    } else if destino.id == origem.id {
        let limite = rules.max_stack(origem.id).max(1);
        let espaco = limite.saturating_sub(destino.qty);
        if espaco == 0 {
            return Err(CoreError::StackFull { slot: to as u16 });
        }
        origem.qty.min(espaco)
    } else {
        return Err(CoreError::BagSlotOccupied { slot: to as u16 });
    };

    store.publish(DomainEvent::ItemMoved {
        entity,
        from: from as u8,
        to: to as u8,
        item_index: origem.id,
        qty: movidos,
    })?;
    let inv = store
        .inventory_mut(entity)
        .expect("entidade viva tem inventário");
    inv.bag[from].qty -= movidos;
    if inv.bag[from].qty == 0 {
        inv.bag[from] = Item::default();
    }
    if inv.bag[to].id == 0 {
        inv.bag[to] = Item {
            qty: movidos,
            ..origem
        };
    } else {
        inv.bag[to].qty += movidos;
    }
    Ok(movidos)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{EntityKind, Inventory, Position, Stats};
    use std::collections::HashMap;

    /// Catálogo de teste: espada no slot 0 (req 10), elmo no 2 (req 40), minério não equipa.
    struct Catalogo(HashMap<u32, (Option<u8>, u16)>);

    impl ItemRules for Catalogo {
        fn equip_slot(&self, item_index: u32) -> Option<u8> {
            self.0.get(&item_index).and_then(|(slot, _)| *slot)
        }
        fn req_level(&self, item_index: u32) -> u16 {
            self.0.get(&item_index).map_or(0, |(_, req)| *req)
        }
        fn max_stack(&self, _item_index: u32) -> u32 {
            10
        }
    }

    fn catalogo() -> Catalogo {
        Catalogo(HashMap::from([
            (100, (Some(0), 10)),
            (101, (Some(2), 40)),
            (200, (None, 0)),
        ]))
    }

    fn stats(level: u16) -> Stats {
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

    fn cenario(level: u16, bag: &[(usize, u32, u32)]) -> (ComponentStore, Entity) {
        let mut inv = Inventory::default();
        for (slot, id, qty) in bag {
            inv.bag[*slot] = Item {
                id: *id,
                qty: *qty,
                ..Item::default()
            };
        }
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(level), inv)
            .expect("spawn válido");
        (store, e)
    }

    #[test]
    fn equipa_no_slot_do_catalogo() {
        let (mut store, e) = cenario(50, &[(0, 100, 1)]);
        let feito =
            equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("equipa");
        assert_eq!((feito.slot, feito.bag_slot), (0, 0));
        assert_eq!(store.inventory(e).expect("vivo").equip[0].id, 100);
        assert_eq!(store.inventory(e).expect("vivo").bag[0].id, 0);
        assert_eq!(
            store.drain_events(),
            vec![DomainEvent::ItemEquipped {
                entity: e,
                bag_slot: 0,
                equip_slot: 0,
                item_index: 100
            }]
        );
    }

    #[test]
    fn slot_pedido_diferente_do_catalogo_e_recusado() {
        let (mut store, e) = cenario(50, &[(0, 100, 1)]);
        assert_eq!(
            equip_into(&mut store, e, 0, 2, &catalogo(), EquipmentRules::default()),
            Err(CoreError::WrongEquipSlot {
                expected: 0,
                requested: 2
            })
        );
        assert_eq!(store.inventory(e).expect("vivo").bag[0].id, 100);
        assert!(store.events().is_empty());
    }

    #[test]
    fn troca_devolve_o_anterior_para_a_mochila() {
        // Duas espadas (mesmo slot 0): a segunda entra no lugar da primeira, que volta para a
        // mochila no slot de onde a nova saiu. O refino distingue as duas.
        let mut inv = Inventory::default();
        inv.bag[0] = Item {
            id: 100,
            qty: 1,
            ..Item::default()
        };
        inv.bag[1] = Item {
            id: 100,
            qty: 1,
            refine: 3,
            ..Item::default()
        };
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(50), inv)
            .expect("spawn válido");

        equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("primeira espada");
        let troca = equip(&mut store, e, 1, &catalogo(), EquipmentRules::default())
            .expect("troca de espada");
        assert_eq!(troca.item.id, 100);
        assert_eq!(troca.item.refine, 3, "a nova espada é a equipada");
        assert_eq!(
            troca.swapped_out.map(|item| item.refine),
            Some(0),
            "a anterior volta para a mochila"
        );
        assert_eq!(store.inventory(e).expect("vivo").equip[0].refine, 3);
        assert_eq!(store.inventory(e).expect("vivo").bag[1].refine, 0);
    }

    #[test]
    fn politica_sem_troca_recusa_slot_ocupado() {
        let (mut store, e) = cenario(50, &[(0, 100, 1), (1, 100, 1)]);
        equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("primeira");
        store.drain_events();
        assert_eq!(
            equip(
                &mut store,
                e,
                1,
                &catalogo(),
                EquipmentRules { allow_swap: false }
            ),
            Err(CoreError::EquipSlotOccupied { slot: 0 })
        );
        assert!(store.events().is_empty());
        assert_eq!(store.inventory(e).expect("vivo").bag[1].id, 100);
    }

    #[test]
    fn desequipa_para_o_primeiro_slot_livre() {
        let (mut store, e) = cenario(50, &[(0, 100, 1)]);
        equip(&mut store, e, 0, &catalogo(), EquipmentRules::default()).expect("equipa");
        store.drain_events();
        let saida = unequip(&mut store, e, 0).expect("desequipa");
        assert_eq!((saida.slot, saida.bag_slot), (0, 0));
        assert_eq!(store.inventory(e).expect("vivo").bag[0].id, 100);
        assert_eq!(store.inventory(e).expect("vivo").equip[0].id, 0);
    }

    #[test]
    fn desequipar_sem_item_e_recusado() {
        let (mut store, e) = cenario(50, &[]);
        assert_eq!(
            unequip(&mut store, e, 3),
            Err(CoreError::EmptySlot { slot: 3 })
        );
        assert!(store.events().is_empty());
    }

    #[test]
    fn mochila_cheia_recusa_desequipar() {
        let mut inv = Inventory::default();
        inv.equip[0] = Item {
            id: 100,
            qty: 1,
            ..Item::default()
        };
        for slot in &mut inv.bag {
            *slot = Item {
                id: 200,
                qty: 1,
                ..Item::default()
            };
        }
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(50), inv)
            .expect("spawn válido");
        assert_eq!(
            unequip(&mut store, e, 0),
            Err(CoreError::BagFull {
                slots: BAG_SLOTS as u16
            })
        );
        assert_eq!(store.inventory(e).expect("vivo").equip[0].id, 100);
        assert!(store.events().is_empty());
    }

    #[test]
    fn mover_empilha_ate_o_limite_do_catalogo() {
        let (mut store, e) = cenario(50, &[(0, 200, 6), (1, 200, 7)]);
        let movidos = move_item(&mut store, e, 0, 1, &catalogo()).expect("empilha");
        assert_eq!(movidos, 3);
        assert_eq!(store.inventory(e).expect("vivo").bag[0].qty, 3);
        assert_eq!(store.inventory(e).expect("vivo").bag[1].qty, 10);
    }
}
