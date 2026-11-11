//! Handlers de operação do portal (FASE-07-6): recompensa diária, loja de donate
//! com crédito idempotente, eventos de mundo, contas e `grant_item` na fila.
//!
//! Sem banco, o `Router` com `FakeOperacao` prova o contrato: `result:1` com dado,
//! `grant_item` enfileira com idempotência (mesma chave, replay), teto de paginação
//! (`result:3`), 401 sem sessão e `result:2` para jogador. Rotas do contrato sem
//! tabela que as sustente (`set_reward_item_enabled`, `donate_revenue_admin/*`)
//! seguem `501` auditado. Com `WYD_TEST_DATABASE_URL` (ou `DATABASE_URL`), o
//! `PgBackend` prova as mesmas operações contra o Postgres real, incluindo o dreno
//! (`claim_item` → slot livre de `Carry`; mochila cheia deixa pendente, nunca duplica).

use std::collections::HashMap;
use std::io::Cursor;
use std::sync::Mutex;

use wyd_admin::transport::{http, json, Router, StaticTokens};
use wyd_admin::{
    AccountSummary, AdminError, ConfigStore, DailyEdit, DailyRewardRow, DonateEdit, DonateItemRow,
    DropEdit, DropRow, EffectRow, EventScheduleRow, MemoryGrantQueue, MobTemplateRow, NpcRow,
    RatesSnapshot, Role, RouteRow, Session, VecSink, WorldEventRow,
};
use wyd_db::{Account, Character, DbError};

struct FakeOperacao {
    creditos: Mutex<HashMap<String, i64>>,
}

impl FakeOperacao {
    fn recompensas() -> Vec<DailyRewardRow> {
        vec![
            DailyRewardRow {
                id: 1,
                day: None,
                item_index: None,
                gold: 5_000,
                qty: 1,
                account_wide: true,
            },
            DailyRewardRow {
                id: 2,
                day: Some(6),
                item_index: Some(3_367),
                gold: 0,
                qty: 2,
                account_wide: true,
            },
        ]
    }

    fn loja() -> Vec<DonateItemRow> {
        vec![
            DonateItemRow {
                id: 1,
                item_index: 3_367,
                price_credits: 100,
                duration_days: None,
                enabled: true,
            },
            DonateItemRow {
                id: 2,
                item_index: 3_368,
                price_credits: 250,
                duration_days: Some(30),
                enabled: false,
            },
        ]
    }

    fn eventos() -> Vec<WorldEventRow> {
        vec![WorldEventRow {
            code: "kefra".to_string(),
            name: "Kefra".to_string(),
            kind: "kefra".to_string(),
            enabled: true,
            requires_city: false,
            description: "sábado".to_string(),
        }]
    }

    fn horarios() -> Vec<EventScheduleRow> {
        vec![EventScheduleRow {
            id: 1,
            event_code: "kefra".to_string(),
            weekday: 6,
            start_time: "20:00:00".to_string(),
            duration_minutes: 60,
            timezone: "America/Sao_Paulo".to_string(),
            every_n_weeks: 1,
            enabled: true,
        }]
    }

    fn contas() -> Vec<AccountSummary> {
        vec![
            AccountSummary {
                id: 11,
                login: "jogador1".to_string(),
                role: Role::Player,
            },
            AccountSummary {
                id: 12,
                login: "mod1".to_string(),
                role: Role::Moderator,
            },
        ]
    }
}

