//! Helpers compartilhados dos testes que precisam de Postgres.
//!
//! O gate é único por crate: qualquer teste de sistema usa `dsn_ou_skip` e recebe
//! a mesma mensagem `SKIP:` com todas as variáveis aceitas.

#![allow(dead_code)]

use std::time::{Duration, Instant};

/// DSN de teste, na ordem de precedência do repositório.
pub fn dsn() -> Option<String> {
    ["WYD_TEST_DSN", "WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty())
}

/// Devolve o DSN ou imprime o `SKIP:` explícito e devolve `None`.
pub fn dsn_ou_skip(contexto: &str) -> Option<String> {
    let dsn = dsn();
    if dsn.is_none() {
        eprintln!(
            "SKIP: WYD_TEST_DSN/WYD_TEST_DATABASE_URL/DATABASE_URL ausente — {contexto} não executado"
        );
    }
    dsn
}

/// Espera `cond` virar verdadeira até o prazo, amostrando a cada 25 ms.
pub fn esperar_ate(prazo: Duration, cond: impl FnMut() -> bool) -> bool {
    esperar_ate_a_cada(prazo, Duration::from_millis(25), cond)
}

/// Espera `cond` com um intervalo próprio (ex.: scrape do Prometheus a cada 1 s).
///
/// Substitui `sleep` fixo de sincronização: o teste só espera o tempo necessário
/// e falha com o prazo esgotado quando a condição nunca acontece.
pub fn esperar_ate_a_cada(
    prazo: Duration,
    intervalo: Duration,
    mut cond: impl FnMut() -> bool,
) -> bool {
    let limite = Instant::now() + prazo;
    loop {
        if cond() {
            return true;
        }
        if Instant::now() >= limite {
            return false;
        }
        std::thread::sleep(intervalo);
    }
}
