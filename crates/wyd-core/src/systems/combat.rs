//! Combate server-authoritative: o cliente pede **contra quem** atacar; o dano é do servidor.
//!
//! Não existe parâmetro de dano em nenhuma assinatura deste módulo. O número sai de
//! [`DamageModel`], que recebe os `Stats` autoritativos das duas entidades — a fórmula aprovada
//! da FASE-10 implementa esse trait; o core não inventa equação.
//!
//! Recusas (alvo morto, fora de alcance, auto-ataque, cooldown, entidade inexistente) são
//! erros tipados e acontecem antes de qualquer escrita: HP, cooldown e fila de eventos ficam
//! idênticos depois de um ataque forjado (FASE-03-5 passo 3).

use crate::ecs::components::{ComponentStore, Entity, Stats};
use crate::ecs::resources::DomainEvent;
use crate::error::CoreError;

/// Fórmula de dano aprovada (FASE-10), injetada pelo chamador.
///
/// Mantém o sistema fora do negócio de balanceamento: trocar a equação não muda a validação.
pub trait DamageModel {
    /// Dano pretendido pelo servidor para `attacker` contra `defender` (0 é permitido).
    fn damage(&self, attacker: Stats, defender: Stats) -> u32;
}

/// Pré-condições do ataque, vindas da configuração do servidor.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AttackRules {
    /// Alcance em tiles (Chebyshev), sempre ≥ 1.
    pub range_tiles: u32,
    /// Intervalo mínimo entre dois ataques da mesma entidade.
    pub cooldown_ms: u64,
}

impl AttackRules {
    /// Regras válidas ou erro explícito — alcance zero não é "corpo a corpo".
    pub fn new(range_tiles: u32, cooldown_ms: u64) -> Result<Self, CoreError> {
        if range_tiles == 0 {
            return Err(CoreError::InvalidAttackRules {
                field: "range_tiles",
                value: 0,
            });
        }
        Ok(Self {
            range_tiles,
            cooldown_ms,
        })
    }
}

/// Resultado de um ataque aceito: o dano que o **servidor** aplicou.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Hit {
    /// Alvo atingido.
    pub target: Entity,
    /// Dano efetivamente aplicado (limitado ao HP restante).
    pub amount: u32,
    /// HP do alvo depois do golpe.
    pub remaining_hp: u32,
    /// Instante (ms) em que o atacante pode atacar de novo.
    pub next_attack_ms: u64,
}