impl ConfigStore for FakeOperacao {
    fn rates_snapshot(&self) -> Result<RatesSnapshot, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn set_rate(
        &self,
        _domain: &str,
        _key: &str,
        _value: i64,
        _source: &str,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn reset_rate(&self, _domain: &str, _key: &str, _actor: i64) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn drops(&self, _template: Option<&str>, _limit: i64) -> Result<Vec<DropRow>, AdminError> {
        Ok(Vec::new())
    }

    fn set_drop(&self, _edit: &DropEdit, _actor: i64) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn reset_drop(
        &self,
        _template: &str,
        _item_index: i32,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn npc(&self, _id: i32) -> Result<Option<NpcRow>, AdminError> {
        Ok(None)
    }

    fn set_npc(
        &self,
        _id: i32,
        _map_id: i16,
        _pos_x: i16,
        _pos_y: i16,
        _enabled: bool,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn npcs(&self, _limit: i64, _after_id: Option<i32>) -> Result<Vec<NpcRow>, AdminError> {
        Ok(Vec::new())
    }

    fn mob_templates(
        &self,
        _limit: i64,
        _after: Option<&str>,
    ) -> Result<Vec<MobTemplateRow>, AdminError> {
        Ok(Vec::new())
    }

    fn effects(&self, _limit: i64, _after: Option<&str>) -> Result<Vec<EffectRow>, AdminError> {
        Ok(Vec::new())
    }

    fn routes(&self, _limit: i64) -> Result<Vec<RouteRow>, AdminError> {
        Ok(Vec::new())
    }

    fn daily_rewards(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<DailyRewardRow>, AdminError> {
        let mut rows = Self::recompensas();
        rows.retain(|row| after_id.is_none_or(|cursor| row.id > cursor));
        rows.truncate(limit.max(0) as usize);
        Ok(rows)
    }

    fn upsert_daily_reward(&self, _edit: &DailyEdit, _actor: i64) -> Result<i64, AdminError> {
        Ok(31)
    }

    fn delete_daily_reward(&self, _id: i32, _actor: i64) -> Result<i64, AdminError> {
        Ok(32)
    }

    fn donate_items(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<DonateItemRow>, AdminError> {
        let mut rows = Self::loja();
        rows.retain(|row| after_id.is_none_or(|cursor| row.id > cursor));
        rows.truncate(limit.max(0) as usize);
        Ok(rows)
    }

    fn upsert_donate_item(&self, _edit: &DonateEdit, _actor: i64) -> Result<i64, AdminError> {
        Ok(33)
    }

    fn set_donate_item_enabled(
        &self,
        _id: i32,
        _enabled: bool,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        Ok(34)
    }

    fn delete_donate_item(&self, _id: i32, _actor: i64) -> Result<i64, AdminError> {
        Ok(35)
    }

    fn donate_revenue(&self) -> Result<wyd_admin::DonateRevenueRow, AdminError> {
        Ok(wyd_admin::DonateRevenueRow {
            credits_total: 1_200,
            debits_total: 200,
            topups: 3,
            debits: 1,
        })
    }

    fn donate_credits(&self, _limit: i64) -> Result<Vec<wyd_admin::DonateLedgerRow>, AdminError> {
        Ok(vec![wyd_admin::DonateLedgerRow {
            id: 9,
            account_id: 42,
            login: "comprador".to_string(),
            credits: 500,
            idempotency_key: "qa-topup".to_string(),
            created_at: "2026-11-11T12:00:00Z".to_string(),
        }])
    }

    fn donate_debits(&self, _limit: i64) -> Result<Vec<wyd_admin::DonateLedgerRow>, AdminError> {
        Ok(Vec::new())
    }

    fn donate_top_buyers(&self, _limit: i64) -> Result<Vec<wyd_admin::DonateBuyerRow>, AdminError> {
        Ok(vec![wyd_admin::DonateBuyerRow {
            account_id: 42,
            login: "comprador".to_string(),
            credits: 500,
            topups: 1,
        }])
    }

    fn debit_donate(
        &self,
        _account_id: i64,
        _credits: i64,
        _key: &str,
        _reason: &str,
        _actor: i64,
    ) -> Result<wyd_admin::DebitResult, AdminError> {
        Ok(wyd_admin::DebitResult::Applied(42))
    }

    fn search_accounts(
        &self,
        _term: &str,
        _limit: i64,
    ) -> Result<Vec<wyd_admin::AccountLookupRow>, AdminError> {
        Ok(vec![wyd_admin::AccountLookupRow {
            id: 42,
            login: "comprador".to_string(),
            role: "player".to_string(),
        }])
    }

    fn world_events(&self) -> Result<Vec<WorldEventRow>, AdminError> {
        Ok(Self::eventos())
    }

    fn event_schedules(&self) -> Result<Vec<EventScheduleRow>, AdminError> {
        Ok(Self::horarios())
    }

    fn set_world_event_enabled(
        &self,
        _code: &str,
        _enabled: bool,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        Ok(36)
    }

    fn accounts(
        &self,
        limit: i64,
        after_id: Option<i64>,
    ) -> Result<Vec<AccountSummary>, AdminError> {
        let mut rows = Self::contas();
        rows.retain(|row| after_id.is_none_or(|cursor| row.id > cursor));
        rows.truncate(limit.max(0) as usize);
        Ok(rows)
    }

    fn account(&self, id: i64) -> Result<Option<AccountSummary>, AdminError> {
        Ok(Self::contas().into_iter().find(|row| row.id == id))
    }

    fn credit_donate(
        &self,
        _account_id: i64,
        _credits: i64,
        key: &str,
        _actor: i64,
    ) -> Result<(i64, bool), AdminError> {
        let mut mapa = self.creditos.lock().expect("creditos");
        if let Some(id) = mapa.get(key) {
            return Ok((*id, true));
        }
        let id = 40 + mapa.len() as i64 + 1;
        mapa.insert(key.to_string(), id);
        Ok((id, false))
    }

    fn set_route(
        &self,
        _id: i32,
        _cost_gold: Option<i32>,
        _req_level: Option<i16>,
        _enabled: Option<bool>,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }

    fn item_prices(
        &self,
        _item_index: Option<i32>,
        _limit: i64,
    ) -> Result<Vec<wyd_admin::ItemPriceRow>, AdminError> {
        Ok(Vec::new())
    }

    fn set_item_price(
        &self,
        _edit: &wyd_admin::ItemPriceEdit,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        unimplemented!("fora do escopo deste teste");
    }
}

struct FakeAccounts;

impl wyd_db::Accounts for FakeAccounts {
    async fn find_by_login(&self, _login: &str) -> Result<Account, DbError> {
        Err(DbError::NotFound {
            entity: "account",
            id: 0,
        })
    }

    async fn find_by_id(&self, id: i64) -> Result<Account, DbError> {
        Err(DbError::NotFound {
            entity: "account",
            id,
        })
    }

    async fn create_account(&self, login: &str, role: Role) -> Result<Account, DbError> {
        Ok(Account {
            id: 99,
            login: login.to_string(),
            role,
        })
    }
}

struct FakeCharacters;

impl wyd_db::Characters for FakeCharacters {
    async fn list_by_account(&self, _account_id: i64) -> Result<Vec<Character>, DbError> {
        Ok(Vec::new())
    }

    async fn top_by_level(&self, _limit: usize) -> Result<Vec<Character>, DbError> {
        Ok(Vec::new())
    }

    async fn search_by_name(&self, _term: &str, _limit: usize) -> Result<Vec<Character>, DbError> {
        Ok(Vec::new())
    }

    async fn create_character(&self, character: &Character) -> Result<Character, DbError> {
        Ok(character.clone())
    }

    async fn soft_delete(&self, _id: i64) -> Result<(), DbError> {
        Ok(())
    }
}

fn tokens() -> StaticTokens {
    let mut tokens = StaticTokens::default();
    tokens.insert(
        "staff",
        Session {
            account_id: 7,
            role: Role::Moderator,
        },
    );
    tokens.insert(
        "player",
        Session {
            account_id: 5,
            role: Role::Player,
        },
    );
    tokens
}

fn router() -> Router<FakeAccounts, FakeCharacters, VecSink> {
    Router::new(FakeAccounts, FakeCharacters, VecSink::default(), tokens())
        .with_config(std::sync::Arc::new(FakeOperacao {
            creditos: Mutex::new(HashMap::new()),
        }))
        .with_grant_queue(std::sync::Arc::new(MemoryGrantQueue::default()))
}

fn raw(method: &str, path: &str, token: Option<&str>, body: &str) -> String {
    let mut request = format!("{method} {path} HTTP/1.1\r\nhost: loopback\r\nx-trace-id: t-1\r\n");
    if let Some(token) = token {
        request.push_str(&format!("authorization: Bearer {token}\r\n"));
    }
    let length = body.len();
    request.push_str(&format!("content-length: {length}\r\n\r\n{body}"));
    request
}

fn dispatch(
    router: &Router<FakeAccounts, FakeCharacters, VecSink>,
    request: &str,
) -> (u16, json::Json) {
    let mut cursor = Cursor::new(request.as_bytes().to_vec());
    let parsed = http::read_request(&mut cursor, 1 << 20).expect("request válida");
    let reply = router.dispatch(&parsed);
    let body = json::Json::parse(&reply.body).expect("corpo é JSON");
    (reply.status, body)
}

fn result_of(body: &json::Json) -> i64 {
    body.get("result")
        .and_then(json::Json::as_i64)
        .expect("result")
}

fn lista(body: &json::Json, chave: &str) -> Vec<json::Json> {
    match body.get(chave).expect("chave da lista") {
        json::Json::Arr(itens) => itens.clone(),
        outro => panic!("{chave} deveria ser array, veio {outro:?}"),
    }
}

fn campo(body: &json::Json, chave: &str) -> json::Json {
    body.get(chave)
        .unwrap_or_else(|| panic!("campo {chave}"))
        .clone()
}

#[test]
fn staff_lista_edita_e_remove_recompensa_diaria() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/daily_reward_admin/list_reward_items",
            Some("staff"),
            r#"{"limit":50}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let recompensas = lista(&body, "rewards");
    assert_eq!(recompensas.len(), 2);
    assert_eq!(
        recompensas[0].get("gold").and_then(json::Json::as_i64),
        Some(5_000)
    );

    // Cursor: depois do id 1 só resta o id 2.
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/daily_reward_admin/list_reward_items",
            Some("staff"),
            r#"{"limit":50,"after_id":1}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(lista(&body, "rewards").len(), 1);

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/daily_reward_admin/upsert_reward_item",
            Some("staff"),
            r#"{"day":6,"item_index":3367,"qty":2}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(
        campo(&body, "revision").as_i64(),
        Some(31),
        "revisão do fake"
    );

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/daily_reward_admin/delete_reward_item",
            Some("staff"),
            r#"{"id":2}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(campo(&body, "revision").as_i64(), Some(32));
}

#[test]
fn loja_lista_edita_credita_com_replay() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/donate_admin/list_shop_items",
            Some("staff"),
            r#"{"limit":50}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let itens = lista(&body, "shop_items");
    assert_eq!(itens.len(), 2);
    assert_eq!(
        itens[1].get("enabled").and_then(json::Json::as_bool),
        Some(false)
    );

    for (rota, corpo, revisao) in [
        (
            "upsert_shop_item",
            r#"{"item_index":3367,"price_credits":100}"#,
            33,
        ),
        ("set_shop_item_enabled", r#"{"id":2,"enabled":true}"#, 34),
        ("delete_shop_item", r#"{"id":2}"#, 35),
    ] {
        let (status, body) = dispatch(
            &router,
            &raw(
                "POST",
                &format!("/admin/v1/donate_admin/{rota}"),
                Some("staff"),
                corpo,
            ),
        );
        assert_eq!(status, 200, "{rota}");
        assert_eq!(result_of(&body), 1, "{rota}");
        assert_eq!(campo(&body, "revision").as_i64(), Some(revisao), "{rota}");
    }

    // Crédito idempotente: mesma chave devolve o mesmo razão com replay.
    let corpo = r#"{"account_id":11,"credits":500,"idempotency_key":"vale-07-6"}"#;
    let (status, primeira) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/donate_admin/credit_donate_balance",
            Some("staff"),
            corpo,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&primeira), 1);
    assert_eq!(
        campo(&primeira, "replay").as_bool(),
        Some(false),
        "primeira vez não é replay"
    );
    let (status, segunda) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/donate_admin/credit_donate_balance",
            Some("staff"),
            corpo,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&segunda), 1);
    assert_eq!(campo(&segunda, "replay").as_bool(), Some(true));
    assert_eq!(
        campo(&primeira, "credit_id"),
        campo(&segunda, "credit_id"),
        "mesma chave, mesmo razão"
    );
}

