package handler

import (
	"fmt"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/world"
)

// Pedra_da_Fúria (item 3020) executa as três quests Celestiais que /destravar90
// e /arcana só atalham: Cythera Arcana e o reset Celestial para um
// tier celestial de nível 199, e o desbloqueio Lv90 para um CELESTIAL
// (_MSG_UseItem.cpp:3470-3690, issue #408). Ela não tem EF_VOLATILE no
// ItemList.csv, então o useItem a roteia por sIndex.
const (
	furyStoneItem = 3020

	furyStoneFame = 500 // Fama que todo ramo custa

	// furyStoneArcanaLevel é o nível armazenado (exibido 200) que a Arcana e
	// o reset exigem; furyStoneLv90Level é o 89 armazenado que o desbloqueio Lv90 exige.
	furyStoneArcanaLevel = 199
	furyStoneLv90Level   = 89

	celestialResetMax = 3          // teto de QuestInfo.Celestial.Reset
	celestialResetExp = 20_000_000 // exp que cada reset retira

	msgFailure = "Resultou em falha." // _NN_FAILURE (476)
)

// useFuryStone porta o bloco 3020 de _MSG_UseItem.cpp:3470-3690 na sua ordem.
// Um uso recusado deixa a pedra no lugar e ressincroniza o slot, como o
// SendItem legado faz; uma vez que uma quest roda, a pedra é gasta mesmo falhando.
func (d *Dispatcher) useFuryStone(w *world.World, s *world.Session, e *world.Entity, src int) {
	resync := func() { d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src]) }
	if isCelestialMaster(e.ClassMaster) && e.Level >= furyStoneArcanaLevel {
		if e.Fame < furyStoneFame {
			resync()
			return
		}
		if e.CelCircle == 0 {
			slots, ok := pedraSecretaSlots(e)
			if !ok {
				resync()
				return
			}
			d.furyStoneArcana(w, s, e, src, slots)
			return
		}
		if e.CelReset < celestialResetMax {
			d.furyStoneReset(w, s, e, src)
			return
		}
		// Um celestial com todos os resets gastos cai nas checagens de Lv90,
		// onde só um CELESTIAL que nunca desbloqueou Lv90 ainda pode usá-la.
	}
	if e.ClassMaster != classMasterCelestial {
		resync()
		return
	}
	if e.Level < furyStoneLv90Level {
		d.sendClientMessage(w, s, msgLevelLimit2)
		resync()
		return
	}
	if e.CelLv90 == 1 {
		d.sendClientMessage(w, s, msgYouveDoneAlready)
		resync()
		return
	}
	if e.Fame < furyStoneFame {
		resync()
		return
	}
	// Aqui o legado sorteia antes de cobrar a Fama.
	success := furyStoneRoll(w)
	e.Fame -= furyStoneFame
	if success {
		e.CelLv90 = 1
		d.playUnlockEmote(w, s)
		d.sendClientMessage(w, s, fmt.Sprintf(msgPlayQuest, "Lv90"))
		consumeOneItem(&e.Carry[src])
		resync()
		d.newbiePutItem(w, s, e, world.Item{Index: cytheraMisticaItem})
	} else {
		d.sendClientMessage(w, s, msgFailure)
		consumeOneItem(&e.Carry[src])
		resync()
	}
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("fury stone lv90", "conn", s.Conn, "name", e.Name, "success", success)
}

// furyStoneRoll é o rand()%115 legado com 101..114 reduzido em 15 para
// 86..99. Esses ainda passam no teste < 100, então só um sorteio de exatamente 100 falha:
// 114 dos 115 resultados dão certo, não os ~87% que a issue estimou.
func furyStoneRoll(w *world.World) bool {
	roll := w.Rand().Intn(115)
	if roll > 100 {
		roll -= 15
	}
	return roll < 100
}

// furyStoneArcana porta a quest Cythera Arcana (_MSG_UseItem.cpp:3480-3568):
// as quatro Pedras Secretas e 500 de Fama saem primeiro, depois o sorteio decide se
// Equip[1] vira o 3507 e QuestInfo.Circle é definido.
func (d *Dispatcher) furyStoneArcana(w *world.World, s *world.Session, e *world.Entity, src int, slots [len(pedraSecretaItems)]int) {
	// O legado percorre o inventário uma vez, então os slots limpam na ordem de slot.
	for i := 0; i < activeCarryLimit(e); i++ {
		for _, slot := range slots {
			if slot == i {
				e.Carry[i] = world.Item{}
				d.sendSlot(w, s, world.ItemPlaceCarry, i, e.Carry[i])
			}
		}
	}
	e.Fame -= furyStoneFame
	success := furyStoneRoll(w)
	if success {
		// Só o sIndex muda: uma armadura vestida mantém seu refino e efeitos
		// sob o novo 3507, já que o legado escreve só o sIndex.
		e.Equip[arcanaEquipSlot].Index = arcanaItemIndex
		d.sendSlot(w, s, world.ItemPlaceEquip, arcanaEquipSlot, e.Equip[arcanaEquipSlot])
		d.sendClientMessage(w, s, fmt.Sprintf(msgPlayQuest, "Cythera Arcana"))
		d.playUnlockEmote(w, s)
		e.CelCircle = 1
	} else {
		d.sendClientMessage(w, s, msgFailure)
	}
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("fury stone arcana", "conn", s.Conn, "name", e.Name, "success", success)
}

// furyStoneReset porta o reset Celestial (_MSG_UseItem.cpp:3579-3615): sem
// sorteio, +1 Reset, -500 de Fama e -20M de exp, o nível consultado de novo na
// curva celestial e os pontos livres recalculados, onde cada reset vale
// +200. O CheckGetLevel logo depois no legado, com SetCircletSubGod quando ele
// devolve 4 (:3596-3599, issue #409), é o applyLevelUps: o nível acabou de
// ser consultado da exp, então normalmente não acha nada a subir.
func (d *Dispatcher) furyStoneReset(w *world.World, s *world.Session, e *world.Entity, src int) {
	e.CelReset++
	e.Fame -= furyStoneFame
	e.Exp -= celestialResetExp
	e.Level = level.ForExpTier(e.Exp, e.ClassMaster)
	// O login em Go deriva BaseAC do nível, então baixe-o agora para manter o
	// valor vivo igual ao que o próximo login reconstrói.
	e.BaseAC = playerBaseAC(e)
	d.applyLevelUps(w, s, e)
	e.ScoreBonus = scoreBonusFor(e)
	d.refreshScore(e)
	d.sendEtc(w, s, e)
	d.sendScore(w, s, e)
	d.playUnlockEmote(w, s)
	d.sendClientMessage(w, s, fmt.Sprintf(msgPlayQuest, "Reset Celestial"))
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("fury stone reset", "conn", s.Conn, "name", e.Name, "reset", e.CelReset, "level", e.Level)
}
