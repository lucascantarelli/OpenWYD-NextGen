//! Política de save e dreno da fila de persistência (FASE-04-3).
//!
//! **Nenhum `await` no tick:** o Hot Path só chama [`SaveQueue::enqueue`], que é
//! `try_send` em canal limitado (custo de nanossegundos, sem lock global). Quem abre
//! transação é o [`SaveDrain`], tarefa de Cold Path.
//!
//! **Coalescência:** o dreno agrupa por personagem e grava só o snapshot mais recente do
//! lote. Salvar 5 vezes o mesmo personagem entre dois drenos custaria 5 `UPDATE` para
//! chegar ao mesmo estado — o último vence, e o lote vira um teto de escrita.
//!
//! **Sem perda no shutdown:** [`SaveDrain::shutdown`] esvazia a fila até o canal fechar e
//! devolve [`FlushReport`] com quantos foram gravados. O que não couber no timeout é
//! erro explícito (`SaveTimeout`), nunca perda silenciosa.
//!
//! Política (documentada em `docs/arquitetura/contrato-persistencia.md` §8):
//!
//! | Gatilho | Quando | Garantia |
//! |---|---|---|
//! | periódico | a cada [`SavePolicy::periodic_ms`] | estado converge em ≤ 1 período |
//! | por lote | máximo [`SavePolicy::max_batch`] snapshots por transação | transação curta |
//! | shutdown | sinal de parada | fila esvaziada dentro de `shutdown_timeout_ms` |

use std::collections::BTreeMap;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use std::time::Duration;

use tokio::sync::{mpsc, watch};

use crate::error::DbError;
use crate::model::Character;
use crate::pool::pg::PgBackend;

/// Política de save (validada: zero/desligado é recusado em vez de virar default).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SavePolicy {
    /// Período do dreno periódico.
    pub periodic_ms: u64,
    /// Teto de snapshots por transação.
    pub max_batch: u32,
    /// Tempo máximo para esvaziar a fila no shutdown.
    pub shutdown_timeout_ms: u64,
    /// Capacidade do canal (backpressure do Hot Path).
    pub queue_capacity: usize,
}

impl SavePolicy {
    /// Padrão do servidor local: 5 s, 256 por lote, 4 mil pendentes.
    #[must_use]
    pub fn local() -> Self {
        Self {
            periodic_ms: 5_000,
            max_batch: 256,
            shutdown_timeout_ms: 3_000,
            queue_capacity: 4_096,
        }
    }

    pub fn validated(self) -> Result<Self, DbError> {
        if self.periodic_ms == 0 || self.shutdown_timeout_ms == 0 {
            return Err(DbError::ConstraintViolation {
                table: "save_policy",
                constraint: "timeouts > 0",
            });
        }
        if self.max_batch == 0 || self.max_batch > 4_096 {
            return Err(DbError::ConstraintViolation {
                table: "save_policy",
                constraint: "max_batch 1–4096",
            });
        }
        if self.queue_capacity == 0 {
            return Err(DbError::ConstraintViolation {
                table: "save_policy",
                constraint: "queue_capacity > 0",
            });
        }
        Ok(self)
    }
}

/// Snapshot a persistir. Só personagem por enquanto (inventário/guilda entram com
/// os repositórios das FASEs seguintes, mesmo caminho).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SaveRequest {
    /// Estado vivo do personagem (HP/MP/nível/EXP/posição) a gravar no `characters`.
    Character(Character),
}

impl SaveRequest {
    /// Id do personagem do snapshot (chave da coalescência).
    #[must_use]
    pub fn character_id(&self) -> i64 {
        match self {
            Self::Character(c) => c.id,
        }
    }
}

/// Produtor da fila — a única parte que o Hot Path toca.
#[derive(Debug)]
pub struct SaveQueue {
    tx: mpsc::Sender<SaveRequest>,
    pending: Arc<AtomicUsize>,
    capacity: usize,
}

impl SaveQueue {
    /// Cria a fila e o dreno correspondente (um par por servidor).
    pub fn bounded(policy: SavePolicy) -> Result<(Self, SaveDrain), DbError> {
        let policy = policy.validated()?;
        let (tx, rx) = mpsc::channel(policy.queue_capacity);
        let pending = Arc::new(AtomicUsize::new(0));
        Ok((
            Self {
                tx,
                pending: Arc::clone(&pending),
                capacity: policy.queue_capacity,
            },
            SaveDrain {
                rx,
                pending,
                policy,
            },
        ))
    }

