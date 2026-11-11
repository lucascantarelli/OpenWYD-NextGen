//! Submódulo de `cpsock` (ADR-0026, etapa 2).

use super::selection::{create_character, delete_character, select_character};
use super::*;

/// Trata um pacote já desofuscado e íntegro. `false` = a escrita falhou, encerrar.
pub(super) fn dispatch(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    opcode: Opcode,
    sessao: Sessao<'_>,
    selection: Option<&SelectionStore>,
    stage: &mut Stage,
) -> bool {
    let Sessao { account } = sessao;
    match opcode {
        // `0x0FDE` depois da seleção: conta sem PIN cadastrado pode receber o pacote
        // ainda assim (cliente que já conhece o fluxo). Confirma sem abrir sessão nova.
        Opcode::AccountSecure => {
            log(
                "info",
                "pin",
                &format!("result=ok account={account} mode=late"),
            );
            reply(stream, key_word, Opcode::AccountSecure)
        }
        Opcode::CreateCharacter => {
            let resultado = match selection {
                Some(store) => create_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok(detalhe) => {
                    log(
                        "info",
                        "character",
                        &format!("result=ok action=create account={account} {detalhe}"),
                    );
                    // O antigo responde `CNFNewCharacter` com a seleção inteira
                    // (`_MSG_DBCNFNewCharacter` → `MSG_CNFNewCharacter.sel`). O corpo é o
                    // `SELCHAR` provisório já enviado no login, agora com o personagem novo.
                    let roster = selection
                        .and_then(|store| store.roster_inventory(account).ok())
                        .unwrap_or_default();
                    let corpo = new_character_body(&roster);
                    reply_with_body(
                        stream,
                        key_word,
                        ID_NEW_CHARACTER,
                        Opcode::CNFNewCharacter,
                        &corpo,
                    )
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=create account={account} reason={error}"),
                    );
                    reply(stream, key_word, Opcode::NewCharacterFail)
                }
            }
        }
        Opcode::DeleteCharacter => {
            let resultado = match selection {
                Some(store) => delete_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok(detalhe) => {
                    log(
                        "info",
                        "character",
                        &format!("result=ok action=delete account={account} {detalhe}"),
                    );
                    // O legado devolve o roster inteiro (`MSG_CNFDeleteCharacter` carrega
                    // `STRUCT_SELCHAR`): é com ele que o cliente apaga o slot da tela.
                    let roster = selection
                        .and_then(|store| store.roster_inventory(account).ok())
                        .unwrap_or_default();
                    let corpo = delete_character_body(&roster);
                    reply_with_body(
                        stream,
                        key_word,
                        ID_DELETE_CHARACTER,
                        Opcode::CNFDeleteCharacter,
                        &corpo,
                    )
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=delete account={account} reason={error}"),
                    );
                    reply(stream, key_word, Opcode::DeleteCharacterFail)
                }
            }
        }
        Opcode::CharacterLogin => {
            // Gancho de operação/teste: força a recusa de zona sem mexer no caminho normal.
            // O código é escolhido aqui e sai pelo mesmo `reply_recusa` (ponto único).
            if std::env::var("WYD_TEST_FORCE_ZONA").is_ok() {
                return reply_recusa(
                    stream,
                    key_word,
                    Opcode::CharacterLoginFail,
                    wyd_protocol::motivo::CodigoRecusa::ZonaIndisponivel,
                );
            }
            let resultado = match selection {
                Some(store) => select_character(store, frame, account),
                None => Err(SelectionError::Unavailable),
            };
            match resultado {
                Ok((detalhe, selected)) => {
                    let personagem = &selected.character;
                    // `STRUCT_MOB` do personagem escolhido: equipamento e mochila
                    // de `character_items`, nunca do template da classe. A montagem vem antes
                    // da entrada no mundo porque a mesma aparência vira `_MSG_CreateMob` na
                    // cena dos outros personagens do canal (FASE-03-9).
                    let mut mob = Mob::from_persisted(personagem, &selected.items);
                    // Atributos, pontos e `LearnedSkill` vêm do banco (`character_scores`
                    // + `character_learned_skill`), não do template da classe.
                    crate::characters::apply_progression(&mut mob, &selected.progression);
                    let skin = crate::world::ActorSkin {
                        name: mob.name,
                        equip: mob.equip.map(|item| u16::try_from(item.index).unwrap_or(0)),
                        guild: mob.guild,
                        attack_run: mob.current_score.attack_run,
                        direction: mob.current_score.direction,
                    };
                    // Encarna no mundo **antes** de confirmar: recusa do motor vira
                    // `CharacterLoginFail` e o personagem não fica pela metade. A cena de quem
                    // já está no canal recebe o `_MSG_CreateMob` na varredura seguinte.
                    let entrada = match selection {
                        Some(store) => store.enter_world(personagem, &skin),
                        None => Ok(None),
                    };
                    match entrada {
                        Ok(entrada) => {
                            let (entity, client_id) = match entrada {
                                Some(entrada) => (Some(entrada.entity), entrada.scene_id),
                                None => (None, 0),
                            };
                            *stage = Stage::InGame {
                                entity,
                                selected: Box::new(selected.clone()),
                                client_id,
                                scene: Scene::default(),
                            };
                            log(
                                "info",
                                "character",
                                &format!(
                                    "result=ok action=select account={account} {detalhe} entity={} client_id={client_id}",
                                    entity.map_or_else(
                                        || "none".to_string(),
                                        |e| e.index.to_string()
                                    )
                                ),
                            );
                            let corpo = character_login_body(
                                &mob,
                                u16::try_from(personagem.slot).unwrap_or(0),
                                client_id,
                                0,
                                &selected.short_skill,
                                account,
                            );
                            let enviado = reply_with_body(
                                stream,
                                key_word,
                                ID_SCENE,
                                Opcode::CNFCharacterLogin,
                                &corpo,
                            )
                                // O legado manda o `UpdateScore` do próprio personagem
                                // logo depois do `CNF` e **antes** da cena
                                // (`ProcessDBMessage` → `enterWorldView` → `sendScoreSelf`,
                                // porte Go `handler/character.go`): é o que pinta as barras
                                // de HP/MP/EXP na entrada. Sem ele o cliente fica com o
                                // score do blob de login.
                                && reply_with_body(
                                    stream,
                                    key_word,
                                    client_id,
                                    Opcode::UpdateScore,
                                    &crate::personagens::update_score_body(&mob),
                                )
                                // `_MSG_UpdateEtc` é quem carrega `Coin@28` (o `SendEtc` do
                                // legado, `SendFunc.cpp`): o `STRUCT_MOB` do `CNF` traz o gold
                                // no blob, mas a moeda que o cliente desenha é a do `UpdateEtc`.
                                // Sem esta metade do par o personagem entra com 5 000 000 no
                                // banco (Coin do `BaseMob`) e mostra 0 de ouro na tela.
                                && reply_with_body(
                                    stream,
                                    key_word,
                                    client_id,
                                    Opcode::UpdateEtc,
                                    &crate::personagens::update_etc_body(&mob),
                                );
                            // Cena logo depois do `CNF`: todo NPC materializado pela revisão
                            // vigente de `npc_definition` chega como `_MSG_CreateMob` com
                            // `HEADER.ID = ESCENE_FIELD` (o id vai em `MobID`).
                            enviado && refresh_scene(stream, key_word, stage, selection)
                        }
                        Err(error) => {
                            log(
                                "warn",
                                "character",
                                &format!(
                                    "result=refused action=enter account={account} reason={error}"
                                ),
                            );
                            reply_recusa(
                                stream,
                                key_word,
                                Opcode::CharacterLoginFail,
                                wyd_protocol::motivo::CodigoRecusa::ZonaIndisponivel,
                            )
                        }
                    }
                }
                Err(error) => {
                    log(
                        "warn",
                        "character",
                        &format!("result=refused action=select account={account} reason={error}"),
                    );
                    let motivo = if error.to_string().contains("payload") {
                        wyd_protocol::motivo::CodigoRecusa::SelecaoInvalida
                    } else {
                        wyd_protocol::motivo::CodigoRecusa::PersonagemIndisponivel
                    };
                    reply_recusa(stream, key_word, Opcode::CharacterLoginFail, motivo)
                }
            }
        }
        // `_MSG_CharacterLogout` (0x0215, C→J): o cliente pediu para sair (voltar à seleção ou
        // fechar a janela). Responder `CNFCharacterLogout` e encerrar aqui libera a sessão e
        // enfileira o save **na hora** (`leave_world` + `Drop` do `SessionLease`), sem esperar
        // o timeout de ociosidade — era o defeito de "cliente fechado continua online".
        Opcode::CharacterLogout => {
            log(
                "info",
                "logout",
                &format!(
                    "result=ok reason=explicit account={account} stage={}",
                    stage.as_str()
                ),
            );
            reply(stream, key_word, Opcode::CNFCharacterLogout);
            false
        }
        // `_MSG_MessageWhisper` (0x0334, C→J) é o canal do `/getout`, que no legado zera a
        // cidadania (`_MSG_MessageWhisper.cpp:57-64`). O **corpo** do sussurro não tem layout
        // provado na árvore (invariante 5): sem captura do cliente 7662 não se lê comando nem
        // nome, e tratar o pacote como `/getout` seria inventar layout. Tolerar e seguir deixa
        // a sessão viva; a cidadania, derivada do canal, não teria o que zerar hoje.
        Opcode::MessageWhisper => {
            log(
                "info",
                "citizen",
                &format!(
                    "result=tolerated action=getout account={account} reason=layout_unverified stage={}",
                    stage.as_str()
                ),
            );
            true
        }
        // `_MSG_UseItem` de refino: `SourType/SourPos` = pó, `DestType/DestPos` = alvo
        // (`Basedef.h:146-148`; `_MSG_UseItem.cpp:42-152`). O legado consome o pó, aplica
        // `BASE_SetItemSanc` e devolve `SendItem` (0x0182) com o alvo; aqui a taxa vem
        // do snapshot vivo, a piedade da tabela `refine_pity` e o inventário é gravado
        // antes de a sessão passar a ver o novo estado.
        Opcode::UseItem => {
            let (
                Stage::InGame {
                    selected,
                    client_id,
                    ..
                },
                Some(store),
            ) = (&mut *stage, selection)
            else {
                log(
                    "warn",
                    "use_item",
                    &format!("result=refused account={account} reason=not_in_game"),
                );
                return true;
            };
            let packet = match UseItem::decode(frame) {
                Ok(packet) => packet,
                Err(error) => {
                    log(
                        "warn",
                        "use_item",
                        &format!("result=refused account={account} reason=layout:{error}"),
                    );
                    return true;
                }
            };
            // Sorteio do processo (não do cliente): duas chamadas de 64 bits
            // independentes, como `rand()%100` e `rand()%4` no legado.
            let seed = std::time::SystemTime::now()
                .duration_since(std::time::UNIX_EPOCH)
                .map(|d| d.as_nanos() as u64)
                .unwrap_or(1);
            let mut die = XorShift64::from_seed(seed);
            let rolls = UseItemRolls {
                roll: (die.next_u64() % 100) as u8,
                pity_roll: (die.next_u64() % 4) as u8,
            };
            let client_id = *client_id;
            match store.use_item(selected.as_mut(), &packet, rolls) {
                Ok(outcome) => {
                    log(
                        "info",
                        "use_item",
                        &format!(
                            "result=refined account={account} powder={:?} slot={} target={} from=+{} to=+{} rate_bps={} success={} destroyed={} powder_left={}",
                            outcome.powder,
                            outcome.source_slot,
                            outcome.target_slot,
                            outcome.resolution.from_level,
                            outcome.resolution.to_level,
                            outcome.resolution.rate_bps,
                            outcome.resolution.success,
                            outcome.destroyed,
                            outcome.powder_left
                        ),
                    );
                    // Alvo atualizado com o par EF_SANC no fio; destruído vira slot vazio
                    // (`sIndex` 0 limpa no cliente). Recusa não emite nada: o cliente já
                    // mostra o estado correto, pois o servidor nada mudou.
                    let quadro = selected
                        .items
                        .iter()
                        .find(|linha| linha.slot == outcome.target_slot)
                        .map(|linha| {
                            Item::from_row_with_pity(linha, outcome.resolution.next_pity).to_bytes()
                        })
                        .unwrap_or_default();
                    reply_send_item(
                        stream,
                        key_word,
                        client_id,
                        packet.dest_type,
                        packet.dest_pos,
                        quadro,
                    )
                }
                Err(error) => {
                    log(
                        "warn",
                        "use_item",
                        &format!(
                            "result=refused account={account} sour=({}, {}) dest=({}, {}) reason={error}",
                            packet.sour_type,
                            packet.sour_pos,
                            packet.dest_type,
                            packet.dest_pos
                        ),
                    );
                    true
                }
            }
        }
        // `_MSG_ApplyBonus`: distribuir ponto de atributo/skill. Responde `UpdateEtc` + `UpdateScore`
        // (o `SendEtc`/`SendScore` do legado) com o estado gravado no banco.
        Opcode::ApplyBonus => {
            let (client_id, aplicado) = match (&*stage, selection) {
                (
                    Stage::InGame {
                        selected,
                        client_id,
                        ..
                    },
                    Some(store),
                ) => match ApplyBonus::decode(frame) {
                    Ok(packet) => match store.apply_bonus(selected, &packet) {
                        Ok(aplicado) => {
                            log(
                                "info",
                                "bonus",
                                &format!(
                                    "result=ok account={account} bonus_type={} detail={} pontos={} skill_bonus={} learned={:#08x}",
                                    packet.bonus_type,
                                    packet.detail,
                                    aplicado.progression.score_bonus,
                                    aplicado.progression.skill_bonus,
                                    aplicado.progression.learned_skill
                                ),
                            );
                            (*client_id, aplicado)
                        }
                        Err(error) => {
                            log(
                                "warn",
                                "bonus",
                                &format!(
                                    "result=refused account={account} bonus_type={} detail={} reason={error}",
                                    packet.bonus_type, packet.detail
                                ),
                            );
                            return true;
                        }
                    },
                    Err(error) => {
                        log(
                            "warn",
                            "bonus",
                            &format!("result=refused account={account} reason=layout:{error}"),
                        );
                        return true;
                    }
                },
                _ => {
                    log(
                        "warn",
                        "bonus",
                        &format!("result=refused account={account} reason=not_in_game"),
                    );
                    return true;
                }
            };
            send_progress(stream, key_word, client_id, &aplicado.mob)
        }
        // `_MSG_Attack`/`_MSG_AttackOne`/`_MSG_AttackTwo`: mesmo corpo, aridade diferente.
        // O cliente aponta o alvo (x/y); o dano é do servidor e a EXP do template só entra
        // no abate, via `SelectionStore::attack` → `gain_exp` (multiplicador do snapshot).
        Opcode::AttackOne => handle_attack::<1>(stream, frame, key_word, account, stage, selection),
        Opcode::AttackTwo => handle_attack::<2>(stream, frame, key_word, account, stage, selection),
        Opcode::Attack => handle_attack::<13>(stream, frame, key_word, account, stage, selection),
        // `_MSG_Action` de NPC: o clique vira diálogo com o dado de `npc_definition`.
        Opcode::Action => handle_action(stream, frame, key_word, account, stage, selection),
        // `_MSG_MessageChat` (0x0333): `/cp` responde no painel; o resto vai para quem está
        // na vista (o cliente resolve o id do falante e desenha `nome : texto`).
        Opcode::MessageChat => handle_chat(stream, frame, key_word, account, stage, selection),
        // `_MSG_Quest` (0x028B): treinadores do Campo de Treinamento (FASE-03-18).
        Opcode::Quest => handle_quest(stream, frame, key_word, account, stage, selection),
        // `_MSG_REQShopList` (0x027B): catálogo do mercador clicado (`npc_shop_item`).
        Opcode::REQShopList => handle_shop_list(stream, frame, key_word, account, stage, selection),
        // `_MSG_Buy` (0x0379): compra em loja de NPC (preço e gold são do servidor).
        Opcode::Buy => handle_buy(stream, frame, key_word, account, stage, selection),
        // Sem layout portado (UNVERIFIED) ou fora do escopo deste processo: tolerar, registrar
        // e seguir. Fechar a conexão aqui era o defeito que derrubava a sessão.
        outro => {
            log(
                "info",
                "packet",
                &format!(
                    "result=tolerated account={account} opcode={} wire={:#06x} stage={}",
                    outro.const_name(),
                    outro.wire(),
                    stage.as_str()
                ),
            );
            true
        }
    }
}