#[test]
fn eventos_foto_com_horarios_e_liga_desliga() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/world_event_admin/get_world_event_config",
            Some("staff"),
            r#"{}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let eventos = lista(&body, "events");
    assert_eq!(eventos.len(), 1);
    assert_eq!(
        eventos[0].get("code").and_then(json::Json::as_str),
        Some("kefra")
    );
    let horarios = match eventos[0].get("schedules").expect("schedules") {
        json::Json::Arr(itens) => itens.clone(),
        outro => panic!("schedules deveria ser array, veio {outro:?}"),
    };
    assert_eq!(horarios.len(), 1);
    assert_eq!(
        horarios[0].get("timezone").and_then(json::Json::as_str),
        Some("America/Sao_Paulo")
    );

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/world_event_admin/set_world_event_config",
            Some("staff"),
            r#"{"code":"kefra","enabled":false}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(campo(&body, "revision").as_i64(), Some(36));
}

#[test]
fn contas_listam_sem_segredo() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/account_admin/list_accounts",
            Some("staff"),
            r#"{"limit":50}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let contas = lista(&body, "accounts");
    assert_eq!(contas.len(), 2);
    assert_eq!(
        contas[0].get("login").and_then(json::Json::as_str),
        Some("jogador1")
    );

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/account_admin/get_account",
            Some("staff"),
            r#"{"id":12}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let conta = campo(&body, "account");
    assert_eq!(
        conta.get("role").and_then(json::Json::as_str),
        Some("moderator")
    );

    for corpo in [&body, &conta] {
        let texto = format!("{corpo:?}");
        assert!(
            !texto.contains("password") && !texto.contains("pin"),
            "sem segredo no JSON: {texto}"
        );
    }
}

