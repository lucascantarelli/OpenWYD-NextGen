//! FASE-03-18 — Campo de Treinamento (sequência A, fiel ao legado do 7662).
//!
//! Fluxo completo sem GUI: personagem com equipamento da classe recebe as provas (drops dos
//! guardiões), entrega nos quatro treinadores e recebe cada recompensa com o valor exato.
//! O boss `OrcWarrior` entra aqui como **fonte do emblema 524** (`ItemDropList.txt`): o
//! combate/drop no mundo depende do template do campo no snapshot (aberto na issue).
//!
//! Recusas cobertas: prova errada, fora de ordem, recompensa repetida e nível acima de 35.

use wyd_db::{ItemRow, CARRY_FIRST_SLOT};
use wyd_server::gameplay::{
    apply_quest_reward, quest_accept, quest_engage, QuestEngage, QuestRefusal, QuestReward,
    MERCHANT_TRAINER_CHIEF, MERCHANT_TRAINER_FIRST, MERCHANT_TRAINER_SECOND,
    MERCHANT_TRAINER_THIRD, QUEST_AFFECT_HEAL, QUEST_CAMPO_MAX_LEVEL, QUEST_CARBUNCLE_RUBY,
    QUEST_ENCHANT_BASE, QUEST_ENCHANT_SANC, QUEST_GROWING_EYE, QUEST_HEAL_KIT, QUEST_PROOF_EMBLEM,
    QUEST_PROOF_FIRST, QUEST_PROOF_LAST, QUEST_PROOF_SECOND,
};

/// `ReqLvl` do equipamento de nível 1 da classe (base do `50 + ReqLvl`).
const REQ_LEVEL: i64 = 1;
/// Bônus esperado: `SetItemBonus(50 + ReqLvl, 1, 0)`.
const BONUS: i64 = QUEST_ENCHANT_BASE + REQ_LEVEL;

fn inventario_da_classe() -> Vec<ItemRow> {
    let mut itens: Vec<ItemRow> = (1..=7)
        .map(|slot| ItemRow {
            slot,
            item_id: 1000 + slot,
            qty: 1,
            ..ItemRow::default()
        })
        .collect();
    // O `BaseMob` já traz a kill mark na mochila: o primeiro slot de mochila não está livre.
    itens.push(ItemRow {
        slot: CARRY_FIRST_SLOT,
        item_id: 401,
        qty: 1,
        ..ItemRow::default()
    });
    itens
}

fn entregar(
    itens: &mut Vec<ItemRow>,
    merchant: i16,
    stage: i16,
    mask: i32,
    level: i64,
    roll: u8,
) -> Result<wyd_server::gameplay::QuestStep, QuestRefusal> {
    let ids: Vec<i64> = itens
        .iter()
        .filter(|linha| linha.item_id != 0)
        .map(|linha| linha.item_id)
        .collect();
    let arma = itens
        .iter()
        .any(|linha| linha.slot == 6 && linha.item_id != 0);
    let passo = quest_engage(&QuestEngage {
        merchant,
        stage,
        reward_mask: mask,
        level,
        inventory: &ids,
        weapon_req_level: arma.then_some(REQ_LEVEL),
        roll,
    })?;
    apply_quest_reward(itens, &passo).expect("recompensa aplicável no inventário de teste");
    Ok(passo)
}

fn prova(itens: &mut Vec<ItemRow>, item_id: i64) {
    let slot = (CARRY_FIRST_SLOT..CARRY_FIRST_SLOT + 64)
        .find(|slot| {
            !itens
                .iter()
                .any(|linha| linha.slot == *slot && linha.item_id != 0)
        })
        .expect("mochila com slot livre");
    itens.push(ItemRow {
        slot,
        item_id,
        qty: 1,
        ..ItemRow::default()
    });
}

fn contem(itens: &[ItemRow], item_id: i64) -> bool {
    itens.iter().any(|linha| linha.item_id == item_id)
}

