//! Recursos globais do mundo — dados compartilhados, sem lógica de sistema.
//!
//! Aqui mora a fila de **eventos de domínio**. O Hot Path publica (sem I/O) e a borda
//! (rede/auditoria) drena **depois** do tick; a fila é limitada, porque crescimento sem teto
//! no processo do jogo é vazamento de memória disfarçado de log.
//!
//! Publicar não aloca no caminho quente: o `Vec` é reservado na construção
//! ([`DomainEvents::new`]) e publicar acima do teto devolve erro tipado, nunca cresce.

use crate::ecs::components::{Entity, Position};
use crate::error::CoreError;

/// Efeito de gema gravado no item: código (`GEM_EFFECT_*`) + valor efetivo.
///
/// Separado do [`crate::ecs::components::AppliedGem`] porque este é o **estado** (antes e
/// depois), não o resultado da operação.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct GemEffect {
    /// Código do efeito (`0` = nenhum).
    pub effect: u8,
    /// Valor efetivo (`8`, `2` ou `40 × (refino − 9)`).
    pub value: u16,
}

/// Evento de domínio publicado pelo core. Nomes no passado: o fato já aconteceu.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DomainEvent {
    /// A gema trocou (ou aplicou pela primeira vez) o efeito do item equipado.
    ///
    /// Carrega o **antes e o depois**, para a borda conseguir auditar a troca sem consultar
    /// o item de novo. O pacote de rede correspondente é outro contrato e ainda é
    /// `UNVERIFIED` (invariante 5 do `AGENTS.md`) — nada de layout inventado aqui.
    GemEffectChanged {
        /// Entidade viva no momento da troca.
        entity: Entity,
        /// Slot de equipamento (0..`EQUIP_SLOTS`).
        slot: u8,
        /// `item_index` da gema consumida (3386..3389).
        gem_item: u32,
        /// Efeito anterior (`0/0` quando o item não tinha gema).
        from: GemEffect,
        /// Efeito que passou a valer.
        to: GemEffect,
        /// Refino usado no cálculo (Anct abaixo de +10 conta como +10).
        refine: u8,
    },
    /// A entidade andou por um passo autorizado (FASE-03-5).
    ///
    /// Publicado **depois** de distância/limites/orçamento validados e **antes** da gravação,
    /// então a borda só vê passo que o servidor aceitou.
    Moved {
        /// Entidade que andou.
        entity: Entity,
        /// Posição autoritativa antes do passo.
        from: Position,
        /// Posição gravada depois do passo.
        to: Position,
    },
    /// O servidor aplicou dano calculado por ele (a fórmula vem da FASE-10).
    DamageDealt {
        /// Autor do golpe.
        attacker: Entity,
        /// Alvo atingido.
        target: Entity,
        /// Dano efetivamente aplicado (já limitado ao HP restante).
        amount: u32,
        /// HP do alvo depois do golpe.
        remaining_hp: u32,
    },
    /// O servidor aceitou o uso de uma skill e cobrou o custo.
    ///
    /// O efeito (dano, cura, buff) não é decidido aqui: o fato registra **quem usou o quê** e o
    /// custo, para a borda auditar e para a fórmula aprovada (FASE-10) aplicar o resto.
    SkillCast {
        /// Entidade que usou a skill.
        caster: Entity,
        /// `skill_index` do catálogo.
        skill: u32,
        /// MP cobrado.
        mana_spent: u32,
        /// MP restante depois do custo.
        mana_left: u32,
    },
    /// O servidor equipou um item (slot derivado do catálogo).
    ItemEquipped {
        /// Entidade dona do inventário.
        entity: Entity,
        /// Slot da mochila de onde o item saiu.
        bag_slot: u8,
        /// Slot de equipamento que passou a valer.
        equip_slot: u8,
        /// `item_index` equipado.
        item_index: u32,
    },
    /// O servidor desequipou e devolveu o item à mochila.
    ItemUnequipped {
        /// Entidade dona do inventário.
        entity: Entity,
        /// Slot de equipamento que ficou vazio.
        equip_slot: u8,
        /// Slot da mochila que recebeu o item.
        bag_slot: u8,
        /// `item_index` devolvido.
        item_index: u32,
    },
    /// O servidor resolveu uma tentativa de composição (Aylin/Anct/Odin).
    ///
    /// Carrega taxa efetiva, sorteio e resultado para a borda auditar a economia sem
    /// recalcular nada; o item em si já foi gravado antes do dreno.
    CombineFinished {
        /// Entidade que compôs.
        entity: Entity,
        /// Família da receita (`Aylin`, `Anct`, `Odin`).
        family: &'static str,
        /// Id da receita dentro da família (0 = Sets do Odin).
        recipe: u8,
        /// A tentativa passou?
        success: bool,
        /// Taxa efetiva acumulada, em pontos percentuais.
        rate_percent: i32,
        /// Valor sorteado (já achatado pelo sorteio da família).
        roll: i32,
        /// Item produzido (0 quando não houve).
        produced_item: u32,
    },
    /// O servidor resolveu o passo de refino com pó (`+10 → +11`).
    RefineFinished {
        /// Entidade que refinou.
        entity: Entity,
        /// `item_index` refinado.
        item_index: u32,
        /// Refino antes.
        from_refine: u8,
        /// Refino depois (inalterado na falha).
        to_refine: u8,
        /// Taxa efetiva do passo, em pontos percentuais.
        rate_percent: i32,
        /// Sorteio de 0..=99.
        roll: u8,
        /// A tentativa passou?
        success: bool,
    },
    /// O servidor moveu/empilhou item dentro da mochila.
    ItemMoved {
        /// Entidade dona do inventário.
        entity: Entity,
        /// Slot de origem.
        from: u8,
        /// Slot de destino.
        to: u8,
        /// `item_index` movido.
        item_index: u32,
        /// Quantidade efetivamente movida.
        qty: u32,
    },
}

