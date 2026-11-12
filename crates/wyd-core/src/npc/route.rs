//! Rota de patrulha — hoje **ancorada**, porque `route_type`/`generator_data` é UNVERIFIED.
//!
//! `docs/arquitetura/ia-bots.md` §9.2: a semântica de rota do `NPCGener` só se confirma com o
//! trace do cliente (CLIENT-06). Até lá o NPC patrulha dentro de um raio da âncora, sem inventar
//! waypoint que o jogo original não tem.

use crate::ecs::components::Position;

/// Rota ancorada: posição de spawn + raio de patrulha em tiles.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Route {
    /// Âncora (spawn).
    pub anchor: Position,
    /// Raio máximo em tiles (0 = ponto fixo).
    pub radius: u32,
}

impl Route {
    /// Rota parada no spawn (`radius = 0`).
    #[must_use]
    pub fn anchor_only(anchor: Position) -> Self {
        Self { anchor, radius: 0 }
    }

    /// Mantém o destino dentro do quadrado da âncora (recorte determinístico, sem RNG).
    #[must_use]
    pub fn clamp(&self, to: Position) -> Position {
        let r = i32::try_from(self.radius).unwrap_or(i32::MAX);
        Position {
            x: to.x.clamp(self.anchor.x - r, self.anchor.x + r),
            y: to.y.clamp(self.anchor.y - r, self.anchor.y + r),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rota_ancorada_nao_sai_do_raio() {
        let rota = Route {
            anchor: Position { x: 100, y: 100 },
            radius: 5,
        };
        assert_eq!(
            rota.clamp(Position { x: 120, y: 90 }),
            Position { x: 105, y: 95 }
        );
        assert_eq!(
            Route::anchor_only(Position { x: 1, y: 1 }).clamp(Position { x: 9, y: 9 }),
            Position { x: 1, y: 1 }
        );
    }
}