#[test]
fn aceite_so_no_primeiro_treinador_e_dentro_do_teto_de_nivel() {
    assert_eq!(quest_accept(1, MERCHANT_TRAINER_FIRST), Ok(()));
    assert_eq!(
        quest_accept(QUEST_CAMPO_MAX_LEVEL, MERCHANT_TRAINER_FIRST),
        Ok(())
    );
    assert_eq!(
        quest_accept(QUEST_CAMPO_MAX_LEVEL + 1, MERCHANT_TRAINER_FIRST),
        Err(QuestRefusal::LevelTooHigh {
            level: QUEST_CAMPO_MAX_LEVEL + 1,
            max: QUEST_CAMPO_MAX_LEVEL
        })
    );
    // Os outros treinadores só avançam etapa: nunca iniciam a quest.
    for merchant in [
        MERCHANT_TRAINER_SECOND,
        MERCHANT_TRAINER_THIRD,
        MERCHANT_TRAINER_CHIEF,
    ] {
        assert_eq!(
            quest_accept(1, merchant),
            Err(QuestRefusal::WrongMerchant { merchant })
        );
    }
}

#[test]
fn fluxo_completo_do_campo_entrega_as_quatro_recompensas() {
    let mut itens = inventario_da_classe();
    let mut mask = 0i32;

    // Etapa 1: GremlinDecoy dropa 451 → FirstTrainer entrega o kit 682 + affect 44.
    prova(&mut itens, QUEST_PROOF_FIRST);
    let passo = entregar(&mut itens, MERCHANT_TRAINER_FIRST, 0, mask, 1, 0).expect("etapa 1");
    assert_eq!(passo.reward, QuestReward::Item(QUEST_HEAL_KIT));
    assert_eq!(passo.stage, 1);
    assert!(!passo.completed);
    assert!(contem(&itens, QUEST_HEAL_KIT), "kit 682 no inventário");
    assert!(
        !contem(&itens, QUEST_PROOF_FIRST),
        "prova consumida na entrega"
    );
    assert_eq!(QUEST_AFFECT_HEAL, 200, "affect 44 (200,200) do legado");
    mask |= passo.reward_bit;

    // Etapa 2: GiantCrill dropa 452 → SecondTrainer encanta a arma (Equip[6]).
    prova(&mut itens, QUEST_PROOF_SECOND);
    let passo = entregar(&mut itens, MERCHANT_TRAINER_SECOND, 1, mask, 1, 0).expect("etapa 2");
    assert_eq!(passo.reward, QuestReward::EnchantWeapon { bonus: BONUS });
    let arma = itens
        .iter()
        .find(|linha| linha.slot == 6)
        .expect("arma equipada");
    assert_eq!(arma.gem_effect, 1, "tipo do bônus");
    assert_eq!(arma.gem_value, BONUS, "50 + ReqLvl");
    assert!(!contem(&itens, QUEST_PROOF_SECOND));
    mask |= passo.reward_bit;

    // Etapa 3: OrcSniper dropa 453 → ThirdTrainer encanta Equip[1..7] + sanc 3.
    prova(&mut itens, QUEST_PROOF_LAST);
    let passo = entregar(&mut itens, MERCHANT_TRAINER_THIRD, 2, mask, 1, 0).expect("etapa 3");
    assert_eq!(
        passo.reward,
        QuestReward::EnchantEquip {
            bonus: BONUS,
            sanc: QUEST_ENCHANT_SANC
        }
    );
    for slot in 1..=7i64 {
        let linha = itens.iter().find(|linha| linha.slot == slot).expect("slot");
        assert_eq!(linha.refine, QUEST_ENCHANT_SANC, "sanc 3 em Equip[{slot}]");
        assert_eq!(linha.gem_value, BONUS, "bônus em Equip[{slot}]");
    }
    assert!(!contem(&itens, QUEST_PROOF_LAST));
    mask |= passo.reward_bit;

    // Etapa 4: OrcWarrior dropa o Emblema 524 → Chefe sorteia 682 / 481 / 652.
    prova(&mut itens, QUEST_PROOF_EMBLEM);
    let passo = entregar(&mut itens, MERCHANT_TRAINER_CHIEF, 3, mask, 1, 1).expect("etapa 4");
    assert_eq!(
        passo.reward,
        QuestReward::Random {
            options: [QUEST_HEAL_KIT, QUEST_GROWING_EYE, QUEST_CARBUNCLE_RUBY],
            chosen: QUEST_GROWING_EYE,
        },
        "roll 1 = Olho Crescente (o legado sorteia com rand()%3)"
    );
    assert!(contem(&itens, QUEST_GROWING_EYE));
    assert!(passo.completed, "Newbie 3→4 conclui a quest");
    assert_eq!(passo.reward_bit, 1 << 3);
}

