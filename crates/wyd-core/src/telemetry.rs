//! Inicialização do `tracing` — nível por ambiente, saída estruturada em `key=value`.
//!
//! Variável: `WYD_LOG` (com fallback para `RUST_LOG`), valores `error|warn|info|debug|trace`.
//! Default `info`. Um `RUST_LOG` composto (`info,wyd_net=debug`) é aceito no primeiro termo —
//! não há `env-filter` (que traria `regex` e peso ao binário) porque o filtro por alvo não é
//! requisito da issue; ver ADR-0018.

use std::env;

use tracing_subscriber::filter::LevelFilter;

/// Variável de ambiente que controla o nível.
pub const ENV_LEVEL: &str = "WYD_LOG";

/// Nível configurado no ambiente (ou `info`).
#[must_use]
pub fn level_from_env() -> LevelFilter {
    let valor = env::var(ENV_LEVEL).or_else(|_| env::var("RUST_LOG")).ok();
    level_from_value(valor.as_deref())
}

/// Interpreta o valor da variável; tolera `info,alvo=debug` usando o primeiro termo.
#[must_use]
pub fn level_from_value(valor: Option<&str>) -> LevelFilter {
    let Some(bruto) = valor else {
        return LevelFilter::INFO;
    };
    let primeiro = bruto.split(',').next().unwrap_or("").trim();
    // `info` e `alvo=debug`: o nível é o último termo; alvo específico não é suportado
    // (exigiria `env-filter`/regex — ADR-0018).
    let nome = primeiro.rsplit('=').next().unwrap_or("").trim();
    match nome.to_ascii_lowercase().as_str() {
        "error" => LevelFilter::ERROR,
        "warn" => LevelFilter::WARN,
        "info" => LevelFilter::INFO,
        "debug" => LevelFilter::DEBUG,
        "trace" => LevelFilter::TRACE,
        "off" => LevelFilter::OFF,
        _ => LevelFilter::INFO,
    }
}

/// Instala o subscriber global com o nível do ambiente e devolve o nível efetivo.
///
/// Segunda chamada no mesmo processo é no-op (o subscriber global é único); devolve o nível
/// pedido para o chamador registrar na inicialização.
pub fn init_from_env() -> LevelFilter {
    let level = level_from_env();
    let _ = tracing_subscriber::fmt()
        .with_max_level(level)
        .with_target(false)
        .with_ansi(false)
        .try_init();
    level
}

/// Instala o subscriber com nível explícito (teste/daemon que já decidiu o nível).
pub fn init_with_level(level: LevelFilter) {
    let _ = tracing_subscriber::fmt()
        .with_max_level(level)
        .with_target(false)
        .with_ansi(false)
        .try_init();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn le_niveis_e_ignora_lixo() {
        assert_eq!(level_from_value(Some("debug")), LevelFilter::DEBUG);
        assert_eq!(level_from_value(Some("TRACE")), LevelFilter::TRACE);
        assert_eq!(
            level_from_value(Some("info,wyd_net=debug")),
            LevelFilter::INFO
        );
        assert_eq!(level_from_value(Some("wyd_net=debug")), LevelFilter::DEBUG);
        assert_eq!(level_from_value(Some("off")), LevelFilter::OFF);
        assert_eq!(level_from_value(Some("nada")), LevelFilter::INFO);
        assert_eq!(level_from_value(None), LevelFilter::INFO);
    }
}
