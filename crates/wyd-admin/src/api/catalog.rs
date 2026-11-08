//! `item_catalog/list_items` — primeira rota de dados servida pelo gateway (ADR-0022).
//!
//! É a rota que o `wyd-portal` usa em `/api/health` (`wyd-portal/src/app/api/health/route.ts`):
//! ela prova o caminho portal → gateway → banco sem inventar dado. A foto do catálogo é carregada
//! do Postgres no boot (`wyd_db::PgBackend::catalog_snapshot`) e servida daqui.
//!
//! **Lacuna declarada:** o portal tipa `CatalogItem` com `icon_key`, `slot_mask`, `slots`,
//! `grade`, `mesh`, `texture` e `icon_url`; essas colunas ainda não existem em `item_catalog`
//! (o schema atual tem `item_index`, `name`, `price`, `req_level`, `kind`). A resposta preenche
//! com vazio/0 em vez de mentir, e o campo aparece como pendência no ADR-0022/runbook.

use wyd_db::CatalogItemRow;

use crate::transport::json;

/// Objetos `CatalogItem` do contrato do portal, na ordem do catálogo.
#[must_use]
pub fn items_json(items: &[CatalogItemRow]) -> String {
    let rendered: Vec<json::Json> = items
        .iter()
        .map(|item| {
            json::Json::Obj(vec![
                (
                    "item_index".to_string(),
                    json::Json::Num(item.item_index as f64),
                ),
                ("name".to_string(), json::Json::Str(item.name.clone())),
                (
                    "display_name".to_string(),
                    json::Json::Str(item.name.clone()),
                ),
                ("icon_key".to_string(), json::Json::Str(String::new())),
                ("slot_mask".to_string(), json::Json::Num(0.0)),
                ("slots".to_string(), json::Json::Arr(Vec::new())),
                ("grade".to_string(), json::Json::Num(0.0)),
                ("mesh".to_string(), json::Json::Num(0.0)),
                ("texture".to_string(), json::Json::Num(0.0)),
                ("icon_url".to_string(), json::Json::Str(String::new())),
            ])
        })
        .collect();
    json::Json::Arr(rendered).render()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn itens_vazios_viram_array_json() {
        assert_eq!(items_json(&[]), "[]");
    }

    #[test]
    fn item_carrega_indice_e_nome() {
        let itens = vec![CatalogItemRow {
            item_index: 7,
            name: "Espada".to_string(),
        }];
        let texto = items_json(&itens);
        assert!(texto.contains("\"item_index\":7"));
        assert!(texto.contains("\"name\":\"Espada\""));
        assert!(texto.contains("\"display_name\":\"Espada\""));
    }
}