/// `_MSG_Attack*`: golpe corpo a corpo server-authoritative e EXP real no abate.
///
/// O `current_exp` que vem no pacote é ignorado (cliente não é autoridade); o valor enviado
/// de volta é o do `Mob` gravado. `UpdateEtc`+`UpdateScore` são o par que o cliente usa para
/// redesenhar a barra de EXP/HP.
pub(super) fn handle_attack<const N: usize>(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            selected,
            entity: Some(entity),
            client_id,
            scene,
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "attack",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Attack::<N>::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "attack",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let alvo = packet.dam.iter().find(|alvo| alvo.target_id != 0);
    let (alvo_id, dano_cliente) = alvo.map_or((0, 0), |alvo| (alvo.target_id, alvo.damage)); // EXP do cliente: nunca creditada.
    let _ = &packet.current_exp;
    // Sorteio do processo (nunca do cliente): o loot do abate sai deste `die` e a seed vai
    // para o log, então o kill é reproduzível a partir da tabela do snapshot.
    let seed = loot_seed();
    let mut die = XorShift64::from_seed(seed);
    match store.attack(
        selected.as_ref(),
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
        now_ms(),
        &mut die,
    ) {
        Ok(Some(report)) => {
            log(
                "info",
                "attack",
                &format!(
                    "result=hit account={account} npc={} at=({}, {}) damage={} hp_left={} killed={} exp_base={} exp_granted={} levels={} client_damage={alvo_id}/{dano_cliente}",
                    report.target_npc,
                    packet.target_x,
                    packet.target_y,
                    report.damage,
                    report.remaining_hp,
                    report.killed,
                    report.base_exp,
                    report.granted_exp,
                    report.levels
                ),
            );
            let enviado = match &report.mob {
                Some(mob) => send_progress(stream, key_word, *client_id, mob),
                None => true,
            };
            if !enviado {
                return false;
            }
            // PvP: a vítima continua viva (o motor restaurou o HP) e não há loot/EXP.
            if report.target_actor {
                log(
                    "info",
                    "pvp",
                    &format!(
                        "result=hit account={account} target={} damage={} killed={} cp={} cp_delta={}",
                        report.target_scene,
                        report.damage,
                        report.killed,
                        report.cp_after,
                        report.cp_delta
                    ),
                );
                return true;
            }
            if !report.killed {
                return true;
            }
            // Morte: o abate já saiu do motor (o mundo não guarda cadáver). O cliente recebe
            // `_MSG_RemoveMob` com `RemoveType = 1` (morte) e a vista local perde o id antes
            // do próximo delta, senão o poll de cena repetiria a remoção.
            scene.mobs.remove(&report.target_scene);
            log(
                "info",
                "death",
                &format!(
                    "result=ok account={account} npc={} template={} exp_base={} exp_granted={} levels={} at=({}, {}) seed={seed}",
                    report.target_npc,
                    report.template,
                    report.base_exp,
                    report.granted_exp,
                    report.levels,
                    packet.target_x,
                    packet.target_y
                ),
            );
            let mut envio_ok = true;
            match &report.loot {
                Some(Ok(loot)) => {
                    log(
                        "info",
                        "loot",
                        &format!(
                            "result=resolved account={account} npc={} template={} rolls={:?} drops={:?}",
                            report.target_npc, loot.template, loot.rolls, loot.drops
                        ),
                    );
                    // Entrega no inventário: empilha/ocupa slot, grava e avisa o cliente por
                    // `_MSG_SendItem` (o legado manda o slot entregue, não a mochila inteira).
                    match store.grant_loot(selected, &loot.drops) {
                        Ok(slots) => {
                            log(
                                "info",
                                "loot",
                                &format!(
                                    "result=inventory account={account} npc={} slots={slots:?}",
                                    report.target_npc
                                ),
                            );
                            for slot in slots {
                                let Some(linha) =
                                    selected.items.iter().find(|linha| linha.slot == slot)
                                else {
                                    continue;
                                };
                                let item = Item::from_row(linha).to_bytes();
                                if !reply_send_item(
                                    stream,
                                    key_word,
                                    *client_id,
                                    0,
                                    i32::try_from(slot).unwrap_or(0),
                                    item,
                                ) {
                                    envio_ok = false;
                                    break;
                                }
                            }
                        }
                        Err(error) => log(
                            "warn",
                            "loot",
                            &format!(
                                "result=refused account={account} npc={} reason=inventory:{error}",
                                report.target_npc
                            ),
                        ),
                    }
                }
                Some(Err(error)) => log(
                    "warn",
                    "loot",
                    &format!(
                        "result=refused account={account} npc={} template={} reason={error}",
                        report.target_npc, report.template
                    ),
                ),
                None => log(
                    "warn",
                    "loot",
                    &format!(
                        "result=skipped account={account} npc={} template={} reason=sem_resolvedor",
                        report.target_npc, report.template
                    ),
                ),
            }
            if !envio_ok {
                return false;
            }
            send_remove_mob(stream, key_word, report.target_scene, RemoveMob::DEATH)
        }
        Ok(None) => {
            log(
                "info",
                "attack",
                &format!(
                    "result=no_target account={account} at=({}, {}) npc={alvo_id}",
                    packet.target_x, packet.target_y
                ),
            );
            true
        }
        Err(error) => {
            log(
                "warn",
                "attack",
                &format!("result=refused account={account} reason={error}"),
            );
            true
        }
    }
}

