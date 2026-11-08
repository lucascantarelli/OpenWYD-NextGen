//! Movimento server-authoritative: o cliente pede um destino, o core decide se é possível.
//!
//! Regra (FASE-03-5): a posição gravada é sempre a do servidor. O pedido traz **destino**;
//! distância, orçamento por tempo e limites do mundo são conferidos aqui. Teleporte (andar mais
//! tiles do que o tempo decorrido permite) e destino fora do mapa são recusados com erro
//! tipado, **sem** mover a entidade e **sem** publicar evento.
//!
//! Os limites vêm de [`MoveRules`] (configuração do servidor), nunca do pacote do cliente.
//! Nenhuma constante de velocidade é inventada neste arquivo: quem monta o mundo informa os
//! tiles por segundo do jogo.

use crate::ecs::components::{ComponentStore, Entity, Position};
use crate::ecs::resources::DomainEvent;
use crate::error::CoreError;

/// Orçamento de movimento: velocidade autorizada + retângulo do mundo.
///
/// `tiles_per_second` é a **única** fonte de velocidade; a distância permitida num pedido é
/// `tiles_per_second × elapsed_ms / 1000`. `min`/`max` são inclusivos e formam o retângulo
/// válido do mapa (o legado usa coordenadas em tiles, eixo Y para baixo).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct MoveRules {
    /// Tiles que a entidade pode andar em um segundo.
    pub tiles_per_second: u16,
    /// Canto superior esquerdo válido (inclusivo).
    pub min: Position,
    /// Canto inferior direito válido (inclusivo).
    pub max: Position,
}

impl MoveRules {
    /// Configuração válida ou erro explícito — velocidade zero nunca vira "sem movimento".
    pub fn new(tiles_per_second: u16, min: Position, max: Position) -> Result<Self, CoreError> {
        if tiles_per_second == 0 {
            return Err(CoreError::InvalidMoveRules {
                field: "tiles_per_second",
                value: 0,
            });
        }
        if min.x > max.x {
            return Err(CoreError::InvalidMoveRules {
                field: "x",
                value: (max.x - min.x).unsigned_abs(),
            });
        }
        if min.y > max.y {
            return Err(CoreError::InvalidMoveRules {
                field: "y",
                value: (max.y - min.y).unsigned_abs(),
            });
        }
        Ok(Self {
            tiles_per_second,
            min,
            max,
        })
    }

    /// Distância permitida para `elapsed_ms` de jogo (arredonda para baixo).
    #[must_use]
    pub fn budget(&self, elapsed_ms: u64) -> u64 {
        u64::from(self.tiles_per_second) * elapsed_ms / 1_000
    }

    /// O destino está dentro do retângulo válido?
    #[must_use]
    pub fn contains(&self, to: Position) -> bool {
        (self.min.x..=self.max.x).contains(&to.x) && (self.min.y..=self.max.y).contains(&to.y)
    }
}

/// Distância em tiles do movimento (Chebyshev: diagonal custa o mesmo que reta).
#[must_use]
pub fn distance(from: Position, to: Position) -> u32 {
    (to.x - from.x)
        .unsigned_abs()
        .max((to.y - from.y).unsigned_abs())
}

