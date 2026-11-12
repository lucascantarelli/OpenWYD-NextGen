//! Integração real do teto de pilha de produção (view `item_stack_cap`, migration `0039`).
//!
//! Não roda no `cargo test` sem banco: a ausência de `WYD_TEST_DSN` **pula** com mensagem
//! explícita (`tests/pg.rs` usa a mesma regra). A evidência de 2026-11-10 foi produzida contra
//! o Postgres do compose (`docker port wyd-db-1` → `127.0.0.1:5432`):
//!
//! ```bash
//! WYD_TEST_DSN="postgres://admin:$SENHA@127.0.0.1:5432/wyd" \
//!   cargo test -p wyd-db --test sistema_stack_caps
//! ```
//!
//! O que este teste cobra é o que a estimativa não provava: **item real sem `kind` empilha**.
//! `416 Antídoto`, `419 Resto_de_Oriharucon` e a poeira `412` não têm `stackable` nem `kind` no
//! catálogo bruto; sem a view eles caíam em teto 1.

#[path = "../common/mod.rs"]
mod common;

use sqlx::Row;
use wyd_db::{PgBackend, PoolConfig};

#[tokio::test]
async fn teto_de_producao_vem_do_banco_e_cobre_item_real_sem_kind() {
    let Some(dsn) = common::dsn_ou_skip("view item_stack_cap") else {
        return;
    };
    let backend = PgBackend::connect(&dsn, PoolConfig::local())
        .await
        .expect("conecta no Postgres");
    let rows = backend.item_stack_caps().await.expect("lê a view");
    assert!(
        rows.len() > 3000,
        "a view cobre o catálogo inteiro, não só os classificados: {}",
        rows.len()
    );

    let row = |index: u32| rows.iter().find(|r| r.item_index == index);
    let cap = |index: u32| row(index).map(|r| r.cap);

    // Aglutinável declarado no catálogo.
    assert_eq!(cap(400), Some(120), "poção de cura empilha");
    assert_eq!(cap(412), Some(120), "poeira de Oriharucon empilha");
    assert_eq!(cap(413), Some(120), "poeira de Lactolerium empilha");

    // Item real **sem `kind`** — o caso que a regra antiga rebaixava a teto 1.
    assert_eq!(cap(419), Some(120), "Resto_de_Oriharucon sem kind empilha");
    assert_eq!(cap(420), Some(120), "Resto_de_Lactolerium sem kind empilha");
    assert_eq!(cap(416), Some(120), "Antídoto (pack de 5) empilha");
    assert_eq!(cap(3373), Some(120), "Ração de Cavalo(P) empilha");

    // Não aglutinável: equipamento e placeholder.
    assert_eq!(cap(831), Some(1), "arma não empilha");
    assert_eq!(cap(2433), Some(1), "placeholder N/A não empilha");

    // Conteúdo de pack é separado do teto.
    assert_eq!(row(429).and_then(|r| r.contained), Some(10), "kit de 10");
    assert_eq!(row(416).and_then(|r| r.contained), Some(5), "antídoto de 5");
    assert_eq!(row(400).and_then(|r| r.contained), None, "poção não é pack");

    // Toda linha tem teto não-nulo e nenhum valor inventado.
    assert!(rows.iter().all(|r| r.cap == 1 || r.cap == 120));
    assert!(rows.iter().all(|r| !r.source.is_empty()));
    let fontes: std::collections::BTreeSet<&str> = rows.iter().map(|r| r.source.as_str()).collect();
    assert!(fontes.contains("catalogo:aglutinavel"));
    assert!(
        !fontes.contains("legado:ef_amount") && !fontes.contains("familia:material"),
        "o backfill tornou o teto explícito no catálogo; a fonte antiga não deve sobrar: {fontes:?}"
    );

    // GAME-12/GAME-15: nenhum `stackable` nulo, nenhum true com teto 1, nenhum equipamento > 1.
    let nulos: i64 =
        sqlx::query_scalar("SELECT count(*) FROM item_catalog WHERE stackable IS NULL")
            .fetch_one(backend.pool())
            .await
            .expect("conta stackable nulo");
    assert_eq!(nulos, 0, "classificação de aglutinação fechada");
    let invalidos: i64 = sqlx::query_scalar(
        "SELECT count(*) FROM item_catalog c JOIN item_stack_cap s USING (item_index) \
         WHERE c.stackable AND s.cap <> 120",
    )
    .fetch_one(backend.pool())
    .await
    .expect("conta stackable=true com teto 1");
    assert_eq!(invalidos, 0, "stackable=true sempre tem teto 120");
    let equipamentos: i64 = sqlx::query_scalar(
        "SELECT count(*) FROM item_catalog c JOIN item_stack_cap s USING (item_index) \
         WHERE c.kind IN ('weapon','shield','armor','helmet','boots','gloves','cloak','accessory') \
           AND s.cap > 1",
    )
    .fetch_one(backend.pool())
    .await
    .expect("conta equipamento empilhável");
    assert_eq!(equipamentos, 0, "equipamento força teto 1");

    // Os seis divergentes da GAME-15, com nome/categoria/balde conferidos no banco.
    let divergentes = [
        (406, "Poção_de_Mana_M.", Some("consumable"), true, "real"),
        (416, "Antídoto", Some("consumable"), true, "real"),
        (419, "Resto_de_Oriharucon", Some("material"), true, "real"),
        (420, "Resto_de_Lactolerium", Some("material"), true, "real"),
        (3385, "not_used", None, false, "placeholder"),
        (2433, "N/A", None, false, "placeholder"),
        (2434, "N/A", None, false, "placeholder"),
        (2435, "N/A", None, false, "placeholder"),
        (4045, "NONAME", None, false, "placeholder"),
    ];
    for (indice, nome, kind, stackable, bucket) in divergentes {
        let linha = sqlx::query(
            "SELECT c.name, c.kind, c.stackable, b.bucket \
               FROM item_catalog c JOIN item_bucket b USING (item_index) \
              WHERE c.item_index = $1",
        )
        .bind(indice)
        .fetch_one(backend.pool())
        .await
        .unwrap_or_else(|erro| panic!("item {indice} no catálogo: {erro}"));
        assert_eq!(
            linha.get::<String, _>("name"),
            nome,
            "nome do item {indice}"
        );
        assert_eq!(
            linha.get::<Option<String>, _>("kind").as_deref(),
            kind,
            "kind do item {indice}"
        );
        assert_eq!(
            linha.get::<bool, _>("stackable"),
            stackable,
            "stackable do item {indice}"
        );
        assert_eq!(
            linha.get::<String, _>("bucket"),
            bucket,
            "balde do item {indice}"
        );
    }

    eprintln!("teto: {} linhas, fontes: {fontes:?}", rows.len());
}
