//! IA de NPC (camada A, FASE-03-6) — determinística, sem I/O e sem alocação no quente.
//!
//! Contrato de `docs/arquitetura/ia-bots.md` §3–§4: a decisão é uma **função pura** de
//! `VisãoDoMundo × NpcState → Intenção`, roda a **5 Hz** (1 decisão a cada 4 ticks de 20 Hz,
//! `fase = entity_id % 4`), desempata sempre pelo **menor `entity`** e **não** move nem causa
//! dano: emite intenção que os `systems/` validam (ADR-0014/ADR-0016).
//!
//! Só declara e re-exporta; cada peça mora no seu arquivo.

pub mod behavior;
pub mod brain;
pub mod budget;
pub mod route;
pub mod state;

pub use brain::{decide, Decision, Intention, NpcProfile, Observed};
pub use budget::{should_decide, DecisionBudget, DECISION_EVERY_TICKS};
pub use state::{NpcMode, NpcState, NpcStates, ThreatEntry, ThreatTable, MAX_THREAT_ENTRIES};
