//! `donate_admin/*` — loja de donate e crédito de saldo (FASE-07-6).
//!
//! A loja vende itens por créditos já conciliados (dinheiro real nunca entra aqui).
//! `credit_donate_balance` credita `account_wallet` (saldo de **conta**, nunca personagem
//! vivo) com idempotência por chave via `donate_credit`: mesma chave devolve o mesmo razão
//! sem creditar duas vezes. Toda escrita é auditada com autor/alvo/antes/depois.

use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, DonateEdit, DonateItemRow};

/// Teto de itens por leitura.
pub const MAX_LIMIT: i64 = 200;

/// Lista itens da loja em páginas por id.
pub fn list_shop_items(
    session: Option<Session>,
    limit: i64,
    after_id: Option<i32>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DonateItemRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.donate_items")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite da loja fora da faixa",
        });
    }
    if after_id.is_some_and(|id| id < 0) {
        return Err(AdminError::InvalidInput {
            reason: "cursor da loja inválido",
        });
    }
    store.donate_items(limit, after_id)
}

/// Cria (`edit.id = None`) ou substitui um item da loja.
pub fn upsert_shop_item(
    session: Option<Session>,
    edit: &DonateEdit,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.set")?;
    validar_edicao(edit)?;
    let revision = store.upsert_donate_item(edit, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.set",
        &edit
            .id
            .map_or_else(|| "shop:novo".to_string(), |id| format!("shop:{id}")),
        &format!(
            "item={} preco={} enabled={}",
            edit.item_index, edit.price_credits, edit.enabled
        ),
    );
    Ok(revision)
}

/// Liga/desliga um item da loja.
pub fn set_shop_item_enabled(
    session: Option<Session>,
    id: i32,
    enabled: bool,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.set")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item da loja inválido",
        });
    }
    let revision = store.set_donate_item_enabled(id, enabled, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.set",
        &format!("shop:{id}"),
        &format!("enabled={enabled}"),
    );
    Ok(revision)
}

/// Remove um item da loja (não apaga compra já feita).
pub fn delete_shop_item(
    session: Option<Session>,
    id: i32,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.delete")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item da loja inválido",
        });
    }
    let revision = store.delete_donate_item(id, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.delete",
        &format!("shop:{id}"),
        "removido",
    );
    Ok(revision)
}

/// Recibo do crédito (mesma chave, mesmo razão — prova de idempotência).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateCreditReceipt {
    pub idempotency_key: String,
    pub credit_id: i64,
    pub replay: bool,
}

/// Credita saldo de donate na conta, idempotente por chave.
pub fn credit_balance(
    session: Option<Session>,
    account_id: i64,
    credits: i64,
    key: &str,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<DonateCreditReceipt, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.credit")?;
    if account_id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "conta inválida",
        });
    }
    if credits <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "créditos precisam ser positivos",
        });
    }
    let key = key.trim();
    if key.is_empty() || key.len() > 120 {
        return Err(AdminError::InvalidInput {
            reason: "chave de idempotência obrigatória (até 120 caracteres)",
        });
    }
    let (credit_id, replay) =
        store.credit_donate(account_id, credits, key, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        if replay {
            "config.donate.replay"
        } else {
            "config.donate.credit"
        },
        &format!("account:{account_id}"),
        &format!("{credits} créditos (chave {key})"),
    );
    Ok(DonateCreditReceipt {
        idempotency_key: key.to_string(),
        credit_id,
        replay,
    })
}

/// Receita de donate consolidada (soma de linhas de razão, nunca estimativa).
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct DonateRevenueRow {
    pub credits_total: i64,
    pub debits_total: i64,
    pub topups: i64,
    pub debits: i64,
}

/// Linha de extrato (crédito ou débito), já com o login da conta.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateLedgerRow {
    pub id: i64,
    pub account_id: i64,
    pub login: String,
    pub credits: i64,
    pub idempotency_key: String,
    /// ISO-8601 em UTC, como o banco devolve.
    pub created_at: String,
}

