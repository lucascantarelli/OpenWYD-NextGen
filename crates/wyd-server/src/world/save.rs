//! Submódulo de `world` (ADR-0026, etapa 3).

use super::*;

impl Engine {
    /// Foto de todos os personagens vivos (save periódico).
    #[must_use]
    pub fn snapshots(&self) -> Vec<LiveState> {
        let mut estados: Vec<LiveState> = self.live.values().copied().collect();
        estados.sort_unstable_by_key(|state| state.character_id);
        estados
    }
}

/// Destino do save (implementado em `main` sobre `SaveQueue`; teste injeta um coletor).
pub trait SaveSink: Send + Sync {
    /// Enfileira o snapshot; falha é registrada, nunca bloqueia a borda.
    fn save(&self, state: LiveState);
}

/// Save real: converte o estado vivo no `Character` que o `SaveDrain` grava.
#[derive(Debug)]
pub struct SaveQueueSink {
    queue: Arc<SaveQueue>,
}

impl SaveQueueSink {
    #[must_use]
    pub fn new(queue: Arc<SaveQueue>) -> Self {
        Self { queue }
    }
}

impl SaveSink for SaveQueueSink {
    fn save(&self, state: LiveState) {
        let character = Character {
            id: state.character_id,
            level: i64::from(state.level),
            cp: i64::from(state.cp),
            exp: state.exp,
            hp: i64::from(state.hp),
            mp: i64::from(state.mp),
            pos_x: i64::from(state.x),
            pos_y: i64::from(state.y),
            ..Character::default()
        };
        // Ouro **não** entra no save do mundo: quem escreve `gold` é o claim de grant, e
        // sobrescrever aqui poderia regredir uma concessão já auditada (FASE-07-4).
        if let Err(error) = self.queue.enqueue(SaveRequest::Character(character)) {
            println!(
                "level=warn event=save result=enqueue_failed character={} reason={error}",
                state.character_id
            );
        }
    }
}
