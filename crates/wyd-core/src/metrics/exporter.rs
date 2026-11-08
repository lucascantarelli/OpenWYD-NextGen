//! Exporter HTTP mínimo do registro — serve `GET /metrics` em texto Prometheus 0.0.4.
//!
//! Sem framework web: `std::net::TcpListener` em thread própria, uma request por conexão e
//! resposta `Connection: close`. Fica fora do tick (thread de scrape), não no Hot Path.
//!
//! Fronteira: o **binário** do servidor decide o endereço (`WYD_METRICS_ADDR`, default
//! `127.0.0.1:9898`); este módulo só expõe o que o registro tem.

use std::io::{Read, Write};
use std::net::{SocketAddr, TcpListener, TcpStream};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::thread::JoinHandle;
use std::time::Duration;

/// Fonte do corpo `/metrics` — o registro do motor, ou uma composição com outras crates.
pub trait MetricsSource: Send + Sync {
    /// Corpo completo em formato de exposição Prometheus.
    fn scrape(&self) -> String;
}

/// Servidor de scrape em thread dedicada; para quando `MetricsServer` cai.
#[derive(Debug)]
pub struct MetricsServer {
    addr: SocketAddr,
    stop: Arc<AtomicBool>,
    handle: Option<JoinHandle<()>>,
}

impl MetricsServer {
    /// Sobe o listener e a thread de atendimento. `addr` com porta 0 escolhe porta livre.
    pub fn bind(addr: &str, source: Arc<dyn MetricsSource>) -> std::io::Result<Self> {
        let listener = TcpListener::bind(addr)?;
        listener.set_nonblocking(true)?;
        let local = listener.local_addr()?;
        let stop = Arc::new(AtomicBool::new(false));
        let flag = Arc::clone(&stop);
        let handle = std::thread::Builder::new()
            .name("wyd-metrics".to_string())
            .spawn(move || {
                while !flag.load(Ordering::Relaxed) {
                    match listener.accept() {
                        Ok((stream, _)) => {
                            let _ = atender(stream, source.as_ref());
                        }
                        Err(ref e) if e.kind() == std::io::ErrorKind::WouldBlock => {
                            std::thread::sleep(Duration::from_millis(5));
                        }
                        Err(_) => break,
                    }
                }
            })?;
        Ok(Self {
            addr: local,
            stop,
            handle: Some(handle),
        })
    }

    /// Endereço efetivo (útil quando a porta foi escolhida pelo SO).
    #[must_use]
    pub fn local_addr(&self) -> SocketAddr {
        self.addr
    }
}

impl Drop for MetricsServer {
    fn drop(&mut self) {
        self.stop.store(true, Ordering::Relaxed);
        if let Some(handle) = self.handle.take() {
            let _ = handle.join();
        }
    }
}

/// Responde uma conexão: `GET /metrics` → 200 texto; resto → 404.
fn atender(mut stream: TcpStream, source: &dyn MetricsSource) -> std::io::Result<()> {
    let _ = stream.set_read_timeout(Some(Duration::from_millis(500)));
    let mut buf = [0u8; 512];
    let lido = stream.read(&mut buf)?;
    let pedido = String::from_utf8_lossy(&buf[..lido]);
    let (status, corpo) = if pedido.starts_with("GET /metrics ") {
        ("200 OK", source.scrape())
    } else {
        ("404 Not Found", "not found\n".to_string())
    };
    let resposta = format!(
        "HTTP/1.1 {status}\r\nContent-Type: text/plain; version=0.0.4; charset=utf-8\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{corpo}",
        corpo.len()
    );
    stream.write_all(resposta.as_bytes())?;
    stream.flush()
}

#[cfg(test)]
mod tests {
    use super::*;

    struct Fixo;

    impl MetricsSource for Fixo {
        fn scrape(&self) -> String {
            "wyd_ticks_total 3\n".to_string()
        }
    }

    fn get(addr: SocketAddr, caminho: &str) -> (u16, String) {
        let mut s = TcpStream::connect(addr).expect("conecta");
        s.set_read_timeout(Some(Duration::from_secs(2)))
            .expect("timeout");
        write!(s, "GET {caminho} HTTP/1.1\r\nHost: localhost\r\n\r\n").expect("pedido");
        let mut corpo = String::new();
        s.read_to_string(&mut corpo).expect("resposta");
        let status = corpo
            .split_whitespace()
            .nth(1)
            .and_then(|s| s.parse().ok())
            .unwrap_or(0);
        (status, corpo)
    }

    #[test]
    fn serve_metrics_e_recusa_outro_caminho() {
        let server = MetricsServer::bind("127.0.0.1:0", Arc::new(Fixo)).expect("bind");
        let addr = server.local_addr();
        let (status, corpo) = get(addr, "/metrics");
        assert_eq!(status, 200);
        assert!(corpo.contains("wyd_ticks_total 3"));
        assert!(corpo.contains("text/plain; version=0.0.4"));
        let (status, _) = get(addr, "/saude");
        assert_eq!(status, 404);
    }
}