#[test]
fn grant_item_enfileira_com_idempotencia() {
    let router = router();
    let corpo = r#"{"idempotency_key":"grant-07-6","character_id":77,"item_id":3367,"qty":3}"#;
    let (status, primeira) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/character_admin/grant_item",
            Some("staff"),
            corpo,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&primeira), 1);
    assert_eq!(campo(&primeira, "replay").as_bool(), Some(false));

    let (status, segunda) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/character_admin/grant_item",
            Some("staff"),
            corpo,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&segunda), 1);
    assert_eq!(campo(&segunda, "replay").as_bool(), Some(true));
    assert_eq!(
        campo(&primeira, "grant_id"),
        campo(&segunda, "grant_id"),
        "mesma chave, mesmo grant"
    );
}

#[test]
fn operacao_recusa_sem_sessao_jogador_e_teto() {
    let router = router();
    let (status, _) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/daily_reward_admin/list_reward_items",
            None,
            r#"{"limit":10}"#,
        ),
    );
    assert_eq!(status, 401);

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/donate_admin/list_shop_items",
            Some("player"),
            r#"{"limit":10}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 2);

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/world_event_admin/get_world_event_config",
            Some("staff"),
            r#"{}"#,
        ),
    );
    assert_eq!(status, 200, "leitura de evento não tem teto de limite");
    assert_eq!(result_of(&body), 1);

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/account_admin/list_accounts",
            Some("staff"),
            r#"{"limit":201}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 3, "limite acima do teto");
}