/// Aplica um passo autorizado e devolve a posição gravada.
///
/// Ordem (transação): entidade viva → orçamento de tempo → limites do mundo → distância → espaço
/// na fila → evento → gravação. Qualquer recusa deixa posição e fila como estavam.
///
/// A API **não** aceita posição "confirmada pelo cliente" nem velocidade: só o destino pedido,
/// o tempo decorrido (do tick) e as regras do servidor.
pub fn step(
    store: &mut ComponentStore,
    entity: Entity,
    to: Position,
    elapsed_ms: u64,
    rules: MoveRules,
) -> Result<Position, CoreError> {
    if !store.is_alive(entity) {
        return Err(CoreError::EntityNotAlive {
            index: entity.index,
            generation: entity.generation,
        });
    }
    let from = store
        .position_of(entity)
        .expect("entidade viva tem posição no SoA");
    let allowed = rules.budget(elapsed_ms);
    if allowed == 0 {
        return Err(CoreError::NoMoveBudget { elapsed_ms });
    }
    if !rules.contains(to) {
        return Err(CoreError::OutOfBounds { x: to.x, y: to.y });
    }
    let distance = distance(from, to);
    if u64::from(distance) > allowed {
        return Err(CoreError::MoveTooFar {
            distance,
            allowed: u32::try_from(allowed).unwrap_or(u32::MAX),
        });
    }
    store.publish(DomainEvent::Moved { entity, from, to })?;
    store.commit_position(entity, to);
    Ok(to)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{EntityKind, Inventory, Stats};

    fn stats() -> Stats {
        Stats {
            level: 1,
            hp: 10,
            max_hp: 10,
            mp: 0,
            max_mp: 0,
            attack: 1,
            defense: 1,
        }
    }

    fn store_com(x: i32, y: i32) -> (ComponentStore, Entity) {
        let mut store = ComponentStore::default();
        let e = store
            .spawn(
                EntityKind::Player,
                Position { x, y },
                stats(),
                Inventory::default(),
            )
            .expect("spawn válido");
        (store, e)
    }

    fn regras() -> MoveRules {
        MoveRules::new(4, Position { x: 0, y: 0 }, Position { x: 100, y: 100 }).expect("regras")
    }

    #[test]
    fn orcamento_e_chebyshev() {
        let regras = regras();
        assert_eq!(regras.budget(500), 2);
        assert_eq!(regras.budget(250), 1);
        assert_eq!(regras.budget(200), 0);
        assert_eq!(
            distance(Position { x: 3, y: 4 }, Position { x: 5, y: 8 }),
            4
        );
    }

    #[test]
    fn passo_valido_movimenta_e_publica_um_evento() {
        let (mut store, e) = store_com(10, 10);
        let destino = Position { x: 12, y: 12 };
        assert_eq!(step(&mut store, e, destino, 500, regras()), Ok(destino));
        assert_eq!(store.position_of(e), Some(destino));
        let eventos = store.drain_events();
        assert_eq!(
            eventos,
            vec![DomainEvent::Moved {
                entity: e,
                from: Position { x: 10, y: 10 },
                to: destino
            }]
        );
    }

    #[test]
    fn teleporte_e_recusado_sem_mover_nem_publicar() {
        let (mut store, e) = store_com(10, 10);
        let pedido = Position { x: 90, y: 10 };
        assert_eq!(
            step(&mut store, e, pedido, 500, regras()),
            Err(CoreError::MoveTooFar {
                distance: 80,
                allowed: 2
            })
        );
        assert_eq!(store.position_of(e), Some(Position { x: 10, y: 10 }));
        assert!(store.events().is_empty(), "recusa não publica evento");
    }

    #[test]
    fn destino_fora_do_mundo_e_recusado() {
        let (mut store, e) = store_com(0, 0);
        assert_eq!(
            step(&mut store, e, Position { x: -1, y: 0 }, 5_000, regras()),
            Err(CoreError::OutOfBounds { x: -1, y: 0 })
        );
        assert_eq!(store.position_of(e), Some(Position { x: 0, y: 0 }));
    }

    #[test]
    fn tempo_insuficiente_para_um_tile_e_recusa() {
        let (mut store, e) = store_com(10, 10);
        assert_eq!(
            step(&mut store, e, Position { x: 11, y: 10 }, 200, regras()),
            Err(CoreError::NoMoveBudget { elapsed_ms: 200 })
        );
    }

    #[test]
    fn regras_invalidas_falham_no_boot() {
        assert_eq!(
            MoveRules::new(0, Position::default(), Position::default()),
            Err(CoreError::InvalidMoveRules {
                field: "tiles_per_second",
                value: 0
            })
        );
        assert!(MoveRules::new(4, Position { x: 10, y: 0 }, Position { x: 5, y: 0 }).is_err());
    }
}
