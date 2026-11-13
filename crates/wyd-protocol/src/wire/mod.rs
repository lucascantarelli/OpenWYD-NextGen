//! Fio: cabeçalho, enquadramento, opcodes e tabela gerada.
//!
//! Só declara e re-exporta; a lógica mora em cada módulo.

pub mod frame;
pub mod generated;
pub mod header;
pub mod opcode;

pub use frame::{inspect, split_frame, validate_size, FrameError, ValidFrame, INITCODE};
pub use header::{
    Header, HeaderError, APP_VERSION, HEADER_SIZE, MAX_MESSAGE_SIZE, MIN_MESSAGE_SIZE,
    SKIPCHECKTICK,
};
pub use opcode::{
    Flow, Opcode, FLAG_CLIENT2GAME, FLAG_DB2GAME, FLAG_DB2NP, FLAG_GAME2CLIENT, FLAG_GAME2DB,
    FLAG_NEW, FLAG_NP2DB,
};
