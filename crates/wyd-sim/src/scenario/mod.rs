//! Cenários de resolução — a ordem do tick, escrita uma vez.
//!
//! `formula/` responde "quanto"; aqui responde "em que ordem". Um golpe atravessa quatro
//! sistemas em sequência fixa (acerto → crítico → dano → efeito), e o tick atravessa os
//! sistemas de estado em ordem fixa também. Nada disso é aleatório: a ordem é contrato, e o
//! teste de replay (`tests/determinism.rs`) prende o resultado.

pub mod combat;

pub use combat::{resolve_attack, AttackOutcome, AttackProfile, Combatant, TICK_PHASES};
