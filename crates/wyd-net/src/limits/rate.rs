//! Tetos por conexão/IP e backpressure — DoS contido, legítimo preservado.
//!
//! Tudo puro (tempo injetado em ms): sem relógio global, sem I/O, sem alloc no
//! `try_consume`. Limites default dimensionados para servidor local; NAT exige
//! teto por conta além do por IP (pendência 06-2, não improviso aqui).

use core::fmt;

/// Motivo da recusa (sem segredo, sem endereço além da chave u32 já conhecida).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Deny {
    /// Rajada acima do balde (pacotes).
    RateLimited,
    /// IP com conexões demais (handshake).
    TooManyPerIp { limit: u32 },
    /// Servidor cheio (handshake).
    TooManyTotal { limit: u32 },
    /// Handshake não concluído no prazo (varrido por `sweep`).
    HandshakeTimeout { after_ms: u64 },
}

impl fmt::Display for Deny {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::RateLimited => write!(f, "rajada acima do limite de pacotes"),
            Self::TooManyPerIp { limit } => write!(f, "conexões demais por IP (teto {limit})"),
            Self::TooManyTotal { limit } => write!(f, "servidor cheio (teto {limit})"),
            Self::HandshakeTimeout { after_ms } => {
                write!(f, "handshake expirou após {after_ms} ms")
            }
        }
    }
}

/// Balde de tokens: rajada `capacity`, recarga `per_sec` por segundo.
/// Tempo em ms injetado pelo chamador (teste determinístico, produção lê relógio).
#[derive(Debug, Clone)]
pub struct TokenBucket {
    capacity: f64,
    per_ms: f64,
    tokens: f64,
    last_ms: u64,
}

impl TokenBucket {
    /// `per_sec` pacotes/s sustentados com rajada `capacity` (ambos > 0).
    #[must_use]
    pub fn new(capacity: u32, per_sec: u32) -> Self {
        let capacity = capacity.max(1) as f64;
        Self {
            capacity,
            per_ms: (per_sec.max(1) as f64) / 1000.0,
            tokens: capacity,
            last_ms: 0,
        }
    }

    /// Tenta consumir 1 token em `now_ms` (monótono; volta no tempo = sem recarga).
    pub fn try_consume(&mut self, now_ms: u64) -> Result<(), Deny> {
        if now_ms > self.last_ms {
            self.tokens =
                (self.tokens + (now_ms - self.last_ms) as f64 * self.per_ms).min(self.capacity);
            self.last_ms = now_ms;
        }
        if self.tokens >= 1.0 {
            self.tokens -= 1.0;
            Ok(())
        } else {
            Err(Deny::RateLimited)
        }
    }
}

/// Padrão do servidor local: rajada 100, 100 pacotes/s por origem.
#[must_use]
pub fn default_bucket() -> TokenBucket {
    TokenBucket::new(100, 100)
}

/// Porte do handshake: teto por IP + teto total + prazo de conclusão.
#[derive(Debug, Clone)]
pub struct ConnectionGate {
    max_per_ip: u32,
    max_total: u32,
    handshake_timeout_ms: u64,
    per_ip: std::collections::HashMap<u32, u32>,
    total: u32,
    pending: std::collections::HashMap<u64, (u32, u64)>,
    next_conn: u64,
}

impl ConnectionGate {
    /// Padrão local: 3 conexões/IP, 1000 totais, handshake em 5 s.
    #[must_use]
    pub fn local() -> Self {
        Self {
            max_per_ip: 3,
            max_total: 1000,
            handshake_timeout_ms: 5_000,
            per_ip: std::collections::HashMap::new(),
            total: 0,
            pending: std::collections::HashMap::new(),
            next_conn: 1,
        }
    }

    /// Admite handshake do `ip` em `now_ms`; devolve o id da conexão pendente.
    pub fn admit(&mut self, ip: u32, now_ms: u64) -> Result<u64, Deny> {
        self.sweep(now_ms);
        if self.total >= self.max_total {
            return Err(Deny::TooManyTotal {
                limit: self.max_total,
            });
        }
        let used = self.per_ip.get(&ip).copied().unwrap_or(0);
        if used >= self.max_per_ip {
            return Err(Deny::TooManyPerIp {
                limit: self.max_per_ip,
            });
        }
        *self.per_ip.entry(ip).or_insert(0) += 1;
        self.total += 1;
        let id = self.next_conn;
        self.next_conn += 1;
        self.pending.insert(id, (ip, now_ms));
        Ok(id)
    }

    /// Handshake concluído: libera a vaga de pendente (conta segue em `total`).
    pub fn complete(&mut self, conn: u64) -> bool {
        self.pending.remove(&conn).is_some()
    }

    /// Encerra conexão (libera teto por IP e total).
    pub fn release(&mut self, ip: u32) {
        if let Some(n) = self.per_ip.get_mut(&ip) {
            *n = n.saturating_sub(1);
            if *n == 0 {
                self.per_ip.remove(&ip);
            }
        }
        self.total = self.total.saturating_sub(1);
    }

    /// Varre pendentes estourados; devolve quantos caíram.
    pub fn sweep(&mut self, now_ms: u64) -> usize {
        let timeout = self.handshake_timeout_ms;
        let expired: Vec<(u64, u32)> = self
            .pending
            .iter()
            .filter(|(_, (_, since))| now_ms.saturating_sub(*since) > timeout)
            .map(|(id, (ip, _))| (*id, *ip))
            .collect();
        let n = expired.len();
        for (id, ip) in expired {
            self.pending.remove(&id);
            self.release(ip);
        }
        n
    }

    #[must_use]
    pub fn pending_count(&self) -> usize {
        self.pending.len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bucket_bursts_then_throttles_then_refills() {
        let mut b = TokenBucket::new(3, 1000);
        assert!(b.try_consume(0).is_ok());
        assert!(b.try_consume(0).is_ok());
        assert!(b.try_consume(0).is_ok());
        assert_eq!(b.try_consume(0), Err(Deny::RateLimited));
        // 1000/s = 1/ms: 2 ms recarregam 2 tokens.
        assert!(b.try_consume(2).is_ok());
        assert!(b.try_consume(2).is_ok());
        assert_eq!(b.try_consume(2), Err(Deny::RateLimited));
        // Volta no tempo não injeta tokens.
        assert_eq!(b.try_consume(0), Err(Deny::RateLimited));
    }

    #[test]
    fn gate_caps_per_ip_and_total_with_release() {
        let mut g = ConnectionGate::local();
        let a1 = g.admit(10, 0).expect("1ª do ip");
        g.admit(10, 0).expect("2ª do ip");
        g.admit(10, 0).expect("3ª do ip");
        assert_eq!(g.admit(10, 0), Err(Deny::TooManyPerIp { limit: 3 }));
        // Outro IP passa; legítimo preservado.
        g.admit(11, 0).expect("ip distinto passa");
        assert!(g.complete(a1));
        g.release(11);
        // Pendentes: 2 do ip .1 (a1 concluído) + 1 do ip .2.
        assert_eq!(g.pending_count(), 3);
    }

    #[test]
    fn handshake_timeout_evicts_and_frees_ip_slot() {
        let mut g = ConnectionGate::local();
        for _ in 0..3 {
            g.admit(10, 0).expect("lota o ip");
        }
        assert_eq!(g.admit(10, 0), Err(Deny::TooManyPerIp { limit: 3 }));
        assert_eq!(g.sweep(5_001), 3);
        g.admit(10, 5_001).expect("vaga liberada pelo sweep");
    }
}
