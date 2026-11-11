//! Submódulo de `world` (ADR-0026, etapa 3).

use super::*;

impl Engine {
    /// Golpe corpo a corpo autoritativo no ponto clicado.
    ///
    /// `Ok(None)` = não há criatura de combate no alcance (o cliente não escolhe alvo fora
    /// do alcance). O dano sai de [`TemplateDamage`] sobre os `Stats` das duas entidades; o
    /// cliente só diz **onde** golpeou. Abate devolve a EXP do template, que a sessão aplica
    /// com o multiplicador do snapshot.
    pub fn attack_at(
        &mut self,
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
    ) -> Result<Option<AttackOutcome>, CoreError> {
        let Some((entity, npc_id, exp, template)) =
            self.npc_near(x, y, ATTACK_RANGE_TILES, true).map(|npc| {
                (
                    npc.entity,
                    npc.spec.id,
                    npc.spec.exp,
                    npc.spec.template.clone(),
                )
            })
        else {
            // Sem criatura no ponto: com PvP ligado, o alvo pode ser outro jogador.
            return self.attack_player(attacker, x, y, now_ms);
        };
        let golpe = combat_system::attack(
            &mut self.store,
            attacker,
            entity,
            now_ms,
            ATTACK_RULES,
            &TemplateDamage,
        )?;
        let killed = golpe.remaining_hp == 0;
        // Morte real: a criatura sai do mundo no mesmo comando. Deixá-la no mapa com HP 0
        // seria um cadáver que o cliente desenha e o jogador não consegue mais golpear; a
        // geração sobe para a borda emitir `_MSG_RemoveMob` (tipo morte). O loot do abate é
        // resolvido na sessão, com a tabela do snapshot e o sorteio do momento.
        if killed {
            self.despawn(entity);
        }
        Ok(Some(AttackOutcome {
            target_npc: npc_id,
            target_scene: self.scene_ids.get(&entity.index).copied().unwrap_or(0),
            template,
            damage: golpe.amount,
            remaining_hp: golpe.remaining_hp,
            killed,
            exp: if killed { exp } else { 0 },
            target_actor: false,
            cp_after: 0,
            cp_delta: 0,
        }))
    }

    /// Golpe em **outro jogador** (PvP) no ponto clicado.
    ///
    /// Só existe com o PvP do canal/mapa ligado e autor com CP acima do piso (`-75`). O
    /// abate desconta o caos do autor (0 para vítima já listada) e devolve a vítima com HP
    /// cheio — sem EXP/item, porque no original o CP é a consequência do kill. O save
    /// periódico leva o CP novo (`LiveState.cp`).
    fn attack_player(
        &mut self,
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
    ) -> Result<Option<AttackOutcome>, CoreError> {
        if !self.pvp {
            return Ok(None);
        }
        let atacante_cp = self.live.get(&attacker.index).map_or(0, |state| state.cp);
        if !wyd_sim::pode_iniciar_pk(atacante_cp) {
            return Ok(None);
        }
        // Menor distância e depois menor índice: mesma ordem determinística de `npc_near`.
        let alvo = self
            .by_character
            .values()
            .copied()
            .filter(|entity| *entity != attacker)
            .filter_map(|entity| {
                let posicao = self.store.position_of(entity)?;
                let hp = self.store.stats_of(entity)?.hp;
                let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
                (hp > 0 && distancia <= ATTACK_RANGE_TILES).then_some((
                    distancia,
                    entity.index,
                    entity,
                ))
            })
            .min_by_key(|(distancia, indice, _)| (*distancia, *indice))
            .map(|(_, _, entity)| entity);
        let Some(alvo) = alvo else {
            return Ok(None);
        };
        let golpe = combat_system::attack(
            &mut self.store,
            attacker,
            alvo,
            now_ms,
            ATTACK_RULES,
            &TemplateDamage,
        )?;
        let killed = golpe.remaining_hp == 0;
        let mut cp_after = atacante_cp;
        let mut cp_delta = 0;
        if killed {
            let vitima_cp = self.live.get(&alvo.index).map_or(0, |state| state.cp);
            cp_delta = wyd_sim::penalidade_de_kill(vitima_cp);
            cp_after = wyd_sim::clamp_cp(atacante_cp - cp_delta);
            if let Some(state) = self.live.get_mut(&attacker.index) {
                state.cp = cp_after;
            }
            if let Some(stats) = self.store.stats_of(alvo) {
                self.store.restore_hp(alvo, stats.max_hp);
                if let Some(state) = self.live.get_mut(&alvo.index) {
                    state.hp = stats.max_hp;
                }
            }
        }
        Ok(Some(AttackOutcome {
            target_npc: 0,
            target_scene: self.scene_ids.get(&alvo.index).copied().unwrap_or(0),
            template: String::new(),
            damage: golpe.amount,
            remaining_hp: golpe.remaining_hp,
            killed,
            exp: 0,
            target_actor: true,
            cp_after,
            cp_delta,
        }))
    }
}

/// Dano do canal: `attack` do atacante contra o `defense` do alvo na cadeia portada do
/// `wyd-sim` (`DamageInput`/`damage_chain`), com piso 1 para o golpe nunca ser nulo.
///
/// O piso é regra de andamento do canal, não balanceamento: ambos os lados podem nascer com
/// ataque 0 antes da FASE-10 e um golpe de 0 travaria o combate para sempre.
struct TemplateDamage;

impl DamageModel for TemplateDamage {
    fn damage(&self, attacker: Stats, defender: Stats) -> u32 {
        let entrada = DamageInput {
            base: attacker.attack,
            defense: defender.defense,
            ..DamageInput::new(0)
        };
        damage_chain(&entrada).max(1)
    }
}
