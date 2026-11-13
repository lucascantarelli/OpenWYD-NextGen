//! Amostragem de memória residente — fora do caminho quente, por plataforma.
//!
//! A amostra entra por uma thread de 1 Hz no boot do `wyd-server` ([`super::Metrics::sample_memory`]),
//! nunca pelo laço do tick: o hot path não faz I/O (invariante 1 do `AGENTS.md`).
//!
//! Em Linux (produção, container distroless) lê `/proc/self/statm` (páginas residentes ×
//! tamanho de página). Em outras plataformas devolve `None`: não existe leitura portátil em
//! `std` sem `unsafe`/FFI, e o projeto só aceita `unsafe` em interop comprovada. O exporter
//! então publica `process_resident_memory_bytes 0` e `wyd_memory_samples_total 0` — ausência
//! declarada, não número inventado.

/// Páginas residentes × 4 KiB (tamanho de página do Linux x86_64/aarch64).
#[cfg(target_os = "linux")]
const PAGE_BYTES: u64 = 4 * 1024;

/// Memória residente do processo, se a plataforma permitir ler sem FFI.
#[must_use]
pub fn sample_rss_bytes() -> Option<u64> {
    #[cfg(target_os = "linux")]
    {
        let statm = std::fs::read_to_string("/proc/self/statm").ok()?;
        let residente: u64 = statm.split_whitespace().nth(1)?.parse().ok()?;
        Some(residente.saturating_mul(PAGE_BYTES))
    }
    #[cfg(not(target_os = "linux"))]
    {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn amostragem_e_ou_none_ou_plausivel() {
        match sample_rss_bytes() {
            Some(bytes) => assert!(bytes > 0, "RSS positivo quando disponível"),
            None => assert_ne!(std::env::consts::OS, "linux", "Linux deve amostrar"),
        }
    }
}
