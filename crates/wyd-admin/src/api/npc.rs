//! `npc_admin/get_npc` e `npc_admin/upsert_npc` — posição e visibilidade de NPC.
//!
//! NPC é definição de mundo (não estado vivo de personagem): mover e desligar aqui é
//! edição de conteúdo auditada, e o servidor recarrega quando `config_revision` muda.
//! O espaço de coordenadas é o do cliente: mapa 0..255, x/y 0..4095.

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, ItemPriceEdit, ItemPriceRow, NpcRow};

/// Teto de NPCs por leitura (a definição tem 6.100 linhas; sem paginação o portal trava).
pub const MAX_LIMIT: i64 = 200;

/// Lista NPCs em páginas por id (só leitura, FASE-07-5).
///
/// Sem escrita em `audit_log`: leitura pura não é ação auditável (a negação de acesso
/// continua auditada por `staff_or_deny`). `after_id` é o cursor (`None` = primeira página).
pub fn list_npcs(
    session: Option<Session>,
    limit: i64,
    after_id: Option<i32>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<NpcRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.npcs")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de NPCs fora da faixa",
        });
    }
    if after_id.is_some_and(|id| id < 0) {
        return Err(AdminError::InvalidInput {
            reason: "cursor de NPC inválido",
        });
    }
    store.npcs(limit, after_id)
}

/// Leitura de um NPC por id.
pub fn get_npc(
    session: Option<Session>,
    id: i32,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Option<NpcRow>, AdminError> {
    let staff = staff_or_deny(session, sink, "config.npc")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "id de NPC inválido",
        });
    }
    let npc = store.npc(id)?;
    log_action(
        sink,
        &staff,
        "config.npc",
        &format!("npc:{id}"),
        if npc.is_some() { "leitura" } else { "ausente" },
    );
    Ok(npc)
}

/// Move/ativa um NPC, validando o espaço de coordenadas antes do banco.
pub fn set_npc(
    session: Option<Session>,
    id: i32,
    (map_id, pos_x, pos_y): (i16, i16, i16),
    enabled: bool,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.npc.set")?;
    if id <= 0 || !(0..=255).contains(&map_id) {
        return Err(AdminError::InvalidInput {
            reason: "NPC ou mapa inválido",
        });
    }
    if !(0..=4095).contains(&pos_x) || !(0..=4095).contains(&pos_y) {
        return Err(AdminError::InvalidInput {
            reason: "posição fora do espaço 0..4095",
        });
    }
    let revision = store.set_npc(id, map_id, pos_x, pos_y, enabled, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.npc.set",
        &format!("npc:{id}"),
        &format!("map={map_id} x={pos_x} y={pos_y} enabled={enabled} (revisão {revision})"),
    );
    Ok(revision)
}

/// Teto de preços por leitura (a tabela é configuração, não catálogo).
pub const MAX_PRICES: i64 = 200;

/// Valida a forma de uma edição de preço **antes** do banco.
///
/// Item e NPC são obrigatórios: a chave de `item_price` é o par, e preço sem mercador
/// não é preço de loja. Os valores aceitam zero (item não vendido/comprado por aquele
/// mercador), mas fonte vazia é recusada — mesmo rigor das taxas: sem procedência não
/// grava.
pub fn validar_preco(edit: &ItemPriceEdit) -> Result<(), AdminError> {
    if edit.item_index <= 0 || edit.npc_id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item e NPC do preço são obrigatórios",
        });
    }
    if edit.buy_price < 0 || edit.sell_price < 0 {
        return Err(AdminError::InvalidInput {
            reason: "preço não pode ser negativo",
        });
    }
    let source = edit.source.trim();
    if source.is_empty() {
        return Err(AdminError::InvalidInput {
            reason: "fonte do preço obrigatória",
        });
    }
    if source.chars().count() > super::rates::MAX_SOURCE {
        return Err(AdminError::InvalidInput {
            reason: "fonte do preço longa demais",
        });
    }
    Ok(())
}

/// Preços de NPC, filtrados por item quando informado (staff, só leitura).
///
/// Sem escrita em `audit_log` para a leitura bem-sucedida (mesma regra de `list_npcs`);
/// a negação de acesso continua auditada por `staff_or_deny`.
pub fn list_item_prices(
    session: Option<Session>,
    item_index: Option<i32>,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<ItemPriceRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.prices")?;
    if !(1..=MAX_PRICES).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de preços fora da faixa",
        });
    }
    if item_index.is_some_and(|item| item <= 0) {
        return Err(AdminError::InvalidInput {
            reason: "item do filtro inválido",
        });
    }
    store.item_prices(item_index, limit)
}

