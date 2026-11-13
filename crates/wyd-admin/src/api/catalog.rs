//! `item_catalog/list_items` — primeira rota de dados servida pelo gateway (ADR-0022).
//!
//! É a rota que o `wyd-portal` usa em `/api/health` (`wyd-portal/src/app/api/health/route.ts`):
//! ela prova o caminho portal → gateway → banco sem inventar dado. A foto do catálogo é carregada
//! do Postgres no boot (`wyd_db::PgBackend::catalog_snapshot`) e servida daqui.
//!
//! Leva o que o schema tem de verdade: preço, grade, requisitos, alcance, efeitos e a
//! classificação de domínio. Campo que o schema **não** tem (`icon_key`, `slot_mask`,
//! `slots`, `mesh`, `texture`, `icon_url`) sai vazio/0 para o contrato do portal continuar
//! válido — vazio é honesto, número inventado não.

use wyd_db::CatalogItemRow;

use crate::transport::json;

/// Objetos `CatalogItem` do contrato do portal, na ordem do catálogo.
#[must_use]
pub fn items_json(items: &[CatalogItemRow]) -> String {
    let rendered: Vec<String> = items.iter().map(item_json).collect();
    format!("[{}]", rendered.join(","))
}

fn item_json(item: &CatalogItemRow) -> String {
    let efeitos: Vec<String> = item
        .effect_names
        .iter()
        .map(|nome| json::quote(nome))
        .collect();
    format!(
        "{{\"item_index\":{},\"name\":{},\"display_name\":{},\
         \"icon_key\":\"\",\"slot_mask\":0,\"slots\":[],\"mesh\":0,\"texture\":0,\"icon_url\":\"\",\
         \"grade\":{},\"price\":{},\
         \"req_level\":{},\"req_str\":{},\"req_int\":{},\"req_dex\":{},\"req_con\":{},\
         \"attack_range\":{},\"volatile\":{},\
         \"kind\":{},\"primary_system\":{},\"event_only\":{},\"tradeable\":{},\"stackable\":{},\
         \"class_mask\":{},\"effect_names\":[{}]}}",
        item.item_index,
        json::quote(&item.name),
        json::quote(&item.name),
        item.grade,
        item.price,
        item.req_level,
        item.req_str,
        item.req_int,
        item.req_dex,
        item.req_con,
        opt_num(item.attack_range),
        opt_num(item.volatile),
        opt_str(item.kind.as_deref()),
        opt_str(item.primary_system.as_deref()),
        opt_bool(item.event_only),
        opt_bool(item.tradeable),
        opt_bool(item.stackable),
        opt_str(item.class_mask.as_deref()),
        efeitos.join(",")
    )
}

fn opt_num(value: Option<i16>) -> String {
    value.map_or_else(|| "null".to_string(), |n| n.to_string())
}

fn opt_str(value: Option<&str>) -> String {
    value.map_or_else(|| "null".to_string(), json::quote)
}

fn opt_bool(value: Option<bool>) -> String {
    value.map_or_else(|| "null".to_string(), |flag| flag.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn item() -> CatalogItemRow {
        CatalogItemRow {
            item_index: 7,
            name: "Espada".to_string(),
            price: 1_200,
            grade: 2,
            req_level: 30,
            req_str: 45,
            req_int: 0,
            req_dex: 0,
            req_con: 0,
            attack_range: Some(2),
            volatile: None,
            kind: Some("weapon".to_string()),
            primary_system: None,
            event_only: Some(false),
            tradeable: Some(true),
            stackable: None,
            class_mask: None,
            effect_names: vec!["EF_NOSANC".to_string()],
        }
    }

    #[test]
    fn itens_vazios_viram_array_json() {
        assert_eq!(items_json(&[]), "[]");
    }

    #[test]
    fn item_carrega_indice_e_nome() {
        let itens = vec![item()];
        let texto = items_json(&itens);
        assert!(texto.contains("\"item_index\":7"));
        assert!(texto.contains("\"name\":\"Espada\""));
        assert!(texto.contains("\"display_name\":\"Espada\""));
    }

    #[test]
    fn item_carrega_atributos_e_requisitos_do_schema() {
        let texto = items_json(&[item()]);
        assert!(texto.contains("\"price\":1200"));
        assert!(texto.contains("\"grade\":2"));
        assert!(texto.contains("\"req_level\":30"));
        assert!(texto.contains("\"req_str\":45"));
        assert!(texto.contains("\"attack_range\":2"));
        assert!(texto.contains("\"kind\":\"weapon\""));
        assert!(texto.contains("\"effect_names\":[\"EF_NOSANC\"]"));
    }

    #[test]
    fn campo_ausente_vira_null_nunca_zero() {
        let texto = items_json(&[item()]);
        assert!(texto.contains("\"volatile\":null"));
        assert!(texto.contains("\"class_mask\":null"));
        assert!(texto.contains("\"stackable\":null"));
    }
}
