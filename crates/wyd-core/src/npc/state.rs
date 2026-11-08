//! Estado da criatura em SoA — dado, não regra.
//!
//! `NpcMode` e a tabela de ameaça são a **cópia presa por golden** da matemática de
//! `crates/wyd-sim/src/formula/threat.rs` (D-IA02): o DAG proíbe `wyd-core → wyd-sim`, e a
//! unificação exige contrato em `wyd-protocol` (dívida da FASE-10-1, ADR-0011). O golden de
//! decisão (`tests/npc.rs`) impede que as duas cópias divirjam em silêncio.
//!
//! Nada aqui consulta banco, relógio de parede ou `HashMap`: o estado é `Copy`, indexado por
//! `Entity.index` e salvo no snapshot do mundo.

use crate::ecs::components::Position;
use crate::error::CoreError;

/// Teto de entidades na tabela de ameaça de uma criatura.
pub const MAX_THREAT_ENTRIES: usize = 8;

/// Estado da criatura (FSM de `ADR-0010` §5).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum NpcMode {
    /// Parada no spawn.
    #[default]
    Idle,
    /// Andando na rota (hoje limitada à âncora — `route_type` UNVERIFIED).
    Patrol,
    /// Detectou alvo.
    Aggro,
    /// Perseguindo.
    Chase,
    /// Ao alcance de ataque.
    Attack,
    /// HP baixo: fugindo.
    Flee,
    /// Fora do leash: voltando à âncora.
    Return,
}

impl NpcMode {
    /// Todos os estados, na ordem documentada.
    pub const ALL: [Self; 7] = [
        Self::Idle,
        Self::Patrol,
        Self::Aggro,
        Self::Chase,
        Self::Attack,
        Self::Flee,
        Self::Return,
    ];

    /// Enxerga o alvo a partir da distância e do alcance (comparação inclusiva, inteira).
    #[must_use]
    pub fn should_aggro(distance: u32, aggro_range: u32) -> bool {
        distance <= aggro_range
    }
}

/// Linha da tabela de ameaça.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ThreatEntry {
    /// `Entity.index` de quem gerou ameaça.
    pub entity: u32,
    /// Ameaça acumulada.
    pub threat: u32,
    /// Tick da última contribuição (expiração por TTL de ticks).
    pub last_tick: u32,
}

/// Tabela de ameaça com teto fixo (sem alocação).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct ThreatTable {
    entries: [Option<ThreatEntry>; MAX_THREAT_ENTRIES],
    len: u8,
}

impl ThreatTable {
    /// Tabela vazia.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Quantas entidades na tabela.
    #[must_use]
    pub fn len(&self) -> usize {
        usize::from(self.len)
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.len == 0
    }

    /// Soma ameaça (cria a linha na primeira vez). Tabela cheia é erro, sem mutação.
    pub fn add(&mut self, entity: u32, amount: u32, tick: u32) -> Result<(), CoreError> {
        for entrada in self.entries.iter_mut().flatten() {
            if entrada.entity == entity {
                entrada.threat = entrada.threat.saturating_add(amount);
                entrada.last_tick = tick;
                return Ok(());
            }
        }
        if let Some(vazio) = self.entries.iter_mut().find(|slot| slot.is_none()) {
            *vazio = Some(ThreatEntry {
                entity,
                threat: amount,
                last_tick: tick,
            });
            self.len = self.len.saturating_add(1);
            return Ok(());
        }
        Err(CoreError::ThreatTableFull {
            capacity: MAX_THREAT_ENTRIES,
        })
    }

    /// Ameaça atual de uma entidade.
    #[must_use]
    pub fn threat_of(&self, entity: u32) -> Option<u32> {
        self.entries
            .iter()
            .flatten()
            .find(|entrada| entrada.entity == entity)
            .map(|entrada| entrada.threat)
    }

    /// Alvo: maior ameaça; empate pelo **menor `entity`**. Ignora linhas expiradas.
    #[must_use]
    pub fn select_target(&self, now_tick: u32, ttl_ticks: u32) -> Option<u32> {
        let mut melhor: Option<ThreatEntry> = None;
        for entrada in self.entries.iter().flatten() {
            if ttl_ticks > 0 && now_tick.saturating_sub(entrada.last_tick) > ttl_ticks {
                continue;
            }
            let vence = match melhor {
                None => true,
                Some(atual) => {
                    entrada.threat > atual.threat
                        || (entrada.threat == atual.threat && entrada.entity < atual.entity)
                }
            };
            if vence {
                melhor = Some(*entrada);
            }
        }
        melhor.map(|entrada| entrada.entity)
    }

    /// Remove linhas mais velhas que `ttl_ticks` (0 desliga); devolve quantas saíram.
    pub fn prune(&mut self, now_tick: u32, ttl_ticks: u32) -> usize {
        if ttl_ticks == 0 {
            return 0;
        }
        let mut removidos = 0usize;
        for slot in &mut self.entries {
            let expirou = match slot {
                Some(entrada) => now_tick.saturating_sub(entrada.last_tick) > ttl_ticks,
                None => false,
            };
            if expirou {
                *slot = None;
                removidos += 1;
            }
        }
        self.len = self.len.saturating_sub(removidos as u8);
        removidos
    }
}

/// Estado de uma criatura (SoA por índice de entidade).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct NpcState {
    /// Estado atual da FSM.
    pub mode: NpcMode,
    /// Alvo atual (`Entity.index`), se houver.
    pub target: Option<u32>,
    /// Âncora de spawn (leash/retorno).
    pub anchor: Position,
    /// Tabela de ameaça.
    pub threat: ThreatTable,
    /// Estado do RNG (`xorshift64`), semeado por `(world_seed, entity_index)`.
    pub rng: u64,
    /// Próximo tick em que pode decidir (cadência de 5 Hz).
    pub next_decision_tick: u32,
}

