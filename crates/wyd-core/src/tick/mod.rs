//! Tick determinístico em frequência fixa.
//!
//! Só declara e re-exporta; o acumulador mora em `scheduler`.

pub mod scheduler;

pub use scheduler::{Scheduler, SchedulerConfig, Tick, MAX_HZ, MAX_STEPS_PER_ADVANCE, MIN_HZ};