/// Grava o preço de (item, NPC) e devolve a revisão.
///
/// O servidor de jogo relê quando `config_revision` muda; nada aqui toca personagem vivo
/// nem inventa preço: item fora do catálogo é recusado pela FK do banco.
pub fn set_item_price(
    session: Option<Session>,
    edit: &ItemPriceEdit,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.price.set")?;
    validar_preco(edit)?;
    let mut limpo = edit.clone();
    limpo.source = edit.source.trim().to_string();
    let revision = store.set_item_price(&limpo, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.price.set",
        &format!("item{}@npc{}", edit.item_index, edit.npc_id),
        &format!(
            "compra={} venda={} fonte={} (revisão {revision})",
            edit.buy_price, edit.sell_price, limpo.source
        ),
    );
    Ok(revision)
}

/// Lista JSON de preços.
#[must_use]
pub fn item_prices_json(rows: &[ItemPriceRow]) -> String {
    let itens: Vec<String> = rows
        .iter()
        .map(|row| {
            format!(
                "{{\"item_index\":{},\"item_name\":{},\"npc_id\":{},\"npc_slug\":{},\"buy_price\":{},\"sell_price\":{},\"source\":{}}}",
                row.item_index,
                json::quote(&row.item_name),
                row.npc_id,
                json::quote(&row.npc_slug),
                row.buy_price,
                row.sell_price,
                json::quote(&row.source)
            )
        })
        .collect();
    format!("[{}]", itens.join(","))
}

/// Lista JSON de NPCs.
#[must_use]
pub fn npcs_json(rows: &[NpcRow]) -> String {
    let itens: Vec<String> = rows.iter().map(npc_json).collect();
    format!("[{}]", itens.join(","))
}

/// JSON de um NPC.
#[must_use]
pub fn npc_json(row: &NpcRow) -> String {
    format!(
        "{{\"id\":{},\"slug\":{},\"display_name\":{},\"map_id\":{},\"pos_x\":{},\
\"pos_y\":{},\"enabled\":{},\"merchant\":{}}}",
        row.id,
        json::quote(&row.slug),
        json::quote(&row.display_name),
        row.map_id,
        row.pos_x,
        row.pos_y,
        row.enabled,
        row.merchant
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    fn edit() -> ItemPriceEdit {
        ItemPriceEdit {
            item_index: 3321,
            npc_id: 1212,
            buy_price: 5_000,
            sell_price: 1_250,
            source: "tabela do dono do servidor".to_string(),
        }
    }

    #[test]
    fn preco_valido_passa() {
        assert!(validar_preco(&edit()).is_ok());
        // Zero é permitido: mercador que não vende (ou não compra) aquele item.
        assert!(validar_preco(&ItemPriceEdit {
            buy_price: 0,
            sell_price: 0,
            ..edit()
        })
        .is_ok());
    }

    #[test]
    fn preco_sem_item_ou_npc_e_recusado_antes_do_banco() {
        for invalido in [
            ItemPriceEdit {
                item_index: 0,
                ..edit()
            },
            ItemPriceEdit {
                npc_id: 0,
                ..edit()
            },
        ] {
            assert!(matches!(
                validar_preco(&invalido),
                Err(AdminError::InvalidInput { .. })
            ));
        }
    }

    #[test]
    fn preco_negativo_e_recusado() {
        assert!(validar_preco(&ItemPriceEdit {
            buy_price: -1,
            ..edit()
        })
        .is_err());
        assert!(validar_preco(&ItemPriceEdit {
            sell_price: -1,
            ..edit()
        })
        .is_err());
    }

    #[test]
    fn fonte_vazia_ou_longa_demais_e_recusada() {
        // Mesma regra das taxas: preço sem procedência não vira configuração.
        assert!(matches!(
            validar_preco(&ItemPriceEdit {
                source: "   ".to_string(),
                ..edit()
            }),
            Err(AdminError::InvalidInput { reason }) if reason == "fonte do preço obrigatória"
        ));
        assert!(validar_preco(&ItemPriceEdit {
            source: "x".repeat(super::super::rates::MAX_SOURCE + 1),
            ..edit()
        })
        .is_err());
    }

    #[test]
    fn json_do_preco_leva_item_mercador_e_fonte() {
        let linha = ItemPriceRow {
            item_index: 3321,
            item_name: "Espada".to_string(),
            npc_id: 1212,
            npc_slug: "ferreiro".to_string(),
            buy_price: 5_000,
            sell_price: 1_250,
            source: "dono".to_string(),
        };
        let json = item_prices_json(&[linha]);
        assert!(json.contains("\"item_index\":3321"));
        assert!(json.contains("\"npc_slug\":\"ferreiro\""));
        assert!(json.contains("\"buy_price\":5000"));
        assert!(json.contains("\"source\":\"dono\""));
    }
}
