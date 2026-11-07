//! Grade espacial: célula fixa, consulta de vizinhança O(1) por célula.
//!
//! O mundo é dividido em células quadradas de lado `cell`; cada consulta
//! toca só as células do quadrado envolvente do raio (varredura limitada,
//! sem varrer o mundo). Inserção/remoção são O(1) amortizado. Travessia entre
//! zonas (thread-per-core futuro): a célula é a unidade de migração — mover
//! entidade de zona = `remove` + `insert`, com a posição antiga sempre
//! conhecida pelo chamador (sem busca).

use crate::ecs::Entity;
use crate::error::CoreError;

/// Dimensões do mundo + lado da célula, em unidades de mapa. Tudo > 0.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct GridConfig {
    pub width: i32,
    pub height: i32,
    pub cell: i32,
}

impl GridConfig {
    pub fn validated(width: i32, height: i32, cell: i32) -> Result<Self, CoreError> {
        if width <= 0 {
            return Err(CoreError::InvalidGrid {
                field: "width",
                value: width,
            });
        }
        if height <= 0 {
            return Err(CoreError::InvalidGrid {
                field: "height",
                value: height,
            });
        }
        if cell <= 0 {
            return Err(CoreError::InvalidGrid {
                field: "cell",
                value: cell,
            });
        }
        Ok(Self {
            width,
            height,
            cell,
        })
    }
}

/// Particionamento espacial por célula fixa.
#[derive(Debug)]
pub struct Grid {
    cols: usize,
    rows: usize,
    cell: i32,
    width: i32,
    height: i32,
    cells: Vec<Vec<Entity>>,
}

impl Grid {
    /// Constrói a grade vazia (fronteira fria; aloca as células uma vez).
    pub fn new(config: GridConfig) -> Self {
        // Teto da divisão em i64 (`div_ceil` de i32 ainda é instável no toolchain).
        let cols = (i64::from(config.width) + i64::from(config.cell) - 1) / i64::from(config.cell);
        let rows = (i64::from(config.height) + i64::from(config.cell) - 1) / i64::from(config.cell);
        let (cols, rows) = (cols as usize, rows as usize);
        let mut cells = Vec::with_capacity(cols * rows);
        cells.resize_with(cols * rows, Vec::new);
        Self {
            cols,
            rows,
            cell: config.cell,
            width: config.width,
            height: config.height,
            cells,
        }
    }

    /// Dimensões da grade em células (largura, altura) — base do fatiamento por zona.
    #[must_use]
    pub fn dimensions(&self) -> (usize, usize) {
        (self.cols, self.rows)
    }

    /// Pré-reserva `n` vagas por célula (densidade esperada de guerra; fria).
    pub fn reserve_per_cell(&mut self, n: usize) {
        for c in &mut self.cells {
            c.reserve(n);
        }
    }

    fn cell_of(&self, x: i32, y: i32) -> Result<usize, CoreError> {
        if !(0..self.width).contains(&x) || !(0..self.height).contains(&y) {
            return Err(CoreError::OutOfBounds { x, y });
        }
        let col = (x / self.cell) as usize;
        let row = (y / self.cell) as usize;
        Ok(row * self.cols + col)
    }

    /// Insere a entidade na célula da posição (O(1) amortizado).
    pub fn insert(&mut self, entity: Entity, x: i32, y: i32) -> Result<(), CoreError> {
        let slot = self.cell_of(x, y)?;
        self.cells[slot].push(entity);
        Ok(())
    }

    /// Remove uma ocorrência da entidade na célula da posição antiga.
    pub fn remove(&mut self, entity: Entity, x: i32, y: i32) -> Result<bool, CoreError> {
        let slot = self.cell_of(x, y)?;
        let cell = &mut self.cells[slot];
        if let Some(i) = cell.iter().position(|&e| e == entity) {
            cell.swap_remove(i);
            Ok(true)
        } else {
            Ok(false)
        }
    }

