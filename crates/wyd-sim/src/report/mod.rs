//! Relatórios comparáveis — a saída do laboratório, sem `f64` no meio da regra.
//!
//! Existe para responder "o que muda se..." com texto diff-ável, não com print solto. A
//! tabela é determinística (mesma entrada → mesmas colunas alinhadas), então também serve de
//! golden de formatação.

pub mod table;

pub use table::{damage_by_defense, Table};
