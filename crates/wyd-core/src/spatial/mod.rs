//! Particionamento espacial: grade de célula fixa (visibilidade O(1) por célula).
//!
//! Só declara e re-exporta; a grade mora em `grid`.

pub mod grid;

pub use grid::{Grid, GridConfig};