    /// Travessia: remove da célula antiga, insere na nova (zonas futuras incluídas).
    pub fn relocate(
        &mut self,
        entity: Entity,
        from: (i32, i32),
        to: (i32, i32),
    ) -> Result<bool, CoreError> {
        let moved = self.remove(entity, from.0, from.1)?;
        self.insert(entity, to.0, to.1)?;
        Ok(moved)
    }

    /// Vizinhança: chama `f` para cada entidade nas células do quadrado
    /// envolvente do raio (inclui cantos — distância exata é do chamador).
    pub fn for_each_in_radius(&self, x: i32, y: i32, radius: i32, mut f: impl FnMut(Entity)) {
        if radius < 0 || !(0..self.width).contains(&x) || !(0..self.height).contains(&y) {
            return;
        }
        let col0 = ((x - radius).max(0) / self.cell) as usize;
        let col1 = ((x + radius).min(self.width - 1) / self.cell) as usize;
        let row0 = ((y - radius).max(0) / self.cell) as usize;
        let row1 = ((y + radius).min(self.height - 1) / self.cell) as usize;
        for row in row0..=row1 {
            for col in col0..=col1 {
                for &e in &self.cells[row * self.cols + col] {
                    f(e);
                }
            }
        }
    }

    #[cfg(test)]
    fn count_in_radius(&self, x: i32, y: i32, radius: i32) -> usize {
        let mut n = 0;
        self.for_each_in_radius(x, y, radius, |_| n += 1);
        n
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::{ComponentStore, EntityKind, Inventory, Position, Stats};

    const CFG: GridConfig = GridConfig {
        width: 4096,
        height: 4096,
        cell: 128,
    };

    fn stats_ok() -> Stats {
        Stats {
            level: 1,
            hp: 10,
            max_hp: 10,
            mp: 1,
            max_mp: 1,
            attack: 1,
            defense: 1,
        }
    }

    fn entity_at(store: &mut ComponentStore, x: i32, y: i32) -> Entity {
        store
            .spawn(
                EntityKind::Monster,
                Position { x, y },
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido")
    }

    #[test]
    fn config_rejects_non_positive_dimensions() {
        assert!(GridConfig::validated(0, 4096, 128).is_err());
        assert!(GridConfig::validated(4096, -1, 128).is_err());
        assert!(GridConfig::validated(4096, 4096, 0).is_err());
        assert!(GridConfig::validated(4096, 4096, 128).is_ok());
    }

    #[test]
    fn borders_insert_query_and_reject_outside() {
        let mut grid = Grid::new(CFG);
        let mut store = ComponentStore::default();
        let a = entity_at(&mut store, 0, 0);
        let b = entity_at(&mut store, 4095, 4095);
        grid.insert(a, 0, 0).expect("origem válida");
        grid.insert(b, 4095, 4095).expect("canto válido");
        assert_eq!(grid.count_in_radius(0, 0, 10), 1);
        assert_eq!(grid.count_in_radius(4095, 4095, 10), 1);
        assert_eq!(grid.count_in_radius(2048, 2048, 10), 0);
        assert_eq!(
            grid.insert(a, -1, 0),
            Err(CoreError::OutOfBounds { x: -1, y: 0 })
        );
        assert_eq!(
            grid.insert(a, 4096, 0),
            Err(CoreError::OutOfBounds { x: 4096, y: 0 })
        );
    }

    #[test]
    fn relocate_moves_between_cells() {
        let mut grid = Grid::new(CFG);
        let mut store = ComponentStore::default();
        let e = entity_at(&mut store, 10, 10);
        grid.insert(e, 10, 10).expect("inserção válida");
        assert!(grid
            .relocate(e, (10, 10), (500, 10))
            .expect("travessia válida"));
        assert_eq!(grid.count_in_radius(10, 10, 10), 0);
        assert_eq!(grid.count_in_radius(500, 10, 10), 1);
        assert!(!grid
            .relocate(e, (10, 10), (600, 10))
            .expect("travessia válida"));
        assert_eq!(grid.count_in_radius(600, 10, 10), 1);
    }
}
