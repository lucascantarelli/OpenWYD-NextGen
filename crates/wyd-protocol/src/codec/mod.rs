//! Codec: transformação `pKeyWord`, checksum e tabela gerada.
//!
//! Só declara e re-exporta; a lógica mora em cada módulo.

pub mod checksum;
pub mod generated;
pub mod keyword;

pub use checksum::{verify as verify_checksum, ChecksumPolicy};
pub use keyword::{decode_in_place, encode_in_place, TransformError, KEYWORD_OFFSET};
