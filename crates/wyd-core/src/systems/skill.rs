//! Uso de skill server-authoritative: o cliente pede **qual** skill, o core decide o custo.
//!
//! Não existe parâmetro de dano, cura ou efeito: o sistema valida pré-condições (skill conhecida,
//! MP, cooldown, alvo no alcance) e publica o fato (`SkillCast`). O efeito em si é da fórmula
//! aprovada (FASE-10) e do domínio da skill — aqui não se inventa nem um nem outro.
//!
//! A lista de skills **aprendidas** vem do servidor (`character_skills` carregado na PROC-16),
//! nunca do pacote: o cliente só manda o id.
//!
//! Os números de balanceamento (custo de MP, cooldown, alcance) vêm de [`SkillBook`], montado no
//! boot a partir do catálogo; o core não os fixa (mesma regra de `MoveRules`/`AttackRules`).

use crate::ecs::components::{ComponentStore, Entity};
use crate::ecs::resources::DomainEvent;
use crate::error::CoreError;
use crate::systems::movement::distance;

/// Definição de uma skill para efeito de validação (subconjunto do catálogo).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SkillDef {
    /// `skill_index` do catálogo (`skill_catalog`).
    pub id: u32,
    /// MP cobrado no uso.
    pub mana_cost: u32,
    /// Intervalo mínimo entre dois usos da mesma skill.
    pub cooldown_ms: u64,
    /// Alcance em tiles (Chebyshev); `1` é corpo a corpo.
    pub range_tiles: u32,
}

/// Livro de skills do servidor: catálogo carregado no boot, sem alocação no tick.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct SkillBook {
    defs: Vec<SkillDef>,
}

impl SkillBook {
    /// Monta o livro; id duplicado ou alcance zero é erro explícito, não "última vence".
    pub fn new(defs: Vec<SkillDef>) -> Result<Self, CoreError> {
        for (i, def) in defs.iter().enumerate() {
            if def.range_tiles == 0 {
                return Err(CoreError::InvalidSkillRules {
                    field: "range_tiles",
                    value: 0,
                });
            }
            if defs[..i].iter().any(|anterior| anterior.id == def.id) {
                return Err(CoreError::InvalidSkillRules {
                    field: "id",
                    value: def.id,
                });
            }
        }
        Ok(Self { defs })
    }

    /// Definição pelo id do catálogo.
    #[must_use]
    pub fn get(&self, id: u32) -> Option<&SkillDef> {
        self.defs.iter().find(|def| def.id == id)
    }

    /// Quantas skills o livro conhece (config, não progressão do jogador).
    #[must_use]
    pub fn len(&self) -> usize {
        self.defs.len()
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.defs.is_empty()
    }
}

/// Resultado de um uso aceito: custo e cooldown que o servidor gravou.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Cast {
    /// Entidade que usou a skill.
    pub caster: Entity,
    /// `skill_index` usada.
    pub skill: u32,
    /// MP que restou depois do custo.
    pub mana_left: u32,
    /// Instante (ms) em que a skill fica liberada de novo.
    pub next_skill_ms: u64,
}

