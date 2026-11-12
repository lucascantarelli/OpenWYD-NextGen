//! Lotação e manutenção do canal (QA-06): a cerca conta **sessões vivas** no
//! `character_session` (`released_at IS NULL`), não conexões TCP.
//!
//! Sem `WYD_TEST_DATABASE_URL`/`DATABASE_URL` os testes **pulam** (o CI sem stack imprime
//! `skip:`); com a URL, cada caso prova o fluxo contra o Postgres de desenvolvimento.
//! O par grupo/canal `7/7` não tem seed: os testes usam sobrescrita por env (`overrides`)
//! e nunca alteram dado de outro canal.

#[path = "../common/mod.rs"]
mod common;

use std::sync::Arc;
use std::time::Duration;

use wyd_db::{Accounts, PgBackend, PoolConfig, Role, Sessions};
use wyd_net::metrics::NetMetrics;
use wyd_server::channels::policy_overrides_from_env;
use wyd_server::session::{PolicyOverrides, SessionError, SessionGate};

fn runtime() -> Arc<tokio::runtime::Runtime> {
    Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(1)
            .enable_all()
            .build()
            .expect("runtime"),
    )
}

/// Aplica a série no banco de desenvolvimento (idempotente, com ledger): o teste que lê o
/// seed da `0040` não depende de boot prévio do canal.
fn migrar(dsn: &str, rt: &Arc<tokio::runtime::Runtime>) {
    let dir = std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("..")
        .join("wyd-db")
        .join("migrations");
    rt.block_on(wyd_db::migrate::run_dir(dsn, &dir, false))
        .expect("migrations aplicadas no banco de desenvolvimento");
}

fn backend(dsn: &str, rt: &Arc<tokio::runtime::Runtime>) -> Arc<PgBackend> {
    Arc::new(
        rt.block_on(PgBackend::connect(dsn, PoolConfig::local()))
            .expect("conecta no Postgres de desenvolvimento"),
    )
}

/// Conta única por execução: a cerca é por login.
static SEQ: std::sync::atomic::AtomicU64 = std::sync::atomic::AtomicU64::new(0);

fn conta_unica() -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let seq = SEQ.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
    format!("lt{:011}{:02}", nanos % 100_000_000_000, seq % 100)
}

fn nova_conta(rt: &tokio::runtime::Runtime, backend: &PgBackend, conta: &str) {
    rt.block_on(backend.create_account(conta, Role::Player))
        .expect("conta criada");
}

fn cerca(
    backend: &Arc<PgBackend>,
    rt: &Arc<tokio::runtime::Runtime>,
    overrides: PolicyOverrides,
) -> SessionGate {
    SessionGate::new(
        Arc::clone(backend),
        Arc::clone(rt),
        7,
        7,
        Duration::from_secs(120),
        Duration::from_secs(180),
    )
    .with_overrides(overrides)
}

/// O seed da migration `0040` publica o teto do legado (`MAX_USER = 1000`) sem manutenção.
#[test]
fn capacidade_do_banco_vem_do_seed() {
    let Some(_dsn) = common::dsn_ou_skip("lotação do canal") else {
        return;
    };
    let dsn = common::dsn_ou_skip("lotação do canal").expect("dsn");
    let rt = runtime();
    migrar(&dsn, &rt);
    let backend = backend(&dsn, &rt);
    let linha = rt
        .block_on(backend.channel_capacity(1, 1))
        .expect("lê channel_capacity");
    assert_eq!(linha.max_players, 1000);
    assert!(!linha.maintenance);
}

/// `max_players = 1`: a segunda sessão é recusada e a vaga volta ao sair (drop = release).
#[test]
fn teto_um_recusa_segunda_sessao_e_libera_ao_sair() {
    let Some(_dsn) = common::dsn_ou_skip("lotação do canal") else {
        return;
    };
    let dsn = common::dsn_ou_skip("lotação do canal").expect("dsn");
    let rt = runtime();
    migrar(&dsn, &rt);
    let backend = backend(&dsn, &rt);
    let gate = cerca(
        &backend,
        &rt,
        PolicyOverrides {
            max_players: Some(1),
            maintenance: None,
        },
    );
    let conta_a = conta_unica();
    let conta_b = conta_unica();
    nova_conta(&rt, &backend, &conta_a);
    nova_conta(&rt, &backend, &conta_b);

    let sessao_a = gate.claim(&conta_a, 12_000).expect("primeira sessão abre");
    assert!(
        matches!(
            gate.claim(&conta_b, 12_000),
            Err(SessionError::Capacity { limit: 1 })
        ),
        "canal com max_players=1 recusa a segunda sessão"
    );
    drop(sessao_a);
    gate.claim(&conta_b, 12_000)
        .expect("vaga liberada ao sair: a segunda conta entra");
}

