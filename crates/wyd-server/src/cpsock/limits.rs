//! Submódulo de `cpsock` (ADR-0026, etapa 2).

use super::*;

/// Limite de conexões simultâneas por origem (`per_ip = 0` desliga).
#[derive(Debug, Clone, Copy)]
pub struct ConnectionLimit {
    pub per_ip: usize,
}

impl ConnectionLimit {
    #[must_use]
    pub fn new(per_ip: usize) -> Self {
        Self { per_ip }
    }

    /// `WYD_MAX_CONNECTIONS_PER_IP` (padrão 6; `0` desliga). Multiboxing custa 4–5.
    #[must_use]
    pub fn from_env() -> Self {
        let per_ip = std::env::var("WYD_MAX_CONNECTIONS_PER_IP")
            .ok()
            .and_then(|valor| valor.trim().parse().ok())
            .unwrap_or(6);
        Self { per_ip }
    }
}

/// Vaga de conexão de um IP: libera a contagem no `Drop`, inclusive em pânico da thread.
pub(super) struct IpSlot {
    ativos: Option<Arc<std::sync::Mutex<std::collections::HashMap<std::net::IpAddr, usize>>>>,
    ip: std::net::IpAddr,
}

impl IpSlot {
    pub(super) fn reservar(
        ativos: &Arc<std::sync::Mutex<std::collections::HashMap<std::net::IpAddr, usize>>>,
        ip: std::net::IpAddr,
        maximo: usize,
    ) -> Option<Self> {
        if maximo == 0 {
            return Some(Self { ativos: None, ip });
        }
        let mut mapa = ativos.lock().unwrap_or_else(|e| e.into_inner());
        let atual = mapa.entry(ip).or_insert(0);
        if *atual >= maximo {
            return None;
        }
        *atual += 1;
        Some(Self {
            ativos: Some(Arc::clone(ativos)),
            ip,
        })
    }
}

impl Drop for IpSlot {
    fn drop(&mut self) {
        let Some(ativos) = &self.ativos else {
            return;
        };
        let mut mapa = ativos.lock().unwrap_or_else(|e| e.into_inner());
        if let Some(atual) = mapa.get_mut(&self.ip) {
            *atual = atual.saturating_sub(1);
            if *atual == 0 {
                mapa.remove(&self.ip);
            }
        }
    }
}