/// `_MSG_Quest` (0x028B): etapa do Campo de Treinamento no treinador clicado.
///
/// `Parm1` é o `MobID` do NPC na cena; o **merchant** vem do mundo (`npc_definition`), não do
/// pacote, e o estágio/prova/recompensa vêm do motor puro (`gameplay::quest_engage`). O
/// sorteio do treinador 4 é do processo (`loot_seed`), nunca do cliente.
/// `_MSG_REQShopList` (0x027B): abre o catálogo do mercador clicado.
///
/// `Target` é o `MobID` da cena; o id de `npc_definition` sai do mundo vivo. A resposta é o
/// `_MSG_ShopList` (236 B) com `HEADER.ID = ESCENE_FIELD`, `ShopType = 0` e `Tax = 0`
/// (imposto de cidade não existe neste servidor — valor explícito, não estimado).
pub(super) fn handle_shop_list(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (Stage::InGame { .. }, Some(store)) = (&mut *stage, selection) else {
        log(
            "warn",
            "shop",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let pedido = match ReqShopList::decode(frame) {
        Ok(pedido) => pedido,
        Err(error) => {
            log(
                "warn",
                "shop",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let Some((npc_id, merchant, nome)) = store.npc_shop(pedido.target) else {
        log(
            "warn",
            "shop",
            &format!(
                "result=refused account={account} npc={} reason=no_npc",
                pedido.target
            ),
        );
        return true;
    };
    if merchant <= 0 {
        log(
            "warn",
            "shop",
            &format!(
                "result=refused account={account} npc={npc_id} name={nome} reason=not_merchant"
            ),
        );
        return true;
    }
    let catalogo = store.shop_items(npc_id).unwrap_or_default();
    let mut lista = ShopList {
        shop_type: 0,
        items: [[0u8; 8]; MAX_SHOPLIST],
        tax: 0,
    };
    for regra in &catalogo {
        let Ok(posicao) = usize::try_from(regra.slot) else {
            continue;
        };
        if posicao >= MAX_SHOPLIST {
            continue;
        }
        let mut item = [0u8; 8];
        item[0..2].copy_from_slice(&i16::try_from(regra.item_index).unwrap_or(0).to_le_bytes());
        for (indice, par) in regra.effects.iter().enumerate() {
            item[2 + indice * 2] = par[0];
            item[3 + indice * 2] = par[1];
        }
        lista.items[posicao] = item;
    }
    let mut corpo = vec![0u8; ShopList::WIRE_LEN];
    if lista.encode_into(&mut corpo).is_err() {
        return false;
    }
    log(
        "info",
        "shop",
        &format!(
            "result=list account={account} npc={npc_id} name={nome} itens={} cheios={}",
            catalogo.len(),
            lista
                .items
                .iter()
                .filter(|item| item[0] != 0 || item[1] != 0)
                .count()
        ),
    );
    reply_with_body(
        stream,
        key_word,
        ID_SCENE,
        Opcode::ShopList,
        &corpo[HEADER_SIZE..],
    )
}

/// `_MSG_Buy` (0x0379): compra em loja de NPC.
///
/// Sucesso entrega `_MSG_SendItem` (mochila) + `UpdateEtc`/`UpdateScore` (gold novo).
/// Recusa não emite pacote de item: o log diz o motivo e o cliente continua com o estado
/// que o servidor tem.
pub(super) fn handle_buy(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            selected,
            client_id,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "shop",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Buy::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "shop",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    match store.buy(
        selected.as_mut(),
        packet.target_id,
        packet.target_inven_pos,
        packet.coin,
    ) {
        Ok(compra) => {
            log(
                "info",
                "shop",
                &format!(
                    "result=bought account={account} npc={} pos={} coin={} slot={} item={}",
                    packet.target_id,
                    packet.target_inven_pos,
                    packet.coin,
                    compra.carry_pos,
                    i16::from_le_bytes([compra.item[0], compra.item[1]])
                ),
            );
            reply_send_item(
                stream,
                key_word,
                *client_id,
                1,
                i32::from(compra.carry_pos),
                compra.item,
            ) && send_progress(stream, key_word, *client_id, &compra.mob)
        }
        Err(error) => {
            log(
                "warn",
                "shop",
                &format!(
                    "result=refused account={account} npc={} pos={} coin={} reason={error}",
                    packet.target_id, packet.target_inven_pos, packet.coin
                ),
            );
            true
        }
    }
}

pub(super) fn handle_quest(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            selected,
            client_id,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "quest",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Quest::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "quest",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let scene_id = u16::try_from(packet.npc_index).unwrap_or(0);
    let Some((merchant, nome)) = store.npc_merchant(scene_id) else {
        log(
            "warn",
            "quest",
            &format!("result=refused account={account} npc={scene_id} reason=no_npc"),
        );
        return true;
    };
    let roll = (loot_seed() % 3) as u8;
    match store.quest(selected.as_mut(), merchant, roll) {
        Ok(resultado) => {
            let recompensa = resultado.step.map_or_else(
                || "aceita".to_string(),
                |passo| format!("{:?}", passo.reward),
            );
            log(
                "info",
                "quest",
                &format!(
                    "result=ok account={account} npc={scene_id} name={nome} merchant={merchant} stage={} reward={recompensa} completed={} roll={roll} accepted={}",
                    resultado.state.stage,
                    resultado.state.completed,
                    resultado.step.is_none()
                ),
            );
            let texto = if resultado.step.is_none() {
                "Campo de Treinamento: etapa 1/4 (aceita)".to_string()
            } else {
                format!(
                    "Campo de Treinamento: etapa {}/4{}",
                    resultado.state.stage,
                    if resultado.state.completed {
                        " (concluido)"
                    } else {
                        ""
                    }
                )
            };
            let mut painel = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
            let bytes = texto.as_bytes();
            let limite = bytes.len().min(painel.len() - 1);
            painel[..limite].copy_from_slice(&bytes[..limite]);
            reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &painel)
                && send_progress(stream, key_word, *client_id, &resultado.mob)
        }
        Err(error) => {
            log(
                "warn",
                "quest",
                &format!(
                    "result=refused account={account} npc={scene_id} merchant={merchant} reason={error}"
                ),
            );
            let mut painel = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
            let texto = error.to_string();
            let bytes = texto.as_bytes();
            let limite = bytes.len().min(painel.len() - 1);
            painel[..limite].copy_from_slice(&bytes[..limite]);
            reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &painel)
        }
    }
}

/// `_MSG_Action`: clique em NPC → `_MSG_MessagePanel` com o nome real da definição.
///
/// Sem tabela de diálogo (`npc_dialogue` ainda é planejada), o texto é o `display_name` do
/// snapshot — não existe frase inventada. NPC fora do alcance não responde.
pub(super) fn handle_action(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            entity: Some(entity),
            selected,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "action",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match Action::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "action",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    // `_MSG_Action` também é o passo do personagem (`Route[24]` do legado): o destino é
    // validado no tick (orçamento por tempo + retângulo do mapa) antes de qualquer NPC.
    match store.move_to(
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
        now_ms(),
    ) {
        Ok((x, y)) => log(
            "info",
            "move",
            &format!("result=ok account={account} to=({x}, {y})"),
        ),
        Err(error) => log(
            "warn",
            "move",
            &format!(
                "result=refused account={account} at=({}, {}) reason={error}",
                packet.target_x, packet.target_y
            ),
        ),
    }
    // Portão no ponto clicado: chave por `EF_KEYID`, consumida no aceite, aberto por tempo.
    match store.use_key(
        selected,
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
        now_ms(),
    ) {
        Ok(Some(portao)) => {
            log(
                "info",
                "gate",
                &format!(
                    "result=open account={account} gate={} key_ef={} key_item={} closes_at_ms={}",
                    portao.gate, portao.key_ef, portao.key_item, portao.closes_at_ms
                ),
            );
            return reply_panel(stream, key_word, &format!("Portao {} aberto", portao.gate));
        }
        Ok(None) => {}
        Err(error) => {
            log(
                "warn",
                "gate",
                &format!(
                    "result=refused account={account} at=({}, {}) reason={error}",
                    packet.target_x, packet.target_y
                ),
            );
            return reply_panel(stream, key_word, &error.to_string());
        }
    }
    match store.talk(
        *entity,
        i32::from(packet.target_x),
        i32::from(packet.target_y),
    ) {
        Ok(Some(alvo)) => {
            log(
                "info",
                "action",
                &format!(
                    "result=ok account={account} npc={} name={} merchant={} at=({}, {})",
                    alvo.id, alvo.name, alvo.merchant, packet.target_x, packet.target_y
                ),
            );
            let mut text = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
            let bytes = alvo.name.as_bytes();
            let limite = bytes.len().min(text.len() - 1);
            text[..limite].copy_from_slice(&bytes[..limite]);
            reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &text)
        }
        Ok(None) => {
            log(
                "info",
                "action",
                &format!(
                    "result=no_npc account={account} at=({}, {})",
                    packet.target_x, packet.target_y
                ),
            );
            true
        }
        Err(error) => {
            log(
                "warn",
                "action",
                &format!("result=refused account={account} reason={error}"),
            );
            true
        }
    }
}

/// `_MSG_MessageChat`: `/cp` no painel e chat social para quem está na vista.
///
/// O cliente monta 140 B com o texto em `+12` (`WYD.exe 0x4695b3` — ver `payload::ui`) e o
/// receptor (`0x48241f`) resolve `HEADER.ID` no mob da vista e mostra `nome : texto`. Por isso
/// a entrega leva o id de cena do falante e vai só para as sessões cuja vista inclui o
/// falante (mesmo raio de cena), cada uma drenando na própria thread com o próprio `key_word`.
pub(super) fn handle_chat(
    stream: &mut TcpStream,
    frame: &[u8],
    key_word: u8,
    account: &str,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            entity: Some(entity),
            client_id,
            ..
        },
        Some(store),
    ) = (&mut *stage, selection)
    else {
        log(
            "warn",
            "chat",
            &format!("result=refused account={account} reason=not_in_game"),
        );
        return true;
    };
    let packet = match MessageChat::decode(frame) {
        Ok(packet) => packet,
        Err(error) => {
            log(
                "warn",
                "chat",
                &format!("result=refused account={account} reason=layout:{error}"),
            );
            return true;
        }
    };
    let texto = String::from_utf8_lossy(packet.text_bytes())
        .trim()
        .to_string();
    if texto.is_empty() {
        return true;
    }
    let (comando, resto) = texto
        .split_once(char::is_whitespace)
        .unwrap_or((texto.as_str(), ""));
    let comando = comando.to_ascii_lowercase();
    if comando == "/cp" {
        let cp = store.cp(*entity).unwrap_or(0);
        log(
            "info",
            "chat",
            &format!("result=cp account={account} cp={cp}"),
        );
        return reply_panel(stream, key_word, &cp_panel(cp));
    }
    if comando == "/mute" || comando == "/unmute" {
        let alvo = resto.trim();
        let resposta = if alvo.is_empty() {
            let lista = store.chat().muted(*client_id);
            if lista.is_empty() {
                "Nenhum jogador silenciado.".to_string()
            } else {
                format!("Silenciados: {}", lista.join(", "))
            }
        } else if comando == "/mute" {
            match store.chat().mute(*client_id, alvo) {
                Ok(()) => format!("Silenciado: {alvo}."),
                Err(crate::session::MuteRefusal::SelfPlayer) => {
                    "Nao da para silenciar voce mesmo.".to_string()
                }
                Err(crate::session::MuteRefusal::NotFound) => {
                    format!("Nao encontrado no canal: {alvo}.")
                }
            }
        } else if store.chat().unmute(*client_id, alvo) {
            format!("Voltou a ouvir: {alvo}.")
        } else {
            format!("Nao estava silenciado: {alvo}.")
        };
        log(
            "info",
            "chat",
            &format!("result=command account={account} cmd={comando} alvo={alvo}"),
        );
        return reply_panel(stream, key_word, &resposta);
    }
    // Anti-flood só para fala social; comandos acima não consomem cota.
    if let Some(espera) = store.chat().allow(*client_id, now_ms()) {
        let segundos = espera.div_ceil(1_000);
        log(
            "warn",
            "chat",
            &format!("result=flood account={account} wait_ms={espera}"),
        );
        return reply_panel(
            stream,
            key_word,
            &format!("Calma: 3 falas a cada 5s. Espere {segundos}s."),
        );
    }
    // Mesmo recorte do pacote: `text[128]` em `+12`, NUL no fim.
    let mut body = [0u8; 128];
    let bytes = texto.as_bytes();
    let limite = bytes.len().min(body.len() - 1);
    body[..limite].copy_from_slice(&bytes[..limite]);
    // Quem está na vista de quem falou: os raios são iguais, então a relação é simétrica.
    let Some(scene) = store.scene_around(*entity, crate::world::DEFAULT_SCENE_VIEW_TILES) else {
        return true;
    };
    let ids: Vec<u16> = scene.mobs.iter().map(|mob| mob.id).collect();
    let entregues = store.chat().deliver(
        &ids,
        &crate::session::ChatLine {
            speaker: *client_id,
            text: body,
        },
    );
    log(
        "info",
        "chat",
        &format!("result=broadcast account={account} listeners={entregues} len={limite}"),
    );
    true
}

