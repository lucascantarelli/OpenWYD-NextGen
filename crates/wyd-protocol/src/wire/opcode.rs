//! Catálogo de `Type`: `Opcode`, direção (`Flow`) e lookup.
//!
//! Todo o conteúdo é GERADO de `.deprecated-go/Source/Code/Basedef.h` por
//! `tools/protocol/gen-protocol.mjs` — a fonte canônica vive em
//! [`super::generated::opcode_table`]; aqui só há re-export com documentação.

pub use super::generated::opcode_table::{
    Flow, Opcode, FLAG_CLIENT2GAME, FLAG_DB2GAME, FLAG_DB2NP, FLAG_GAME2CLIENT, FLAG_GAME2DB,
    FLAG_NEW, FLAG_NP2DB,
};