/// Executa um ataque autorizado e devolve o dano aplicado.
///
/// Ordem (transação): entidades vivas → alvo ≠ atacante → alvo com HP → alcance → cooldown →
/// espaço na fila → evento → cooldown + HP. Recusa não muda nada.
pub fn attack<M: DamageModel>(
    store: &mut ComponentStore,
    attacker: Entity,
    target: Entity,
    now_ms: u64,
    rules: AttackRules,
    model: &M,
) -> Result<Hit, CoreError> {
    let atacante = store.stats_of(attacker).ok_or(CoreError::EntityNotAlive {
        index: attacker.index,
        generation: attacker.generation,
    })?;
    let alvo = store.stats_of(target).ok_or(CoreError::EntityNotAlive {
        index: target.index,
        generation: target.generation,
    })?;
    if attacker == target {
        return Err(CoreError::SelfAttack {
            index: attacker.index,
        });
    }
    if alvo.hp == 0 {
        return Err(CoreError::TargetAlreadyDead {
            index: target.index,
        });
    }
    let distancia = crate::systems::movement::distance(
        store
            .position_of(attacker)
            .expect("entidade viva tem posição no SoA"),
        store
            .position_of(target)
            .expect("entidade viva tem posição no SoA"),
    );
    if distancia > rules.range_tiles {
        return Err(CoreError::TargetOutOfRange {
            distance: distancia,
            range: rules.range_tiles,
        });
    }
    let liberado_em = store
        .next_attack_ms(attacker)
        .expect("entidade viva tem cooldown");
    if liberado_em > now_ms {
        return Err(CoreError::AttackCooldown {
            remaining_ms: liberado_em - now_ms,
        });
    }

    let dano = model.damage(atacante, alvo).min(alvo.hp);
    let restante = alvo.hp - dano;
    let proximo = now_ms.saturating_add(rules.cooldown_ms);
    store.publish(DomainEvent::DamageDealt {
        attacker,
        target,
        amount: dano,
        remaining_hp: restante,
    })?;
    store.commit_next_attack(attacker, proximo);
    store.commit_damage(target, dano);
    Ok(Hit {
        target,
        amount: dano,
        remaining_hp: restante,
        next_attack_ms: proximo,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{EntityKind, Inventory, Position};

    fn stats() -> Stats {
        Stats {
            level: 10,
            hp: 50,
            max_hp: 50,
            mp: 0,
            max_mp: 0,
            attack: 20,
            defense: 5,
        }
    }

    /// Modelo determinístico de teste: dano fixo, independente do cliente.
    struct Fixo(u32);

    impl DamageModel for Fixo {
        fn damage(&self, _attacker: Stats, _defender: Stats) -> u32 {
            self.0
        }
    }

    struct Metade;

    impl DamageModel for Metade {
        fn damage(&self, attacker: Stats, defender: Stats) -> u32 {
            attacker.attack.saturating_sub(defender.defense) / 2
        }
    }

    fn cenario(pos_alvo: Position) -> (ComponentStore, Entity, Entity) {
        let mut store = ComponentStore::default();
        let a = store
            .spawn(
                EntityKind::Player,
                Position { x: 10, y: 10 },
                stats(),
                Inventory::default(),
            )
            .expect("spawn válido");
        let b = store
            .spawn(EntityKind::Monster, pos_alvo, stats(), Inventory::default())
            .expect("spawn válido");
        (store, a, b)
    }

    fn regras() -> AttackRules {
        AttackRules::new(2, 1_000).expect("regras")
    }

    #[test]
    fn ataque_valido_aplica_dano_do_modelo_e_publica_um_evento() {
        let (mut store, a, b) = cenario(Position { x: 11, y: 11 });
        let golpe = attack(&mut store, a, b, 5_000, regras(), &Metade).expect("ataque válido");
        assert_eq!(golpe.amount, 7, "(20 − 5) / 2: dano calculado no servidor");
        assert_eq!(golpe.remaining_hp, 43);
        assert_eq!(golpe.next_attack_ms, 6_000);
        assert_eq!(store.stats_of(b).expect("vivo").hp, 43);
        assert_eq!(store.next_attack_ms(a), Some(6_000));
        assert_eq!(
            store.drain_events(),
            vec![DomainEvent::DamageDealt {
                attacker: a,
                target: b,
                amount: 7,
                remaining_hp: 43
            }]
        );
    }

    #[test]
    fn cooldown_recusa_sem_tocar_em_hp_ou_fila() {
        let (mut store, a, b) = cenario(Position { x: 11, y: 10 });
        attack(&mut store, a, b, 5_000, regras(), &Fixo(3)).expect("primeiro ataque");
        store.drain_events();
        assert_eq!(
            attack(&mut store, a, b, 5_500, regras(), &Fixo(3)),
            Err(CoreError::AttackCooldown { remaining_ms: 500 })
        );
        assert_eq!(store.stats_of(b).expect("vivo").hp, 47, "HP intacto");
        assert!(store.events().is_empty(), "recusa não publica evento");
    }

    #[test]
    fn alvo_fora_de_alcance_e_recusado() {
        let (mut store, a, b) = cenario(Position { x: 30, y: 10 });
        assert_eq!(
            attack(&mut store, a, b, 5_000, regras(), &Fixo(3)),
            Err(CoreError::TargetOutOfRange {
                distance: 20,
                range: 2
            })
        );
        assert_eq!(store.stats_of(b).expect("vivo").hp, 50);
    }

    #[test]
    fn auto_ataque_e_alvo_morto_sao_recusados() {
        let (mut store, a, b) = cenario(Position { x: 11, y: 10 });
        assert_eq!(
            attack(&mut store, a, a, 5_000, regras(), &Fixo(3)),
            Err(CoreError::SelfAttack { index: a.index })
        );
        // mata o alvo com o modelo fixo (50 de dano em 50 de HP)
        let golpe = attack(&mut store, a, b, 5_000, regras(), &Fixo(99)).expect("golpe");
        assert_eq!((golpe.amount, golpe.remaining_hp), (50, 0));
        assert_eq!(
            attack(&mut store, a, b, 9_000, regras(), &Fixo(1)),
            Err(CoreError::TargetAlreadyDead { index: b.index })
        );
    }

    #[test]
    fn dano_nunca_ultrapassa_o_hp_restante() {
        let (mut store, a, b) = cenario(Position { x: 11, y: 10 });
        let golpe = attack(&mut store, a, b, 5_000, regras(), &Fixo(1_000)).expect("golpe");
        assert_eq!((golpe.amount, golpe.remaining_hp), (50, 0));
        assert_eq!(store.stats_of(b).expect("vivo").hp, 0);
    }

    #[test]
    fn fila_cheia_recusa_antes_de_aplicar_dano() {
        let mut store = ComponentStore::with_event_capacity(0);
        let a = store
            .spawn(
                EntityKind::Player,
                Position { x: 10, y: 10 },
                stats(),
                Inventory::default(),
            )
            .expect("spawn válido");
        let b = store
            .spawn(
                EntityKind::Monster,
                Position { x: 11, y: 10 },
                stats(),
                Inventory::default(),
            )
            .expect("spawn válido");
        assert_eq!(
            attack(&mut store, a, b, 5_000, regras(), &Fixo(4)),
            Err(CoreError::EventQueueFull { capacity: 0 })
        );
        assert_eq!(store.stats_of(b).expect("vivo").hp, 50);
        assert_eq!(store.next_attack_ms(a), Some(0), "cooldown não avança");
    }

    #[test]
    fn regras_de_alcance_zero_sao_invalidas() {
        assert_eq!(
            AttackRules::new(0, 100),
            Err(CoreError::InvalidAttackRules {
                field: "range_tiles",
                value: 0
            })
        );
    }
}
