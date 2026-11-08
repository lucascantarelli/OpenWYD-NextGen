//! Sistemas do tick — aqui mora a **autoridade** do servidor (FASE-03-5).
//!
//! Cada sistema recebe a intenção vinda da rede (destino, alvo) e **nunca** o resultado
//! (posição final, dano, HP restante): o resultado é calculado no core a partir do estado
//! autoritativo. Toda recusa é erro tipado e acontece **antes** da primeira escrita, então
//! estado, fila de eventos e cooldown ficam idênticos depois de uma ação forjada.
//!
//! Só declara e re-exporta; a regra mora em cada sistema.
//!
//! Fórmulas de dano **não** moram aqui: entram por [`combat::DamageModel`], implementado pelas
//! equações aprovadas na FASE-10. Enquanto não houver fórmula aprovada, o sistema não inventa
//! nenhuma — quem chama fornece o modelo (nos testes, um modelo determinístico).

pub mod combat;
pub mod compose;
pub mod inventory;
pub mod movement;
pub mod npc;
pub mod skill;

pub use combat::{attack, AttackRules, DamageModel, Hit};
pub use compose::{
    anct, apply_powder, aylin, downgrade_refine, jewel_effect, odin, plus12_failure,
    powder_failure, roll_115, roll_199, ComposeOutcome, ComposeRules, ItemFacts, OdinRecipe,
    Plus12Failure, PowderFailure, PowderOutcome, ANCT_MAX_SACRIFICES, AYLIN_SLOTS, COMBINE_SLOTS,
    GEM_LEVEL, REFINE_MAX,
};
pub use inventory::{
    equip, equip_into, move_item, unequip, EquipmentRules, Equipped, ItemRules, Unequipped,
};
pub use movement::{step, MoveRules};
pub use npc::decide as decide_npc;
pub use skill::{cast, Cast, SkillBook, SkillDef};