impl DomainEvent {
    /// Linha de log/auditoria em português (a tradução para pacote é da camada de rede).
    ///
    /// Aloca `String` de propósito: é chamada na borda fria, nunca no tick.
    #[must_use]
    pub fn describe(&self) -> String {
        match self {
            Self::GemEffectChanged {
                entity,
                slot,
                gem_item,
                from,
                to,
                refine,
            } => format!(
                "gema {gem_item} na entidade {} (geração {}) slot {slot}: efeito {} ({} → {}), refino +{refine}",
                entity.index, entity.generation, to.effect, from.value, to.value
            ),
            Self::Moved { entity, from, to } => format!(
                "entidade {} (geração {}) andou de ({}, {}) para ({}, {})",
                entity.index, entity.generation, from.x, from.y, to.x, to.y
            ),
            Self::DamageDealt {
                attacker,
                target,
                amount,
                remaining_hp,
            } => format!(
                "entidade {} (geração {}) causou {amount} de dano em {} (geração {}), HP restante {remaining_hp}",
                attacker.index, attacker.generation, target.index, target.generation
            ),
            Self::SkillCast {
                caster,
                skill,
                mana_spent,
                mana_left,
            } => format!(
                "entidade {} (geração {}) usou skill {skill} por {mana_spent} de MP, MP restante {mana_left}",
                caster.index, caster.generation
            ),
            Self::ItemEquipped {
                entity,
                bag_slot,
                equip_slot,
                item_index,
            } => format!(
                "entidade {} (geração {}) equipou item {item_index} da mochila {bag_slot} no slot {equip_slot}",
                entity.index, entity.generation
            ),
            Self::ItemUnequipped {
                entity,
                equip_slot,
                bag_slot,
                item_index,
            } => format!(
                "entidade {} (geração {}) desequipou o slot {equip_slot} (item {item_index}) para a mochila {bag_slot}",
                entity.index, entity.generation
            ),
            Self::CombineFinished {
                entity,
                family,
                recipe,
                success,
                rate_percent,
                roll,
                produced_item,
            } => format!(
                "entidade {} (geração {}) compôs {family} receita {recipe}: taxa {rate_percent}%, sorteio {roll}, {} (item {produced_item})",
                entity.index,
                entity.generation,
                if *success { "sucesso" } else { "falha" }
            ),
            Self::RefineFinished {
                entity,
                item_index,
                from_refine,
                to_refine,
                rate_percent,
                roll,
                success,
            } => format!(
                "entidade {} (geração {}) refinou item {item_index}: +{from_refine} → +{to_refine} (taxa {rate_percent}%, sorteio {roll}, {})",
                entity.index,
                entity.generation,
                if *success { "sucesso" } else { "falha" }
            ),
            Self::ItemMoved {
                entity,
                from,
                to,
                item_index,
                qty,
            } => format!(
                "entidade {} (geração {}) moveu {qty}× item {item_index} do slot {from} para {to}",
                entity.index, entity.generation
            ),
        }
    }
}