/// Executa o uso de skill autorizado e devolve o custo aplicado.
///
/// `known` é a lista de skills **aprendidas** (fonte: servidor). `target` é `None` para skill
/// sem alvo (buff próprio, invocação). Ordem transacional: entidade viva → skill no livro →
/// skill aprendida → MP → cooldown → alvo (vivo e no alcance) → fila → evento → MP + cooldown.
pub fn cast(
    store: &mut ComponentStore,
    caster: Entity,
    skill: u32,
    known: &[u32],
    target: Option<Entity>,
    now_ms: u64,
    book: &SkillBook,
) -> Result<Cast, CoreError> {
    let stats = store.stats_of(caster).ok_or(CoreError::EntityNotAlive {
        index: caster.index,
        generation: caster.generation,
    })?;
    let def = book.get(skill).ok_or(CoreError::UnknownSkill { skill })?;
    if !known.contains(&skill) {
        return Err(CoreError::SkillNotLearned { skill });
    }
    if stats.mp < def.mana_cost {
        return Err(CoreError::NotEnoughMana {
            needed: def.mana_cost,
            available: stats.mp,
        });
    }
    let liberado_em = store
        .next_skill_ms(caster)
        .expect("entidade viva tem cooldown de skill");
    if liberado_em > now_ms {
        return Err(CoreError::SkillCooldown {
            remaining_ms: liberado_em - now_ms,
        });
    }
    if let Some(alvo) = target {
        if !store.is_alive(alvo) {
            return Err(CoreError::EntityNotAlive {
                index: alvo.index,
                generation: alvo.generation,
            });
        }
        let distancia = distance(
            store
                .position_of(caster)
                .expect("entidade viva tem posição no SoA"),
            store
                .position_of(alvo)
                .expect("entidade viva tem posição no SoA"),
        );
        if distancia > def.range_tiles {
            return Err(CoreError::TargetOutOfRange {
                distance: distancia,
                range: def.range_tiles,
            });
        }
    }

    let mana_restante = stats.mp - def.mana_cost;
    let proximo = now_ms.saturating_add(def.cooldown_ms);
    store.publish(DomainEvent::SkillCast {
        caster,
        skill,
        mana_spent: def.mana_cost,
        mana_left: mana_restante,
    })?;
    store.commit_mana(caster, def.mana_cost);
    store.commit_next_skill(caster, proximo);
    Ok(Cast {
        caster,
        skill,
        mana_left: mana_restante,
        next_skill_ms: proximo,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{EntityKind, Inventory, Position, Stats};

    fn stats(mp: u32) -> Stats {
        Stats {
            level: 40,
            hp: 100,
            max_hp: 100,
            mp,
            max_mp: 100,
            attack: 10,
            defense: 5,
        }
    }

    fn livro() -> SkillBook {
        SkillBook::new(vec![
            SkillDef {
                id: 32,
                mana_cost: 7,
                cooldown_ms: 500,
                range_tiles: 5,
            },
            SkillDef {
                id: 5,
                mana_cost: 20,
                cooldown_ms: 0,
                range_tiles: 1,
            },
        ])
        .expect("livro válido")
    }

    fn cenario(pos_alvo: Position, mp: u32) -> (ComponentStore, Entity, Entity) {
        let mut store = ComponentStore::default();
        let e = store
            .spawn(
                EntityKind::Player,
                Position { x: 10, y: 10 },
                stats(mp),
                Inventory::default(),
            )
            .expect("spawn válido");
        let alvo = store
            .spawn(
                EntityKind::Monster,
                pos_alvo,
                stats(0),
                Inventory::default(),
            )
            .expect("spawn válido");
        (store, e, alvo)
    }

    #[test]
    fn uso_valido_cobra_mp_e_arma_cooldown() {
        let (mut store, e, alvo) = cenario(Position { x: 12, y: 10 }, 50);
        let uso =
            cast(&mut store, e, 32, &[32], Some(alvo), 1_000, &livro()).expect("skill válida");
        assert_eq!((uso.mana_left, uso.next_skill_ms), (43, 1_500));
        assert_eq!(store.stats_of(e).expect("vivo").mp, 43);
        assert_eq!(store.next_skill_ms(e), Some(1_500));
        assert_eq!(
            store.drain_events(),
            vec![DomainEvent::SkillCast {
                caster: e,
                skill: 32,
                mana_spent: 7,
                mana_left: 43
            }]
        );
    }

    #[test]
    fn skill_nao_aprendida_e_desconhecida_sao_recusadas() {
        let (mut store, e, _alvo) = cenario(Position { x: 11, y: 10 }, 50);
        assert_eq!(
            cast(&mut store, e, 32, &[5], None, 0, &livro()),
            Err(CoreError::SkillNotLearned { skill: 32 })
        );
        assert_eq!(
            cast(&mut store, e, 999, &[999], None, 0, &livro()),
            Err(CoreError::UnknownSkill { skill: 999 })
        );
        assert!(store.events().is_empty());
    }

    #[test]
    fn mp_insuficiente_e_cooldown_nao_alteram_estado() {
        let (mut store, e, alvo) = cenario(Position { x: 11, y: 10 }, 5);
        assert_eq!(
            cast(&mut store, e, 32, &[32], Some(alvo), 0, &livro()),
            Err(CoreError::NotEnoughMana {
                needed: 7,
                available: 5
            })
        );
        let (mut store, e, alvo) = cenario(Position { x: 11, y: 10 }, 50);
        cast(&mut store, e, 32, &[32], Some(alvo), 1_000, &livro()).expect("primeiro uso");
        store.drain_events();
        assert_eq!(
            cast(&mut store, e, 32, &[32], Some(alvo), 1_200, &livro()),
            Err(CoreError::SkillCooldown { remaining_ms: 300 })
        );
        assert_eq!(store.stats_of(e).expect("vivo").mp, 43, "MP intacto");
        assert!(store.events().is_empty());
    }

    #[test]
    fn alvo_fora_do_alcance_e_alvo_morto_sao_recusados() {
        let (mut store, e, alvo) = cenario(Position { x: 40, y: 10 }, 50);
        assert_eq!(
            cast(&mut store, e, 32, &[32], Some(alvo), 0, &livro()),
            Err(CoreError::TargetOutOfRange {
                distance: 30,
                range: 5
            })
        );
        let (mut store, e, alvo) = cenario(Position { x: 11, y: 10 }, 50);
        assert!(store.despawn(alvo));
        assert!(matches!(
            cast(&mut store, e, 32, &[32], Some(alvo), 0, &livro()),
            Err(CoreError::EntityNotAlive { .. })
        ));
    }

    #[test]
    fn skill_sem_alvo_ignora_alcance() {
        let (mut store, e, _alvo) = cenario(Position { x: 10, y: 10 }, 50);
        let uso = cast(&mut store, e, 5, &[5], None, 0, &livro()).expect("buff próprio");
        assert_eq!(uso.mana_left, 30);
    }

    #[test]
    fn livro_com_id_duplicado_ou_alcance_zero_falha_no_boot() {
        assert!(SkillBook::new(vec![
            SkillDef {
                id: 1,
                mana_cost: 1,
                cooldown_ms: 0,
                range_tiles: 1
            },
            SkillDef {
                id: 1,
                mana_cost: 1,
                cooldown_ms: 0,
                range_tiles: 1
            }
        ])
        .is_err());
        assert!(SkillBook::new(vec![SkillDef {
            id: 1,
            mana_cost: 1,
            cooldown_ms: 0,
            range_tiles: 0
        }])
        .is_err());
    }
}
