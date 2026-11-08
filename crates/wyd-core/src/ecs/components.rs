//! Componentes contíguos das entidades — dados, sem lógica de jogo.
//!
//! Layout DOD: cada componente mora num `Vec` próprio dentro de
//! [`ComponentStore`]; sistemas varrem fatias (`&[T]` / `&mut [T]`), nunca
//! objetos espalhados no heap. Fórmulas e regras vivem na FASE-10; aqui só a
//! forma dos dados + invariantes estruturais verificadas no spawn.

use crate::ecs::resources::{DomainEvent, DomainEvents, GemEffect};
use crate::error::CoreError;

/// Posição no plano do mundo (unidades do mapa legado, eixo Y para baixo).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Position {
    pub x: i32,
    pub y: i32,
}

/// Atributos vitais e de combate (valores brutos; fórmulas na FASE-10).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Stats {
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    pub mp: u32,
    pub max_mp: u32,
    pub attack: u32,
    pub defense: u32,
}

impl Stats {
    /// Invariantes estruturais mínimas — falha explícita, nunca default silencioso.
    pub fn validate(&self) -> Result<(), CoreError> {
        if self.level == 0 {
            return Err(CoreError::InvalidStats {
                field: "level",
                value: 0,
            });
        }
        if self.max_hp == 0 {
            return Err(CoreError::InvalidStats {
                field: "max_hp",
                value: 0,
            });
        }
        if self.hp > self.max_hp {
            return Err(CoreError::InvalidStats {
                field: "hp",
                value: self.hp,
            });
        }
        if self.mp > self.max_mp {
            return Err(CoreError::InvalidStats {
                field: "mp",
                value: self.mp,
            });
        }
        Ok(())
    }
}

/// Código do efeito extra de gema no item (0 = nenhum).
///
/// O item carrega o **efeito aplicado**, não a gema: a gema é consumida na troca e o que
/// passa a valer no combate é o efeito. O código é `u8` para o componente continuar `Copy`
/// e caber num array contíguo sem enum de tamanho variável.
pub const GEM_EFFECT_NONE: u8 = 0;
pub const GEM_EFFECT_DROP: u8 = 1;
pub const GEM_EFFECT_PIERCE: u8 = 2;
pub const GEM_EFFECT_EXP: u8 = 3;
pub const GEM_EFFECT_ABSORB: u8 = 4;

/// Um item empilhável (forma; efeitos de gema aplicados, semântica de combate na FASE-10).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Item {
    pub id: u32,
    pub qty: u32,
    /// Refino atual (`+0`..`+15`); `0` é item sem refino.
    pub refine: u8,
    /// Item Ancient: aceita gema mesmo abaixo de `+10`.
    pub ancient: bool,
    /// Efeito de gema aplicado (`GEM_EFFECT_*`).
    pub gem_effect: u8,
    /// Valor do efeito: 8 (drop), 2 (EXP) ou 40 × (refino − 9).
    pub gem_value: u16,
}

/// Dimensões do `STRUCT_MOB` legado (confirmar contra `data-formats.md` em 03-4).
pub const EQUIP_SLOTS: usize = 16;
pub const BAG_SLOTS: usize = 64;

/// Inventário de uma entidade: equipamento + mochila em arrays fixos (sem alloc).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Inventory {
    pub equip: [Item; EQUIP_SLOTS],
    pub bag: [Item; BAG_SLOTS],
}

impl Default for Inventory {
    fn default() -> Self {
        Self {
            equip: [Item::default(); EQUIP_SLOTS],
            bag: [Item::default(); BAG_SLOTS],
        }
    }
}

/// Efeito aplicado por uma gema, com o valor que passou a valer no item.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AppliedGem {
    /// Slot de equipamento alterado.
    pub slot: usize,
    /// Efeito que o item tinha antes (`0/0` quando não havia gema).
    pub previous: GemEffect,
    /// Código `GEM_EFFECT_*`.
    pub effect: u8,
    /// Valor efetivo (8, 2 ou 40 × (refino − 9)).
    pub value: u16,
    /// Refino efetivo usado no cálculo (Anct abaixo de +10 conta como +10).
    pub refine: u8,
}

/// Refino mínimo para gema em item comum.
pub const GEM_MIN_REFINE: u8 = 10;
/// Refino máximo conhecido.
pub const GEM_MAX_REFINE: u8 = 15;

