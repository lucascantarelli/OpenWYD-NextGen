//! Leitura de conteúdo do portal (WEB-05): NPCs, templates de mob e efeitos.
//!
//! Sem banco, o `Router` com `FakeConteudo` prova o contrato: `result:1` com dado,
//! lista vazia sem erro, teto de paginação (`result:3`), 401 sem sessão e `result:2`
//! para jogador. Com `WYD_TEST_DATABASE_URL` (ou `DATABASE_URL`), o `PgBackend` prova
//! as mesmas rotas contra o Postgres semeado (6.100 NPCs, 1.991 templates, `EF_*`).

#[path = "../common/mod.rs"]
mod common;

use std::io::Cursor;

use wyd_admin::transport::{http, json, Router, StaticTokens};
use wyd_admin::{
    AccountSummary, AdminError, ConfigStore, DailyEdit, DailyRewardRow, DonateEdit, DonateItemRow,
    DropEdit, DropRow, EffectRow, EventScheduleRow, MobTemplateRow, NpcRow, RatesSnapshot,
    RouteRow, Session, VecSink, WorldEventRow,
};
use wyd_db::{Account, Character, DbError, Role};

const EFEITOS_FAKE: [(&str, &str, &str); 3] = [
    ("EF_AC", "ac", "score"),
    ("EF_BONUS", "bonus", "score"),
    ("EF_CLASS", "class", "restricao"),
];

struct FakeConteudo;

impl FakeConteudo {
    fn npcs() -> Vec<NpcRow> {
        (1..=3)
            .map(|id| NpcRow {
                id,
                slug: format!("npc-{id}"),
                display_name: format!("NPC {id}"),
                map_id: 0,
                pos_x: 100 * id as i16,
                pos_y: 200 * id as i16,
                enabled: true,
                merchant: 0,
            })
            .collect()
    }

    fn templates() -> Vec<MobTemplateRow> {
        ["GOLEM", "ORC", "URUK"]
            .into_iter()
            .map(|name| MobTemplateRow {
                template_name: name.to_string(),
                name: name.to_string(),
                source_size: 816,
            })
            .collect()
    }

    fn efeitos() -> Vec<EffectRow> {
        EFEITOS_FAKE
            .iter()
            .map(|(name, label, category)| EffectRow {
                name: name.to_string(),
                label: label.to_string(),
                category: category.to_string(),
                description: String::new(),
                confirmed: true,
            })
            .collect()
    }
}

impl ConfigStore for FakeConteudo {
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

    fn npcs(&self, limit: i64, after_id: Option<i32>) -> Result<Vec<NpcRow>, AdminError> {
        let mut rows = Self::npcs();
        rows.retain(|row| after_id.is_none_or(|cursor| row.id > cursor));
        rows.truncate(limit.max(0) as usize);
        Ok(rows)
    }

    fn mob_templates(
        &self,
        limit: i64,
        after: Option<&str>,
    ) -> Result<Vec<MobTemplateRow>, AdminError> {
        let mut rows = Self::templates();
        rows.retain(|row| after.is_none_or(|cursor| row.template_name.as_str() > cursor));
        rows.truncate(limit.max(0) as usize);
        Ok(rows)
    }

    fn effects(&self, limit: i64, after: Option<&str>) -> Result<Vec<EffectRow>, AdminError> {
        let mut rows = Self::efeitos();
        rows.retain(|row| after.is_none_or(|cursor| row.name.as_str() > cursor));
        rows.truncate(limit.max(0) as usize);
        Ok(rows)
    }

    fn routes(&self, _limit: i64) -> Result<Vec<RouteRow>, AdminError> {
        Ok(Vec::new())
    }

    fn daily_rewards(
        &self,
        _limit: i64,
        _after_id: Option<i32>,
    ) -> Result<Vec<DailyRewardRow>, AdminError> {
        Ok(Vec::new())
    }

    fn upsert_daily_reward(&self, _edit: &DailyEdit, _actor: i64) -> Result<i64, AdminError> {
        Ok(14)
    }

    fn delete_daily_reward(&self, _id: i32, _actor: i64) -> Result<i64, AdminError> {
        Ok(15)
    }

