//! Presença por canal: `live_sessions` filtra **grupo e canal**, nunca o grupo inteiro.
//!
//! Este é o ponto em que um vazamento de presença entre canais apareceria primeiro: uma
//! consulta de "online" sem `channel` no `WHERE` faria o canal 2 contar jogadores do canal 1.
//! Roda contra Postgres de verdade (sem `WYD_TEST_DSN` **pula** com mensagem explícita) e
//! compara **deltas** sobre a linha de base: a stack pode ter sessão viva de outro teste.
//!
//! ```bash
//! WYD_TEST_DSN=postgres://… cargo test -p wyd-db --test sessions
//! ```

#[path = "../common/mod.rs"]
mod common;

use wyd_db::{ClaimOutcome, PgBackend, PoolConfig, Sessions};

fn conta(prefixo: &str) -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let nome = format!("{prefixo}{}", nanos % 100_000_000_000_000);
    nome.chars().take(16).collect()
}

async fn backend() -> Option<PgBackend> {
    let dsn = common::dsn_ou_skip("presença por canal")?;
    Some(
        PgBackend::connect(&dsn, PoolConfig::local())
            .await
            .expect("conecta no Postgres de teste"),
    )
}

#[tokio::test]
async fn presenca_de_um_canal_nao_aparece_no_outro() {
    let Some(backend) = backend().await else {
        return;
    };

    // Linha de base: outros testes/containers podem ter sessão viva no mesmo banco.
    let base_1_1 = backend.live_sessions(1, 1).await.expect("presença canal 1");
    let base_1_2 = backend.live_sessions(1, 2).await.expect("presença canal 2");
    let base_2_1 = backend.live_sessions(2, 1).await.expect("presença grupo 2");

    let a = conta("pa");
    let b = conta("pb");

    // 1) Sessão no canal 1: aparece no canal 1, **não** no canal 2 nem em outro grupo.
    let ClaimOutcome::Acquired { id: id_a } = backend
        .claim_session(&a, 1, 1, 12_000, 180)
        .await
        .expect("claim no canal 1")
    else {
        panic!("canal 1 deveria abrir sessão para a conta de teste");
    };
    assert_eq!(
        backend.live_sessions(1, 1).await.expect("canal 1"),
        base_1_1 + 1,
        "canal 1 conta a própria sessão"
    );
    assert_eq!(
        backend.live_sessions(1, 2).await.expect("canal 2"),
        base_1_2,
        "canal 2 NÃO enxerga a presença do canal 1"
    );
    assert_eq!(
        backend.live_sessions(2, 1).await.expect("grupo 2"),
        base_2_1,
        "outro grupo NÃO enxerga a presença desta sessão"
    );

    // 2) Sessão no canal 2: as duas presenças continuam separadas.
    let ClaimOutcome::Acquired { id: id_b } = backend
        .claim_session(&b, 1, 2, 12_000, 180)
        .await
        .expect("claim no canal 2")
    else {
        panic!("canal 2 deveria abrir sessão para a conta de teste");
    };
    assert_eq!(
        backend.live_sessions(1, 2).await.expect("canal 2"),
        base_1_2 + 1
    );
    assert_eq!(
        backend.live_sessions(1, 1).await.expect("canal 1"),
        base_1_1 + 1,
        "presença do canal 1 não muda com sessão no canal 2"
    );

    // 3) Mesma conta em outro canal: conflito tipado, com o canal de origem.
    match backend.claim_session(&a, 1, 2, 12_000, 180).await {
        Ok(ClaimOutcome::Conflict { channel }) => assert_eq!(channel, 1),
        outro => panic!("esperava Conflict{{channel: 1}}, veio {outro:?}"),
    }

    // 4) Liberar devolve a presença do canal ao valor de base (sem contar o outro canal).
    backend.release_session(id_a).await.expect("release a");
    backend.release_session(id_b).await.expect("release b");
    assert_eq!(
        backend.live_sessions(1, 1).await.expect("canal 1"),
        base_1_1
    );
    assert_eq!(
        backend.live_sessions(1, 2).await.expect("canal 2"),
        base_1_2
    );
}
