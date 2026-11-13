//! Aggro e estado de criatura — utilidade inteira, sem I/O, com desempate determinístico.
//!
//! Esta é a **matemática** da camada A de `docs/arquitetura/adr/0010-ia-npc-e-bots.md`: a
//! tabela de ameaça e a escolha de alvo. A máquina de estados (`idle → patrol → aggro →
//! chase → attack → flee → return`) e a cadência de 5 Hz vivem em `crates/wyd-core/src/npc/`
//! (GAME-06), que consome estas funções puras — a mesma função roda no laboratório.
//!
//! **Hipótese (D-09):** ameaça = dano causado, tabela com teto fixo, alvo = maior ameaça e,
//! no empate, **menor `entity`**. Sem desempate por ordem de chegada ou por `HashMap`, dois
//! replays com a mesma sequência de golpes escolhem o mesmo alvo — é o que torna o golden de
//! decisão possível (`docs/rust/07-ia-e-bots.md` §8).

use crate::error::SimError;

/// Teto de entidades na tabela de ameaça de uma criatura.
pub const MAX_THREAT_ENTRIES: usize = 8;

/// Linha da tabela: quem gerou ameaça, quanta e em que tick.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ThreatEntry {
    /// Índice da entidade que ameaça.
    pub entity: u32,
    /// Ameaça acumulada.
    pub threat: u32,
    /// Tick da última contribuição (para expiração).
    pub last_tick: u32,
}

/// Tabela de ameaça de uma criatura.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ThreatTable {
    entries: [Option<ThreatEntry>; MAX_THREAT_ENTRIES],
    len: u8,
}

impl Default for ThreatTable {
    fn default() -> Self {
        Self {
            entries: [None; MAX_THREAT_ENTRIES],
            len: 0,
        }
    }
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

    /// Tabela sem ninguém.
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.len == 0
    }

    /// Soma ameaça de uma entidade (cria a linha se for a primeira vez).
    ///
    /// # Errors
    /// [`SimError::ThreatTableFull`] com a tabela cheia; nada é alterado.
    pub fn add(&mut self, entity: u32, amount: u32, tick: u32) -> Result<(), SimError> {
        for entrada in self.entries.iter_mut().flatten() {
            if entrada.entity == entity {
                entrada.threat = entrada.threat.saturating_add(amount);
                entrada.last_tick = tick;
                return Ok(());
            }
        }
        if let Some(vazio) = self.entries.iter_mut().find(|s| s.is_none()) {
            *vazio = Some(ThreatEntry {
                entity,
                threat: amount,
                last_tick: tick,
            });
            self.len = self.len.saturating_add(1);
            return Ok(());
        }
        Err(SimError::ThreatTableFull {
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

    /// Remove linhas mais antigas que `ttl_ticks`; devolve quantas saíram.
    ///
    /// `ttl_ticks = 0` desliga a expiração (ameaça nunca decai).
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
        self.len = self
            .len
            .saturating_sub(u8::try_from(removidos).unwrap_or(u8::MAX));
        removidos
    }

    /// Alvo escolhido: maior ameaça; empate resolvido pelo menor `entity`.
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
}

/// Ameaça gerada por um golpe (provisório: 1 ponto de ameaça por 1 de dano).
#[must_use]
pub fn threat_from_damage(damage: u32) -> u32 {
    damage
}

/// Estado da criatura (a FSM de `ADR-0010` §5, sem a lógica de transição).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NpcMode {
    /// Parada no spawn.
    Idle,
    /// Andando na rota.
    Patrol,
    /// Detectou alvo e vai persegui-lo.
    Aggro,
    /// Perseguindo.
    Chase,
    /// Atacando/solvendo skill.
    Attack,
    /// Fugindo com HP baixo.
    Flee,
    /// Voltando ao spawn.
    Return,
}

impl NpcMode {
    /// Todos os estados, na ordem do ciclo documentado.
    pub const ALL: [Self; 7] = [
        Self::Idle,
        Self::Patrol,
        Self::Aggro,
        Self::Chase,
        Self::Attack,
        Self::Flee,
        Self::Return,
    ];

    /// Enxerga o alvo a partir da distância e do alcance de aggro (comparação inteira).
    #[must_use]
    pub fn should_aggro(distance: u32, aggro_range: u32) -> bool {
        distance <= aggro_range
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn threat_accumulates_per_entity() {
        let mut tabela = ThreatTable::new();
        tabela.add(7, 100, 1).expect("cabe");
        tabela.add(7, 50, 2).expect("acumula");
        tabela.add(9, 120, 3).expect("cabe");
        assert_eq!(tabela.threat_of(7), Some(150));
        assert_eq!(tabela.threat_of(9), Some(120));
        assert_eq!(tabela.len(), 2);
    }

    #[test]
    fn highest_threat_wins() {
        let mut tabela = ThreatTable::new();
        tabela.add(7, 100, 1).expect("cabe");
        tabela.add(9, 120, 1).expect("cabe");
        assert_eq!(tabela.select_target(1, 0), Some(9));
    }

    #[test]
    fn tie_breaks_by_lowest_entity_for_determinism() {
        let mut a = ThreatTable::new();
        a.add(5, 100, 1).expect("cabe");
        a.add(3, 100, 1).expect("cabe");
        assert_eq!(a.select_target(1, 0), Some(3));

        let mut b = ThreatTable::new();
        b.add(3, 100, 1).expect("cabe");
        b.add(5, 100, 1).expect("cabe");
        assert_eq!(
            b.select_target(1, 0),
            Some(3),
            "ordem de inserção não decide"
        );
    }

    #[test]
    fn stale_entries_are_ignored_and_pruned() {
        let mut tabela = ThreatTable::new();
        tabela.add(7, 100, 1).expect("cabe");
        tabela.add(9, 200, 100).expect("cabe");
        assert_eq!(tabela.select_target(100, 50), Some(9));
        assert_eq!(tabela.prune(100, 50), 1);
        assert_eq!(tabela.threat_of(7), None);
        assert_eq!(tabela.len(), 1);
    }

    #[test]
    fn full_table_refuses_without_mutating() {
        let mut tabela = ThreatTable::new();
        for entity in 0..MAX_THREAT_ENTRIES {
            tabela
                .add(u32::try_from(entity).expect("cabe"), 1, 1)
                .expect("cabe");
        }
        assert_eq!(
            tabela.add(999, 1, 1),
            Err(SimError::ThreatTableFull {
                capacity: MAX_THREAT_ENTRIES
            })
        );
        assert_eq!(tabela.len(), MAX_THREAT_ENTRIES);
    }

    #[test]
    fn empty_table_has_no_target() {
        assert_eq!(ThreatTable::new().select_target(1, 0), None);
    }

    #[test]
    fn aggro_range_is_inclusive() {
        assert!(NpcMode::should_aggro(10, 10));
        assert!(!NpcMode::should_aggro(11, 10));
        assert_eq!(NpcMode::ALL.len(), 7);
    }

    #[test]
    fn threat_equals_damage_by_provisional_rule() {
        assert_eq!(threat_from_damage(37), 37);
    }
}
