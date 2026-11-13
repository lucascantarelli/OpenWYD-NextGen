//! Portas do Hot Path: o core pede (traits), a borda entrega (DIP).
//!
//! Só declara e re-exporta; cada porta mora no seu módulo.

pub mod clock;
pub mod content;

pub use clock::{elapsed_ms, Clock, ManualClock, SystemClock};
pub use content::{
    ContentCounts, ContentError, ContentIssue, ContentSource, ContentTables, ContentVersion,
    ItemRow, SettingRow, SkillRow, MAX_CONTENT_ISSUES,
};