/// Comprador com maior crédito acumulado (agregado por conta).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateBuyerRow {
    pub account_id: i64,
    pub login: String,
    pub credits: i64,
    pub topups: i64,
}

/// Conta encontrada pela busca por login (id e papel; nunca segredo).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountLookupRow {
    pub id: i64,
    pub login: String,
    pub role: String,
}

/// Resultado da baixa de saldo pedida pelo staff.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DebitResult {
    Applied(i64),
    Replay(i64),
    Insufficient,
}

/// Recibo do débito (mesma chave, mesmo razão — prova de idempotência).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateDebitReceipt {
    pub idempotency_key: String,
    pub debit_id: i64,
    pub replay: bool,
}

/// Receita de donate para a tela de Economia (staff; leitura auditada).
pub fn revenue_summary(
    session: Option<Session>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<DonateRevenueRow, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.revenue")?;
    let row = store.donate_revenue()?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.revenue",
        "donate",
        "leitura da receita",
    );
    Ok(row)
}

/// Pedidos de recarga (extrato de créditos), mais recentes primeiro.
pub fn topup_orders(
    session: Option<Session>,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DonateLedgerRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.donate.revenue")?;
    validar_limite(limit)?;
    store.donate_credits(limit)
}

/// Débitos de saldo (extrato de gastos), mais recentes primeiro.
pub fn spend_orders(
    session: Option<Session>,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DonateLedgerRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.donate.revenue")?;
    validar_limite(limit)?;
    store.donate_debits(limit)
}

/// Contas que mais compraram créditos.
pub fn top_buyers(
    session: Option<Session>,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DonateBuyerRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.donate.revenue")?;
    if !(1..=50).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de compradores fora da faixa",
        });
    }
    store.donate_top_buyers(limit)
}

/// Busca conta por prefixo de login, para a tela de débito achar o id.
pub fn search_accounts(
    session: Option<Session>,
    term: &str,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<AccountLookupRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.donate.revenue")?;
    if !(1..=20).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite da busca fora da faixa",
        });
    }
    let termo = term.trim();
    if termo.len() > 40 {
        return Err(AdminError::InvalidInput {
            reason: "busca longa demais",
        });
    }
    store.search_accounts(termo, limit)
}

/// Debita saldo de donate da conta, idempotente por chave.
///
/// Saldo insuficiente é recusa de negócio (nada gravado), nunca débito parcial.
pub fn debit_balance(
    session: Option<Session>,
    account_id: i64,
    credits: i64,
    key: &str,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<DonateDebitReceipt, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.debit")?;
    if account_id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "conta inválida",
        });
    }
    if credits <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "créditos a debitar precisam ser positivos",
        });
    }
    let key = key.trim();
    if key.is_empty() || key.len() > 120 {
        return Err(AdminError::InvalidInput {
            reason: "chave de idempotência obrigatória (até 120 caracteres)",
        });
    }
    match store.debit_donate(account_id, credits, key, "portal", staff.moderator_id())? {
        DebitResult::Insufficient => Err(AdminError::InvalidInput {
            reason: "saldo de donate insuficiente na conta",
        }),
        DebitResult::Applied(debit_id) => {
            crate::audit::log_action(
                sink,
                &staff,
                "config.donate.debit",
                &format!("account:{account_id}"),
                &format!("{credits} créditos debitados (chave {key})"),
            );
            Ok(DonateDebitReceipt {
                idempotency_key: key.to_string(),
                debit_id,
                replay: false,
            })
        }
        DebitResult::Replay(debit_id) => {
            crate::audit::log_action(
                sink,
                &staff,
                "config.donate.debit.replay",
                &format!("account:{account_id}"),
                &format!("{credits} créditos (chave {key} já aplicada)"),
            );
            Ok(DonateDebitReceipt {
                idempotency_key: key.to_string(),
                debit_id,
                replay: true,
            })
        }
    }
}

