//! Auditoria de ações administrativas (quem/o quê/quando).
//!
//! Só declara e re-exporta; o registro e os destinos moram em `log`.

pub mod log;

pub use log::{log_action, log_denied, log_panel, AuditEntry, AuditSink, VecSink};