#[test]
fn sem_tabela_sem_handler_501() {
    let router = router();
    // `daily_reward_item` não tem coluna de liga/desliga (série proíbe ALTER no up).
    let (status, _) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/daily_reward_admin/set_reward_item_enabled",
            Some("staff"),
            r#"{"id":1,"enabled":false}"#,
        ),
    );
    assert_eq!(status, 501);

    // Receita de donate tem razão no banco (0042 crédito, 0045 débito): as rotas
    // respondem 200 com os números do store — nenhuma delas pode sobrar em 501.
    for rota in [
        "get_revenue_summary",
        "list_topup_orders",
        "list_top_buyers",
        "list_donate_spend",
    ] {
        let (status, body) = dispatch(
            &router,
            &raw(
                "POST",
                &format!("/admin/v1/donate_revenue_admin/{rota}"),
                Some("staff"),
                r#"{"limit":10}"#,
            ),
        );
        assert_eq!(status, 200, "{rota}");
        assert_eq!(result_of(&body), 1, "{rota}");
    }

    // Busca de conta e débito de saldo também têm handler (antes: 501).
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/donate_revenue_admin/search_accounts",
            Some("staff"),
            r#"{"login":"comprador","limit":5}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert!(
        body.render().contains("comprador"),
        "a busca devolve o login"
    );

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/donate_admin/debit_donate_balance",
            Some("staff"),
            r#"{"account_id":42,"credits":100,"idempotency_key":"qa-debit-1"}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert!(body.render().contains("\"debit_id\":42"));
}