/// Manutenção recusa entrada nova sem derrubar a sessão que já está dentro.
#[test]
fn manutencao_recusa_entrada_nova_e_preserva_viva() {
    let Some(_dsn) = common::dsn_ou_skip("lotação do canal") else {
        return;
    };
    let dsn = common::dsn_ou_skip("lotação do canal").expect("dsn");
    let rt = runtime();
    migrar(&dsn, &rt);
    let backend = backend(&dsn, &rt);
    let aberta = cerca(&backend, &rt, PolicyOverrides::default());
    let conta_dentro = conta_unica();
    let conta_fora = conta_unica();
    nova_conta(&rt, &backend, &conta_dentro);
    nova_conta(&rt, &backend, &conta_fora);

    let _sessao_dentro = aberta
        .claim(&conta_dentro, 12_000)
        .expect("sessão abre fora de manutenção");
    let em_manutencao = cerca(
        &backend,
        &rt,
        PolicyOverrides {
            max_players: None,
            maintenance: Some(true),
        },
    );
    assert!(
        matches!(
            em_manutencao.claim(&conta_fora, 12_000),
            Err(SessionError::Maintenance)
        ),
        "entrada nova é recusada em manutenção"
    );
    let vivas = rt
        .block_on(backend.live_sessions(7, 7))
        .expect("conta sessões vivas");
    assert!(
        vivas >= 1,
        "a sessão que já estava dentro continua viva: vivas={vivas}"
    );
}

/// Env sobrescreve o banco (e parse quebrado é ignorado, nunca `panic!`).
#[test]
fn env_sobrescreve_politica_do_banco() {
    std::env::set_var("WYD_SERVER_MAX_PLAYERS", "3");
    std::env::set_var("WYD_SERVER_MAINTENANCE", "on");
    let lida = policy_overrides_from_env();
    assert_eq!(
        lida,
        PolicyOverrides {
            max_players: Some(3),
            maintenance: Some(true),
        }
    );

    std::env::set_var("WYD_SERVER_MAX_PLAYERS", "muito");
    std::env::set_var("WYD_SERVER_MAINTENANCE", "0");
    let tolerante = policy_overrides_from_env();
    assert_eq!(
        tolerante,
        PolicyOverrides {
            max_players: None,
            maintenance: Some(false),
        },
        "teto inválido cai para o banco; manutenção '0' desliga"
    );

    std::env::set_var("WYD_SERVER_MAX_PLAYERS", "");
    std::env::set_var("WYD_SERVER_MAINTENANCE", "  ");
    assert_eq!(
        policy_overrides_from_env(),
        PolicyOverrides::default(),
        "env vazio vale o banco"
    );

    std::env::remove_var("WYD_SERVER_MAX_PLAYERS");
    std::env::remove_var("WYD_SERVER_MAINTENANCE");
    assert_eq!(
        policy_overrides_from_env(),
        PolicyOverrides::default(),
        "sem env vale o banco"
    );
}

/// A recusa de lotação/manutenção tem motivo próprio no Prometheus (QA-06 §4).
#[test]
fn recusa_de_lotacao_e_manutencao_tem_metrica_propria() {
    let metricas = NetMetrics::new();
    metricas.packet_rejected("canal_lotado");
    metricas.packet_rejected("manutencao");
    assert_eq!(metricas.rejected(), 2);
    let texto = metricas.render_prometheus();
    assert!(
        texto.contains("reason=\"canal_lotado\""),
        "motivo canal_lotado declarado"
    );
    assert!(
        texto.contains("reason=\"manutencao\""),
        "motivo manutencao declarado"
    );
}