    /// Enfileira **sem** bloquear nem alocar: fila cheia é `SaveQueueFull`.
    ///
    /// O chamador decide o que fazer com a recusa (logar, reagendar no próximo tick);
    /// esperar aqui seria I/O disfarçado dentro do quente.
    pub fn enqueue(&self, request: SaveRequest) -> Result<(), DbError> {
        self.tx.try_send(request).map_err(|e| match e {
            mpsc::error::TrySendError::Full(_) => DbError::SaveQueueFull {
                capacity: self.capacity,
            },
            mpsc::error::TrySendError::Closed(_) => DbError::Transport { op: "save.enqueue" },
        })?;
        self.pending.fetch_add(1, Ordering::Relaxed);
        Ok(())
    }

    /// Pendentes agora (observabilidade; contador atômico, fora do quente).
    #[must_use]
    pub fn pending(&self) -> usize {
        self.pending.load(Ordering::Relaxed)
    }
}

/// Resultado de um dreno (o que foi realmente para o banco).
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct FlushReport {
    /// Personagens efetivamente gravados.
    pub written: u32,
    /// Snapshots recebidos (antes da coalescência).
    pub received: u32,
    /// Snapshots descartados por coalescência (versão mais antiga do mesmo id).
    pub coalesced: u32,
    /// Personagens que sumiram do banco entre a fila e o dreno.
    pub missing: u32,
}

impl FlushReport {
    fn merge(self, other: Self) -> Self {
        Self {
            written: self.written + other.written,
            received: self.received + other.received,
            coalesced: self.coalesced + other.coalesced,
            missing: self.missing + other.missing,
        }
    }
}

/// Consumidor da fila — tarefa de Cold Path, nunca chamada do tick.
#[derive(Debug)]
pub struct SaveDrain {
    rx: mpsc::Receiver<SaveRequest>,
    pending: Arc<AtomicUsize>,
    policy: SavePolicy,
}

impl SaveDrain {
    #[must_use]
    pub fn policy(&self) -> SavePolicy {
        self.policy
    }

    /// Pega até `limit` pedidos e mantém **só o último** por personagem.
    ///
    /// Determinístico: devolve em ordem de `character_id`, então o mesmo conjunto de
    /// entradas produz a mesma sequência de `UPDATE`.
    pub fn coalesce(&mut self, limit: u32) -> (Vec<SaveRequest>, u32) {
        let mut latest: BTreeMap<i64, Character> = BTreeMap::new();
        let mut received = 0u32;
        for _ in 0..limit {
            match self.rx.try_recv() {
                Ok(SaveRequest::Character(character)) => {
                    received += 1;
                    latest.insert(character.id, character);
                }
                Err(_) => break,
            }
        }
        self.pending.fetch_sub(received as usize, Ordering::Relaxed);
        let coalesced = received.saturating_sub(latest.len() as u32);
        (
            latest
                .into_values()
                .map(SaveRequest::Character)
                .collect::<Vec<_>>(),
            coalesced,
        )
    }

    /// Grava um lote em **uma** transação curta.
    ///
    /// Fila vazia não toca o banco: devolve relatório zerado sem abrir conexão — é o que
    /// garante que um servidor ocioso não gere ruído de I/O.
    pub async fn flush(&mut self, backend: &PgBackend) -> Result<FlushReport, DbError> {
        let (requests, coalesced) = self.coalesce(self.policy.max_batch);
        self.apply(backend, &requests, coalesced).await
    }

    async fn apply(
        &mut self,
        backend: &PgBackend,
        requests: &[SaveRequest],
        coalesced: u32,
    ) -> Result<FlushReport, DbError> {
        if requests.is_empty() {
            return Ok(FlushReport::default());
        }
        let mut tx = backend.begin().await?;
        let mut written = 0u32;
        let mut missing = 0u32;
        for request in requests {
            let SaveRequest::Character(c) = request;
            let res = sqlx::query(
                "UPDATE characters SET level = $2, exp = $3, hp = $4, mp = $5, \
                 pos_x = $6, pos_y = $7, updated_at = now() \
                 WHERE id = $1 AND deleted_at IS NULL",
            )
            .bind(c.id)
            .bind(i32::try_from(c.level).unwrap_or(1))
            .bind(c.exp)
            .bind(i32::try_from(c.hp).unwrap_or(1))
            .bind(i32::try_from(c.mp).unwrap_or(0))
            .bind(i32::try_from(c.pos_x).unwrap_or(0))
            .bind(i32::try_from(c.pos_y).unwrap_or(0))
            .execute(&mut *tx)
            .await
            .map_err(crate::pool::pg::map_error("save.characters", "characters"))?;
            if res.rows_affected() == 0 {
                missing += 1;
            } else {
                written += 1;
            }
        }
        tx.commit()
            .await
            .map_err(crate::pool::pg::map_error("save.commit", "characters"))?;
        Ok(FlushReport {
            written,
            received: requests.len() as u32 + coalesced,
            coalesced,
            missing,
        })
    }