fn validar_limite(limit: i64) -> Result<(), AdminError> {
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite do extrato fora da faixa",
        });
    }
    Ok(())
}

/// JSON da receita (saldo = créditos − débitos).
#[must_use]
pub fn revenue_json(row: &DonateRevenueRow) -> String {
    format!(
        "{{\"credits_total\":{},\"debits_total\":{},\"balance\":{},\"topups\":{},\"debits\":{}}}",
        row.credits_total,
        row.debits_total,
        row.credits_total - row.debits_total,
        row.topups,
        row.debits
    )
}

/// JSON de uma linha de extrato.
#[must_use]
pub fn ledger_row_json(row: &DonateLedgerRow) -> String {
    format!(
        "{{\"id\":{},\"account_id\":{},\"login\":{},\"credits\":{},\"idempotency_key\":{},\"created_at\":{}}}",
        row.id,
        row.account_id,
        json_escape(&row.login),
        row.credits,
        json_escape(&row.idempotency_key),
        json_escape(&row.created_at)
    )
}

/// Lista JSON de extrato.
#[must_use]
pub fn ledger_json(rows: &[DonateLedgerRow]) -> String {
    let linhas: Vec<String> = rows.iter().map(ledger_row_json).collect();
    format!("[{}]", linhas.join(","))
}

/// Lista JSON de compradores.
#[must_use]
pub fn buyers_json(rows: &[DonateBuyerRow]) -> String {
    let linhas: Vec<String> = rows
        .iter()
        .map(|row| {
            format!(
                "{{\"account_id\":{},\"login\":{},\"credits\":{},\"topups\":{}}}",
                row.account_id,
                json_escape(&row.login),
                row.credits,
                row.topups
            )
        })
        .collect();
    format!("[{}]", linhas.join(","))
}

/// Lista JSON da busca de conta.
#[must_use]
pub fn lookup_json(rows: &[AccountLookupRow]) -> String {
    let linhas: Vec<String> = rows
        .iter()
        .map(|row| {
            format!(
                "{{\"id\":{},\"login\":{},\"role\":{}}}",
                row.id,
                json_escape(&row.login),
                json_escape(&row.role)
            )
        })
        .collect();
    format!("[{}]", linhas.join(","))
}

/// Escapa texto para dentro de string JSON.
fn json_escape(input: &str) -> String {
    let mut saida = String::with_capacity(input.len() + 2);
    saida.push('"');
    for caractere in input.chars() {
        match caractere {
            '"' => saida.push_str("\\\""),
            '\\' => saida.push_str("\\\\"),
            '\n' => saida.push_str("\\n"),
            '\r' => saida.push_str("\\r"),
            '\t' => saida.push_str("\\t"),
            outro if (outro as u32) < 0x20 => {
                saida.push_str(&format!("\\u{:04x}", outro as u32));
            }
            outro => saida.push(outro),
        }
    }
    saida.push('"');
    saida
}

fn validar_edicao(edit: &DonateEdit) -> Result<(), AdminError> {
    if edit.item_index <= 0 || edit.price_credits <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item ou preço inválido",
        });
    }
    if edit.duration_days.is_some_and(|d| d <= 0) {
        return Err(AdminError::InvalidInput {
            reason: "duração inválida",
        });
    }
    Ok(())
}

/// JSON de um item da loja.
#[must_use]
pub fn shop_item_json(row: &DonateItemRow) -> String {
    let duration = row
        .duration_days
        .map_or_else(|| "null".to_string(), |d| d.to_string());
    format!(
        "{{\"id\":{},\"item_index\":{},\"price_credits\":{},\"duration_days\":{duration},\"enabled\":{}}}",
        row.id, row.item_index, row.price_credits, row.enabled
    )
}

/// Lista JSON da loja.
#[must_use]
pub fn shop_items_json(rows: &[DonateItemRow]) -> String {
    let itens: Vec<String> = rows.iter().map(shop_item_json).collect();
    format!("[{}]", itens.join(","))
}
