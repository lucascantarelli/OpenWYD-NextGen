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
///
/// Com `WYD_TEST_REQUIRED=1` (ligado no job `sistema` do CI, DEVOPS-07) a ausência de DSN vira
/// **falha**: teste que precisa de banco não pode passar por omissão — era assim que o CI ficava
/// verde com zero cobertura dos caminhos mais frágeis (login, entrada no mundo, cena).
///
/// # Panics
///
/// Se `WYD_TEST_REQUIRED` estiver ligado e nenhum DSN for encontrado.
pub fn dsn_ou_skip(contexto: &str) -> Option<String> {
    if let Some(dsn) = dsn() {
        return Some(dsn);
    }
    if obrigatorio_sem_skip() {
        panic!(
            "WYD_TEST_REQUIRED=1 sem DSN — {contexto} não pode ser pulado; defina \
             WYD_TEST_DSN/WYD_TEST_DATABASE_URL/DATABASE_URL"
        );
    }
    eprintln!(
        "SKIP: WYD_TEST_DSN/WYD_TEST_DATABASE_URL/DATABASE_URL ausente — {contexto} não executado"
    );
    None
}

/// `WYD_TEST_REQUIRED` ligado? (qualquer valor diferente de vazio/`0`).
fn obrigatorio_sem_skip() -> bool {
    std::env::var("WYD_TEST_REQUIRED")
        .map(|valor| {
            let valor = valor.trim();
            !valor.is_empty() && valor != "0"
        })
        .unwrap_or(false)
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