impl Inventory {
    /// Valida a aplicação **sem escrever nada** — item, refino, gema e faixa.
    ///
    /// Separar validação de escrita é o que garante recusa sem mutação parcial: quem chama
    /// pode falhar em qualquer pré-condição e o inventário continua idêntico.
    pub(crate) fn plan_gem(&self, slot: usize, gem_item: u32) -> Result<AppliedGem, CoreError> {
        if slot >= EQUIP_SLOTS {
            return Err(CoreError::SlotOutOfRange {
                slot: slot as u16,
                max: (EQUIP_SLOTS - 1) as u16,
            });
        }
        let item = &self.equip[slot];
        if item.id == 0 {
            return Err(CoreError::EmptySlot { slot: slot as u16 });
        }
        let effect = gem_effect_code(gem_item).ok_or(CoreError::UnknownGem {
            item_index: gem_item,
        })?;
        if item.refine > GEM_MAX_REFINE {
            return Err(CoreError::GemRefineTooHigh {
                refine: item.refine,
            });
        }
        if !item.ancient && item.refine < GEM_MIN_REFINE {
            return Err(CoreError::GemRefineTooLow {
                refine: item.refine,
            });
        }
        // Anct abaixo de +10 conta como +10 (fonte: `docs/wyd/sistemas/refino.md` §3).
        let effective_refine = item.refine.max(GEM_MIN_REFINE);
        let value = gem_value(effect, effective_refine).expect("faixa validada acima");
        Ok(AppliedGem {
            slot,
            previous: GemEffect {
                effect: item.gem_effect,
                value: item.gem_value,
            },
            effect,
            value,
            refine: effective_refine,
        })
    }

    /// Grava o resultado planejado no item — operação infalível, só depois das validações.
    pub(crate) fn commit_gem(&mut self, plan: AppliedGem) {
        let item = &mut self.equip[plan.slot];
        item.refine = plan.refine;
        item.gem_effect = plan.effect;
        item.gem_value = plan.value;
    }

    /// Quantas unidades da gema existem na mochila (soma das pilhas).
    #[must_use]
    pub fn bag_qty(&self, item_index: u32) -> u32 {
        self.bag
            .iter()
            .filter(|item| item.id == item_index)
            .fold(0u32, |total, item| total.saturating_add(item.qty))
    }

    /// Há ao menos uma unidade da gema na mochila?
    #[must_use]
    pub fn has_gem(&self, item_index: u32) -> bool {
        self.bag
            .iter()
            .any(|item| item.id == item_index && item.qty > 0)
    }

    /// Consome **uma** unidade da gema da primeira pilha com estoque.
    ///
    /// Devolve `false` quando não há nada a consumir — o chamador traduz em
    /// [`CoreError::GemNotInBag`]. Pilha zerada vira slot vazio, para não sobrar item
    /// fantasma com `qty = 0` na mochila.
    pub(crate) fn consume_gem(&mut self, item_index: u32) -> bool {
        for slot in &mut self.bag {
            if slot.id == item_index && slot.qty > 0 {
                slot.qty -= 1;
                if slot.qty == 0 {
                    *slot = Item::default();
                }
                return true;
            }
        }
        false
    }

    /// Consome a gema e grava o efeito **já planejado** — a parte mutável da transação.
    ///
    /// Recebe o plano de [`Inventory::plan_gem`], então não repete validação nem abre janela
    /// entre "pode aplicar" e "aplica". A ordem é consumo → escrita: sem estoque devolve
    /// [`CoreError::GemNotInBag`] com o item intocado.
    pub(crate) fn apply_gem_from_bag(
        &mut self,
        plan: AppliedGem,
        gem_item: u32,
    ) -> Result<(), CoreError> {
        if !self.consume_gem(gem_item) {
            return Err(CoreError::GemNotInBag {
                item_index: gem_item,
            });
        }
        self.commit_gem(plan);
        Ok(())
    }
}

/// Gema do `item_catalog` → código de efeito (`None` se não for uma das quatro).
#[must_use]
pub const fn gem_effect_code(item_index: u32) -> Option<u8> {
    match item_index {
        3386 => Some(GEM_EFFECT_DROP),
        3387 => Some(GEM_EFFECT_PIERCE),
        3388 => Some(GEM_EFFECT_EXP),
        3389 => Some(GEM_EFFECT_ABSORB),
        _ => None,
    }
}

