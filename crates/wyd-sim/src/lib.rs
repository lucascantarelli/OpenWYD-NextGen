//! Laboratório matemático do WYD — fachada do crate.
//!
//! Este arquivo **não** carrega regra: só declara módulos e reexporta. A matemática mora em
//! [`formula`] (um arquivo por sistema), a ordem do tick em [`scenario`], o sorteio
//! determinístico em [`roll`] e a saída comparável em [`report`].
//!
//! Documento de arquitetura (entradas, saídas, unidades, ordem no tick, arredondamento e
//! decisões provisórias): `docs/arquitetura/matematica-sim.md`. Inventário rápido:
//! `docs/rust/06-matematica.md`. Árvore normativa: `docs/arquitetura/estrutura-e-padroes.md`
//! §14 (ADR-0011).
//!
//! Determinismo é contrato: nenhuma fórmula usa `rand`, relógio, `f32` ou `f64`; a seed entra
//! por [`XorShift64::from_seed`] e o estado do gerador é inspecionável.

pub mod error;
pub mod formula;
pub mod report;
pub mod roll;
pub mod scenario;
pub mod units;

pub use error::SimError;
pub use formula::*;
pub use report::{damage_by_defense, Table};
pub use roll::{Die, XorShift64, CHANCE_SCALE};
pub use scenario::{resolve_attack, AttackOutcome, AttackProfile, Combatant, TICK_PHASES};
pub use units::{mul_bps, Bps, Tenths};