impl NpcState {
    /// Estado inicial de uma criatura: âncora no spawn, RNG determinístico, sem alvo.
    #[must_use]
    pub fn spawn(anchor: Position, world_seed: u64, entity_index: u32) -> Self {
        // Mistura estável: dois NPCs do mesmo mundo nunca compartilham sequência.
        let semente = world_seed
            ^ (u64::from(entity_index).wrapping_mul(0x9E37_79B9_7F4A_7C15))
            ^ 0xD1B5_4A32_D192_ED03;
        Self {
            mode: NpcMode::Idle,
            target: None,
            anchor,
            threat: ThreatTable::new(),
            rng: if semente == 0 { 1 } else { semente },
            next_decision_tick: 0,
        }
    }

    /// Próximo valor do `xorshift64` (determinístico, sem dependência externa).
    pub fn next_random(&mut self) -> u64 {
        let mut x = self.rng;
        x ^= x << 13;
        x ^= x >> 7;
        x ^= x << 17;
        self.rng = x;
        x
    }
}

/// Vetor contíguo de estados de criatura, indexado por `Entity.index`.
///
/// O dono é o estado do mundo (`world/state.rs`, reservado); o sistema recebe `&mut` e nunca
/// aloca no caminho quente, porque o vetor cresce só no spawn (chamada fria).
#[derive(Debug, Default)]
pub struct NpcStates {
    states: Vec<NpcState>,
}

impl NpcStates {
    /// Garante um estado para o índice (spawn frio) e devolve referência mutável.
    ///
    /// O primeiro `ensure` de um índice fixa a âncora na posição de spawn e semeia o RNG por
    /// `(world_seed, index)`; chamadas seguintes não reinicializam alvo/ameaça/modo.
    pub fn ensure(&mut self, index: u32, anchor: Position, world_seed: u64) -> &mut NpcState {
        while self.states.len() <= index as usize {
            let slot = self.states.len() as u32;
            self.states
                .push(NpcState::spawn(Position::default(), world_seed, slot));
        }
        let i = index as usize;
        if self.states[i].anchor == Position::default() {
            self.states[i] = NpcState::spawn(anchor, world_seed, index);
        }
        &mut self.states[i]
    }

    /// Estado de um índice, se existir.
    #[must_use]
    pub fn get(&self, index: u32) -> Option<&NpcState> {
        self.states.get(index as usize)
    }

    /// Estado mutável de um índice, se existir.
    pub fn get_mut(&mut self, index: u32) -> Option<&mut NpcState> {
        self.states.get_mut(index as usize)
    }

    /// Quantos estados existem (índices já vistos no spawn).
    #[must_use]
    pub fn len(&self) -> usize {
        self.states.len()
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.states.is_empty()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ameaca_acumula_e_desempata_pelo_menor_entity() {
        let mut tabela = ThreatTable::new();
        tabela.add(9, 100, 1).expect("cabe");
        tabela.add(9, 50, 2).expect("acumula");
        tabela.add(7, 150, 2).expect("cabe");
        assert_eq!(tabela.threat_of(9), Some(150));
        assert_eq!(tabela.select_target(2, 0), Some(7), "empate → menor entity");

        let mut invertida = ThreatTable::new();
        invertida.add(7, 150, 2).expect("cabe");
        invertida.add(9, 150, 2).expect("cabe");
        assert_eq!(
            invertida.select_target(2, 0),
            Some(7),
            "ordem de inserção não decide"
        );
    }

    #[test]
    fn tabela_cheia_recusa_sem_mutar() {
        let mut tabela = ThreatTable::new();
        for entity in 0..MAX_THREAT_ENTRIES as u32 {
            tabela.add(entity, 1, 1).expect("cabe");
        }
        assert_eq!(
            tabela.add(99, 1, 1),
            Err(CoreError::ThreatTableFull {
                capacity: MAX_THREAT_ENTRIES
            })
        );
        assert_eq!(tabela.len(), MAX_THREAT_ENTRIES);
    }

    #[test]
    fn ameaca_expira_por_ticks() {
        let mut tabela = ThreatTable::new();
        tabela.add(1, 10, 1).expect("cabe");
        tabela.add(2, 20, 100).expect("cabe");
        assert_eq!(tabela.select_target(100, 50), Some(2));
        assert_eq!(tabela.prune(100, 50), 1);
        assert_eq!(tabela.threat_of(1), None);
    }

    #[test]
    fn rng_e_deterministico_por_entidade() {
        let a = NpcState::spawn(Position { x: 5, y: 5 }, 42, 3);
        let b = NpcState::spawn(Position { x: 5, y: 5 }, 42, 3);
        assert_eq!(a, b, "mesmo seed e índice → mesmo estado");

        let mut x = a;
        let mut y = b;
        let sequencia_x: Vec<u64> = (0..4).map(|_| x.next_random()).collect();
        let sequencia_y: Vec<u64> = (0..4).map(|_| y.next_random()).collect();
        assert_eq!(sequencia_x, sequencia_y);
        assert!(sequencia_x.windows(2).all(|par| par[0] != par[1]));

        let outro = NpcState::spawn(Position { x: 5, y: 5 }, 42, 4);
        assert_ne!(a.rng, outro.rng, "índice diferente → sequência diferente");
    }

    #[test]
    fn aggro_inclusivo_e_sete_estados() {
        assert!(NpcMode::should_aggro(10, 10));
        assert!(!NpcMode::should_aggro(11, 10));
        assert_eq!(NpcMode::ALL.len(), 7);
    }
}
