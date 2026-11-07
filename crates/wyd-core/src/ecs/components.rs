//! Componentes contíguos das entidades — dados, sem lógica de jogo.
//!
//! Layout DOD: cada componente mora num `Vec` próprio dentro de
//! [`ComponentStore`]; sistemas varrem fatias (`&[T]` / `&mut [T]`), nunca
//! objetos espalhados no heap. Fórmulas e regras vivem na FASE-10; aqui só a
//! forma dos dados + invariantes estruturais verificadas no spawn.

use crate::error::CoreError;

/// Posição no plano do mundo (unidades do mapa legado, eixo Y para baixo).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Position {
    pub x: i32,
    pub y: i32,
}

/// Atributos vitais e de combate (valores brutos; fórmulas na FASE-10).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Stats {
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    pub mp: u32,
    pub max_mp: u32,
    pub attack: u32,
    pub defense: u32,
}

impl Stats {
    /// Invariantes estruturais mínimas — falha explícita, nunca default silencioso.
    pub fn validate(&self) -> Result<(), CoreError> {
        if self.level == 0 {
            return Err(CoreError::InvalidStats {
                field: "level",
                value: 0,
            });
        }
        if self.max_hp == 0 {
            return Err(CoreError::InvalidStats {
                field: "max_hp",
                value: 0,
            });
        }
        if self.hp > self.max_hp {
            return Err(CoreError::InvalidStats {
                field: "hp",
                value: self.hp,
            });
        }
        if self.mp > self.max_mp {
            return Err(CoreError::InvalidStats {
                field: "mp",
                value: self.mp,
            });
        }
        Ok(())
    }
}

/// Um item empilhável (forma; semântica de efeitos na FASE-10/economia).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Item {
    pub id: u32,
    pub qty: u32,
}

/// Dimensões do `STRUCT_MOB` legado (confirmar contra `data-formats.md` em 03-4).
pub const EQUIP_SLOTS: usize = 16;
pub const BAG_SLOTS: usize = 64;

/// Inventário de uma entidade: equipamento + mochila em arrays fixos (sem alloc).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Inventory {
    pub equip: [Item; EQUIP_SLOTS],
    pub bag: [Item; BAG_SLOTS],
}

impl Default for Inventory {
    fn default() -> Self {
        Self {
            equip: [Item::default(); EQUIP_SLOTS],
            bag: [Item::default(); BAG_SLOTS],
        }
    }
}

/// As 4 espécies de entidade do mundo (03-2: sem hierarquia OOP).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EntityKind {
    Player,
    Npc,
    Monster,
    GroundItem,
}

/// Identidade estável: índice SoA + geração (detecta uso-após-despawn).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Entity {
    pub index: u32,
    pub generation: u32,
}

/// Armazém SoA: um `Vec` contíguo por componente + vida/geração por índice.
///
/// Spawn/despawn aqui são operação de estrutura de dados (fronteira fria);
/// o caminho quente só itera as fatias. Spawn valida `Stats` e nunca inventa
/// valores: dado ruim é `Err`, não correção silenciosa.
#[derive(Debug, Default)]
pub struct ComponentStore {
    kind: Vec<EntityKind>,
    position: Vec<Position>,
    stats: Vec<Stats>,
    inventory: Vec<Inventory>,
    alive: Vec<bool>,
    generation: Vec<u32>,
    free: Vec<u32>,
}

impl ComponentStore {
    /// Reserva espaço para `n` entidades sem inicializar (chamada fria).
    pub fn reserve(&mut self, n: usize) {
        self.kind.reserve(n);
        self.position.reserve(n);
        self.stats.reserve(n);
        self.inventory.reserve(n);
        self.alive.reserve(n);
        self.generation.reserve(n);
    }

    /// Número de índices (vivos + mortos com geração antiga).
    #[must_use]
    pub fn len(&self) -> usize {
        self.alive.len()
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.alive.is_empty()
    }