#[test]
fn sorteio_do_chefe_cobre_as_tres_opcoes() {
    for (roll, esperado) in [
        (0u8, QUEST_HEAL_KIT),
        (1, QUEST_GROWING_EYE),
        (2, QUEST_CARBUNCLE_RUBY),
        (3, QUEST_HEAL_KIT),
    ] {
        let mut itens = inventario_da_classe();
        prova(&mut itens, QUEST_PROOF_EMBLEM);
        let passo = entregar(&mut itens, MERCHANT_TRAINER_CHIEF, 3, 0b0111, 1, roll);
        match passo.expect("etapa 4").reward {
            QuestReward::Random { chosen, .. } => assert_eq!(chosen, esperado, "roll {roll}"),
            outro => panic!("recompensa inesperada: {outro:?}"),
        }
        assert!(contem(&itens, esperado));
    }
}

#[test]
fn nivel_acima_de_35_e_recusado_em_qualquer_etapa() {
    assert_eq!(QUEST_CAMPO_MAX_LEVEL, 35);
    for (merchant, stage) in [
        (MERCHANT_TRAINER_FIRST, 0),
        (MERCHANT_TRAINER_SECOND, 1),
        (MERCHANT_TRAINER_THIRD, 2),
        (MERCHANT_TRAINER_CHIEF, 3),
    ] {
        let ids = [
            QUEST_PROOF_FIRST,
            QUEST_PROOF_SECOND,
            QUEST_PROOF_LAST,
            QUEST_PROOF_EMBLEM,
        ];
        let recusa = quest_engage(&QuestEngage {
            merchant,
            stage,
            reward_mask: 0,
            level: 36,
            inventory: &ids,
            weapon_req_level: Some(REQ_LEVEL),
            roll: 0,
        })
        .expect_err("nível 36 não entra no Campo");
        assert_eq!(recusa, QuestRefusal::LevelTooHigh { level: 36, max: 35 });
    }
    // Nível 35 (o teto) continua aceito.
    let ids = [QUEST_PROOF_FIRST];
    assert!(quest_engage(&QuestEngage {
        merchant: MERCHANT_TRAINER_FIRST,
        stage: 0,
        reward_mask: 0,
        level: 35,
        inventory: &ids,
        weapon_req_level: Some(REQ_LEVEL),
        roll: 0,
    })
    .is_ok());
}

#[test]
fn prova_errada_nao_avanca_nem_consome() {
    let mut itens = inventario_da_classe();
    // Entrega a chave da 2ª porta na 1ª etapa: falta a 451.
    prova(&mut itens, QUEST_PROOF_SECOND);
    let antes = itens.len();
    let ids: Vec<i64> = itens.iter().map(|linha| linha.item_id).collect();
    let recusa = quest_engage(&QuestEngage {
        merchant: MERCHANT_TRAINER_FIRST,
        stage: 0,
        reward_mask: 0,
        level: 1,
        inventory: &ids,
        weapon_req_level: Some(REQ_LEVEL),
        roll: 0,
    })
    .expect_err("prova errada");
    assert_eq!(
        recusa,
        QuestRefusal::MissingProof {
            item: QUEST_PROOF_FIRST
        }
    );
    assert_eq!(itens.len(), antes, "nada é consumido sem aceite");
    assert!(contem(&itens, QUEST_PROOF_SECOND));
    assert!(!contem(&itens, QUEST_HEAL_KIT));
}