/// Valor do efeito: fixo para drop/EXP, `40 × (refino − 9)` para perfuração/absorção.
#[must_use]
pub fn gem_value(effect: u8, refine: u8) -> Option<u16> {
    if !(GEM_MIN_REFINE..=GEM_MAX_REFINE).contains(&refine) {
        return None;
    }
    let steps = u16::from(refine - (GEM_MIN_REFINE - 1));
    match effect {
        GEM_EFFECT_DROP => Some(8),
        GEM_EFFECT_EXP => Some(2),
        GEM_EFFECT_PIERCE | GEM_EFFECT_ABSORB => Some(40 * steps),
        _ => None,
    }
}

/// As 4 espécies de entidade do mundo (03-2: sem hierarquia OOP).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EntityKind {
    Player,
    Npc,
    Monster,
    GroundItem,
}

/// Identidade estável: índice SoA + geração (detecta uso-após-despawn).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Entity {
    pub index: u32,
    pub generation: u32,
}

/// Armazém SoA: um `Vec` contíguo por componente + vida/geração por índice.
///
/// Spawn/despawn aqui são operação de estrutura de dados (fronteira fria);
/// o caminho quente só itera as fatias. Spawn valida `Stats` e nunca inventa
/// valores: dado ruim é `Err`, não correção silenciosa.
#[derive(Debug, Default)]
pub struct ComponentStore {
    kind: Vec<EntityKind>,
    position: Vec<Position>,
    stats: Vec<Stats>,
    inventory: Vec<Inventory>,
    next_attack_ms: Vec<u64>,
    next_skill_ms: Vec<u64>,
    /// Moedas do personagem (custo de composição do Aylin).
    wallet: Vec<u32>,
    /// Classe celestial ou acima (`ClassMaster` do legado): gate do Odin.
    celestial: Vec<bool>,
    /// Destrave Lv40 celestial já comprado.
    celestial_lv40: Vec<bool>,
    events: DomainEvents,
    alive: Vec<bool>,
    generation: Vec<u32>,
    free: Vec<u32>,
}

impl ComponentStore {
    /// Reserva espaço para `n` entidades sem inicializar (chamada fria).
    pub fn reserve(&mut self, n: usize) {
        self.kind.reserve(n);
        self.position.reserve(n);
        self.stats.reserve(n);
        self.inventory.reserve(n);
        self.next_attack_ms.reserve(n);
        self.next_skill_ms.reserve(n);
        self.wallet.reserve(n);
        self.celestial.reserve(n);
        self.celestial_lv40.reserve(n);
        self.alive.reserve(n);
        self.generation.reserve(n);
    }

    /// Número de índices (vivos + mortos com geração antiga).
    #[must_use]
    pub fn len(&self) -> usize {
        self.alive.len()
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.alive.is_empty()
    }

    /// Entidades vivas agora.
    #[must_use]
    pub fn live_count(&self) -> usize {
        self.alive.iter().filter(|&&a| a).count()
    }

    /// Spawn com validação estrutural; reutiliza índice liberado quando há.
    pub fn spawn(
        &mut self,
        kind: EntityKind,
        position: Position,
        stats: Stats,
        inventory: Inventory,
    ) -> Result<Entity, CoreError> {
        stats.validate()?;
        if let Some(index) = self.free.pop() {
            let i = index as usize;
            self.generation[i] += 1;
            self.kind[i] = kind;
            self.position[i] = position;
            self.stats[i] = stats;
            self.inventory[i] = inventory;
            self.next_attack_ms[i] = 0;
            self.next_skill_ms[i] = 0;
            self.wallet[i] = 0;
            self.celestial[i] = false;
            self.celestial_lv40[i] = false;
            self.alive[i] = true;
            Ok(Entity {
                index,
                generation: self.generation[i],
            })
        } else {
            let index = self.alive.len() as u32;
            self.kind.push(kind);
            self.position.push(position);
            self.stats.push(stats);
            self.inventory.push(inventory);
            self.next_attack_ms.push(0);
            self.next_skill_ms.push(0);
            self.wallet.push(0);
            self.celestial.push(false);
            self.celestial_lv40.push(false);
            self.alive.push(true);
            self.generation.push(0);
            Ok(Entity {
                index,
                generation: 0,
            })
        }
    }

    /// Despawn idempotente quanto a índice morto; geração nova invalida o id antigo.
    pub fn despawn(&mut self, entity: Entity) -> bool {
        let i = entity.index as usize;
        if i < self.alive.len() && self.alive[i] && self.generation[i] == entity.generation {
            self.alive[i] = false;
            self.free.push(entity.index);
            true
        } else {
            false
        }
    }

