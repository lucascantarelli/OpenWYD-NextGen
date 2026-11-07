package handler

import (
	"fmt"

	"openwyd-nextgen/tmserver/internal/level"
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// Sub-Celestial (issue #410): a Pedra Ideal transforma um Celestial num
// CELESTIALCS escrevendo uma segunda ficha nova, SaveCelestial[1], e o Urnammu
// muda a classe dessa ficha. Ambos deixam a ficha ativa (Celestial) intacta; a
// Pedra Misteriosa 4148 troca a ficha ativa pela estacionada.
const (
	subCelestialMinLevel   = 120 // CurrentScore.Level, inclusive (_MSG_UseItem.cpp:3032)
	subCelestialAC         = 230
	subCelestialSpecialPts = 855
	// learnedSkillKeepMask mantém o byte alto de LearnedSkill (os bits
	// sem classe, por exemplo o bit 30 que a virada Celestial define) e descarta as skills.
	learnedSkillKeepMask int32 = -0x01000000 // 0xFF000000

	urnammuMerchant     = 76   // MOB.Merchant de Urnammu/Urnamu (npcMerc == 76, _MSG_Quest.cpp:154)
	urnammuFameCost     = 2000 // _MSG_Quest.cpp:1262
	urnammuSpecialReset = 55   // teto de reembolso por Special (_MSG_Quest.cpp:1285)
	subCelestialSheet   = 1    // índice de SaveCelestial do Sub-Celestial
)

// Linhas do Language.txt, nos bytes Windows-1252 da tabela.
const (
	msgWantEquipItem = "Voc\xea deve equipar %s."              // _SN_WANTEQUIPITEM (477)
	msgNeedFame      = "Voc\xea precisa de %d pontos de fama." // _DN_NEEDFAME (523)
)

// equippedSephirotClass devolve a classe base 0 do Sephirot (1760..1763)
// em Equip[11].
func equippedSephirotClass(e *world.Entity) (int32, bool) {
	idx := e.Equip[sephirotEquipSlot].Index
	if idx < archSephirotMin || idx > archSephirotMax {
		return 0, false
	}
	return int32(idx - archSephirotMin), true
}

// createSubCelestial é a perna CELESTIAL da Pedra Ideal (Vol 211,
// _MSG_UseItem.cpp:3026-3085). Sem Sephirot ou abaixo do nível 120 o legado
// só reenvia a pedra (SendItem). Caso contrário o personagem vira CELESTIALCS
// e SaveCelestial[1] recebe uma ficha de nível 0 da classe do Sephirot; a ficha
// Celestial ativa não é tocada, então nada que o jogador tem é sobrescrito.
//
// Peculiaridades de paridade mantidas de propósito: Str/Int/Dex/Con vêm da classe *atual*
// (BaseSIDCHM[MOB.Class]) enquanto HP/MP vêm da classe do Sephirot, e os
// campos que o legado não atribui (SPX/SPY, ScoreBonus, SkillBonus, as barras
// de skill, SecLearnedSkill) mantêm o que o slot já tinha.
func (d *Dispatcher) createSubCelestial(w *world.World, s *world.Session, e *world.Entity, src int) {
	cls, ok := equippedSephirotClass(e)
	if !ok || e.Level < subCelestialMinLevel {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	e.ClassMaster = classMasterCelestialCS

	attrs := level.BaseStats(e.Class)
	hpmp := level.BaseStats(uint8(cls))
	sub := &e.SaveCelestial[subCelestialSheet]
	sub.Class = cls
	sub.LearnedSkill = e.LearnedSkill & learnedSkillKeepMask
	sub.Level = 0
	sub.Str, sub.Int, sub.Dex, sub.Con = int16(attrs[0]), int16(attrs[1]), int16(attrs[2]), int16(attrs[3])
	sub.Special = [4]int16{}
	sub.AC, sub.Damage = subCelestialAC, 0
	sub.HP, sub.MaxHP = hpmp[4], hpmp[4]
	sub.MP, sub.MaxMP = hpmp[5], hpmp[5]
	sub.SpecialBonus = subCelestialSpecialPts
	sub.Exp = 0
	sub.Soul = e.Soul

	e.Equip[sephirotEquipSlot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceEquip, sephirotEquipSlot, e.Equip[sephirotEquipSlot])
	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	d.sendClientMessage(w, s, msgKingBless1)
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("ideal stone used to make god2", "conn", s.Conn, "name", e.Name, "subclass", cls)
}

// isUrnammuNPC lê MOB.Merchant do template como o legado (byte 17).
func isUrnammuNPC(npc *world.Entity) bool {
	return npc != nil && len(npc.Template) > 17 && npc.Template[17] == urnammuMerchant
}

// turnammu porta o caso URNAMMU (_MSG_Quest.cpp:1248-1311): um CELESTIALCS com
// um Sephirot equipado e 2000 de Fama muda a classe da ficha Sub estacionada.
// O legado nunca checa confirmação. Os atributos são resetados a partir da classe
// *ativa* (BaseSIDCHM[MOB.Class]), cada Special reembolsa no máximo 55 pontos no
// SpecialBonus da ficha, e as barras de skill e skills aprendidas são limpas.
func (d *Dispatcher) urnammu(w *world.World, s *world.Session, e *world.Entity) {
	if e.ClassMaster != classMasterCelestialCS {
		d.sendClientMessage(w, s, fmt.Sprintf(msgNeedClass, "Sub Celestial"))
		return
	}
	cls, ok := equippedSephirotClass(e)
	if !ok {
		d.sendClientMessage(w, s, fmt.Sprintf(msgWantEquipItem, "o Sephirot da classe escolhida"))
		return
	}
	if e.Fame < urnammuFameCost {
		d.sendClientMessage(w, s, fmt.Sprintf(msgNeedFame, urnammuFameCost))
		return
	}
	d.log.Info("change SubClass", "conn", s.Conn, "name", e.Name, "class", e.Class, "to", cls, "level", e.Level)
	e.Fame -= urnammuFameCost

	sub := &e.SaveCelestial[subCelestialSheet]
	sub.Class = cls
	attrs := level.BaseStats(e.Class)
	sub.Str, sub.Int, sub.Dex, sub.Con = int16(attrs[0]), int16(attrs[1]), int16(attrs[2]), int16(attrs[3])
	sub.SkillBar1, sub.SkillBar2 = [4]uint8{}, [16]uint8{}

	special := int32(sub.SpecialBonus)
	for i, v := range sub.Special {
		if v <= urnammuSpecialReset {
			special += int32(v)
			sub.Special[i] = 0
		} else {
			special += urnammuSpecialReset
			sub.Special[i] -= urnammuSpecialReset
		}
	}
	sub.SpecialBonus = uint16(special) // unsigned short na ficha legada
	sub.LearnedSkill &= learnedSkillKeepMask

	e.Equip[sephirotEquipSlot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceEquip, sephirotEquipSlot, e.Equip[sephirotEquipSlot])
	d.sendClientMessage(w, s, msgKingBless1)
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
}

// pedraMisteriosaKeptAffects são os tipos de affect que a troca deixa no lugar
// (_MSG_UseItem.cpp:3244-3250): 4, 8, 29, o Frango Assado, as poções Divina e
// de Vigor e o Baú de Experiência. Todo o resto é limpo.
var pedraMisteriosaKeptAffects = map[uint8]bool{
	4: true, 8: true, 29: true,
	world.AffectForceMobDamage: true, world.AffectDivine: true,
	world.AffectVigor: true, world.AffectExpChest: true,
}

// usePedraMisteriosa porta a Pedra Misteriosa (4148, _MSG_UseItem.cpp:3186-3289):
// dentro de uma vila, um CELESTIALCS ou SCELESTIAL estaciona a ficha ativa em
// SaveCelestial e toca a outra. Fora desses níveis o legado só
// reenvia a pedra; fora de uma vila ele também diz _NN_OnlyVillage.
//
// Peculiaridades de paridade mantidas de propósito:
// - HP/MP vivos passam da ficha que está sendo deixada (GetCurrentScore mantém
// CurrentScore.Hp/Mp) e são apenas limitados aos novos máximos.
// - BASE_GetHpMp roda antes de GetCurrentScore, então os novos BaseMaxHP/MaxMP usam
// o nível da ficha sendo *deixada* (CurrentScore.Level ainda está obsoleto).
// SkillBonus não é persistido aqui e é rederivado a cada login, então
// usa o nível novo em vez de um obsoleto que não sobreviveria a um relog.
// - A pedra é consumida depois da troca, e nada é salvo pelo legado
// aqui; nós salvamos mesmo assim para um crash não poder desfazer a troca que o cliente viu.
func (d *Dispatcher) usePedraMisteriosa(w *world.World, s *world.Session, e *world.Entity, src int) {
	if e.ClassMaster != classMasterCelestialCS && e.ClassMaster != classMasterSCelestial {
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}
	if v := world.Village(e.X, e.Y); v < 0 || v >= 5 {
		d.notify(w, s, NoticeOnlyVillage)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	cl, ncl := 0, subCelestialSheet
	if e.ClassMaster == classMasterCelestialCS {
		e.ClassMaster = classMasterSCelestial
		e.CelestialLevel = int16(e.Level)
	} else {
		cl, ncl = subCelestialSheet, 0
		e.ClassMaster = classMasterCelestialCS
		e.SubCelestialLevel = int16(e.Level)
	}
	oldClass, oldLevel := e.Class, e.Level
	e.SaveCelestial[cl] = parkCelestialSheet(s, e)
	loadCelestialSheet(s, e, e.SaveCelestial[ncl])

	for i := range e.Affect {
		if !pedraMisteriosaKeptAffects[e.Affect[i].Type] {
			e.Affect[i] = world.Affect{}
		}
	}

	// Equip[0] = MortalFace + 5 + Class. O tmServer não mantém MortalFace,
	// mas nada reescreve o rosto de um jogador depois que o Arch foi criado com essa mesma
	// fórmula, então ele é recuperado do rosto e da classe atuais.
	if face := e.Equip[0].Index; face > 0 {
		if mortalFace := int32(face) - 5 - int32(oldClass); mortalFace > 0 {
			e.Equip[0].Index = int16(mortalFace + 5 + int32(e.Class))
		}
	}
	d.sendSlot(w, s, world.ItemPlaceEquip, 0, e.Equip[0])
	w.SendTo(s, protocol.Header{Type: protocol.MsgSetShortSkill, ID: protocol.IDScene},
		protocol.EncodeSetShortSkill(e.SkillBar, s.ShortSkill))

	d.deriveSkillBonus(e)
	e.ScoreBonus = scoreBonusFor(e)
	e.BaseMaxHP, e.BaseMaxMP = celestialBaseHpMp(e.Class, e.BaseCon, e.BaseInt, oldLevel)
	d.refreshEquip(w, s, e) // rosto novo no modelo + GetCurrentScore + SendScore
	d.sendAffect(w, s, e)
	d.sendEtc(w, s, e)

	consumeOneItem(&e.Carry[src])
	d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("mystery stone change god2god", "conn", s.Conn, "name", e.Name,
		"classMaster", e.ClassMaster, "class", oldClass, "to", e.Class)
}

// parkCelestialSheet copia a ficha ativa para uma entrada de SaveCelestial.
func parkCelestialSheet(s *world.Session, e *world.Entity) world.CelestialSheet {
	return world.CelestialSheet{
		Class:           int32(e.Class),
		Exp:             e.Exp,
		SPX:             e.SaveX,
		SPY:             e.SaveY,
		Level:           e.Level,
		AC:              e.BaseAC,
		Damage:          e.BaseDamage,
		MaxHP:           e.BaseMaxHP,
		MaxMP:           e.BaseMaxMP,
		HP:              e.HP,
		MP:              e.MP,
		Str:             e.BaseStr,
		Int:             e.BaseInt,
		Dex:             e.BaseDex,
		Con:             e.BaseCon,
		Special:         e.BaseSpecial,
		LearnedSkill:    e.LearnedSkill,
		SecLearnedSkill: e.SecLearnedSkill,
		ScoreBonus:      e.ScoreBonus,
		SpecialBonus:    e.SpecialBonus,
		SkillBonus:      e.SkillBonus,
		SkillBar1:       e.SkillBar,
		SkillBar2:       s.ShortSkill,
		Soul:            e.Soul,
	}
}

// loadCelestialSheet torna sh a ficha ativa. HP/MP vivos são mantidos (ver
// usePedraMisteriosa); BaseAC/BaseDamage são rederivados do novo tier e
// nível, como no login, já que o tmServer nunca os persiste.
func loadCelestialSheet(s *world.Session, e *world.Entity, sh world.CelestialSheet) {
	e.Class = uint8(sh.Class)
	e.Exp = sh.Exp
	e.SaveX, e.SaveY = sh.SPX, sh.SPY
	e.Level = sh.Level
	e.BaseMaxHP, e.BaseMaxMP = sh.MaxHP, sh.MaxMP
	e.BaseStr, e.BaseInt, e.BaseDex, e.BaseCon = sh.Str, sh.Int, sh.Dex, sh.Con
	e.BaseSpecial = sh.Special
	e.LearnedSkill = sh.LearnedSkill
	e.SecLearnedSkill = sh.SecLearnedSkill
	e.ScoreBonus = sh.ScoreBonus
	e.SpecialBonus = sh.SpecialBonus
	e.SkillBonus = sh.SkillBonus
	e.SkillBar = sh.SkillBar1
	s.ShortSkill = sh.SkillBar2
	e.Soul = sh.Soul
	e.BaseAC = playerBaseAC(e)
	e.BaseDamage = playerBaseDamage(e)
}

// celestialBaseHpMp é BASE_GetHpMp (Basedef.cpp:1214-1248) para os níveis
// celestiais, cujo termo de nível soma MAX_LEVEL: base da classe + 2 por Con/Int acima da
// base da classe + IncHP/IncMP por nível, com teto em MAX_HP/MAX_MP.
func celestialBaseHpMp(cls uint8, baseCon, baseInt int16, lvl int32) (maxHP, maxMP int32) {
	base := level.BaseStats(cls)
	maxHP = base[4] + (int32(baseCon)-base[3])*2 + (lvl+level.MaxLevel)*level.IncHP(cls)
	maxMP = base[5] + (int32(baseInt)-base[1])*2 + (lvl+level.MaxLevel)*level.IncMP(cls)
	return min(maxHP, level.MaxHPCap), min(maxMP, level.MaxMPCap)
}
