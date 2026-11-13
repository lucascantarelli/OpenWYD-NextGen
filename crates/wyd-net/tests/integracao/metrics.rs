#![cfg(feature = "metrics")]
//! Contadores de rede sob tráfego real dos limites: pacotes, recusas por motivo e gauges.
//!
//! Os contadores são globais do processo (como no daemon), então as asserções usam o delta
//! observado ao redor do tráfego e toleram incrementos concorrentes de outros testes.

use wyd_net::metrics::NET;
use wyd_net::{ConnectionGate, Deny, TokenBucket};

#[test]
fn contadores_movem_sob_trafego_e_renderizam_prometheus() {
    let pacotes_antes = NET.packets();
    let recusas_antes = NET.rejected();

    let mut balde = TokenBucket::new(2, 1000);
    assert!(balde.try_consume(0).is_ok());
    assert!(balde.try_consume(0).is_ok());
    assert_eq!(balde.try_consume(0), Err(Deny::RateLimited));
    assert!(balde.try_consume(5).is_ok(), "recarga em 5 ms");

    let aceitos = NET.packets().saturating_sub(pacotes_antes);
    let recusas = NET.rejected().saturating_sub(recusas_antes);
    assert!(
        aceitos >= 3,
        "3 consumos aceitos contabilizados ({aceitos})"
    );
    assert!(recusas >= 1, "1 recusa contabilizada ({recusas})");

    let mut porte = ConnectionGate::local();
    for _ in 0..3 {
        porte.admit(7, 0).expect("cabe no teto por IP");
    }
    assert_eq!(porte.admit(7, 0), Err(Deny::TooManyPerIp { limit: 3 }));

    let texto = NET.render_prometheus();
    println!("--- fragmento /metrics do wyd-net ---\n{texto}");
    assert!(texto.contains("wyd_packets_total "));
    assert!(texto.contains("wyd_packets_rejected_total{reason=\"rate\"}"));
    assert!(texto.contains("wyd_packets_rejected_total{reason=\"ip\"}"));
    assert!(texto.contains("wyd_handshakes_total "));
    assert!(texto.contains("wyd_connections_pending "));
    assert!(texto.contains("wyd_sessions_open "));

    let contagem: u64 = texto
        .lines()
        .find_map(|l| l.strip_prefix("wyd_packets_total "))
        .and_then(|v| v.trim().parse().ok())
        .expect("linha de contador");
    assert!(contagem >= 3, "contador exportado reflete o tráfego");
}
