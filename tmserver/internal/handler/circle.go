package handler

import (
	"openwyd-nextgen/tmserver/internal/world"
)

// Círculo Divino (_MSG_Quest.cpp:514-691): o Jeffi purifica o círculo vestido
// em Equip[13] (ou agrupa pó de Ori/Lac) e o Shaman gasta o círculo puro
// para resetar as masteries e uma árvore de skill. O legado roteia ambos por
// MOB.Merchant (byte 17 do template): 12 é o Jeffi (mais Alchemy_Jeffi e
// Torre_Real, que compartilham o byte) e 13 o Shaman. O Curandeiro vem com
// CurrentScore.Merchant 13 mas byte 17 = 200, então rotear por npc.Merchant o
// mandaria para cá por engano (issue #405).
const (
	jeffiMerchant = 12
	shamaMerchant = 13

	circleEquipSlot = 13

	circleCommonRaw  = 447 // Círculo Divino, purificado por 1.000.000
	circleCommonPure = 448 // 448..450: círculo comum puro (um por árvore de skill)
	circleFullRaw    = 692 // círculo completo, purificado por 5.000.000
	circleFullPure   = 693 // 693..695: círculo completo puro

	circleCommonPrice = 1000000
	circleFullPrice   = 5000000

	circleCommonReset = 50  // Pontos de Special devolvidos por mastery (comum)
	circleFullReset   = 100 // ... e com o círculo completo

	itemOriDust = 419 // Poeira de Ori, 10 -> Ori (412)
	itemLacDust = 420 // Poeira de Lac, 10 -> Lac (413)
	itemOri     = 412
	itemLac     = 413
	dustPerItem = 10

	circleDustPrice = 1000000

	// SetAffect(conn, 44, 20, 20) depois de todo serviço de Jeffi/Shaman.
	circleAffectSkill = 44
	circleAffectTime  = 20

	combinePrizeEffect = 59 // EF dos três efeitos sorteados de um prêmio de Combine
)

// Linhas do Language.txt, nos bytes Windows-1252 da tabela.
const (
	msgNeed1MGold         = "Requer 1000000 gold."                                      // _NN_Need_1000000_Gold (152)
	msgNeed5MGold         = "Para refinar o item s\xe3o necess\xe1rios 5000000 Gold."   // _NN_Need_5000000_Gold (166)
	msgNeed10Particle     = "Voc\xea pode agrupar Ori/Lac em 10 unidades."              // _NN_Need_10_Particle (157)
	msgProcessingComplete = "Processo de combina\xe7\xe3o foi conclu\xeddo."            // _NN_Processing_Complete (158)
	msgNeedPureDivine     = "Requer C\xedrculo Divino Puro para a reiniciar as skills." // _NN_Need_Pure_Divine (155)
	msgInitializeSkill    = "Skills foram reiniciadas."                                 // _NN_Initialize_Skill (156)
	msgQuest06Complete    = "Quest: Voc\xea completou a \"Quest 6\"."                   // _NN_Qest06Complete (6)
)

func isJeffiNPC(npc *world.Entity) bool {
	return npc != nil && len(npc.Template) > 17 && npc.Template[17] == jeffiMerchant
}

func isShamaNPC(npc *world.Entity) bool {
	return npc != nil && len(npc.Template) > 17 && npc.Template[17] == shamaMerchant
}

// jeffi é o caso JEFFI (_MSG_Quest.cpp:514-610). O legado não checa nem
// a classe nem a confirmação.
func (d *Dispatcher) jeffi(w *world.World, s *world.Session, e, npc *world.Entity) {
	circle := e.Equip[circleEquipSlot].Index
	if circle == circleCommonRaw || circle == circleFullRaw {
		price, base, need := int32(circleCommonPrice), circleCommonPure, msgNeed1MGold
		if circle == circleFullRaw {
			price, base, need = circleFullPrice, circleFullPure, msgNeed5MGold
		}
		if e.Coin < price {
			d.sendNPCChatText(w, s, npc.ID, need)
			return
		}
		e.Coin -= price
		// Só o sIndex muda: o círculo mantém seus efeitos, como no legado.
		e.Equip[circleEquipSlot].Index = int16(w.Rand().Intn(3) + base)
		d.sendSlot(w, s, world.ItemPlaceEquip, circleEquipSlot, e.Equip[circleEquipSlot])
		d.sendEtc(w, s, e)
		d.sendNPCChatText(w, s, npc.ID, msgProcessingComplete)
		d.circleAffect(w, s, e)
		w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
		d.log.Info("jeffi circle", "conn", s.Conn, "name", e.Name, "from", circle, "to", e.Equip[circleEquipSlot].Index)
		return
	}

	// A contagem para no inventário acessível (pMob.MaxCarry) enquanto
	// o Combine varre todos os slots MAX_CARRY, como no legado.
	ori, lac := 0, 0
	limit := activeCarryLimit(e)
	for i := 0; i < limit; i++ {
		switch e.Carry[i].Index {
		case itemOriDust:
			ori += stackAmount(e.Carry[i])
		case itemLacDust:
			lac += stackAmount(e.Carry[i])
		}
	}
	if ori < dustPerItem && lac < dustPerItem {
		d.sendNPCChatText(w, s, npc.ID, msgNeed10Particle)
		return
	}
	if e.Coin < circleDustPrice {
		d.sendNPCChatText(w, s, npc.ID, msgNeed1MGold)
		return
	}
	// Uma cobrança de 1.000.000 por mais grupos que sejam feitos.
	for ori >= dustPerItem || lac >= dustPerItem {
		if ori >= dustPerItem {
			d.combineDust(w, e, itemOriDust, itemOri)
			ori -= dustPerItem
		}
		if lac >= dustPerItem {
			d.combineDust(w, e, itemLacDust, itemLac)
			lac -= dustPerItem
		}
	}
	d.sendNPCChatText(w, s, npc.ID, msgProcessingComplete)
	e.Coin -= circleDustPrice
	d.circleAffect(w, s, e)
	d.sendCarry(w, s, e)
	d.sendEtc(w, s, e)
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("jeffi lac_ori", "conn", s.Conn, "name", e.Name)
}