#[test]
fn etapa_fora_de_ordem_e_recusada() {
    let mut itens = inventario_da_classe();
    prova(&mut itens, QUEST_PROOF_SECOND);
    let ids: Vec<i64> = itens.iter().map(|linha| linha.item_id).collect();
    let recusa = quest_engage(&QuestEngage {
        merchant: MERCHANT_TRAINER_SECOND,
        stage: 0, // treinador 2 sem ter passado pelo 1
        reward_mask: 0,
        level: 1,
        inventory: &ids,
        weapon_req_level: Some(REQ_LEVEL),
        roll: 0,
    })
    .expect_err("fora de ordem");
    assert_eq!(
        recusa,
        QuestRefusal::OutOfOrder {
            merchant: MERCHANT_TRAINER_SECOND,
            expected: 1,
            current: 0,
        }
    );
    assert!(contem(&itens, QUEST_PROOF_SECOND), "prova preservada");
}

#[test]
fn recompensa_repetida_e_recusada() {
    let mut itens = inventario_da_classe();
    prova(&mut itens, QUEST_PROOF_FIRST);
    // Máscara já tem o bit da etapa 0: o estágio voltou (replay de pacote).
    let ids: Vec<i64> = itens.iter().map(|linha| linha.item_id).collect();
    let recusa = quest_engage(&QuestEngage {
        merchant: MERCHANT_TRAINER_FIRST,
        stage: 0,
        reward_mask: 1,
        level: 1,
        inventory: &ids,
        weapon_req_level: Some(REQ_LEVEL),
        roll: 0,
    })
    .expect_err("reentrega");
    assert_eq!(recusa, QuestRefusal::AlreadyRewarded { stage: 0 });
    // Segundo caminho de idempotência: com o estágio já em 1, o treinador 1 fica fora de ordem.
    let recusa = quest_engage(&QuestEngage {
        merchant: MERCHANT_TRAINER_FIRST,
        stage: 1,
        reward_mask: 1,
        level: 1,
        inventory: &ids,
        weapon_req_level: Some(REQ_LEVEL),
        roll: 0,
    })
    .expect_err("etapa já concluída");
    assert_eq!(
        recusa,
        QuestRefusal::OutOfOrder {
            merchant: MERCHANT_TRAINER_FIRST,
            expected: 0,
            current: 1,
        }
    );
    assert!(!contem(&itens, QUEST_HEAL_KIT), "nenhum kit entregue");
}

#[test]
fn encantamento_sem_arma_e_recusado() {
    let itens = [ItemRow {
        slot: CARRY_FIRST_SLOT,
        item_id: QUEST_PROOF_SECOND,
        qty: 1,
        ..ItemRow::default()
    }];
    let ids: Vec<i64> = itens.iter().map(|linha| linha.item_id).collect();
    let recusa = quest_engage(&QuestEngage {
        merchant: MERCHANT_TRAINER_SECOND,
        stage: 1,
        reward_mask: 1,
        level: 1,
        inventory: &ids,
        weapon_req_level: None,
        roll: 0,
    })
    .expect_err("sem arma");
    assert_eq!(recusa, QuestRefusal::MissingWeapon);
}

#[test]
fn portao_abre_com_o_ef_keyid_da_chave_e_fecha_por_tempo() {
    use wyd_server::gameplay::{gate_at, proof_key_ef, GateBook, GateError, CAMP_GATE_OPEN_MS};

    let portao = gate_at(2075, 2016).expect("primeiro portao do Campo");
    assert_eq!(
        portao.key_ef,
        proof_key_ef(QUEST_PROOF_FIRST).expect("451 tem EF_KEYID")
    );
    let mut livro = GateBook::default();
    let aberto = livro.open(portao, &[2], 1_000).expect("abre com a chave 2");
    assert_eq!(
        aberto.key_item, QUEST_PROOF_FIRST,
        "chave consumida e a 451"
    );
    assert!(livro.is_open(portao.id, 1_000));
    assert!(
        livro.is_open(portao.id, 1_000 + CAMP_GATE_OPEN_MS - 1),
        "aberto ate o instante do fechamento"
    );
    assert!(
        !livro.is_open(portao.id, 1_000 + CAMP_GATE_OPEN_MS),
        "fecha por tempo sem tick dedicado"
    );

    // Chave sem o EF_KEYID do portao e mochila sem chave: recusa e nada abre.
    let mut livro = GateBook::default();
    assert_eq!(livro.open(portao, &[3], 0), Err(GateError::WrongKey));
    assert_eq!(livro.open(portao, &[], 0), Err(GateError::WrongKey));
    assert_eq!(livro.abertos(0), 0);

    // Portao aberto nao aceita segunda chave (a chave nao pode sumir duas vezes).
    let mut livro = GateBook::default();
    livro.open(portao, &[2], 0).expect("abre");
    assert_eq!(livro.open(portao, &[2], 1), Err(GateError::AlreadyOpen));
}