    fn donate_items(
        &self,
        _limit: i64,
        _after_id: Option<i32>,
    ) -> Result<Vec<DonateItemRow>, AdminError> {
        Ok(Vec::new())
    }

    fn upsert_donate_item(&self, _edit: &DonateEdit, _actor: i64) -> Result<i64, AdminError> {
        Ok(16)
    }

    fn set_donate_item_enabled(
        &self,
        _id: i32,
        _enabled: bool,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        Ok(17)
    }

    fn delete_donate_item(&self, _id: i32, _actor: i64) -> Result<i64, AdminError> {
        Ok(18)
    }

    fn donate_revenue(&self) -> Result<wyd_admin::DonateRevenueRow, AdminError> {
        Ok(wyd_admin::DonateRevenueRow::default())
    }

    fn donate_credits(&self, _limit: i64) -> Result<Vec<wyd_admin::DonateLedgerRow>, AdminError> {
        Ok(Vec::new())
    }

    fn donate_debits(&self, _limit: i64) -> Result<Vec<wyd_admin::DonateLedgerRow>, AdminError> {
        Ok(Vec::new())
    }

    fn donate_top_buyers(&self, _limit: i64) -> Result<Vec<wyd_admin::DonateBuyerRow>, AdminError> {
        Ok(Vec::new())
    }

    fn debit_donate(
        &self,
        _account_id: i64,
        _credits: i64,
        _key: &str,
        _reason: &str,
        _actor: i64,
    ) -> Result<wyd_admin::DebitResult, AdminError> {
        Ok(wyd_admin::DebitResult::Applied(41))
    }

    fn search_accounts(
        &self,
        _term: &str,
        _limit: i64,
    ) -> Result<Vec<wyd_admin::AccountLookupRow>, AdminError> {
        Ok(Vec::new())
    }

    fn world_events(&self) -> Result<Vec<WorldEventRow>, AdminError> {
        Ok(Vec::new())
    }

    fn event_schedules(&self) -> Result<Vec<EventScheduleRow>, AdminError> {
        Ok(Vec::new())
    }

    fn set_world_event_enabled(
        &self,
        _code: &str,
        _enabled: bool,
        _actor: i64,
    ) -> Result<i64, AdminError> {
        Ok(19)
    }

    fn accounts(
        &self,
        _limit: i64,
        _after_id: Option<i64>,
    ) -> Result<Vec<AccountSummary>, AdminError> {
        Ok(Vec::new())
    }

    fn account(&self, _id: i64) -> Result<Option<AccountSummary>, AdminError> {
        Ok(None)
    }

    fn credit_donate(
        &self,
        _account_id: i64,
        _credits: i64,
        _key: &str,
        _actor: i64,
    ) -> Result<(i64, bool), AdminError> {
        Ok((20, false))
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
        .with_config(std::sync::Arc::new(FakeConteudo))
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

#[test]
fn staff_lista_npcs_com_paginacao_por_id() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/npc_admin/list_npcs",
            Some("staff"),
            r#"{"limit":2}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let npcs = lista(&body, "npcs");
    assert_eq!(npcs.len(), 2);
    assert_eq!(
        npcs[0].get("display_name").and_then(json::Json::as_str),
        Some("NPC 1")
    );

    // Cursor: depois do id 2 só resta o 3.
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/npc_admin/list_npcs",
            Some("staff"),
            r#"{"limit":50,"after_id":2}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let npcs = lista(&body, "npcs");
    assert_eq!(npcs.len(), 1);
    assert_eq!(npcs[0].get("id").and_then(json::Json::as_i64), Some(3));
}

#[test]
fn staff_lista_templates_e_efeitos_com_cursor_por_nome() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/mob_admin/list_mob_templates",
            Some("staff"),
            r#"{"limit":50}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(lista(&body, "mob_templates").len(), 3);

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/mob_admin/list_mob_templates",
            Some("staff"),
            r#"{"limit":50,"after":"GOLEM"}"#,
        ),
    );
    assert_eq!(status, 200);
    let templates = lista(&body, "mob_templates");
    assert_eq!(templates.len(), 2);
    assert!(templates
        .iter()
        .all(|item| item.get("template_name").and_then(json::Json::as_str) != Some("GOLEM")));

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/item_effect_admin/list_effects",
            Some("staff"),
            r#"{"limit":50,"after":"EF_BONUS"}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let efeitos = lista(&body, "effects");
    assert_eq!(efeitos.len(), 1);
    assert_eq!(
        efeitos[0].get("name").and_then(json::Json::as_str),
        Some("EF_CLASS")
    );
}