// stackAmount é a contagem de unidades ciente de EF_AMOUNT que o Jeffi e o Combine usam.
func stackAmount(it world.Item) int {
	if it.Effects[0].Effect == efAmount {
		return int(it.Effects[0].Value)
	}
	return 1
}

// combineDust é o Combine (Server.cpp:9448-9527): come 10 unidades do item e
// entrega PrizeItem com três efeitos EF 59 sorteados. Peculiaridades mantidas: a primeira passagem
// para no slot que chega a 10, então pilhas de 5 + 8 contam 13 e os dois
// slots são limpos; uma pilha acima de 10 só perde 10. Sem slot livre o
// prêmio cai no chão (CreateItem com rotação rand() % 4).
func (d *Dispatcher) combineDust(w *world.World, e *world.Entity, item, prize int16) {
	found := 0
	for i := 0; i < world.MaxCarry; i++ {
		if e.Carry[i].Index == item {
			found += stackAmount(e.Carry[i])
		}
		if found >= dustPerItem {
			break
		}
	}
	if found < dustPerItem {
		return
	}
	cleared := 0
	for i := 0; i < world.MaxCarry; i++ {
		it := &e.Carry[i]
		if it.Index != item {
			continue
		}
		if cleared >= found {
			break
		}
		cleared += stackAmount(*it)
		if it.Effects[0].Effect == efAmount && it.Effects[0].Value > dustPerItem {
			it.Effects[0].Value -= dustPerItem
		} else {
			*it = world.Item{}
		}
	}

	r := w.Rand()
	p := world.Item{Index: prize}
	p.Effects[0] = world.Effect{Effect: combinePrizeEffect, Value: uint8(r.Intn(100))}
	p.Effects[1] = world.Effect{Effect: combinePrizeEffect, Value: uint8(r.Intn(100))}
	p.Effects[2] = world.Effect{Effect: combinePrizeEffect, Value: uint8(r.Intn(50) + 50)}
	for i := 0; i < world.MaxCarry; i++ {
		if e.Carry[i].Index == 0 {
			e.Carry[i] = p
			return
		}
	}
	r.Rand() // rotação do CreateItem
	w.CreateGroundItem(p, e.X, e.Y)
}

// shama é o caso SHAMA (_MSG_Quest.cpp:612-691): gasta um círculo puro para
// devolver até 50 (comum) / 100 (completo) pontos de cada mastery e esquecer
// a árvore de skill que o círculo nomeia. O círculo comum funciona uma vez por personagem
// (QuestInfo.Circle, CelCircle aqui); o completo não tem limite.
func (d *Dispatcher) shama(w *world.World, s *world.Session, e, npc *world.Entity) {
	circle := int(e.Equip[circleEquipSlot].Index)
	var tree int
	var common bool
	switch {
	case circle >= circleCommonPure && circle < circleCommonPure+3:
		tree, common = circle-circleCommonPure, true
	case circle >= circleFullPure && circle < circleFullPure+3:
		tree = circle - circleFullPure
	default:
		d.sendNPCChatText(w, s, npc.ID, msgNeedPureDivine)
		return
	}
	if common && e.CelCircle == 1 {
		d.sendNPCChatText(w, s, npc.ID, msgYouveDoneAlready)
		return
	}

	reset := int16(circleFullReset)
	if common {
		reset = circleCommonReset
	}
	special := int(e.SpecialBonus)
	for i := range e.BaseSpecial {
		if e.BaseSpecial[i] <= reset {
			special += int(e.BaseSpecial[i])
			e.BaseSpecial[i] = 0
		} else {
			special += int(reset)
			e.BaseSpecial[i] -= reset
		}
	}
	e.SpecialBonus = uint16(special)
	if common {
		e.CelCircle = 1
	}
	e.LearnedSkill &^= int32(0xFF) << (8 * tree)
	d.deriveSkillBonus(e)
	e.Equip[circleEquipSlot] = world.Item{}

	d.sendSlot(w, s, world.ItemPlaceEquip, circleEquipSlot, e.Equip[circleEquipSlot])
	d.sendEtc(w, s, e)
	d.sendNPCChatText(w, s, npc.ID, msgInitializeSkill)
	d.circleAffect(w, s, e)
	d.sendClientMessage(w, s, msgQuest06Complete)
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("divine circle", "conn", s.Conn, "name", e.Name, "item", circle, "common", common, "tree", tree)
}

// circleAffect é a cauda SetAffect(conn, 44, 20, 20) + SendScore que os dois NPCs
// compartilham; o score é recalculado primeiro para as masteries resetadas aparecerem.
func (d *Dispatcher) circleAffect(w *world.World, s *world.Session, e *world.Entity) {
	d.questAffect(e, circleAffectSkill, circleAffectTime, circleAffectTime)
	d.refreshScore(e)
	d.sendScore(w, s, e)
	d.sendAffect(w, s, e)
}