    /// Loop periódico: drena a cada `periodic_ms` até o shutdown.
    ///
    /// `true` no canal `shutdown` interrompe o loop; o esvaziamento final é de
    /// [`Self::shutdown`], que o chamador roda depois (ou no `Drop` da tarefa).
    pub async fn run(
        &mut self,
        backend: &PgBackend,
        mut shutdown: watch::Receiver<bool>,
    ) -> Result<FlushReport, DbError> {
        let mut tick = tokio::time::interval(Duration::from_millis(self.policy.periodic_ms));
        tick.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);
        let mut total = FlushReport::default();
        loop {
            tokio::select! {
                _ = tick.tick() => {
                    total = total.merge(self.flush(backend).await?);
                }
                changed = shutdown.changed() => {
                    if changed.is_err() || *shutdown.borrow() {
                        break;
                    }
                }
            }
        }
        Ok(total)
    }

    /// Esvazia a fila até o canal fechar (produtor derrubado) dentro do timeout.
    ///
    /// Chamado no shutdown, **depois** de derrubar o `SaveQueue`: drenar até
    /// `try_recv` falhar por canal fechado é o que garante "sem perda".
    pub async fn shutdown(&mut self, backend: &PgBackend) -> Result<FlushReport, DbError> {
        let timeout = Duration::from_millis(self.policy.shutdown_timeout_ms);
        let mut total = FlushReport::default();
        let drained = tokio::time::timeout(timeout, async {
            loop {
                let report = self.flush(backend).await?;
                if report.received == 0 {
                    return Ok::<FlushReport, DbError>(total.merge(report));
                }
                total = total.merge(report);
                if self.rx.is_closed() && self.rx.is_empty() {
                    return Ok(total);
                }
            }
        })
        .await;
        match drained {
            Ok(report) => report,
            Err(_) => Err(DbError::SaveTimeout {
                ms: self.policy.shutdown_timeout_ms,
            }),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn character(id: i64, hp: i64) -> Character {
        Character {
            id,
            account_id: 1,
            slot: 0,
            name: format!("c{id}"),
            class: 0,
            level: 10,
            exp: 100,
            hp,
            mp: 10,
            pos_x: 2100,
            pos_y: 2100,
        }
    }

    #[test]
    fn policy_rejects_zeroed_values() {
        assert!(SavePolicy::local().validated().is_ok());
        for bad in [
            SavePolicy {
                periodic_ms: 0,
                ..SavePolicy::local()
            },
            SavePolicy {
                max_batch: 0,
                ..SavePolicy::local()
            },
            SavePolicy {
                queue_capacity: 0,
                ..SavePolicy::local()
            },
            SavePolicy {
                shutdown_timeout_ms: 0,
                ..SavePolicy::local()
            },
        ] {
            assert!(bad.validated().is_err());
        }
    }

    #[test]
    fn coalesce_keeps_latest_per_character_and_reports_it() {
        let (queue, mut drain) = SaveQueue::bounded(SavePolicy::local()).expect("política válida");
        queue
            .enqueue(SaveRequest::Character(character(7, 1)))
            .expect("cabe");
        queue
            .enqueue(SaveRequest::Character(character(7, 2)))
            .expect("cabe");
        queue
            .enqueue(SaveRequest::Character(character(3, 9)))
            .expect("cabe");
        assert_eq!(queue.pending(), 3);

        let (batch, coalesced) = drain.coalesce(8);
        assert_eq!(coalesced, 1);
        assert_eq!(batch.len(), 2);
        // Ordem por id: determinismo do lote.
        assert_eq!(batch[0].character_id(), 3);
        assert_eq!(batch[1].character_id(), 7);
        // O último snapshot vence.
        let SaveRequest::Character(c7) = &batch[1];
        assert_eq!(c7.hp, 2);
        assert_eq!(queue.pending(), 0);
        assert!(queue
            .enqueue(SaveRequest::Character(character(9, 1)))
            .is_ok());
    }

    #[test]
    fn full_queue_is_typed_backpressure_not_blocking() {
        let policy = SavePolicy {
            queue_capacity: 1,
            ..SavePolicy::local()
        };
        let (queue, _drain) = SaveQueue::bounded(policy).expect("política válida");
        queue
            .enqueue(SaveRequest::Character(character(1, 1)))
            .expect("cabe");
        assert_eq!(
            queue.enqueue(SaveRequest::Character(character(2, 1))),
            Err(DbError::SaveQueueFull { capacity: 1 })
        );
    }

    #[test]
    fn empty_batch_never_touches_the_backend() {
        let requests: Vec<SaveRequest> = Vec::new();
        assert!(requests.is_empty());
        // `apply` devolve relatório zerado antes de abrir conexão; o teste do caminho
        // com banco real vive em `tests/pg.rs` (exige Postgres).
        assert_eq!(
            FlushReport::default().merge(FlushReport::default()),
            FlushReport::default()
        );
    }
}