/// DSN dos testes com Postgres (mesma convenção dos demais testes do repositório).
fn dsn() -> Option<String> {
    ["WYD_TEST_DATABASE_URL", "DATABASE_URL", "WYD_TEST_DSN"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty())
}

fn runtime() -> tokio::runtime::Runtime {
    tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .expect("runtime")
}

fn backend() -> Option<wyd_db::PgBackend> {
    let dsn = dsn()?;
    let rt = runtime();
    Some(
        rt.block_on(wyd_db::PgBackend::connect(
            &dsn,
            wyd_db::PoolConfig::local(),
        ))
        .expect("conecta no Postgres de teste"),
    )
}

fn conta_unica() -> String {
    format!(
        "op_07_6_{}_{}",
        std::process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .expect("relógio")
            .as_nanos()
    )
}

#[test]
fn daily_e_loja_reais_no_postgres() {
    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — operação real diária/donate não executada");
        return;
    };
    let rt = runtime();

    // Recompensa só de ouro (sem item) evita FK com o catálogo. O upsert devolve a
    // revisão (`config_revision`), não o id da linha: a linha é achada pelo ouro.
    let rev = rt
        .block_on(backend.config_upsert_daily_reward(
            &wyd_db::DailyRewardEdit {
                id: None,
                day: None,
                item_index: None,
                gold: 1_000,
                qty: 1,
                account_wide: true,
            },
            7,
        ))
        .expect("upsert recompensa");
    assert!(rev > 0);
    let pagina: Vec<wyd_db::DailyRewardRow> = rt
        .block_on(backend.config_daily_rewards(200, None))
        .expect("lista recompensas");
    let achada = pagina
        .iter()
        .find(|row| row.gold == 1_000)
        .expect("recompensa criada aparece na lista");
    assert!(achada.account_wide);
    // O delete devolve a revisão, não o id: a prova é a linha sumir da lista.
    let baixa = rt
        .block_on(backend.config_delete_daily_reward(achada.id, 7))
        .expect("remove recompensa");
    assert!(baixa > 0);
    let depois: Vec<wyd_db::DailyRewardRow> = rt
        .block_on(backend.config_daily_rewards(200, None))
        .expect("lista após delete");
    assert!(
        depois.iter().all(|row| row.id != achada.id),
        "linha removida não lista mais"
    );

    // Loja precisa de um item real do catálogo semeado.
    let item: i32 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar("SELECT item_index FROM item_catalog ORDER BY item_index LIMIT 1")
                .fetch_one(pool)
                .await
        })
        .expect("catálogo semeado");
    // O upsert devolve a revisão, não o id: a linha é achada pelo preço único.
    let rev = rt
        .block_on(backend.config_upsert_donate_item(None, item, 150, None, true, 7))
        .expect("upsert loja");
    assert!(rev > 0);
    let itens: Vec<wyd_db::DonateItemRow> = rt
        .block_on(backend.config_donate_items(200, None))
        .expect("lista loja");
    let achado = itens
        .iter()
        .find(|row| row.price_credits == 150)
        .expect("item criado aparece na lista");
    let loja = achado.id;
    let rev = rt
        .block_on(backend.config_set_donate_item_enabled(loja, false, 7))
        .expect("desliga item");
    assert!(rev > 0);
    let itens: Vec<wyd_db::DonateItemRow> = rt
        .block_on(backend.config_donate_items(200, None))
        .expect("lista após toggle");
    assert!(
        itens
            .iter()
            .find(|row| row.id == loja)
            .is_some_and(|row| !row.enabled),
        "toggle persiste"
    );
    let baixa = rt
        .block_on(backend.config_delete_donate_item(loja, 7))
        .expect("remove item");
    assert!(baixa > 0);
    let depois: Vec<wyd_db::DonateItemRow> = rt
        .block_on(backend.config_donate_items(200, None))
        .expect("lista após delete");
    assert!(
        depois.iter().all(|row| row.id != loja),
        "item removido não lista mais"
    );
}