#[test]
fn lista_vazia_nao_vira_erro() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/item_effect_admin/list_effects",
            Some("staff"),
            r#"{"limit":50,"after":"ZZZ"}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert!(lista(&body, "effects").is_empty());
}

#[test]
fn limite_fora_do_teto_vira_result_3() {
    let router = router();
    for corpo in [r#"{"limit":201}"#, r#"{"limit":0}"#, r#"{"limit":-5}"#] {
        let (status, body) = dispatch(
            &router,
            &raw(
                "POST",
                "/admin/v1/npc_admin/list_npcs",
                Some("staff"),
                corpo,
            ),
        );
        assert_eq!(status, 200, "recusa de negócio viaja no corpo");
        assert_eq!(result_of(&body), 3, "limite fora de 1..=200: {corpo}");
    }
}

#[test]
fn sem_sessao_401_e_jogador_result_2() {
    let router = router();
    let (status, _) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/npc_admin/list_npcs",
            None,
            r#"{"limit":10}"#,
        ),
    );
    assert_eq!(status, 401);

    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/mob_admin/list_mob_templates",
            Some("player"),
            r#"{"limit":10}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 2);
}

fn runtime() -> tokio::runtime::Runtime {
    tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .expect("runtime")
}

fn backend() -> Option<wyd_db::PgBackend> {
    let dsn = common::dsn_ou_skip("leitura real de conteúdo")?;
    let rt = runtime();
    Some(
        rt.block_on(wyd_db::PgBackend::connect(
            &dsn,
            wyd_db::PoolConfig::local(),
        ))
        .expect("conecta no Postgres de teste"),
    )
}

#[test]
fn npcs_reais_vem_do_postgres_com_paginacao() {
    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — leitura real de NPCs não executada");
        return;
    };
    let rt = runtime();
    let pagina: Vec<wyd_db::NpcListRow> = rt
        .block_on(backend.config_npc_page(2, None))
        .expect("página 1");
    assert_eq!(pagina.len(), 2, "seed tem 6.100 NPCs");
    assert!(pagina[0].id > 0);
    assert!(!pagina[0].display_name.is_empty());
    assert!(pagina[1].id > pagina[0].id, "ordem por id");

    let proxima: Vec<wyd_db::NpcListRow> = rt
        .block_on(backend.config_npc_page(2, Some(pagina[1].id)))
        .expect("página 2");
    assert_eq!(proxima.len(), 2);
    assert!(
        proxima.iter().all(|row| row.id > pagina[1].id),
        "cursor avança"
    );
}

#[test]
fn templates_e_efeitos_reais_vem_do_postgres() {
    let Some(backend) = backend() else {
        eprintln!("SKIP: sem DSN — leitura real de templates/efeitos não executada");
        return;
    };
    let rt = runtime();
    let templates: Vec<wyd_db::MobTemplateRow> = rt
        .block_on(backend.config_mob_templates(5, None))
        .expect("templates");
    assert!(!templates.is_empty(), "seed tem 1.991 templates");
    assert!(!templates[0].template_name.is_empty());
    assert!(!templates[0].name.is_empty());

    let efeitos: Vec<wyd_db::EffectRow> = rt
        .block_on(backend.config_effects(5, None))
        .expect("efeitos");
    assert!(!efeitos.is_empty(), "seed tem EF_*");
    assert!(
        efeitos.iter().all(|row| row.name.starts_with("EF_")),
        "chave é o símbolo EF_*"
    );
    let depois: Vec<wyd_db::EffectRow> = rt
        .block_on(backend.config_effects(5, Some("EF_CON")))
        .expect("cursor");
    assert!(
        depois.iter().all(|row| row.name.as_str() > "EF_CON"),
        "cursor avança por nome"
    );
}
