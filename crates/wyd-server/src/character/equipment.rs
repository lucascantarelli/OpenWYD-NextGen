//! Regras de equipamento do catálogo real (`item_catalog`).
//!
//! O core já valida o nível no caminho `equip`/`equip_into`, mas ele não pode adivinhar o
//! mapa `kind → slot` nem o `ReqLvl`: isso é dado do catálogo. Esta é a implementação de
//! [`ItemRules`] usada pelo servidor para que a validação seja feita com o mesmo dado que o
//! portal exibe, sem constante de balanceamento no handler.

use std::collections::BTreeMap;

use wyd_core::ItemRules;
use wyd_db::CatalogItemRow;

/// Regra de um item do catálogo para equipamento.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct EquipRule {
    pub slot: u8,
    pub req_level: u16,
    pub max_stack: u32,
}

/// Catálogo de equipamento montado uma vez a partir de `catalog_snapshot`.
///
/// Itens sem `kind` ou de categoria que não é equipamento não entram: o core recebe `None`
/// e recusa (`NotEquippable`), em vez de inventar um slot.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EquipmentCatalog {
    rules: BTreeMap<u32, EquipRule>,
}

impl EquipmentCatalog {
    /// Monta as regras de equipamento da foto do catálogo.
    #[must_use]
    pub fn from_catalog(rows: &[CatalogItemRow]) -> Self {
        let mut rules = BTreeMap::new();
        for row in rows {
            let Some(slot) = row.kind.as_deref().and_then(slot_for_kind) else {
                continue;
            };
            let Ok(item_index) = u32::try_from(row.item_index) else {
                continue;
            };
            rules.insert(
                item_index,
                EquipRule {
                    slot,
                    req_level: u16::try_from(row.req_level.max(0)).unwrap_or(0),
                    max_stack: if row.stackable == Some(true) { 120 } else { 1 },
                },
            );
        }
        Self { rules }
    }

    /// Regra de um item, quando ele é equipável.
    #[must_use]
    pub fn rule(&self, item_index: u32) -> Option<EquipRule> {
        self.rules.get(&item_index).copied()
    }

    /// Quantos itens equipáveis o catálogo carregou (observabilidade/teste).
    #[must_use]
    pub fn len(&self) -> usize {
        self.rules.len()
    }

    /// Catálogo sem itens equipáveis.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.rules.is_empty()
    }
}

impl ItemRules for EquipmentCatalog {
    fn equip_slot(&self, item_index: u32) -> Option<u8> {
        self.rule(item_index).map(|rule| rule.slot)
    }

    fn req_level(&self, item_index: u32) -> u16 {
        self.rule(item_index).map_or(0, |rule| rule.req_level)
    }

    fn max_stack(&self, item_index: u32) -> u32 {
        self.rule(item_index).map_or(1, |rule| rule.max_stack)
    }
}

/// Slots do `STRUCT_MOB.Equip[16]` usados pelo 7662.
///
/// A categoria vem de `item_kind.slot` no banco (texto) e aqui vira o índice numérico do
/// fio. `armor` cobre o corpo (2); calça não tem categoria própria no catálogo e continua
/// sem equipamento automático até o dado existir.
fn slot_for_kind(kind: &str) -> Option<u8> {
    match kind {
        "helmet" => Some(1),
        "armor" => Some(2),
        "gloves" => Some(4),
        "boots" => Some(5),
        "weapon" => Some(6),
        "shield" => Some(7),
        "cloak" => Some(8),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn row(item_index: i64, kind: &str, req_level: i16) -> CatalogItemRow {
        CatalogItemRow {
            item_index,
            name: format!("item-{item_index}"),
            price: 0,
            grade: 0,
            req_level,
            req_str: 0,
            req_int: 0,
            req_dex: 0,
            req_con: 0,
            attack_range: None,
            volatile: None,
            kind: Some(kind.to_string()),
            primary_system: None,
            event_only: None,
            tradeable: Some(true),
            stackable: Some(false),
            class_mask: None,
            effect_names: Vec::new(),
        }
    }

    #[test]
    fn mapeia_slots_e_requisitos_do_catalogo() {
        let catalogo = EquipmentCatalog::from_catalog(&[
            row(831, "weapon", 4),
            row(1006, "weapon", 319),
            row(1251, "helmet", 0),
            row(451, "quest", 0),
        ]);
        assert_eq!(catalogo.equip_slot(831), Some(6));
        assert_eq!(catalogo.req_level(831), 4);
        assert_eq!(catalogo.req_level(1006), 319);
        assert_eq!(catalogo.equip_slot(1251), Some(1));
        assert_eq!(catalogo.equip_slot(451), None, "quest não equipa");
        assert_eq!(catalogo.max_stack(831), 1);
    }
}
