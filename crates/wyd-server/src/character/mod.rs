//! Personagens do canal: CRUD/seleção, snapshot de wire e sessão única por conta.
//!
//! Árvore da ADR-0026 (etapa 4): `crud` (Create/Delete/CharacterLogin e
//! `SelectionStore`), `snapshot` (`STRUCT_SELCHAR`/`STRUCT_MOB`) e `session` (cerca
//! `character_session` entre canais). Nenhuma regra vive neste arquivo.

mod crud;
pub mod session;
pub mod snapshot;

pub use crud::*;
pub use session::*;
pub use snapshot::*;