#[test]
fn credito_donate_idempotente_no_postgres() {
    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — crédito real de donate não executado");
        return;
    };
    let rt = runtime();
    // `donate_credit.created_by` referencia `accounts`: o staff 7 precisa existir.
    rt.block_on(async {
        let pool = backend.pool();
        sqlx::query(
            "INSERT INTO accounts (id, login, password_hash, role) VALUES (7, 'staff_07_6', 'hash-teste', 'moderator') \
             ON CONFLICT (id) DO NOTHING",
        )
        .execute(pool)
        .await
        .expect("staff 7");
    });
    let login = conta_unica();
    let conta: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar(
                "INSERT INTO accounts (login, password_hash) VALUES ($1, 'hash-teste') RETURNING id",
            )
            .bind(&login)
            .fetch_one(pool)
            .await
        })
        .expect("conta de teste");
    let chave = format!("op-07-6-{conta}");

    let (primeiro, replay) = rt
        .block_on(backend.config_credit_donate(conta, 500, &chave, 7))
        .expect("primeiro crédito");
    assert!(!replay);
    let (segundo, replay) = rt
        .block_on(backend.config_credit_donate(conta, 500, &chave, 7))
        .expect("mesma chave");
    assert!(replay, "segunda vez é replay");
    assert_eq!(primeiro, segundo, "mesma chave, mesmo razão");

    let saldo: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar("SELECT coin FROM account_wallet WHERE account_id = $1")
                .bind(conta)
                .fetch_one(pool)
                .await
        })
        .expect("saldo");
    assert_eq!(saldo, 500, "creditou uma vez só");

    rt.block_on(async {
        let pool = backend.pool();
        sqlx::query("DELETE FROM donate_credit WHERE idempotency_key = $1")
            .bind(&chave)
            .execute(pool)
            .await
            .expect("limpa razão");
        sqlx::query("DELETE FROM account_wallet WHERE account_id = $1")
            .bind(conta)
            .execute(pool)
            .await
            .expect("limpa carteira");
        sqlx::query("DELETE FROM accounts WHERE id = $1")
            .bind(conta)
            .execute(pool)
            .await
            .expect("limpa conta");
    });
}

#[test]
fn eventos_reais_vem_do_postgres_com_horarios() {
    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — eventos reais não executados");
        return;
    };
    let rt = runtime();
    let eventos: Vec<wyd_db::WorldEventRow> =
        rt.block_on(backend.config_world_events()).expect("eventos");
    assert!(!eventos.is_empty(), "seed de eventos existe");
    let horarios: Vec<wyd_db::EventScheduleRow> = rt
        .block_on(backend.config_event_schedules())
        .expect("horários");
    assert!(
        horarios
            .iter()
            .all(|h| eventos.iter().any(|e| e.code == h.event_code)),
        "todo horário pertence a um evento"
    );
    assert!(
        horarios.iter().all(|h| !h.timezone.is_empty()),
        "timezone explícita por linha"
    );
    let primeiro = &eventos[0];
    let rev = rt
        .block_on(backend.config_set_world_event_enabled(&primeiro.code, !primeiro.enabled, 7))
        .expect("toggle evento");
    assert!(rev > 0);
    // Restaura o estado original para não vazar mutação entre suítes.
    rt.block_on(backend.config_set_world_event_enabled(&primeiro.code, primeiro.enabled, 7))
        .expect("restaura evento");
}

#[test]
fn contas_reais_listam_sem_segredo() {
    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — contas reais não executadas");
        return;
    };
    let rt = runtime();
    // Banco descartável começa vazio: cria a conta e prova que ela lista sem segredo.
    let login = conta_unica();
    let conta: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar(
                "INSERT INTO accounts (login, password_hash, role) VALUES ($1, 'hash-teste', 'moderator') RETURNING id",
            )
            .bind(&login)
            .fetch_one(pool)
            .await
        })
        .expect("conta de teste");
    let contas: Vec<wyd_db::AccountListRow> = rt
        .block_on(backend.config_accounts(200, None))
        .expect("contas");
    let achada = contas
        .iter()
        .find(|row| row.id == conta)
        .expect("conta criada aparece na lista");
    assert_eq!(achada.login, login);
    assert_eq!(achada.role, "moderator");
    let uma = rt
        .block_on(backend.config_account(conta))
        .expect("uma conta")
        .expect("conta existe");
    assert_eq!(uma.id, conta);
    assert_eq!(uma.login, login);
    assert!(["player", "moderator", "admin"].contains(&uma.role.as_str()));
    rt.block_on(async {
        let pool = backend.pool();
        sqlx::query("DELETE FROM accounts WHERE id = $1")
            .bind(conta)
            .execute(pool)
            .await
            .expect("limpa conta");
    });
}

