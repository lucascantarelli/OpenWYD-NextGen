//! Autenticação/autorização do control plane.
//!
//! Só declara e re-exporta; sessão e papéis moram nos módulos.

pub mod roles;
pub mod session;

pub use roles::{may_change_role, may_target};
pub use session::{Session, StaffSession};