    /// Inventário de uma entidade viva (leitura para rede/persistência).
    #[must_use]
    pub fn inventory(&self, entity: Entity) -> Option<&Inventory> {
        if !self.is_alive(entity) {
            return None;
        }
        self.inventory.get(entity.index as usize)
    }

    /// Inventário mutável de uma entidade viva (sistemas de inventário/equipamento).
    pub(crate) fn inventory_mut(&mut self, entity: Entity) -> Option<&mut Inventory> {
        if !self.is_alive(entity) {
            return None;
        }
        self.inventory.get_mut(entity.index as usize)
    }

    /// **Caminho único** de aplicação de gema: opera no inventário real da entidade,
    /// consome uma unidade da mochila, grava o efeito e publica **um** evento da troca.
    ///
    /// Ordem (transação): entidade viva → plano validado **sem escrita** → estoque na
    /// mochila → espaço na fila → evento → consumo → gravação do efeito. Toda recusa
    /// acontece antes da primeira escrita, então item e mochila ficam idênticos; o evento é
    /// o registro do que foi validado, e o efeito só é gravado depois dele.
    ///
    /// Fonte da regra: `docs/wyd/sistemas/refino.md` §3 (gema aplicada sobre o item, vale
    /// para Anct e +10+, efeito trocável). Decisões D-12/D-13 em
    /// `docs/arquitetura/matematica-sim.md` §4.
    pub fn apply_gem(
        &mut self,
        entity: Entity,
        slot: usize,
        gem_item: u32,
    ) -> Result<AppliedGem, CoreError> {
        if !self.is_alive(entity) {
            return Err(CoreError::EntityNotAlive {
                index: entity.index,
                generation: entity.generation,
            });
        }
        let indice = entity.index as usize;
        let plan = self.inventory[indice].plan_gem(slot, gem_item)?;
        if !self.inventory[indice].has_gem(gem_item) {
            return Err(CoreError::GemNotInBag {
                item_index: gem_item,
            });
        }
        if !self.events.has_room() {
            return Err(CoreError::EventQueueFull {
                capacity: self.events.capacity(),
            });
        }
        self.events.publish(DomainEvent::GemEffectChanged {
            entity,
            slot: plan.slot as u8,
            gem_item,
            from: plan.previous,
            to: GemEffect {
                effect: plan.effect,
                value: plan.value,
            },
            refine: plan.refine,
        })?;
        self.inventory[indice].apply_gem_from_bag(plan, gem_item)?;
        Ok(plan)
    }

    /// Eventos publicados e ainda não drenados (transparência para a borda e para o teste).
    #[must_use]
    pub fn events(&self) -> &[DomainEvent] {
        self.events.as_slice()
    }

    /// Drena os eventos para a borda (rede/auditoria) — chamada fria, fora do tick.
    pub fn drain_events(&mut self) -> Vec<DomainEvent> {
        self.events.drain().collect()
    }

    /// Store com teto de fila explícito (teste de saturação do orçamento de eventos).
    #[must_use]
    pub fn with_event_capacity(capacity: usize) -> Self {
        Self {
            events: DomainEvents::new(capacity),
            ..Self::default()
        }
    }

    /// Varredura em lote (caminho quente): posições contíguas, sem alloc.
    #[must_use]
    pub fn positions(&self) -> &[Position] {
        &self.position
    }

    /// Varredura mutável em lote (caminho quente): um sistema por vez (`&mut` exclusivo).
    #[must_use]
    pub fn positions_mut(&mut self) -> &mut [Position] {
        &mut self.position
    }

    #[must_use]
    pub fn stats(&self) -> &[Stats] {
        &self.stats
    }

    /// Posição autoritativa de uma entidade viva (leitura pelos sistemas).
    #[must_use]
    pub fn position_of(&self, entity: Entity) -> Option<Position> {
        self.is_alive(entity)
            .then(|| self.position.get(entity.index as usize).copied())
            .flatten()
    }

    /// Stats de uma entidade viva (`None` se o id está morto/obsoleto).
    #[must_use]
    pub fn stats_of(&self, entity: Entity) -> Option<Stats> {
        self.is_alive(entity)
            .then(|| self.stats.get(entity.index as usize).copied())
            .flatten()
    }

