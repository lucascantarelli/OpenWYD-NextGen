//! Autenticação/autorização do control plane.
//!
//! Só declara e re-exporta; sessão e papéis moram nos módulos.

pub mod password;
pub mod roles;
pub mod session;

pub use password::{hash_password, hash_secret, verify_password, PasswordError, MIN_PASSWORD_LEN};
pub use roles::{may_change_role, may_target};
pub use session::{Session, StaffSession};
