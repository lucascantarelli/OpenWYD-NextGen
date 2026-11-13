//! Erros do Hot Path — nenhum caminho do tick causa panic.
//!
//! Toda fronteira (config, spawn, grade, relógio) falha explícita com contexto;
//! default silencioso é violação (ver FASE-03 passo 4).

use core::fmt;

/// Erro de domínio da engine (fronteira que o produziu no nome da variante).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CoreError {
    /// Frequência fora da faixa contratada (20–50 Hz).
    BadTickHz { hz: u32 },
    /// Componente com invariante estrutural violado no spawn.
    InvalidStats { field: &'static str, value: u32 },
    /// Posição fora dos limites do mundo (nunca clamp silencioso).
    OutOfBounds { x: i32, y: i32 },
    /// Dimensão da grade espacial inválida no boot.
    InvalidGrid { field: &'static str, value: i32 },
    /// Relógio voltou no tempo além do tolerável pelo acumulador.
    TimeWentBackwards { before_ms: u64, now_ms: u64 },
    /// Slot de inventário fora da faixa do array fixo.
    SlotOutOfRange { slot: u16, max: u16 },
    /// Slot sem item: não existe o que melhorar.
    EmptySlot { slot: u16 },
    /// Item usado não é uma das quatro gemas de efeito.
    UnknownGem { item_index: u32 },
    /// Gema exige `+10` ou item Ancient.
    GemRefineTooLow { refine: u8 },
    /// Refino acima do máximo conhecido.
    GemRefineTooHigh { refine: u8 },
    /// A gema não está na mochila (ou a última unidade já foi aplicada).
    GemNotInBag { item_index: u32 },
    /// Entidade já morta/gerada de novo: não existe inventário para alterar.
    EntityNotAlive { index: u32, generation: u32 },
    /// Fila de eventos cheia: a borda está atrasada e a operação é recusada.
    EventQueueFull { capacity: usize },
    /// Configuração de movimento inválida no boot (velocidade zero ou retângulo invertido).
    InvalidMoveRules { field: &'static str, value: u32 },
    /// Tempo decorrido insuficiente para andar um tile sequer (pedido prematuro).
    NoMoveBudget { elapsed_ms: u64 },
    /// Passo maior que o orçamento de velocidade (teleporte/override de cliente).
    MoveTooFar { distance: u32, allowed: u32 },
    /// Entidade tentou atacar a si mesma.
    SelfAttack { index: u32 },
    /// Alvo fora do alcance autorizado da arma/habilidade.
    TargetOutOfRange { distance: u32, range: u32 },
    /// Ataque pedido antes do fim do cooldown autoritativo.
    AttackCooldown { remaining_ms: u64 },
    /// Alvo com HP zero: não recebe dano nem evento.
    TargetAlreadyDead { index: u32 },
    /// Configuração de combate inválida no boot (alcance zero).
    InvalidAttackRules { field: &'static str, value: u32 },
    /// Skill pedida não existe no livro do servidor.
    UnknownSkill { skill: u32 },
    /// Skill existe no catálogo, mas o personagem não a aprendeu.
    SkillNotLearned { skill: u32 },
    /// MP abaixo do custo da skill.
    NotEnoughMana { needed: u32, available: u32 },
    /// Skill pedida antes do fim do cooldown autoritativo.
    SkillCooldown { remaining_ms: u64 },
    /// Livro de skills inválido no boot (id duplicado ou alcance zero).
    InvalidSkillRules { field: &'static str, value: u32 },
    /// Item do catálogo não tem slot de equipamento (material, consumível, quest…).
    NotEquippable { item_index: u32 },
    /// Slot de equipamento pedido diverge do slot que o catálogo define para o item.
    WrongEquipSlot { expected: u8, requested: u8 },
    /// Nível do personagem abaixo do requisito do item.
    LevelTooLow { required: u16, current: u16 },
    /// Slot de equipamento ocupado e a política do servidor não permite troca.
    EquipSlotOccupied { slot: u8 },
    /// Mochila sem slot livre para receber o item desequipado.
    BagFull { slots: u16 },
    /// Destino da mochila ocupado por item diferente (sem troca silenciosa).
    BagSlotOccupied { slot: u16 },
    /// Pilha de destino no limite do catálogo.
    StackFull { slot: u16 },
    /// Origem e destino do movimento são o mesmo slot.
    SameSlot { slot: u16 },
    /// Entidade não é criatura: a IA não decide por jogador nem por item no chão.
    NotACreature { index: u32 },
    /// Tabela de ameaça no teto: nova fonte é recusada sem mutação.
    ThreatTableFull { capacity: usize },
    /// Perfil de IA inválido no boot (alcance zero, leash < aggro, fuga > 100%).
    InvalidNpcProfile { field: &'static str, value: u32 },
    /// Sorteio fora da faixa que a família usa (115 ou 199 no combine).
    RollOutOfRange { roll: u16, max: u16 },
    /// A mesma posição apareceu duas vezes na lista de composição.
    ComposeSlotRepeated { slot: u16 },
    /// Lista de composição maior que a mensagem suporta.
    ComposeTooManySlots { count: usize, max: usize },
    /// A combinação de itens não é nenhuma receita da família.
    ComposeRecipeMismatch { family: &'static str },
    /// Sacrifícios acima do teto da régua (recusa própria, não "receita inválida").
    ComposeSacrificeLimit { count: usize, max: usize },
    /// Moedas insuficientes para a receita (custo cobrado antes do sorteio).
    NotEnoughCoin { needed: u32, available: u32 },
    /// Item com teto de refino por nome (769) tentou passar do limite.
    RefineCappedItem { item_index: u32 },
    /// O passo de pó só existe em `+10` (`GEM_LEVEL`).
    RefineNotAtGemLevel { refine: u8 },
    /// Item usado não é pó de refino aceito (`413`/`4141`).
    WrongPowder { item_index: u32 },
}

impl fmt::Display for CoreError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::BadTickHz { hz } => {
                write!(f, "frequência de tick inválida: {hz} Hz (contrato: 20–50)")
            }
            Self::InvalidStats { field, value } => {
                write!(f, "stats inválido no spawn: {field} = {value}")
            }
            Self::OutOfBounds { x, y } => {
                write!(f, "posição fora do mundo: ({x}, {y})")
            }
            Self::InvalidGrid { field, value } => {
                write!(f, "grade espacial inválida: {field} = {value}")
            }
            Self::TimeWentBackwards { before_ms, now_ms } => {
                write!(
                    f,
                    "relógio voltou no tempo: antes {before_ms} ms, agora {now_ms} ms"
                )
            }
            Self::SlotOutOfRange { slot, max } => {
                write!(f, "slot {slot} fora da faixa 0..={max}")
            }
            Self::EmptySlot { slot } => write!(f, "slot {slot} está vazio"),
            Self::UnknownGem { item_index } => {
                write!(f, "item {item_index} não é gema de efeito")
            }
            Self::GemRefineTooLow { refine } => {
                write!(f, "gema exige +10 ou item Anct (refino atual: +{refine})")
            }
            Self::GemRefineTooHigh { refine } => {
                write!(f, "refino +{refine} acima do máximo +15")
            }
            Self::GemNotInBag { item_index } => {
                write!(f, "gema {item_index} não está na mochila")
            }
            Self::EntityNotAlive { index, generation } => {
                write!(f, "entidade {index} (geração {generation}) não está viva")
            }
            Self::EventQueueFull { capacity } => {
                write!(f, "fila de eventos cheia (teto {capacity})")
            }
            Self::InvalidMoveRules { field, value } => {
                write!(f, "regras de movimento inválidas: {field} = {value}")
            }
            Self::NoMoveBudget { elapsed_ms } => {
                write!(f, "tempo insuficiente para andar um tile: {elapsed_ms} ms")
            }
            Self::MoveTooFar { distance, allowed } => {
                write!(
                    f,
                    "passo de {distance} tiles acima do permitido ({allowed})"
                )
            }
            Self::SelfAttack { index } => {
                write!(f, "entidade {index} tentou atacar a si mesma")
            }
            Self::TargetOutOfRange { distance, range } => {
                write!(f, "alvo a {distance} tiles, fora do alcance {range}")
            }
            Self::AttackCooldown { remaining_ms } => {
                write!(f, "ataque em cooldown por {remaining_ms} ms")
            }
            Self::TargetAlreadyDead { index } => {
                write!(f, "alvo {index} já está morto")
            }
            Self::InvalidAttackRules { field, value } => {
                write!(f, "regras de combate inválidas: {field} = {value}")
            }
            Self::UnknownSkill { skill } => {
                write!(f, "skill {skill} não existe no catálogo do servidor")
            }
            Self::SkillNotLearned { skill } => {
                write!(f, "skill {skill} não foi aprendida pelo personagem")
            }
            Self::NotEnoughMana { needed, available } => {
                write!(f, "MP insuficiente: custa {needed}, disponível {available}")
            }
            Self::SkillCooldown { remaining_ms } => {
                write!(f, "skill em cooldown por {remaining_ms} ms")
            }
            Self::InvalidSkillRules { field, value } => {
                write!(f, "regras de skill inválidas: {field} = {value}")
            }
            Self::NotEquippable { item_index } => {
                write!(f, "item {item_index} não é equipável")
            }
            Self::WrongEquipSlot {
                expected,
                requested,
            } => write!(
                f,
                "slot {requested} não é o slot {expected} do item no catálogo"
            ),
            Self::LevelTooLow { required, current } => {
                write!(f, "nível {current} abaixo do requisito {required}")
            }
            Self::EquipSlotOccupied { slot } => {
                write!(f, "slot de equipamento {slot} ocupado (troca desabilitada)")
            }
            Self::BagFull { slots } => {
                write!(f, "mochila cheia ({slots} slots)")
            }
            Self::BagSlotOccupied { slot } => {
                write!(f, "slot {slot} da mochila ocupado por outro item")
            }
            Self::StackFull { slot } => {
                write!(f, "pilha do slot {slot} já está no limite")
            }
            Self::SameSlot { slot } => {
                write!(f, "origem e destino são o mesmo slot {slot}")
            }
            Self::NotACreature { index } => {
                write!(f, "entidade {index} não é criatura (NPC/monstro)")
            }
            Self::ThreatTableFull { capacity } => {
                write!(f, "tabela de ameaça cheia (teto {capacity})")
            }
            Self::InvalidNpcProfile { field, value } => {
                write!(f, "perfil de IA inválido no boot: {field} = {value}")
            }
            Self::RollOutOfRange { roll, max } => {
                write!(f, "sorteio {roll} fora da faixa 0..={max}")
            }
            Self::ComposeSlotRepeated { slot } => {
                write!(f, "slot {slot} repetido na composição")
            }
            Self::ComposeTooManySlots { count, max } => {
                write!(f, "{count} slots na composição, acima do teto {max}")
            }
            Self::ComposeRecipeMismatch { family } => {
                write!(f, "itens não formam nenhuma receita de {family}")
            }
            Self::ComposeSacrificeLimit { count, max } => {
                write!(f, "{count} sacrifícios acima do teto {max} da régua")
            }
            Self::NotEnoughCoin { needed, available } => {
                write!(
                    f,
                    "moedas insuficientes: custa {needed}, disponível {available}"
                )
            }
            Self::RefineCappedItem { item_index } => {
                write!(f, "item {item_index} tem teto de +9 e não passa daí")
            }
            Self::RefineNotAtGemLevel { refine } => {
                write!(f, "passo de pó exige +10 (refino atual: +{refine})")
            }
            Self::WrongPowder { item_index } => {
                write!(f, "item {item_index} não é pó de refino aceito")
            }
        }
    }
}

impl std::error::Error for CoreError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_variant_displays_in_portuguese() {
        assert_eq!(
            CoreError::BadTickHz { hz: 10 }.to_string(),
            "frequência de tick inválida: 10 Hz (contrato: 20–50)"
        );
        assert_eq!(
            CoreError::InvalidStats {
                field: "max_hp",
                value: 0
            }
            .to_string(),
            "stats inválido no spawn: max_hp = 0"
        );
        assert_eq!(
            CoreError::OutOfBounds { x: -1, y: 5000 }.to_string(),
            "posição fora do mundo: (-1, 5000)"
        );
        assert_eq!(
            CoreError::InvalidGrid {
                field: "cell",
                value: 0
            }
            .to_string(),
            "grade espacial inválida: cell = 0"
        );
        assert!(CoreError::TimeWentBackwards {
            before_ms: 100,
            now_ms: 50
        }
        .to_string()
        .contains("voltou no tempo"));
        assert_eq!(
            CoreError::EntityNotAlive {
                index: 7,
                generation: 2
            }
            .to_string(),
            "entidade 7 (geração 2) não está viva"
        );
        assert_eq!(
            CoreError::EventQueueFull { capacity: 256 }.to_string(),
            "fila de eventos cheia (teto 256)"
        );
        assert_eq!(
            CoreError::MoveTooFar {
                distance: 80,
                allowed: 2
            }
            .to_string(),
            "passo de 80 tiles acima do permitido (2)"
        );
        assert_eq!(
            CoreError::AttackCooldown { remaining_ms: 500 }.to_string(),
            "ataque em cooldown por 500 ms"
        );
        assert_eq!(
            CoreError::TargetOutOfRange {
                distance: 20,
                range: 2
            }
            .to_string(),
            "alvo a 20 tiles, fora do alcance 2"
        );
    }
}