    /// Espécie da entidade viva (`None` se o id está morto/obsoleto).
    ///
    /// Leitura para IA e visibilidade; nenhuma regra de jogo é decidida aqui.
    #[must_use]
    pub fn kind_of(&self, entity: Entity) -> Option<EntityKind> {
        self.is_alive(entity)
            .then(|| self.kind.get(entity.index as usize).copied())
            .flatten()
    }

    /// Instante (ms) em que o próximo ataque fica liberado.
    #[must_use]
    pub fn next_attack_ms(&self, entity: Entity) -> Option<u64> {
        self.is_alive(entity)
            .then(|| self.next_attack_ms.get(entity.index as usize).copied())
            .flatten()
    }

    /// Instante (ms) em que a próxima skill fica liberada.
    #[must_use]
    pub fn next_skill_ms(&self, entity: Entity) -> Option<u64> {
        self.is_alive(entity)
            .then(|| self.next_skill_ms.get(entity.index as usize).copied())
            .flatten()
    }

    /// Moedas da entidade viva (`None` se o id está morto/obsoleto).
    #[must_use]
    pub fn coin(&self, entity: Entity) -> Option<u32> {
        self.is_alive(entity)
            .then(|| self.wallet.get(entity.index as usize).copied())
            .flatten()
    }

    /// Grava o saldo de moedas (chamada fria: login, persistência, composição).
    ///
    /// Devolve `false` para entidade morta — nunca escreve em índice obsoleto.
    pub fn set_coin(&mut self, entity: Entity, coin: u32) -> bool {
        if !self.is_alive(entity) {
            return false;
        }
        if let Some(slot) = self.wallet.get_mut(entity.index as usize) {
            *slot = coin;
            return true;
        }
        false
    }

    /// Classe celestial ou acima (`ClassMaster`): gate das receitas do Odin.
    #[must_use]
    pub fn celestial(&self, entity: Entity) -> Option<bool> {
        self.is_alive(entity)
            .then(|| self.celestial.get(entity.index as usize).copied())
            .flatten()
    }

    /// Marca a classe celestial (login/persistência; o core só lê).
    pub fn set_celestial(&mut self, entity: Entity, unlocked: bool) -> bool {
        if !self.is_alive(entity) {
            return false;
        }
        if let Some(slot) = self.celestial.get_mut(entity.index as usize) {
            *slot = unlocked;
            return true;
        }
        false
    }

    /// Destrave Lv40 celestial já comprado.
    #[must_use]
    pub fn celestial_lv40(&self, entity: Entity) -> Option<bool> {
        self.is_alive(entity)
            .then(|| self.celestial_lv40.get(entity.index as usize).copied())
            .flatten()
    }

    /// Grava o destrave Lv40 (receita do Odin; chamada infalível pós-validação).
    pub fn set_celestial_lv40(&mut self, entity: Entity, unlocked: bool) -> bool {
        if !self.is_alive(entity) {
            return false;
        }
        if let Some(slot) = self.celestial_lv40.get_mut(entity.index as usize) {
            *slot = unlocked;
            return true;
        }
        false
    }

    /// Publica um evento na mesma fila limitada do core (usado pelos sistemas).
    pub(crate) fn publish(&mut self, event: DomainEvent) -> Result<(), CoreError> {
        self.events.publish(event)
    }

    /// Grava a posição **depois** de validada pelo sistema de movimento.
    pub(crate) fn commit_position(&mut self, entity: Entity, to: Position) {
        if let Some(slot) = self.position.get_mut(entity.index as usize) {
            *slot = to;
        }
    }

    /// Marca o próximo ataque liberado (depois de validado pelo sistema de combate).
    pub(crate) fn commit_next_attack(&mut self, entity: Entity, at_ms: u64) {
        if let Some(slot) = self.next_attack_ms.get_mut(entity.index as usize) {
            *slot = at_ms;
        }
    }

    /// Marca a próxima skill liberada (depois de validada pelo sistema de skill).
    pub(crate) fn commit_next_skill(&mut self, entity: Entity, at_ms: u64) {
        if let Some(slot) = self.next_skill_ms.get_mut(entity.index as usize) {
            *slot = at_ms;
        }
    }

    /// Cobra o custo de MP e devolve o MP restante (custo já validado).
    pub(crate) fn commit_mana(&mut self, entity: Entity, cost: u32) -> u32 {
        if let Some(stats) = self.stats.get_mut(entity.index as usize) {
            stats.mp = stats.mp.saturating_sub(cost);
            return stats.mp;
        }
        0
    }