/// Teto padrão da fila: 256 eventos por drenagem. Um tick de 20 Hz com 13 trocas de gema
/// por ciclo ainda cabe; acima disso a borda está atrasada e o erro aparece.
pub const EVENT_QUEUE_CAPACITY: usize = 256;

/// Fila limitada de eventos de domínio (recurso do mundo, não componente de entidade).
#[derive(Debug)]
pub struct DomainEvents {
    queue: Vec<DomainEvent>,
    capacity: usize,
}

impl Default for DomainEvents {
    fn default() -> Self {
        Self::new(EVENT_QUEUE_CAPACITY)
    }
}

impl DomainEvents {
    /// Fila com teto explícito; `capacity` zero é aceito e recusa toda publicação.
    #[must_use]
    pub fn new(capacity: usize) -> Self {
        Self {
            queue: Vec::with_capacity(capacity),
            capacity,
        }
    }

    #[must_use]
    pub fn capacity(&self) -> usize {
        self.capacity
    }

    #[must_use]
    pub fn len(&self) -> usize {
        self.queue.len()
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.queue.is_empty()
    }

    /// Há espaço para mais um evento? O chamador consulta antes de mutar estado.
    #[must_use]
    pub fn has_room(&self) -> bool {
        self.queue.len() < self.capacity
    }

    /// Publica um evento; fila cheia é erro tipado, não descarte silencioso.
    pub fn publish(&mut self, event: DomainEvent) -> Result<(), CoreError> {
        if !self.has_room() {
            return Err(CoreError::EventQueueFull {
                capacity: self.capacity,
            });
        }
        self.queue.push(event);
        Ok(())
    }

    /// Drena na ordem de publicação (chamada fria, depois do tick).
    pub fn drain(&mut self) -> std::vec::Drain<'_, DomainEvent> {
        self.queue.drain(..)
    }

    #[must_use]
    pub fn as_slice(&self) -> &[DomainEvent] {
        &self.queue
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn evento() -> DomainEvent {
        DomainEvent::GemEffectChanged {
            entity: Entity {
                index: 1,
                generation: 0,
            },
            slot: 3,
            gem_item: 3386,
            from: GemEffect::default(),
            to: GemEffect {
                effect: 1,
                value: 8,
            },
            refine: 10,
        }
    }

    #[test]
    fn fila_limite_recusa_acima_do_teto() {
        let mut fila = DomainEvents::new(2);
        assert!(fila.publish(evento()).is_ok());
        assert!(fila.publish(evento()).is_ok());
        assert_eq!(
            fila.publish(evento()),
            Err(CoreError::EventQueueFull { capacity: 2 })
        );
        assert_eq!(fila.len(), 2, "recusa não altera o que já estava na fila");
    }

    #[test]
    fn drenar_esvazia_e_preserva_ordem() {
        let mut fila = DomainEvents::new(4);
        fila.publish(evento()).expect("cabe");
        let drenados: Vec<_> = fila.drain().collect();
        assert_eq!(drenados, vec![evento()]);
        assert!(fila.is_empty());
        assert!(fila.has_room());
    }

    #[test]
    fn descreve_a_troca_em_portugues() {
        let texto = evento().describe();
        assert!(texto.contains("gema 3386"), "texto: {texto}");
        assert!(texto.contains("slot 3"), "texto: {texto}");
        assert!(texto.contains("refino +10"), "texto: {texto}");
    }
}
