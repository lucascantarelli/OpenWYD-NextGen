//! Integração real do teto de pilha de produção (view `item_stack_cap`, migration `0039`).
//!
//! Não roda no `cargo test` sem banco: a ausência de `WYD_TEST_DSN` **pula** com mensagem
//! explícita (`tests/pg.rs` usa a mesma regra). A evidência de 2026-11-10 foi produzida contra
//! o Postgres do compose (`docker port wyd-db-1` → `127.0.0.1:5432`):
//!
//! ```bash
//! WYD_TEST_DSN="postgres://admin:$SENHA@127.0.0.1:5432/wyd" \
//!   cargo test -p wyd-db --test stack_caps
//! ```
//!
//! O que este teste cobra é o que a estimativa não provava: **item real sem `kind` empilha**.
//! `416 Antídoto`, `419 Resto_de_Oriharucon` e a poeira `412` não têm `stackable` nem `kind` no
//! catálogo bruto; sem a view eles caíam em teto 1.

use wyd_db::{PgBackend, PoolConfig};

fn dsn() -> Option<String> {
    match std::env::var("WYD_TEST_DSN") {
        Ok(v) if !v.trim().is_empty() => Some(v),
        _ => {
            eprintln!("SKIP: WYD_TEST_DSN ausente — view item_stack_cap não consultada");
            None
        }
    }
}

#[tokio::test]
async fn teto_de_producao_vem_do_banco_e_cobre_item_real_sem_kind() {
    let Some(dsn) = dsn() else {
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
    assert!(fontes.contains("legado:ef_amount"));
    assert!(fontes.contains("familia:material"));
    eprintln!("teto: {} linhas, fontes: {fontes:?}", rows.len());
}