    /// Aplica dano já calculado pelo servidor e devolve o HP restante.
    pub(crate) fn commit_damage(&mut self, entity: Entity, amount: u32) -> u32 {
        if let Some(stats) = self.stats.get_mut(entity.index as usize) {
            stats.hp = stats.hp.saturating_sub(amount);
            return stats.hp;
        }
        0
    }

    #[must_use]
    pub fn is_alive(&self, entity: Entity) -> bool {
        let i = entity.index as usize;
        i < self.alive.len() && self.alive[i] && self.generation[i] == entity.generation
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn stats_ok() -> Stats {
        Stats {
            level: 1,
            hp: 100,
            max_hp: 100,
            mp: 10,
            max_mp: 10,
            attack: 5,
            defense: 3,
        }
    }

    #[test]
    fn spawn_rejects_bad_stats_explicitly() {
        let mut store = ComponentStore::default();
        let zero_hp = Stats {
            max_hp: 0,
            ..stats_ok()
        };
        assert_eq!(
            store.spawn(
                EntityKind::Player,
                Position { x: 0, y: 0 },
                zero_hp,
                Inventory::default()
            ),
            Err(CoreError::InvalidStats {
                field: "max_hp",
                value: 0
            })
        );
        let over_hp = Stats {
            hp: 200,
            ..stats_ok()
        };
        assert!(store
            .spawn(
                EntityKind::Player,
                Position { x: 0, y: 0 },
                over_hp,
                Inventory::default()
            )
            .is_err());
        assert_eq!(store.live_count(), 0);
    }

    #[test]
    fn spawn_reuses_slot_with_new_generation() {
        let mut store = ComponentStore::default();
        let a = store
            .spawn(
                EntityKind::Monster,
                Position { x: 1, y: 2 },
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert!(store.despawn(a));
        assert!(!store.is_alive(a));
        let b = store
            .spawn(
                EntityKind::Npc,
                Position { x: 3, y: 4 },
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert_eq!((b.index, store.len()), (a.index, 1));
        assert_ne!(a.generation, b.generation);
        assert_eq!(store.live_count(), 1);
    }

    #[test]
    fn stale_entity_never_despawns_twice() {
        let mut store = ComponentStore::default();
        let a = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats_ok(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert!(store.despawn(a));
        assert!(!store.despawn(a));
    }
}

/// Gema de efeito: autoridade do servidor sobre o item equipado.
/// Mesma tabela de `wyd-sim::formula::refine` (golden nos dois lados; dívida de unificação
/// registrada em `docs/rust/06-matematica.md` §4).
#[cfg(test)]
mod gem_tests {
    use super::*;

    /// Armadura no slot 3 **e** as quatro gemas na mochila (2 de cada).
    ///
    /// O fixture carrega estoque porque o caminho de jogo consome a gema: aplicar sem
    /// gema na mochila é recusa tipada, não aplicação silenciosa.
    fn armadura(refine: u8, ancient: bool) -> Inventory {
        let mut inv = Inventory::default();
        inv.equip[3] = Item {
            id: 1668,
            qty: 1,
            refine,
            ancient,
            gem_effect: GEM_EFFECT_NONE,
            gem_value: 0,
        };
        for (slot, gema) in [3386u32, 3387, 3388, 3389].into_iter().enumerate() {
            inv.bag[slot] = Item {
                id: gema,
                qty: 2,
                ..Item::default()
            };
        }
        inv
    }

    const DIAMANTE: u32 = 3386;
    const ESMERALDA: u32 = 3387;
    const CORAL: u32 = 3388;
    const GARNET: u32 = 3389;

    /// Mesma composição do caminho de jogo, sem entidade/evento: valida, consome, grava.
    fn aplicar(inv: &mut Inventory, slot: usize, gema: u32) -> Result<AppliedGem, CoreError> {
        let plano = inv.plan_gem(slot, gema)?;
        inv.apply_gem_from_bag(plano, gema)?;
        Ok(plano)
    }

    #[test]
    fn comum_abaixo_de_dez_e_recusado() {
        let mut inv = armadura(9, false);
        assert_eq!(
            aplicar(&mut inv, 3, DIAMANTE),
            Err(CoreError::GemRefineTooLow { refine: 9 })
        );
        assert_eq!(inv.equip[3].gem_effect, GEM_EFFECT_NONE);
    }

    #[test]
    fn anct_abaixo_de_dez_recebe_valor_base() {
        let mut inv = armadura(9, true);
        let aplicado = aplicar(&mut inv, 3, ESMERALDA).expect("Anct aceita gema");
        assert_eq!(aplicado.value, 40);
        assert_eq!(aplicado.refine, 10);
        assert_eq!(inv.equip[3].gem_effect, GEM_EFFECT_PIERCE);
        assert_eq!(inv.equip[3].gem_value, 40);
    }

    #[test]
    fn garnet_escala_com_o_refino() {
        let mut inv = armadura(13, false);
        let aplicado = aplicar(&mut inv, 3, GARNET).expect("gema válida");
        assert_eq!(aplicado.value, 160);
        assert_eq!(inv.equip[3].gem_value, 160);
    }

    #[test]
    fn drop_e_exp_sao_fixos() {
        let mut inv = armadura(15, false);
        assert_eq!(
            aplicar(&mut inv, 3, DIAMANTE).expect("gema válida").value,
            8
        );
        assert_eq!(aplicar(&mut inv, 3, CORAL).expect("gema válida").value, 2);
    }

    #[test]
    fn gema_desconhecida_e_slot_vazio_sao_erro() {
        let mut inv = armadura(12, false);
        assert_eq!(
            aplicar(&mut inv, 3, 2441),
            Err(CoreError::UnknownGem { item_index: 2441 })
        );
        assert_eq!(
            aplicar(&mut inv, 5, DIAMANTE),
            Err(CoreError::EmptySlot { slot: 5 })
        );
        assert_eq!(
            aplicar(&mut inv, EQUIP_SLOTS, DIAMANTE),
            Err(CoreError::SlotOutOfRange { slot: 16, max: 15 })
        );
    }

    #[test]
    fn refino_acima_do_maximo_e_recusado() {
        let mut inv = armadura(16, true);
        assert_eq!(
            aplicar(&mut inv, 3, DIAMANTE),
            Err(CoreError::GemRefineTooHigh { refine: 16 })
        );
    }

    #[test]
    fn gema_sai_da_mochila_ao_aplicar() {
        let mut inv = armadura(12, false);
        assert_eq!(inv.bag_qty(DIAMANTE), 2);
        aplicar(&mut inv, 3, DIAMANTE).expect("gema válida");
        assert_eq!(inv.bag_qty(DIAMANTE), 1, "uma unidade consumida");
        aplicar(&mut inv, 3, DIAMANTE).expect("segunda unidade");
        assert_eq!(inv.bag_qty(DIAMANTE), 0);
        assert_eq!(
            aplicar(&mut inv, 3, DIAMANTE),
            Err(CoreError::GemNotInBag {
                item_index: DIAMANTE
            }),
            "sem estoque é recusa tipada"
        );
    }

    #[test]
    fn recusa_nao_consome_gema() {
        let mut inv = armadura(9, false);
        assert_eq!(
            aplicar(&mut inv, 3, ESMERALDA),
            Err(CoreError::GemRefineTooLow { refine: 9 })
        );
        assert_eq!(inv.bag_qty(ESMERALDA), 2, "recusa não gasta estoque");
        assert_eq!(inv.equip[3].gem_effect, GEM_EFFECT_NONE);
        assert_eq!(inv.equip[3].gem_value, 0);
    }

    #[test]
    fn mochila_sem_a_gema_recusa_sem_trocar_o_item() {
        let mut inv = Inventory::default();
        inv.equip[3] = Item {
            id: 1668,
            qty: 1,
            refine: 12,
            ancient: false,
            gem_effect: GEM_EFFECT_NONE,
            gem_value: 0,
        };
        assert_eq!(
            aplicar(&mut inv, 3, GARNET),
            Err(CoreError::GemNotInBag { item_index: GARNET })
        );
        assert_eq!(inv.equip[3].gem_effect, GEM_EFFECT_NONE);
        assert_eq!(inv.equip[3].gem_value, 0);
    }

    #[test]
    fn pilha_zerada_libera_o_slot_da_mochila() {
        let mut inv = armadura(12, false);
        inv.bag[0].qty = 1;
        aplicar(&mut inv, 3, DIAMANTE).expect("gema válida");
        assert_eq!(inv.bag[0].id, 0, "sem item fantasma com qty zero");
        assert_eq!(inv.bag[0].qty, 0);
    }

    fn stats() -> Stats {
        Stats {
            level: 1,
            hp: 100,
            max_hp: 100,
            mp: 10,
            max_mp: 10,
            attack: 5,
            defense: 3,
        }
    }

    /// Caso de negócio: Anct **contra** comum. O que muda é efeito + refino efetivo + evento;
    /// o comum não recebe efeito nenhum porque a regra o recusa antes de qualquer escrita.
    #[test]
    fn anct_e_comum_divergem_no_evento_de_troca() {
        let mut store = ComponentStore::default();
        let anct = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(),
                armadura(9, true),
            )
            .expect("spawn válido");
        let applied = store
            .apply_gem(anct, 3, DIAMANTE)
            .expect("Anct aceita gema");
        assert_eq!(
            applied.previous,
            GemEffect::default(),
            "Anct +9 não tinha efeito antes"
        );
        assert_eq!((applied.effect, applied.value), (GEM_EFFECT_DROP, 8));
        assert_eq!(applied.refine, 10, "Anct +9 conta como +10");
        let eventos = store.drain_events();
        assert_eq!(eventos.len(), 1, "uma troca, um evento");
        match eventos[0] {
            DomainEvent::GemEffectChanged {
                entity,
                slot,
                gem_item,
                from,
                to,
                refine,
            } => {
                assert_eq!(entity, anct);
                assert_eq!((slot, gem_item, refine), (3, DIAMANTE, 10));
                assert_eq!(from, GemEffect::default());
                assert_eq!(
                    to,
                    GemEffect {
                        effect: GEM_EFFECT_DROP,
                        value: 8
                    }
                );
            }
            outro => panic!("evento inesperado: {outro:?}"),
        }
        assert!(store.events().is_empty(), "drenar esvazia a fila");

        // Mesmo cenário em item comum +9: recusa e NENHUM evento é publicado.
        let mut store = ComponentStore::default();
        let comum = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(),
                armadura(9, false),
            )
            .expect("spawn válido");
        assert_eq!(
            store.apply_gem(comum, 3, DIAMANTE),
            Err(CoreError::GemRefineTooLow { refine: 9 })
        );
        assert!(store.events().is_empty(), "recusa não publica evento");
        assert_eq!(
            store.inventory(comum).expect("vivo").equip[3].gem_effect,
            GEM_EFFECT_NONE,
            "comum +9 continua sem efeito"
        );
    }

    #[test]
    fn segunda_gema_troca_o_efeito_e_o_evento_carrega_o_antes() {
        let mut store = ComponentStore::default();
        let e = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(),
                armadura(12, false),
            )
            .expect("spawn válido");
        store.apply_gem(e, 3, DIAMANTE).expect("gema válida");
        store.drain_events();
        store.apply_gem(e, 3, GARNET).expect("troca válida");
        let eventos = store.drain_events();
        match eventos[0] {
            DomainEvent::GemEffectChanged { from, to, .. } => {
                assert_eq!(
                    from,
                    GemEffect {
                        effect: GEM_EFFECT_DROP,
                        value: 8
                    }
                );
                assert_eq!(
                    to,
                    GemEffect {
                        effect: GEM_EFFECT_ABSORB,
                        value: 120
                    }
                );
            }
            outro => panic!("evento inesperado: {outro:?}"),
        }
    }

    #[test]
    fn fila_cheia_recusa_antes_de_mudar_o_item() {
        let mut store = ComponentStore::with_event_capacity(1);
        let e = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(),
                armadura(12, false),
            )
            .expect("spawn válido");
        assert!(store.apply_gem(e, 3, DIAMANTE).is_ok());
        assert_eq!(
            store.apply_gem(e, 3, GARNET),
            Err(CoreError::EventQueueFull { capacity: 1 })
        );
        assert_eq!(
            store.inventory(e).expect("vivo").equip[3].gem_effect,
            GEM_EFFECT_DROP,
            "item fica como estava quando a fila recusa"
        );
    }

    #[test]
    fn entidade_morta_nao_aplica_gema() {
        let mut store = ComponentStore::default();
        let e = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(),
                armadura(12, false),
            )
            .expect("spawn válido");
        assert!(store.despawn(e));
        assert_eq!(
            store.apply_gem(e, 3, DIAMANTE),
            Err(CoreError::EntityNotAlive {
                index: e.index,
                generation: e.generation
            })
        );
        assert!(store.events().is_empty());
    }
}