/// Drena a fila de chat da sessão no socket **dela** (o `key_word` é o da conexão).
pub(super) fn drain_chat(
    stream: &mut TcpStream,
    key_word: u8,
    rx: &std::sync::mpsc::Receiver<crate::session::ChatLine>,
) -> bool {
    while let Ok(line) = rx.try_recv() {
        if !reply_with_body(
            stream,
            key_word,
            line.speaker,
            Opcode::MessageChat,
            &line.text,
        ) {
            return false;
        }
    }
    true
}

/// Texto do `/cp`: valor, faixa e efeito. ASCII e curto para caber no painel (127 B úteis).
pub(super) fn cp_panel(cp: i16) -> String {
    let faixa = if cp < 0 {
        "listado"
    } else if cp == 0 {
        "neutro"
    } else {
        "positivo"
    };
    let mut texto = format!(
        "CP {cp} ({faixa}, faixa -75..75). Matar jogador neutro custa 2; vitima listada nao rende caos."
    );
    if cp <= -75 {
        texto.push_str(" PK bloqueado no piso.");
    }
    texto
}

/// `_MSG_MessagePanel` com texto curto (resposta de portão/recusa ao cliente).
pub(super) fn reply_panel(stream: &mut TcpStream, key_word: u8, texto: &str) -> bool {
    let mut painel = [0u8; MessagePanel::WIRE_LEN - HEADER_SIZE];
    let bytes = texto.as_bytes();
    let limite = bytes.len().min(painel.len() - 1);
    painel[..limite].copy_from_slice(&bytes[..limite]);
    reply_with_body(stream, key_word, ID_SCENE, Opcode::MessagePanel, &painel)
}

