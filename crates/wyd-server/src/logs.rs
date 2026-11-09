//! Anel de log do processo: as linhas estruturadas também ficam em memória.
//!
//! Serve para provar o que **não** pode aparecer no log (PIN, senha): o teste lê o anel e
//! falha se o segredo estiver lá. Também evita que o teste dependa de captura de `stdout`
//! do harness.

use std::collections::VecDeque;
use std::sync::Mutex;

/// Quantas linhas ficam retidas (suficiente para uma sessão de teste inteira).
const CAPACIDADE: usize = 512;

static ANEL: Mutex<VecDeque<String>> = Mutex::new(VecDeque::new());

/// Registra a linha que também vai para o `stdout`.
pub fn record(linha: &str) {
    let mut anel = ANEL.lock().unwrap_or_else(|e| e.into_inner());
    if anel.len() == CAPACIDADE {
        anel.pop_front();
    }
    anel.push_back(linha.to_string());
}

/// Últimas `n` linhas (mais antigas primeiro).
#[must_use]
pub fn tail(n: usize) -> Vec<String> {
    let anel = ANEL.lock().unwrap_or_else(|e| e.into_inner());
    let inicio = anel.len().saturating_sub(n);
    anel.iter().skip(inicio).cloned().collect()
}
