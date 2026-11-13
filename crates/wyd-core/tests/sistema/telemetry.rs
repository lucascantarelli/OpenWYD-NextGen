//! Telemetria em execução real: tick de 20 Hz simulado, log estruturado e `/metrics` por TCP.
//!
//! Este teste é o banco de ensaio do que o binário do servidor fará com o mesmo registro:
//! sobe o exporter em porta efêmera, roda um laço de ticks, e faz um `GET /metrics` HTTP/1.1
//! de verdade, imprimindo a resposta para a evidência da issue. Rode com `--nocapture`:
//!
//! ```text
//! WYD_LOG=debug cargo test -p wyd-core --test telemetry -- --nocapture
//! ```

use std::io::{Read, Write};
use std::net::TcpStream;
use std::sync::Arc;
use std::time::{Duration, Instant};

use wyd_core::metrics::{mem, Metrics, MetricsServer, MetricsSource};
use wyd_core::telemetry;

/// Ticks do ensaio: 40 × ~2 ms ≈ 80 ms de laço real.
const TICKS: u64 = 40;

#[test]
fn telemetria_mede_tick_e_serve_metrics() {
    let nivel = telemetry::init_from_env();
    println!("nivel efetivo: {nivel}");

    let metrics = Arc::new(Metrics::new());
    let fonte: Arc<dyn MetricsSource> = metrics.clone();
    let servidor = MetricsServer::bind("127.0.0.1:0", fonte).expect("porta efêmera");
    let endereco = servidor.local_addr();
    println!("exporter em http://{endereco}/metrics");

    for tick in 0..TICKS {
        let inicio = Instant::now();
        std::thread::sleep(Duration::from_millis(2));
        let micros = u64::try_from(inicio.elapsed().as_micros()).unwrap_or(u64::MAX);
        metrics.record_tick(micros, 100 + tick % 5);
        if tick % 10 == 0 {
            tracing::debug!(tick, micros, "tick concluido");
        }
        if tick % 20 == 0 {
            if let Some(rss) = mem::sample_rss_bytes() {
                metrics.set_heap_bytes(rss);
            }
        }
    }

    let resumo = metrics.tick_snapshot();
    tracing::info!(
        ticks = metrics.ticks(),
        p50_us = resumo.p50,
        p95_us = resumo.p95,
        p99_us = resumo.p99,
        rss_bytes = metrics.heap_bytes(),
        "resumo de telemetria"
    );

    let resposta = buscar_metrics(&endereco);
    println!("--- GET /metrics ---\n{resposta}");

    assert!(
        resposta.starts_with("HTTP/1.1 200 OK"),
        "status do exporter"
    );
    assert!(
        resposta.contains(&format!("wyd_ticks_total {TICKS}")),
        "contador de ticks acompanha o laço"
    );
    assert!(
        resposta.contains(&format!("wyd_tick_duration_microseconds_count {TICKS}")),
        "histograma recebeu todas as amostras"
    );
    assert!(
        resposta.contains("wyd_tick_duration_microseconds_bucket{le=\"+Inf\"}"),
        "bucket +Inf é obrigatório no formato"
    );
    assert!(resposta.contains("wyd_entities_active 104"), "gauge final");
    assert!(
        resposta.contains("wyd_memory_samples_total"),
        "amostragem de memória declarada mesmo quando indisponível"
    );
    assert_eq!(telemetry::level_from_env(), nivel, "nível vem do ambiente");
}

/// `GET /metrics` HTTP/1.1 em loopback, lendo até o servidor fechar a conexão.
fn buscar_metrics(endereco: &std::net::SocketAddr) -> String {
    let mut stream = TcpStream::connect(endereco).expect("conecta no exporter");
    stream
        .set_read_timeout(Some(Duration::from_secs(2)))
        .expect("timeout");
    write!(
        stream,
        "GET /metrics HTTP/1.1\r\nHost: {endereco}\r\nConnection: close\r\n\r\n"
    )
    .expect("pedido");
    let mut corpo = String::new();
    stream.read_to_string(&mut corpo).expect("le resposta");
    corpo
}