/// Emite o delta da cena de NPCs desde o último envio desta sessão.
///
/// Só busca a foto quando a **geração** do motor mudou (recarga de configuração mexeu em
/// `npc_definition`). Sai do snapshot do mundo, então a posição é a autoritativa: o cliente
/// não vê NPC onde o servidor não o tem.
pub(super) fn refresh_scene(
    stream: &mut TcpStream,
    key_word: u8,
    stage: &mut Stage,
    selection: Option<&SelectionStore>,
) -> bool {
    let (
        Stage::InGame {
            scene,
            client_id,
            entity: Some(entity),
            ..
        },
        Some(store),
    ) = (stage, selection)
    else {
        return true;
    };
    // Entrega por **proximidade**: o mapa tem 6 110 NPCs e a rajada de 1,4 MB no login não
    // é desenhada pelo cliente 7662. A varredura é conferida a cada poll (a geração não muda
    // quando o personagem anda), e o delta tira quem saiu do raio e traz quem entrou.
    let Some(atual) = store.scene_around(*entity, scene_view_tiles()) else {
        return true;
    };
    let mut plano = plan_scene(&scene.mobs, &atual, scene_budget());
    // Perto primeiro: quem entra precisa ver a área onde está antes do resto do mapa. Sem
    // isso, um teto de varredura cortava justamente os NPCs criados por último (o Campo).
    if let Some(eu) = atual.mobs.iter().find(|mob| mob.id == *client_id) {
        let (px, py) = (eu.position.x, eu.position.y);
        plano.enter.sort_by_key(|mob| {
            (
                mob.position.x.abs_diff(px).max(mob.position.y.abs_diff(py)),
                mob.id,
            )
        });
    }
    let mut removidos = 0usize;
    for (id, remove_type) in &plano.leave {
        if !send_remove_mob(stream, key_word, *id, *remove_type) {
            return false;
        }
        scene.mobs.remove(id);
        removidos += 1;
    }
    let mut criados = 0usize;
    for mob in &plano.enter {
        // `CreateType` 2 ("acabou de entrar") só para o dono da sessão; os demais
        // entram como 0 (`SendFunc`/`GetCreateMob` do legado).
        let create_type = if mob.id == *client_id {
            CreateMob::CREATE_JUST_ENTERED
        } else {
            CreateMob::CREATE_NORMAL
        };
        if !send_create_mob(stream, key_word, mob, create_type) {
            return false;
        }
        scene.mobs.insert(mob.id, mob.actor);
        criados += 1;
    }
    if criados == 0 && removidos == 0 {
        // Nada mudou na vista: nenhum pacote, nenhuma linha de log por segundo.
        return true;
    }
    if plano.complete {
        scene.generation = atual.generation;
    } else {
        // Teto atingido: mantém a geração antiga para a próxima varredura continuar o delta.
        log(
            "warn",
            "scene",
            &format!(
                "result=budget limit={} created={criados} removed={removidos} pending=1",
                scene_budget()
            ),
        );
        return true;
    }
    log(
        "info",
        "scene",
        &format!(
            "result=applied generation={} mobs={} created={criados} removed={removidos}",
            scene.generation,
            scene.mobs.len()
        ),
    );
    true
}

