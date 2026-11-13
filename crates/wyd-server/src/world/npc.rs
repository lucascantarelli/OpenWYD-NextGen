//! Submódulo de `world` (ADR-0026, etapa 3).

use super::*;

impl Engine {
    /// Reconcilia o mundo com a foto de `npc_definition` do snapshot.
    ///
    /// Operação fria (boot e recarga de configuração), nunca por tick com tamanho variável.
    /// NPC que sumiu ou mudou de posição/Stats sai e volta a nascer; o que não mudou fica
    /// com a mesma entidade. Recusa do motor (posição fora do mapa) é contada, não silenciada.
    pub fn sync_npcs(&mut self, specs: Vec<NpcSpec>) -> NpcSync {
        let mut report = NpcSync::default();
        let mut querem: HashMap<i32, NpcSpec> =
            specs.into_iter().map(|spec| (spec.id, spec)).collect();
        let existentes: Vec<i32> = self.npcs.keys().copied().collect();
        for id in existentes {
            let manter = match (self.npcs.get(&id), querem.get(&id)) {
                (Some(atual), Some(novo)) => {
                    novo.position == atual.spec.position && novo.stats == atual.spec.stats
                }
                _ => false,
            };
            if manter {
                querem.remove(&id);
                report.kept += 1;
                continue;
            }
            if let Some(atual) = self.npcs.remove(&id) {
                self.despawn(atual.entity);
                report.removed += 1;
            }
        }
        for (id, spec) in querem {
            match self.spawn(SpawnSpec {
                kind: spec.kind,
                position: spec.position,
                stats: spec.stats,
            }) {
                Ok(entity) => {
                    // Id de cena no espaço de criatura: jogadores ficam abaixo de
                    // `MAX_USER`. Sem id livre o NPC não entra (nunca com id de jogador).
                    if self.alloc_scene_id(entity, false).is_none() {
                        self.despawn(entity);
                        report.rejected += 1;
                        continue;
                    }
                    self.npcs.insert(id, NpcLive { entity, spec });
                    report.spawned += 1;
                }
                Err(_) => report.rejected += 1,
            }
        }
        report
    }

    /// NPC/monstro mais próximo do ponto clicado, dentro de `range` tiles.
    ///
    /// Desempate pelo menor id — a mesma regra determinística usada na IA. `only_monster`
    /// separa combate (`_MSG_Attack`) de diálogo (`_MSG_Action`).
    pub(super) fn npc_near(
        &self,
        x: i32,
        y: i32,
        range: u32,
        only_monster: bool,
    ) -> Option<&NpcLive> {
        self.npcs
            .values()
            .filter(|npc| !only_monster || npc.spec.kind == EntityKind::Monster)
            .filter_map(|npc| {
                let posicao = self.store.position_of(npc.entity)?;
                let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
                (distancia <= range).then_some((distancia, npc.spec.id, npc))
            })
            .min_by_key(|(distancia, id, _)| (*distancia, *id))
            .map(|(_, _, npc)| npc)
    }

    /// Quem o personagem clicou em `_MSG_Action`, se houver NPC no ponto clicado.
    ///
    /// Clique no sprite (`CLICK_RANGE_TILES`) resolve o NPC mesmo com o personagem longe:
    /// o `_MSG_Action` também é o destino da caminhada, então a loja/diálogo abre quando ele
    /// chega. Criatura de combate continua exigindo `TALK_RANGE_TILES` — atacar longe é
    /// recusado como antes.
    pub fn talk_at(&self, speaker: Entity, x: i32, y: i32) -> Option<TalkTarget> {
        let posicao = self.store.position_of(speaker)?;
        let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
        let npc = self.npc_near(x, y, CLICK_RANGE_TILES, false)?;
        if distancia > TALK_RANGE_TILES && npc.spec.kind == EntityKind::Monster {
            return None;
        }
        Some(TalkTarget {
            id: npc.spec.id,
            name: npc.spec.name.clone(),
            merchant: npc.spec.merchant,
            exp: npc.spec.exp,
        })
    }

    /// Quantos NPCs vivem neste mundo (observabilidade/teste).
    #[must_use]
    pub fn npc_actors(&self) -> usize {
        self.npcs.len()
    }
}

/// Materializa `npc_definition` do snapshot como pedidos de spawn do mapa dado.
///
/// Nome do diálogo: `display_name` (texto do portal); template só preenche quando ele está
/// vazio. Criatura de combate é a que tem `Exp` no template e não é comerciante.
pub fn npc_specs(config: &GameConfig, map_id: i16) -> Vec<NpcSpec> {
    config
        .npcs_on_map(map_id)
        .map(|npc| NpcSpec {
            id: npc.id,
            name: if npc.display_name.is_empty() {
                npc.name.clone()
            } else {
                npc.display_name.clone()
            },
            template: npc.template_name.clone(),
            position: Position {
                x: i32::from(npc.pos_x),
                y: i32::from(npc.pos_y),
            },
            stats: Stats {
                level: npc.level,
                hp: npc.hp,
                max_hp: npc.max_hp,
                mp: 0,
                max_mp: 0,
                attack: npc.attack,
                defense: npc.defense,
            },
            kind: if npc.monster {
                EntityKind::Monster
            } else {
                EntityKind::Npc
            },
            exp: npc.exp,
            merchant: npc.merchant,
            equip: npc.equip,
            attack_run: npc.attack_run,
            direction: npc.direction,
        })
        .collect()
}