    /// Entidades vivas agora.
    #[must_use]
    pub fn live_count(&self) -> usize {
        self.alive.iter().filter(|&&a| a).count()
    }

    /// Spawn com validação estrutural; reutiliza índice liberado quando há.
    pub fn spawn(
        &mut self,
        kind: EntityKind,
        position: Position,
        stats: Stats,
        inventory: Inventory,
    ) -> Result<Entity, CoreError> {
        stats.validate()?;
        if let Some(index) = self.free.pop() {
            let i = index as usize;
            self.generation[i] += 1;
            self.kind[i] = kind;
            self.position[i] = position;
            self.stats[i] = stats;
            self.inventory[i] = inventory;
            self.alive[i] = true;
            Ok(Entity {
                index,
                generation: self.generation[i],
            })
        } else {
            let index = self.alive.len() as u32;
            self.kind.push(kind);
            self.position.push(position);
            self.stats.push(stats);
            self.inventory.push(inventory);
            self.alive.push(true);
            self.generation.push(0);
            Ok(Entity {
                index,
                generation: 0,
            })
        }
    }

    /// Despawn idempotente quanto a índice morto; geração nova invalida o id antigo.
    pub fn despawn(&mut self, entity: Entity) -> bool {
        let i = entity.index as usize;
        if i < self.alive.len() && self.alive[i] && self.generation[i] == entity.generation {
            self.alive[i] = false;
            self.free.push(entity.index);
            true
        } else {
            false
        }
    }

    /// Varredura em lote (caminho quente): posições contíguas, sem alloc.
    #[must_use]
    pub fn positions(&self) -> &[Position] {
        &self.position
    }

    /// Varredura mutável em lote (caminho quente): um sistema por vez (`&mut` exclusivo).
    #[must_use]
    pub fn positions_mut(&mut self) -> &mut [Position] {
        &mut self.position
    }

    #[must_use]
    pub fn stats(&self) -> &[Stats] {
        &self.stats
    }

    #[must_use]
    pub fn is_alive(&self, entity: Entity) -> bool {
        let i = entity.index as usize;
        i < self.alive.len() && self.alive[i] && self.generation[i] == entity.generation
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn stats_ok() -> Stats {
        Stats {
            level: 1,
            hp: 100,
            max_hp: 100,
            mp: 10,
            max_mp: 10,
            attack: 5,
            defense: 3,
        }
    }

    #[test]
    fn spawn_rejects_bad_stats_explicitly() {
        let mut store = ComponentStore::default();
        let zero_hp = Stats {
            max_hp: 0,
            ..stats_ok()
        };
        assert_eq!(
            store.spawn(
                EntityKind::Player,
                Position { x: 0, y: 0 },
                zero_hp,
                Inventory::default()
            ),
            Err(CoreError::InvalidStats {
                field: "max_hp",
                value: 0
            })
        );
        let over_hp = Stats {
            hp: 200,
            ..stats_ok()
        };
        assert!(store
            .spawn(
                EntityKind::Player,
                Position { x: 0, y: 0 },
                over_hp,
                Inventory::default()
            )
            .is_err());
        assert_eq!(store.live_count(), 0);
    }

    #[test]
    fn spawn_reuses_slot_with_new_generation() {
        let mut store = ComponentStore::default();
        let a = store
            .spawn(
                EntityKind::Monster,
                Position { x: 1, y: 2 },
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert!(store.despawn(a));
        assert!(!store.is_alive(a));
        let b = store
            .spawn(
                EntityKind::Npc,
                Position { x: 3, y: 4 },
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert_eq!((b.index, store.len()), (a.index, 1));
        assert_ne!(a.generation, b.generation);
        assert_eq!(store.live_count(), 1);
    }

    #[test]
    fn stale_entity_never_despawns_twice() {
        let mut store = ComponentStore::default();
        let a = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert!(store.despawn(a));
        assert!(!store.despawn(a));
    }
}