/// Raio da cena visível (`WYD_SCENE_VIEW_TILES`), lido uma vez.
pub(super) fn scene_view_tiles() -> u32 {
    use std::sync::OnceLock;
    static RAIO: OnceLock<u32> = OnceLock::new();
    *RAIO.get_or_init(crate::world::scene_view_tiles_from_env)
}

/// Teto de pacotes de cena por varredura (`WYD_SCENE_MAX_PER_REFRESH`), lido uma vez.
pub(super) fn scene_budget() -> usize {
    use std::sync::OnceLock;
    static BUDGET: OnceLock<usize> = OnceLock::new();
    *BUDGET.get_or_init(|| match crate::world::scene_budget_from_env() {
        Ok(teto) => teto,
        Err(error) => {
            // O boot já recusa valor inválido; aqui é rede de segurança para o teste.
            log("error", "config", &format!("result=invalid {error}"));
            crate::world::DEFAULT_SCENE_MAX_PER_REFRESH
        }
    })
}

/// `_MSG_CreateMob`: uma entidade na vista (NPC ou personagem), `HEADER.ID = ESCENE_FIELD`.
///
/// O corpo é o de `Basedef.h`/porte Go (232 B, offsets em `payload/scene.rs`). `Affect`,
/// `AnctCode`, `Tab` e `Hold` vão zerados porque a origem não os tem — não se inventa
/// efeito nem aura de refino que o dado não sustenta. Do `STRUCT_MOB` do personagem saem
/// nome, `Equip[i].sIndex`, `Guild` e `CurrentScore`; `Merchant` de jogador é 0.
pub(super) fn send_create_mob(
    stream: &mut TcpStream,
    key_word: u8,
    mob: &SceneMob,
    create_type: u16,
) -> bool {
    let packet = CreateMob {
        pos_x: u16::try_from(mob.position.x).unwrap_or(0),
        pos_y: u16::try_from(mob.position.y).unwrap_or(0),
        mob_id: mob.id,
        mob_name: mob.name,
        equip: mob.equip,
        affect: [0; 32],
        guild: mob.guild,
        guild_member_type: 0,
        unknow: [0; 3],
        level: i32::from(mob.level),
        ac: i32::try_from(mob.defense).unwrap_or(i32::MAX),
        damage: i32::try_from(mob.attack).unwrap_or(i32::MAX),
        merchant: mob.merchant,
        attack_run: mob.attack_run,
        direction: mob.direction,
        // CP + 75 do personagem: é este byte que o cliente usa para a cor do nome.
        chaos_rate: mob.chaos_rate,
        max_hp: i32::try_from(mob.max_hp).unwrap_or(i32::MAX),
        max_mp: 0,
        hp: i32::try_from(mob.hp).unwrap_or(i32::MAX),
        mp: 0,
        strength: 0,
        intelligence: 0,
        dexterity: 0,
        constitution: 0,
        create_type,
        anct_code: [0; 16],
        tab: [0; 26],
        hold: 0,
    };
    let mut frame = vec![0u8; CreateMob::WIRE_LEN];
    if packet.encode_into(&mut frame).is_err() {
        return false;
    }
    reply_with_body(
        stream,
        key_word,
        ID_SCENE,
        Opcode::CreateMob,
        &frame[HEADER_SIZE..],
    )
}