#[test]
fn claim_item_entrega_na_carry_e_mochila_cheia_pende() {
    use wyd_db::Grants;

    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — dreno real não executado");
        return;
    };
    let rt = runtime();
    let login = conta_unica();
    let conta: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar(
                "INSERT INTO accounts (login, password_hash) VALUES ($1, 'hash-teste') RETURNING id",
            )
            .bind(&login)
            .fetch_one(pool)
            .await
        })
        .expect("conta de teste");
    let personagem: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar(
                "INSERT INTO characters (account_id, slot, name) VALUES ($1, 0, $2) RETURNING id",
            )
            .bind(conta)
            .bind(format!("opchar_{conta}"))
            .fetch_one(pool)
            .await
        })
        .expect("personagem de teste");

    // Linha pendente de item na fila → dreno entrega no primeiro slot de Carry (16).
    let grant: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar(
                "INSERT INTO delivery_queue (kind, character_id, item_id, qty, status, idempotency_key) \
                 VALUES ('item', $1, 3367, 3, 'pending', $2) RETURNING id",
            )
            .bind(personagem)
            .bind(format!("op-grant-{personagem}"))
            .fetch_one(pool)
            .await
        })
        .expect("grant de teste");
    let entregue = rt.block_on(backend.claim_item(grant)).expect("dreno");
    assert_eq!(
        entregue,
        Some((personagem, 16)),
        "primeiro slot livre de Carry"
    );
    let mesma = rt.block_on(backend.claim_item(grant)).expect("repete");
    assert_eq!(mesma, None, "entregue não entrega de novo");

    // Mochila cheia (16–79 ocupados): fica pendente, soma tentativa, não perde.
    rt.block_on(async {
        let pool = backend.pool();
        for slot in 16..=79 {
            sqlx::query(
                "INSERT INTO character_items (character_id, slot, item_id, qty) \
                 VALUES ($1, $2, 3367, 1) ON CONFLICT (character_id, slot) DO NOTHING",
            )
            .bind(personagem)
            .bind(slot)
            .execute(pool)
            .await
            .expect("enche mochila");
        }
    });
    let grant_cheio: i64 = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_scalar(
                "INSERT INTO delivery_queue (kind, character_id, item_id, qty, status, idempotency_key) \
                 VALUES ('item', $1, 3367, 1, 'pending', $2) RETURNING id",
            )
            .bind(personagem)
            .bind(format!("op-grant-cheio-{personagem}"))
            .fetch_one(pool)
            .await
        })
        .expect("grant com mochila cheia");
    let pendente = rt
        .block_on(backend.claim_item(grant_cheio))
        .expect("dreno cheio");
    assert_eq!(pendente, None, "mochila cheia não entrega");
    let estado: (String, i32) = rt
        .block_on(async {
            let pool = backend.pool();
            sqlx::query_as("SELECT status, attempts FROM delivery_queue WHERE id = $1")
                .bind(grant_cheio)
                .fetch_one(pool)
                .await
        })
        .expect("estado da linha");
    assert_eq!(estado.0, "pending", "linha continua pendente");
    assert!(estado.1 >= 1, "tentativa registrada");

    rt.block_on(async {
        let pool = backend.pool();
        sqlx::query("DELETE FROM delivery_queue WHERE character_id = $1")
            .bind(personagem)
            .execute(pool)
            .await
            .expect("limpa fila");
        sqlx::query("DELETE FROM character_items WHERE character_id = $1")
            .bind(personagem)
            .execute(pool)
            .await
            .expect("limpa mochila");
        sqlx::query("DELETE FROM characters WHERE id = $1")
            .bind(personagem)
            .execute(pool)
            .await
            .expect("limpa personagem");
        sqlx::query("DELETE FROM accounts WHERE id = $1")
            .bind(conta)
            .execute(pool)
            .await
            .expect("limpa conta");
    });
}