#[test]
fn os_tres_portoes_tem_ef_keyid_distintos_e_chaves_do_campo() {
    use wyd_server::gameplay::{gate_at, proof_for_key_ef, CAMP_GATES};

    assert_eq!(CAMP_GATES.len(), 3);
    let mut chaves: Vec<i32> = CAMP_GATES.iter().map(|gate| gate.key_ef).collect();
    chaves.sort_unstable();
    assert_eq!(chaves, vec![2, 3, 4], "EF_KEYID 2/3/4 (itens 451/452/453)");
    assert!(gate_at(0, 0).is_none(), "fora dos portoes nao abre nada");
    for (x, y, item) in [(2075, 2016, 451i64), (2143, 1986, 452), (2081, 1962, 453)] {
        let portao = gate_at(x, y).expect("portao documentado");
        assert_eq!(proof_for_key_ef(portao.key_ef), Some(item));
    }
}

#[test]
fn loot_empilha_ate_o_teto_e_ocupa_slot_livre() {
    use wyd_server::gameplay::{add_item, MAX_STACK};

    let mut itens = inventario_da_classe();
    // Primeira entrega: slot de mochila livre (o 16 já tem a kill mark).
    let slot = add_item(&mut itens, QUEST_PROOF_FIRST as i32, 1).expect("entrega");
    assert_eq!(slot, CARRY_FIRST_SLOT + 1);
    assert_eq!(
        itens
            .iter()
            .find(|linha| linha.slot == slot)
            .map(|linha| linha.qty),
        Some(1)
    );

    // Segunda entrega empilha na mesma linha, sem gastar slot novo.
    add_item(&mut itens, QUEST_PROOF_FIRST as i32, 5).expect("empilha");
    assert_eq!(
        itens
            .iter()
            .find(|linha| linha.slot == slot)
            .map(|linha| linha.qty),
        Some(6)
    );

    // Teto de pilha: enche a linha e o excedente abre outra.
    add_item(&mut itens, QUEST_PROOF_FIRST as i32, (MAX_STACK - 6) as i32).expect("enche");
    assert_eq!(
        itens
            .iter()
            .find(|linha| linha.slot == slot)
            .map(|linha| linha.qty),
        Some(MAX_STACK)
    );
    let segundo = add_item(&mut itens, QUEST_PROOF_FIRST as i32, 2).expect("transborda");
    assert_ne!(segundo, slot, "excedente vai para outro slot");
    assert_eq!(
        itens
            .iter()
            .find(|linha| linha.slot == segundo)
            .map(|linha| linha.qty),
        Some(2)
    );
}

#[test]
fn loot_sem_slot_nao_e_inventado() {
    use wyd_server::gameplay::add_item;

    let mut cheio = inventario_da_classe();
    for slot in CARRY_FIRST_SLOT..CARRY_FIRST_SLOT + 64 {
        if cheio.iter().any(|linha| linha.slot == slot) {
            continue;
        }
        cheio.push(ItemRow {
            slot,
            item_id: 401,
            qty: 1,
            ..ItemRow::default()
        });
    }
    assert_eq!(add_item(&mut cheio, QUEST_PROOF_EMBLEM as i32, 1), None);
    assert!(!contem(&cheio, QUEST_PROOF_EMBLEM), "nada inventado");
}