/// `_MSG_RemoveMob`: `HEADER.ID` = id de cena, corpo `RemoveType` (0 = fora de vista,
/// 2 = logout — `ADR-0025`).
pub(super) fn send_remove_mob(
    stream: &mut TcpStream,
    key_word: u8,
    id: u16,
    remove_type: i32,
) -> bool {
    let packet = RemoveMob { remove_type };
    let mut frame = vec![0u8; RemoveMob::WIRE_LEN];
    if packet.encode_into(&mut frame).is_err() {
        return false;
    }
    reply_with_body(
        stream,
        key_word,
        id,
        Opcode::RemoveMob,
        &frame[HEADER_SIZE..],
    )
}

/// Semente do sorteio de loot: relógio + contador do processo, nunca do cliente.
pub(super) fn loot_seed() -> u64 {
    use std::sync::atomic::{AtomicU64, Ordering};
    static SEQ: AtomicU64 = AtomicU64::new(0);
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|duracao| u64::try_from(duracao.as_nanos()).unwrap_or(u64::MAX))
        .unwrap_or(1);
    nanos
        ^ SEQ
            .fetch_add(1, Ordering::Relaxed)
            .wrapping_mul(0x9E37_79B9_7F4A_7C15)
}

/// Relógio do processo em ms (cooldown de ataque); nunca no tick.
pub(super) fn now_ms() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|duracao| u64::try_from(duracao.as_millis()).unwrap_or(u64::MAX))
        .unwrap_or(0)
}

