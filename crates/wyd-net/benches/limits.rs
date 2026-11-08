//! Custo dos limites de transporte (FASE-05-1) — balde por conexão e porte de handshake.
//!
//! `cargo bench -p wyd-net --bench limits` (`harness = false`, sem dependência nova).
//! Mede o caminho quente de `limits/rate.rs`: cada pacote que chega passa por
//! `TokenBucket::try_consume`; cada conexão nova passa por `ConnectionGate::admit`/`complete`.
//! Não há I/O: só a aritmética que roda antes de qualquer parse.

use std::hint::black_box;
use std::time::Instant;

use wyd_net::{default_bucket, ConnectionGate, TokenBucket};

/// Consumos medidos no balde (um pacote cada).
const CONSUMOS: u32 = 1_000_000;
/// Admissões de handshake medidas.
const ADMISSOES: u32 = 200_000;
/// IPs distintos no cenário de porte.
const IPS: u32 = 256;

fn medir_balde() -> f64 {
    let mut balde: TokenBucket = default_bucket();
    let mut now_ms = 0u64;
    let inicio = Instant::now();
    for i in 0..CONSUMOS {
        // 1 ms por pacote: abaixo da taxa do balde padrão, o caminho é o de aceite.
        now_ms += 1;
        // Estouro de rajada é resultado esperado do cenário, não erro do bench.
        black_box(balde.try_consume(now_ms).is_ok());
        black_box(i);
    }
    let decorrido = inicio.elapsed();
    decorrido.as_nanos() as f64 / f64::from(CONSUMOS)
}

fn medir_porte() -> f64 {
    let mut porte = ConnectionGate::local();
    let mut abertas = 0u64;
    let mut now_ms = 0u64;
    let inicio = Instant::now();
    for i in 0..ADMISSOES {
        now_ms += 1;
        let ip = i % IPS;
        if let Ok(conn) = porte.admit(ip, now_ms) {
            if porte.complete(conn) {
                abertas += 1;
                porte.release(ip);
            }
        }
        black_box(abertas);
    }
    let decorrido = inicio.elapsed();
    decorrido.as_nanos() as f64 / f64::from(ADMISSOES)
}

fn main() {
    println!("=== limites de transporte — custo por operação (FASE-05-1) ===");
    println!("avalie antes de qualquer parse: balde por conexão + porte de handshake");
    let balde = medir_balde();
    let porte = medir_porte();
    println!("TokenBucket::try_consume: {balde:.1} ns/op ({CONSUMOS} consumos)");
    println!("ConnectionGate::admit+complete: {porte:.1} ns/op ({ADMISSOES} admissões, {IPS} IPs)");
    println!(
        "veredito: por pacote (balde) {balde:.1} ns/op; por conexão nova (porte) {porte:.1} ns/op — o porte roda uma vez por handshake, não por pacote"
    );
}