/// `_MSG_SendItem` (0x0182): atualiza um slot do inventário no cliente.
///
/// É a resposta do refino: o alvo com `BASE_SetItemSanc` aplicado no fio e `HEADER.ID`
/// da conexão do dono. `inv_type` segue `ITEM_PLACE_*` e `slot` é relativo ao lugar.
pub(super) fn reply_send_item(
    stream: &mut TcpStream,
    key_word: u8,
    client_id: u16,
    inv_type: i32,
    slot: i32,
    item: [u8; 8],
) -> bool {
    let packet = SendItem {
        inv_type: inv_type as u16,
        slot: slot as u16,
        item,
    };
    let mut frame = vec![0u8; SendItem::WIRE_LEN];
    if packet.encode_into(&mut frame).is_err() {
        return false;
    }
    reply_with_body(
        stream,
        key_word,
        client_id,
        Opcode::SendItem,
        &frame[HEADER_SIZE..],
    )
}

/// Envia o estado do personagem ao cliente: `MSG_UpdateEtc` e `MSG_UpdateScore`.
///
/// É o par `SendEtc`/`SendScore` do legado, usado quando a progressão muda (bônus e nível).
/// Os dois levam o `HEADER.ID` do próprio dono (`ClientID` do login).
pub fn send_progress(stream: &mut TcpStream, key_word: u8, client_id: u16, mob: &Mob) -> bool {
    let etc = crate::personagens::update_etc_body(mob);
    let score = crate::personagens::update_score_body(mob);
    reply_with_body(stream, key_word, client_id, Opcode::UpdateEtc, &etc)
        && reply_with_body(stream, key_word, client_id, Opcode::UpdateScore, &score)
}
