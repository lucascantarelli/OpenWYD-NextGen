package handler

import (
	"context"
	"fmt"
	"time"

	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/world"
)

// mountExpiryDays é o tempo de vida concedido a uma montaria de recompensa Perzen ("X(30dias)",
// BASE_SetItemDate(30)).
const mountExpiryDays = 30

// applyBonus trata _MSG_ApplyBonus (0x0277): gastar um ponto grátis
// (protocol-spec.md §3.5, _MSG_ApplyBonus.cpp). BonusType roteia: 0 = ponto de atributo
// (Str/Int/Dex/Con), 1 = ponto de maestria (árvore Special), 2 = aprender uma skill
// com um mestre de classe.
func (d *Dispatcher) applyBonus(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	var body protocol.MsgApplyBonusBody
	if err := body.Decode(payload); err != nil {
		return
	}
	switch body.BonusType {
	case protocol.BonusScore:
		d.applyScoreBonus(w, s, e, int(body.Detail))
	case protocol.BonusSpecial:
		d.applySpecialBonus(w, s, e, int(body.Detail))
	case protocol.BonusSkill:
		d.learnSkill(w, s, e, int(body.Detail), int(body.TargetID))
	}
}

// applyScoreBonus é ApplyBonus BonusType==0 (_MSG_ApplyBonus.cpp:32-84):
// alocar pontos de atributo no BaseScore; refreshScore os dobra no
// CurrentScore vivo (= base + equipamento) e sendScore mostra. Peculiaridades
// de paridade: um estoque de ScoreBonus ≥300 gasta 100 pontos por clique, e Int/Con
// também somam 2×pontos a MaxMP/MaxHP diretamente.
func (d *Dispatcher) applyScoreBonus(w *world.World, s *world.Session, e *world.Entity, detail int) {
	if e.ScoreBonus == 0 {
		d.sendScore(w, s, e)
		return
	}
	points := int16(1)
	if e.ScoreBonus >= 300 {
		points = 100
	}
	switch detail {
	case protocol.DetailStr:
		e.BaseStr += points
	case protocol.DetailInt:
		e.BaseInt += points
		e.BaseMaxMP += 2 * int32(points)
	case protocol.DetailDex:
		e.BaseDex += points
	case protocol.DetailCon:
		e.BaseCon += points
		e.BaseMaxHP += 2 * int32(points)
	default:
		return
	}
	e.ScoreBonus -= uint16(points)
	d.refreshScore(e)
	d.sendEtc(w, s, e) // os pontos restantes vivem no UpdateEtc (B10)
	d.sendScore(w, s, e)
}

// accountSecure trata _MSG_AccountSecure (0x0FDE): o PIN numérico
// (lote2-sessao-conta.md). ChangeNumeric seleciona verificar (0) vs definir/trocar (1). O
// PIN é validado contra um hash argon2id no dbServer (issue #120) — nunca
// texto plano, nunca logado. No sucesso o cliente recebe um _MSG_AccountSecure só de header
// (ID=ESCENE_FIELD) para avançar além da tela de senha segura; com PIN errado
// recebe _MSG_AccountSecureFail e fica onde está.
//
// Primeira definição: uma verificação contra uma conta sem PIN ainda usa o token
// informado como novo PIN (comportamento legado NumericToken[0]==-1). Sem um dbServer o
// passo é reconhecido para o bring-up ainda funcionar (allow-all, como o billing).
func (d *Dispatcher) accountSecure(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	var body protocol.MsgAccountSecureBody
	if err := body.Decode(payload); err != nil {
		return
	}
	if s.AccountID == 0 {
		return // nenhuma conta vinculada ainda — ignorar
	}
	pin := body.PIN()
	change := body.ChangeNumeric == 1
	accountID := s.AccountID
	p := w.Persistence()

	w.Go(s, func() func(*world.World, *world.Session) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var ok bool
		var err error
		switch {
		case change:
			ok, err = p.SetPin(ctx, accountID, pin)
		default:
			var res world.PinResult
			res, err = p.VerifyPin(ctx, accountID, pin)
			if res == world.PinNotSet && err == nil {
				ok, err = p.SetPin(ctx, accountID, pin) // primeira definição: o PIN digitado vira o PIN
			} else {
				ok = res == world.PinOK
			}
		}
		return func(w *world.World, s *world.Session) { d.completeAccountSecure(w, s, ok, err) }
	})
}

// completeAccountSecure roda de volta no loop com o resultado do PIN. Um backend
// ausente (errNoPersistence) degrada para allow-all para o bring-up inicial não ficar bloqueado
// na tela de senha segura.
func (d *Dispatcher) completeAccountSecure(w *world.World, s *world.Session, ok bool, err error) {
	if err != nil {
		d.log.Warn("account secure check failed", "conn", s.Conn, "err", err)
		ok = true // degradar para permitir (sem dbServer, ou erro transitório) — nunca vazar o PIN
	}
	if ok {
		w.SendTo(s, protocol.Header{Type: protocol.MsgAccountSecure, ID: protocol.IDScene}, nil)
		return
	}
	w.SendTo(s, protocol.Header{Type: protocol.MsgAccountSecureFail, ID: protocol.IDScene}, nil)
}

// quest trata _MSG_Quest (0x028B): interação de quest com NPC. O corpo é
// MSG_STANDARDPARM2 (Parm1 = npcIndex, Parm2 = confirm). O subtipo do NPC vem
// de MOB.Merchant (+ EF_GRADE0 para Merchant==100), veja _MSG_Quest-npcs.md.
//
// Roteado até agora: toda a cadeia da Quest 256 (grade 0-4), os NPCs de
// troca de itens PERZEN (grade 7/8/9), o Mestre Grifo (Merchant 23) — um atalho
// de conteúdo do servidor para as arenas da Quest 256 — e o caminho de criação do
// Rei Arch, mais os NPCs de serviço avulsos abaixo. Os demais tipos de NPC de quest (tutoriais, teleportes) ainda
// não estão roteados.
func (d *Dispatcher) quest(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	if s.Trade.Active {
		d.removeTrade(w, s) // interagir com um NPC de quest cancela uma troca
	}
	npcIndex, confirm, ok := protocol.StandardParm2(payload)
	if !ok || npcIndex < world.MaxUser || int(npcIndex) >= world.MaxMob {
		return
	}
	npc := w.Entity(int(npcIndex))
	if npc == nil || npc.Mode == world.MobEmpty {
		return
	}
	if isSurvivorNPC(npc) {
		d.survivorExchange(w, s, e)
		return
	}
	// TREINADORNEWBIE1..4 (_MSG_Quest.cpp:82-101): roteado pelo
	// MOB.Merchant do template antes do Coveiro, cujo CurrentScore.Merchant 100 /
	// EF_GRADE0 0 Treinador1 também tem (issue #395).
	if step, ok := newbieTrainerStep(npc); ok {
		d.newbieQuest(w, s, e, npc, step)
		return
	}
	// QUEST_COVEIRO (Merchant 100, EF_GRADE0 0): passo 1 da cadeia da Quest 256.
	// Grade também é 0 para templates Merchant-100 sem EF_GRADE0; rotear esses
	// aqui casa com BASE_GetItemAbilityNosanc no servidor legado.
	if npc.Merchant == 100 && npc.Grade == 0 {
		d.quest256NPC(w, s, e, quest256Steps[0], itemVelaDoCoveiro)
		return
	}
	// QUEST_JARDINEIRO (Merchant 100, EF_GRADE0 1): segunda arena da Quest 256.
	if npc.Merchant == 100 && npc.Grade == 1 {
		d.quest256NPC(w, s, e, quest256Steps[1], itemColheitaDoJardineiro)
		return
	}
	// QUEST_KAIZEN (Merchant 100, EF_GRADE0 2): terceira arena da Quest 256, a
	// quest "Ressureicao do Cavaleiro Negro" (_MSG_Quest.cpp:382). A Patrulha do
	// Dungeon Negro pede a Cura do Batedor.
	if npc.Merchant == 100 && npc.Grade == 2 {
		d.quest256NPC(w, s, e, quest256Steps[2], itemCuraDoBatedor)
		return
	}
	// QUEST_HIDRA (Merchant 100, EF_GRADE0 3): quarta arena da Quest 256, a
	// quest "Hidra Imortal" (_MSG_Quest.cpp:426). Mesmo nome de NPC Patrulha do
	// passo Kaizen, um tier acima: pede a Mana do Batedor.
	if npc.Merchant == 100 && npc.Grade == 3 {
		d.quest256NPC(w, s, e, quest256Steps[3], itemManaDoBatedor)
		return
	}
	// QUEST_ELFOS (Merchant 100, EF_GRADE0 4): quinta e última arena da Quest 256, a
	// quest "Inicio da Infelicidade" (_MSG_Quest.cpp:470). A Guarda do
	// Submundo pede o Emblema do Guarda.
	if npc.Merchant == 100 && npc.Grade == 4 {
		d.quest256NPC(w, s, e, quest256Steps[4], itemEmblemaDoGuarda)
		return
	}
	// LIDER_APRENDIZ (Merchant 100, EF_GRADE0 5): a capa verde do Arch
	// (issue #397, _MSG_Quest.cpp:2625).
	if isLiderAprendizNPC(npc) {
		d.liderAprendiz(w, s, e, npc)
		return
	}
	// QUEST_CAPAREAL (Merchant 100, EF_GRADE0 13): entrada da quest da Capa Real.
	if npc.Merchant == 100 && npc.Grade == 13 {
		d.royalCapeQuest(w, s, e)
		return
	}
	// PERZEN (Merchant 100, EF_GRADE0 ∈ {7,8,9}): a troca de itens.
	if npc.Merchant == 100 && npc.Grade >= 7 && npc.Grade <= 9 {
		d.perzenExchange(w, s, npc)
		return
	}
	if npc.Merchant == 23 {
		d.mestreGrifo(w, s, e, npc)
		return
	}
	// CAPAVERDE_TELEPORT (Merchant 100, EF_GRADE0 14): teleporte da arena de aprendizes
	// (issue #139, _MSG_Quest.cpp:2168).
	if npc.Merchant == 100 && npc.Grade == 14 {
		d.capaverdeTeleport(w, s, e)
		return
	}
	// MOLARGARGULA (Merchant 100, EF_GRADE0 15): _MSG_Quest.cpp:2236.
	if npc.Merchant == 100 && npc.Grade == 15 {
		d.molarGargula(w, s, e)
		return
	}
	// CAPAVERDE_TRADE (Merchant 8, "Chefe_Treina."): entrega da capa verde
	// (issue #139, _MSG_Quest.cpp:2189).
	if npc.Merchant == 8 {
		d.capaverdeTrade(w, s, e)
		return
	}
	// AMU_MISTICO (Merchant 10, "Cap.Mercenario"): quest de grupo da Terra Mistica
	// (_MSG_Quest.cpp:254).
	if npc.Merchant == 10 {
		d.amuMistico(w, s, e, int(confirm))
		return
	}
	// EXPLOIT_LEADER (MOB.Merchant 11, "Cap.Explor."): passo 3 da Terra Mistica
	// (issue #403, _MSG_Quest.cpp:1626).
	if isExploitLeaderNPC(npc) {
		d.exploitLeader(w, s, e, npc)
		return
	}
	// JEFFI / SHAMA (MOB.Merchant 12 / 13): o Círculo Divino (issue #405,
	// _MSG_Quest.cpp:514-691).
	if isJeffiNPC(npc) {
		d.jeffi(w, s, e, npc)
		return
	}
	if isShamaNPC(npc) {
		d.shama(w, s, e, npc)
		return
	}
	// BLACKORACLE (Merchant 78): funde os dois itens de alma na Pedra da Imortalidade
	// (1742, _MSG_Quest.cpp:2257-2332).
	if npc.Merchant == 78 {
		d.blackOracle(w, s, e)
		return
	}
	// COMP_SEPHI (Merchant 19): fabrica o Sephirot da classe a partir de 8 Pedras + moeda
	// (_MSG_Quest.cpp:2103).
	if npc.Merchant == 19 {
		d.compSephi(w, s, e, npc, int(confirm))
		return
	}
	// KIBITA (Merchant 74): cidadania primeiro, depois a Mortal Soul permanente
	// com a pedra Secreta específica da classe (_MSG_Quest.cpp:2430-2562).
	if npc.Merchant == 74 {
		d.kibita(w, s, e)
		return
	}
	if isKingQuestNPC(npc) {
		d.kingQuest(w, s, e, npc, int(confirm))
		return
	}
	// URNAMMU (MOB.Merchant 76): troca de classe Sub-Celestial (issue #410,
	// _MSG_Quest.cpp:1248).
	if isUrnammuNPC(npc) {
		d.urnammu(w, s, e)
		return
	}
	// UXMAL (MOB.Merchant 72): registro da Pista de Runas (_MSG_Quest.cpp:1313).
	if isUxmalNPC(npc) {
		d.uxmal(w, s, e)
		return
	}
	d.log.Debug("quest NPC not implemented", "conn", s.Conn, "npc", npcIndex, "merchant", npc.Merchant, "grade", npc.Grade)
}

// kingQuest porta o caso KING (_MSG_Quest.cpp:694-1124) na ordem legada:
// o gate de Clã (:740-741), o orçamento de preço no confirm 0 (:762-767), depois, com
// Equip[10]=1742 e um Sephirot em Equip[11] (:769), o craft da Pedra Ideal
// (:771-840) antes da criação do Arch do Mortal (:842-866), e por fim o
// serviço de capa (:869-1124). O orçamento é um global no legado; o Go o busca
// de forma assíncrona só para as pernas que o usam (confirm 0 e a capa),
// o que não muda nada observável para as pernas da pedra e do Arch.
func (d *Dispatcher) kingQuest(w *world.World, s *world.Session, e, npc *world.Entity, confirm int) {
	if clan := kingClanFromCape(e.Clan, e.Equip[capeEquipSlot].Index); clan != 0 && clan != npc.Clan {
		return
	}
	if confirm != 0 && e.Equip[immortalityStoneEquipSlot].Index == immortalityStoneItem &&
		e.Equip[sephirotEquipSlot].Index >= archSephirotMin && e.Equip[sephirotEquipSlot].Index <= archSephirotMax {
		if d.kingPedraIdeal(w, s, e, npc) {
			return
		}
		if e.ClassMaster == classMasterMortal && e.Level >= 299 {
			d.kingArch(w, s, e, npc, confirm)
			return
		}
	}
	d.kingCapeService(w, s, e, npc, confirm)
}

// Pedra_Ideal (5338, EF_VOLATILE 211) e as quatro Pedras Secretas que o Rei
// funde nela (_MSG_Quest.cpp:771-840). O 1742 atrás de immortalityStoneItem
// é a Pedra_da_Imortalidade, o equipamento Equip[10] do Rei, não esta pedra;
// clicar com o botão direito no 5338 é o que transforma um Arch em Celestial (useIdealStone).
const pedraIdealItem = 5338

var pedraSecretaItems = [...]int16{5334, 5335, 5336, 5337}

// kingPedraIdeal porta o craft da Pedra Ideal (_MSG_Quest.cpp:771-840), aberto a
// todo ClassMaster: com as quatro Pedras Secretas no inventário, a primeira
// 5334 vira um 5338 novo no lugar, as primeiras 5335/5336/5337 são apagadas e
// o equipamento do Rei (Equip[10]/[11]) é consumido. Informa se rodou, para
// o chamador cair na criação do Arch caso contrário, como o legado faz.
func (d *Dispatcher) kingPedraIdeal(w *world.World, s *world.Session, e, npc *world.Entity) bool {
	slots, ok := pedraSecretaSlots(e)
	if !ok {
		return false
	}
	// O legado percorre o inventário uma vez e envia cada slot conforme o reescreve,
	// então os refreshes saem em ordem de slot.
	for i := 0; i < activeCarryLimit(e); i++ {
		for k, slot := range slots {
			if slot != i {
				continue
			}
			e.Carry[i] = world.Item{}
			if k == 0 {
				e.Carry[i].Index = pedraIdealItem
			}
			d.sendSlot(w, s, world.ItemPlaceCarry, i, e.Carry[i])
		}
	}
	e.Equip[immortalityStoneEquipSlot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceEquip, immortalityStoneEquipSlot, e.Equip[immortalityStoneEquipSlot])
	e.Equip[sephirotEquipSlot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceEquip, sephirotEquipSlot, e.Equip[sephirotEquipSlot])
	// SendSay(_NN_My_King_Bless1) independentemente do reino do Rei (:832).
	d.sendNPCChatText(w, s, npc.ID, msgKingBless1)
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("king make ideal_stone", "conn", s.Conn, "name", e.Name, "level", e.Level)
	return true
}

// pedraSecretaSlots acha o primeiro slot de carry acessível que guarda cada uma das
// quatro Pedras Secretas, na ordem de pedraSecretaItems. ok é false quando alguma
// está faltando. Tanto o Rei (_MSG_Quest.cpp:776-800) quanto a Pedra da Furia
// (_MSG_UseItem.cpp:3480-3536) apagam a primeira de cada.
func pedraSecretaSlots(e *world.Entity) (slots [len(pedraSecretaItems)]int, ok bool) {
	for k := range slots {
		slots[k] = -1
	}
	limit := activeCarryLimit(e)
	for i := 0; i < limit; i++ {
		for k, stone := range pedraSecretaItems {
			if slots[k] < 0 && e.Carry[i].Index == stone {
				slots[k] = i
			}
		}
	}
	for _, slot := range slots {
		if slot < 0 {
			return slots, false
		}
	}
	return slots, true
}

// kingCapeService porta a metade da capa do caso KING (_MSG_Quest.cpp:694-1124)
// na ordem legada: gate de reino, orçamento de preço (confirm 0), bênção da capa
// concluída, exigências Celestial/Mortal, contagem de safiras (ou o
// Emblema_do_Reino equipado), pagamento, a nova capa e a bênção do Rei.
func (d *Dispatcher) kingCapeService(w *world.World, s *world.Session, e, npc *world.Entity, confirm int) {
	npcID, kingdom := npc.ID, npc.Clan
	if capeClan, _ := capeKingdomMode(e.Equip[capeEquipSlot].Index); capeClan != 0 && capeClan != kingdom {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	if kingdom != clanHekalotia && kingdom != clanAkelonia {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	p := w.Persistence()
	w.Go(s, func() func(*world.World, *world.Session) {
		quote, err := p.QuoteKingdomCape(context.Background())
		return func(w *world.World, s *world.Session) {
			if err != nil {
				d.notify(w, s, NoticeReqNotMet)
				return
			}
			cost := quote.HekalotiaCost
			if kingdom == clanAkelonia {
				cost = quote.AkeloniaCost
			}
			if confirm == 0 {
				d.sendNPCChatText(w, s, npcID, fmt.Sprintf(msgNeedSapphire, cost))
				return
			}
			d.kingCapePurchase(w, s, e, npcID, kingdom, cost, quote.Revision)
		}
	})
}

// kingCapePurchase é a perna confirmada do serviço de capa
// (_MSG_Quest.cpp:869-1118).
func (d *Dispatcher) kingCapePurchase(w *world.World, s *world.Session, e *world.Entity, npcID int, kingdom uint8, cost int, revision int64) {
	current := e.Equip[capeEquipSlot]
	if _, mode := capeKingdomMode(current.Index); mode >= 2 {
		d.sendNPCChatText(w, s, npcID, kingBlessing(kingdom))
		return
	}
	target, refusal := kingdomCapeTarget(e.ClassMaster, e.Level, current, kingdom)
	switch refusal {
	case capeRefusedSilently:
		return
	case capeRefusedLevel:
		d.sendNPCChatText(w, s, npcID, msgNeedLevel)
		return
	}
	// Um Emblema_do_Reino equipado substitui as safiras: o legado pula
	// tanto a checagem de safiras quanto o laço de consumo enquanto Equip[13] é 4081,
	// depois o apaga (_MSG_Quest.cpp:997,1016,1038-1042). O ramo
	// Celestial não tem essa isenção (_MSG_Quest.cpp:893).
	emblem := !isCelestialMaster(e.ClassMaster) && e.Equip[kingdomEmblemEquipSlot].Index == kingdomEmblemItem
	if !emblem && sapphireValue(e) < cost {
		d.sendNPCChatText(w, s, npcID, fmt.Sprintf(msgNeedSapphire, cost))
		return
	}
	staged := *e
	var changed []int
	if emblem {
		staged.Equip[kingdomEmblemEquipSlot] = world.Item{}
	} else {
		changed = paySapphires(&staged, cost)
	}
	staged.Equip[capeEquipSlot] = target
	// O legado nunca escreve MOB.Clan aqui; GetCurrentScore o deriva do
	// novo manto na próxima passagem de score (Basedef.cpp:3222-3247), o que o
	// refresh de score do Go não modela, então ele é aplicado junto da compra.
	staged.Clan = kingClanFromCape(staged.Clan, target.Index)
	save := w.CharacterSaveFor(s, &staged)
	p := w.Persistence()
	w.Go(s, func() func(*world.World, *world.Session) {
		_, committed, purchaseErr := p.PurchaseKingdomCape(context.Background(), revision, kingdom, save)
		return func(w *world.World, s *world.Session) {
			if purchaseErr != nil || !committed {
				d.notify(w, s, NoticeReqNotMet)
				return
			}
			*e = staged
			for _, slot := range changed {
				d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])
			}
			if emblem {
				d.sendSlot(w, s, world.ItemPlaceEquip, kingdomEmblemEquipSlot, e.Equip[kingdomEmblemEquipSlot])
			}
			d.sendSlot(w, s, world.ItemPlaceEquip, capeEquipSlot, e.Equip[capeEquipSlot])
			d.sendNPCChatText(w, s, npcID, kingBlessing(kingdom))
			d.refreshScore(e)
			d.sendScore(w, s, e)
		}
	})
}

// Textos legados (Language.txt, Windows-1252); o orçamento mantém o texto do Go.
const (
	msgNeedSapphire = "Sao necessarias %d safiras."                            // _DN_Need_D_Sapphire (177)
	msgKingBless1   = "Que a luz de Sephira o ilumine."                        // _NN_My_King_Bless1 (178)
	msgKingBless2   = "Que a for\xe7a divina o aben\xe7oe."                    // _NN_My_King_Bless2 (179)
	msgNeedLevel    = "N\xe3o possui n\xedvel suficiente, retorne mais tarde." // _NN_Need_Level (180)
)

// kingBlessing é a fala do Rei depois de uma venda ou para uma capa concluída:
// Bless1 para ReiClan 7, Bless2 caso contrário (_MSG_Quest.cpp:870-874,1086-1089).
func kingBlessing(kingdom uint8) string {
	if kingdom == clanHekalotia {
		return msgKingBless1
	}
	return msgKingBless2
}

func isCelestialMaster(classMaster uint8) bool {
	return classMaster == classMasterCelestial || classMaster == classMasterCelestialCS || classMaster == classMasterSCelestial
}

// capeKingdomMode devolve o reino a que uma capa prende seu portador e o
// CapeMode legado: 2 = capa de reino concluída, 1 = melhorável, 0 = nenhuma
// (_MSG_Quest.cpp:700-738). 734-737 definem o Clã mas mantêm CapeMode 0.
func capeKingdomMode(index int16) (uint8, int) {
	switch index {
	case 543, 3191, 3194, 3197:
		return clanHekalotia, 2
	case 544, 3192, 3195, 3198:
		return clanAkelonia, 2
	case 545:
		return clanHekalotia, 1
	case 546:
		return clanAkelonia, 1
	case 734, 736:
		return clanHekalotia, 0
	case 735, 737:
		return clanAkelonia, 0
	case 549, 3193, 3196:
		return 0, 1
	default:
		return 0, 0
	}
}

type capeRefusal int

const (
	capeGranted         capeRefusal = iota
	capeRefusedSilently             // Celestial sem o 3199 (_MSG_Quest.cpp:879)
	capeRefusedLevel                // _NN_Need_Level (_MSG_Quest.cpp:977,1118)
)

// kingdomCapeTarget calcula a capa que o Rei entrega, mantendo os efeitos
// do item antigo onde o legado só reescreve sIndex (548/549/3193/3196 e
// o Celestial 3199) e limpando no resto (_MSG_Quest.cpp:933-940,1044-1080).
func kingdomCapeTarget(classMaster uint8, level int32, current world.Item, kingdom uint8) (world.Item, capeRefusal) {
	pick := func(hekalotia, akelonia int16) int16 {
		if kingdom == clanHekalotia {
			return hekalotia
		}
		return akelonia
	}
	if isCelestialMaster(classMaster) {
		if current.Index != 3199 {
			return world.Item{}, capeRefusedSilently
		}
		current.Index = pick(3197, 3198)
		return current, capeGranted
	}
	if level < 219 {
		return world.Item{}, capeRefusedLevel
	}
	_, mode := capeKingdomMode(current.Index)
	if mode == 1 && level < 255 {
		return world.Item{}, capeRefusedLevel
	}
	if mode == 0 {
		if current.Index != 548 {
			current = world.Item{}
		}
		current.Index = pick(545, 546)
		return current, capeGranted
	}
	switch current.Index {
	case 3193:
		current.Index = pick(3191, 3192)
	case 3196:
		current.Index = pick(3194, 3195)
	default:
		if current.Index != 549 {
			current = world.Item{}
		}
		current.Index = pick(543, 544)
	}
	return current, capeGranted
}

// sapphireValue conta as safiras da bolsa: 697 vale 1, 4131 vale 10
// (_MSG_Quest.cpp:986-995).
func sapphireValue(e *world.Entity) int {
	total := 0
	for i := 0; i < activeCarryLimit(e); i++ {
		switch e.Carry[i].Index {
		case sapphireUnit1:
			total++
		case sapphireUnit10:
			total += 10
		}
	}
	return total
}

// paySapphires consome safiras na ordem da bolsa até o preço ser coberto,
// devolvendo os slots limpos. Um 4131 é apagado inteiro mesmo quando menos de 10
// ainda era devido; não há troco (_MSG_Quest.cpp:1016-1035).
func paySapphires(e *world.Entity, cost int) []int {
	var changed []int
	for i := 0; i < activeCarryLimit(e) && cost > 0; i++ {
		switch e.Carry[i].Index {
		case sapphireUnit1:
			cost--
		case sapphireUnit10:
			cost = max(cost-10, 0)
		default:
			continue
		}
		e.Carry[i] = world.Item{}
		changed = append(changed, i)
	}
	return changed
}

const (
	kingQuestMerchant = 111

	immortalityStoneItem      = 1742 // Pedra_da_Imortalidade (sem EF_VOLATILE)
	immortalityStoneEquipSlot = 10
	sephirotEquipSlot         = 11
	archSephirotMin           = 1760
	archSephirotMax           = 1763
	capeEquipSlot             = 15
	pedraIdealsCraftCoin      = 30_000_000
)

// Emblema_do_Reino, dropado pelo Cav._Mortal da quest "Cavaleiro do Reino"
// equipado, ele paga a capa de reino (_MSG_Quest.cpp:997).
const (
	kingdomEmblemItem      = 4081
	kingdomEmblemEquipSlot = 13
)

const (
	kibitaCitizenshipCost = 4_000_000 // _MSG_Quest.cpp:2433, corrigido no legado
	kibitaSoulMinLevel    = 369
	kibitaSoulSkillBit    = int32(1 << 30)
	kibitaCapeHekalotia   = int16(3194)
	kibitaCapeAkelonia    = int16(3195)
	kibitaCapeNeutral     = int16(3196)
)

var kibitaSoulStones = [...]int16{5334, 5336, 5335, 5337}

// kibita é o ponto de entrada KIBITA na ordem legada: a venda da cidadania
// primeiro, a Soul permanente depois (:2430-2562). O buff de evento
// KIBITA_SOUL em tempo de compilação ainda não está portado.
func (d *Dispatcher) kibita(w *world.World, s *world.Session, e *world.Entity) {
	// Um personagem sem cidade e com a taxa em ouro compra a cidadania DESTE
	// canal, e o legado retorna ali mesmo mesmo que também se qualifique para o
	// ramo da Soul (:2433-2459). Citizen é base 1: canal N é N+1.
	if e.Citizen == 0 && e.Coin >= kibitaCitizenshipCost && d.serverIndex >= 0 && d.serverIndex < 255 {
		e.Coin -= kibitaCitizenshipCost
		e.Citizen = uint8(d.serverIndex + 1)
		d.sendEtc(w, s, e) // a taxa muda o ouro que o cliente mostra
		d.sendClientMessage(w, s, joinedTownText)
		d.log.Info("citizenship granted", "conn", s.Conn, "name", e.Name,
			"citizen", e.Citizen, "server_index", d.serverIndex, "coin", e.Coin)
		w.SaveCharacterAsync(s)
		return
	}
	d.kibitaSoul(w, s, e)
}

// kibitaSoul porta o ramo da Soul permanente do KIBITA.
func (d *Dispatcher) kibitaSoul(w *world.World, s *world.Session, e *world.Entity) {
	if e.ClassMaster != classMasterMortal || e.Level < kibitaSoulMinLevel || e.LearnedSkill&kibitaSoulSkillBit != 0 {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	if int(e.Class) >= len(kibitaSoulStones) {
		d.notify(w, s, NoticeReqNotMet)
		return
	}

	stone := kibitaSoulStones[e.Class]
	slot := -1
	for i := 0; i < activeCarryLimit(e); i++ {
		if e.Carry[i].Index == stone {
			slot = i
			break
		}
	}
	if slot < 0 {
		d.notify(w, s, NoticeReqNotMet)
		return
	}

	e.Carry[slot] = world.Item{}
	e.LearnedSkill |= kibitaSoulSkillBit
	cape := kibitaCapeNeutral
	switch e.Clan {
	case clanHekalotia:
		cape = kibitaCapeHekalotia
	case clanAkelonia:
		cape = kibitaCapeAkelonia
	}
	e.Equip[capeEquipSlot] = world.Item{Index: cape}
	d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])
	d.sendSlot(w, s, world.ItemPlaceEquip, capeEquipSlot, e.Equip[capeEquipSlot])
	d.refreshScore(e)
	d.sendScore(w, s, e)
	d.sendEtc(w, s, e)

	d.log.Info("mortal soul learned", "conn", s.Conn, "name", e.Name, "class", e.Class, "stone", stone, "cape", cape)
	d.returnToCharacterSelection(w, s, func(w *world.World, s *world.Session) {
		w.SendTo(s, protocol.Header{Type: protocol.MsgSendArchEffect, ID: protocol.IDScene},
			protocol.EncodeStandardParm(int32(s.Slot)))
	})
}

func isKingQuestNPC(npc *world.Entity) bool {
	if npc == nil {
		return false
	}
	// O loader do Go guarda templates reais de Rei por CurrentScore.Merchant (111),
	// enquanto o switch legado roteia KING pelo MOB.Merchant de topo 14/15.
	// Aceitar as duas formas para conteúdo e loaders futuros poderem usar qualquer fonte.
	return npc.Merchant == kingQuestMerchant || npc.Merchant == 14 || npc.Merchant == 15
}

// Os dois valores de Clã de reino (REINO_BLUE / REINO_RED, Source/Code/Basedef.h:235-236).
const (
	clanHekalotia = 7 // REINO_BLUE
	clanAkelonia  = 8 // REINO_RED
)

// kingClanFromCape deriva o Clã efetivo do jogador a partir da capa: certas
// capas sobrepõem o Clã da guilda do portador para 7 ou 8 independentemente do
// clã real (_MSG_Quest.cpp:704-726, Basedef.cpp:3212-3247). CapeMode (os ramos
// de recompensa de capa mais abaixo no caso legado) está fora de escopo aqui — só a
// sobreposição de Clã importa, para o gate do Arch do Rei e o roteamento de /reino.
func kingClanFromCape(clan uint8, capeIndex int16) uint8 {
	switch capeIndex {
	case 543, 545, 734, 736, 3191, 3194, 3197:
		return clanHekalotia
	case 544, 546, 735, 737, 3192, 3195, 3198:
		return clanAkelonia
	default:
		return clan
	}
}

// kingArch trata o caminho de criação do Arch do npcMode KING (_MSG_Quest.cpp:694-867).
// Portado: o gate de facção Clã/capa (:696-741), a perna de pedido confirm==0
// (:762-767, sem o texto de preço em safiras — formato de pacote UNVERIFIED), e o
// gate Equip[10]==1742 (Pedra da Imortalidade) + Equip[11]∈[1760,1763] (Sephirot) + Mortal/nível
// (:769, :842-866). kingQuest roda o craft da Pedra Ideal (:771-840,
// kingPedraIdeal) primeiro, então um Mortal carregando as quatro Pedras Secretas ganha o
// 5338 em vez do Arch, e envia os ramos de capa (:869-1124) para
// kingCapeService.
func (d *Dispatcher) kingArch(w *world.World, s *world.Session, e, npc *world.Entity, confirm int) {
	clan := kingClanFromCape(e.Clan, e.Equip[capeEquipSlot].Index)
	if clan != 0 && clan != npc.Clan {
		return
	}
	if confirm == 0 {
		return
	}
	if e.ClassMaster != classMasterMortal || e.Level < 299 {
		return
	}
	if e.Equip[immortalityStoneEquipSlot].Index != immortalityStoneItem {
		return
	}
	sephirot := int(e.Equip[sephirotEquipSlot].Index)
	if sephirot < archSephirotMin || sephirot > archSephirotMax {
		return
	}
	class := sephirot - archSephirotMin
	mortalFace := int(e.Equip[0].Index)
	if mortalFace <= 0 {
		if body := d.classBody(int(e.Class)); !body.Empty() {
			mortalFace = int(body.Index)
		}
	}
	if mortalFace <= 0 {
		d.log.Warn("king arch: missing mortal face", "conn", s.Conn, "name", e.Name, "class", e.Class)
		return
	}

	accID := s.AccountID
	name := e.Name
	mortalSlot := s.Slot
	s.Mode = world.UserWaitDB
	p := w.Persistence()
	w.Go(s, func() func(*world.World, *world.Session) {
		slot, ok, err := p.CreateArchCharacter(context.Background(), accID, name, class, mortalFace, mortalSlot, int(e.Level))
		var chars []world.CharSummary
		var listErr error
		if err == nil && ok {
			chars, listErr = p.ListCharacters(context.Background(), accID)
		}
		return func(w *world.World, s *world.Session) {
			d.completeKingArch(w, s, slot, ok, err, chars, listErr)
		}
	})
}

func (d *Dispatcher) completeKingArch(w *world.World, s *world.Session, archSlot int, ok bool, err error, chars []world.CharSummary, listErr error) {
	if err != nil {
		s.Mode = world.UserPlay
		d.log.Warn("king arch: db create failed", "conn", s.Conn, "account", s.AccountID, "err", err)
		d.notify(w, s, NoticeDBError)
		return
	}
	if !ok {
		s.Mode = world.UserPlay
		d.notify(w, s, NoticeNoEmptySlot)
		return
	}
	if listErr != nil {
		d.log.Warn("king arch: list after create failed", "conn", s.Conn, "account", s.AccountID, "err", listErr)
	}

	s.Mode = world.UserPlay
	e := w.Entity(s.Conn)
	if e == nil {
		return
	}
	e.Equip[immortalityStoneEquipSlot] = world.Item{}
	e.Equip[sephirotEquipSlot] = world.Item{}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceEquip, immortalityStoneEquipSlot, itemToSel(e.Equip[immortalityStoneEquipSlot])))
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceEquip, sephirotEquipSlot, itemToSel(e.Equip[sephirotEquipSlot])))

	body := protocol.EncodeRemoveMobBody(2)
	w.ForEachInView(s.Conn, func(vs *world.Session, _ *world.Entity) {
		w.SendTo(vs, protocol.Header{Type: protocol.MsgRemoveMob, ID: uint16(s.Conn)}, body)
	})

	w.SaveCharacterThen(s, func(w *world.World, s *world.Session) {
		w.SaveCargoThen(s, func(w *world.World, s *world.Session) {
			if e := w.Entity(s.Conn); e != nil {
				e.Mode = world.MobUserDock
				e.ResetAffects()
			}
			s.Mode = world.UserSelChar
			w.Send(s, protocol.MsgCNFCharacterLogout, nil)
			if listErr == nil {
				w.SendTo(s, protocol.Header{Type: protocol.MsgCNFNewCharacter, ID: protocol.IDNewCharacter},
					protocol.EncodeCNFNewCharacterBody(d.selCharsFrom(chars)))
			}
			w.SendTo(s, protocol.Header{Type: protocol.MsgSendArchEffect, ID: protocol.IDScene},
				protocol.EncodeStandardParm(int32(archSlot)))
		})
	})
}

const (
	capaverdeMinLevel = 99
	capaverdeMaxLevel = 150

	capaverdeTicketEquipSlot = 13
	capaverdeTicketItem      = 4080
	greenCapeItem            = 4006
)

// capaverdeTeleport trata CAPAVERDE_TELEPORT (Merchant 100, EF_GRADE0 14):
// o passo de teleporte da arena de aprendizes da cadeia da capa verde
// (_MSG_Quest.cpp:2168-2186). Nenhum item é consumido e o caso legado nunca
// checa confirm.
func (d *Dispatcher) capaverdeTeleport(w *world.World, s *world.Session, e *world.Entity) {
	if e.ClassMaster != classMasterMortal || e.Level < capaverdeMinLevel || e.Level >= capaverdeMaxLevel {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	d.doTeleport(w, s, 2245+int16(w.Rand().Intn(5)-3), 1576+int16(w.Rand().Intn(5)-3))
}

const (
	royalCapeMinLevel = 199
	royalCapeMaxLevel = 254
)

// royalCapeQuest trata o gate de entrada da Guarda Real de _MSG_Quest.cpp.
func (d *Dispatcher) royalCapeQuest(w *world.World, s *world.Session, e *world.Entity) {
	if e.ClassMaster != classMasterMortal || e.Level < royalCapeMinLevel || e.Level >= royalCapeMaxLevel {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	d.doTeleport(w, s, 1740+int16(w.Rand().Intn(5)-3), 1725+int16(w.Rand().Intn(5)-3))
}

// capaverdeTrade trata CAPAVERDE_TRADE (Merchant 8, "Chefe_Treina."): a
// entrega da capa verde (_MSG_Quest.cpp:2189-2234). Exige o item 4080 equipado
// no slot 13 (o token de formatura do aprendiz) e nenhuma capa já equipada;
// consome o token e concede a capa verde (item 4006) no slot
// de capa. O caso legado nunca checa confirm.
func (d *Dispatcher) capaverdeTrade(w *world.World, s *world.Session, e *world.Entity) {
	if e.ClassMaster != classMasterMortal || e.Level < capaverdeMinLevel || e.Level >= capaverdeMaxLevel {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	if e.Equip[capaverdeTicketEquipSlot].Index != capaverdeTicketItem {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	if e.Equip[capeEquipSlot].Index != 0 {
		d.notify(w, s, NoticeAlreadyDone)
		return
	}
	e.Equip[capaverdeTicketEquipSlot] = world.Item{}
	e.Equip[capeEquipSlot] = world.Item{Index: greenCapeItem}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceEquip, capaverdeTicketEquipSlot, itemToSel(e.Equip[capaverdeTicketEquipSlot])))
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceEquip, capeEquipSlot, itemToSel(e.Equip[capeEquipSlot])))
	d.refreshScore(e)
	d.sendScore(w, s, e)
	d.log.Info("capaverde trade complete", "conn", s.Conn, "name", e.Name)
}

const (
	molarGargulaMinLevel = 199
	molarGargulaMaxLevel = 254
)

// molarGargula trata MOLARGARGULA (Merchant 100, EF_GRADE0 15):
// _MSG_Quest.cpp:2236-2255. Sem item, sem flag persistida, sem gate de confirm — um
// teleporte puro condicionado a nível/classe.
func (d *Dispatcher) molarGargula(w *world.World, s *world.Session, e *world.Entity) {
	if e.ClassMaster != classMasterMortal || e.Level < molarGargulaMinLevel || e.Level >= molarGargulaMaxLevel {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	d.doTeleport(w, s, 817+int16(w.Rand().Intn(5)-3), 4062+int16(w.Rand().Intn(5)-3))
}

// amuMistico trata AMU_MISTICO (Merchant 10, "Cap.Mercenario"): a quest
// de grupo da Terra Mistica (_MSG_Quest.cpp:254-292). Exige ClassMaster Mortal,
// a quest ainda não feita, e o chamador ser líder de um grupo
// não solo. confirm==0 é um pedido puro no caso legado (só SendSay, sem mudança
// de estado) — espelhado aqui como no-op, igual à perna de pedido de kingArch.
func (d *Dispatcher) amuMistico(w *world.World, s *world.Session, e *world.Entity, confirm int) {
	if e.TerraMistica != 0 || e.ClassMaster != classMasterMortal {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	if confirm == 0 {
		return
	}
	// Líderes de grupo legados mantêm Leader==0; a prova de não-solo é PartyList.
	if e.Leader != 0 || partyMemberCount(e) == 0 {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	e.TerraMistica = 1
	w.SaveCharacterThen(s, func(*world.World, *world.Session) {})
	d.log.Info("terra mistica complete", "conn", s.Conn, "name", e.Name)
}

const (
	soulItem1               = 1740
	soulItem2               = 1741
	sapphireUnit1           = 697  // 1 unidade de safira
	sapphireUnit10          = 4131 // 10 unidades de safira
	blackOracleCost         = 10   // unidades de safira necessárias
	pedraImortalidadeReward = immortalityStoneItem
)

// blackOracle trata BLACKORACLE (Merchant 78): funde os dois itens de alma
// (1740 imediatamente seguido de 1741 no Carry) mais 10 unidades de safira na
// Pedra da Imortalidade (item 1742) que o Rei depois exige em Equip[10]
// (_MSG_Quest.cpp:2257-2334; a recompensa é definida em :2332).
// Um 697 conta como 1 unidade, um 4131 como 10. O caso legado nunca checa confirm.
func (d *Dispatcher) blackOracle(w *world.World, s *world.Session, e *world.Entity) {
	soulSlot := -1
	limit := activeCarryLimit(e)
	for i := 0; i < limit-1; i++ {
		if e.Carry[i].Index == soulItem1 && e.Carry[i+1].Index == soulItem2 {
			soulSlot = i
			break
		}
	}
	if soulSlot < 0 {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	units := 0
	for i := 0; i < limit; i++ {
		it := e.Carry[i]
		switch it.Index {
		case sapphireUnit1:
			units++
		case sapphireUnit10:
			units += 10
		}
	}
	if units < blackOracleCost {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	remaining := blackOracleCost
	for i := 0; i < limit; i++ {
		if remaining <= 0 {
			break
		}
		switch e.Carry[i].Index {
		case sapphireUnit1:
			e.Carry[i] = world.Item{}
			w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, i, itemToSel(e.Carry[i])))
			remaining--
		case sapphireUnit10:
			e.Carry[i] = world.Item{}
			w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, i, itemToSel(e.Carry[i])))
			if remaining -= 10; remaining < 0 {
				remaining = 0
			}
		}
	}
	e.Carry[soulSlot] = world.Item{}
	e.Carry[soulSlot+1] = world.Item{}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, soulSlot, itemToSel(e.Carry[soulSlot])))
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, soulSlot+1, itemToSel(e.Carry[soulSlot+1])))

	slot := firstEmptyCarry(&e.Carry)
	if slot < 0 {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	e.Carry[slot] = world.Item{Index: pedraImortalidadeReward}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, slot, itemToSel(e.Carry[slot])))
	d.log.Info("black oracle soul merge", "conn", s.Conn, "name", e.Name)
}

const compSephiCoinCost = pedraIdealsCraftCoin

// compSephi trata COMP_SEPHI (Merchant 19): fabrica o Sephirot da classe
// (item 1759+npc.Class, ou seja 1760-1763) a partir de 8 "Pedras" (itens 1744-1751, uma
// de cada) mais 30.000.000 de moeda (_MSG_Quest.cpp:2103-2166). confirm==0 é um
// pedido puro no caso legado (só SendSay) — espelhado como no-op.
func (d *Dispatcher) compSephi(w *world.World, s *world.Session, e, npc *world.Entity, confirm int) {
	if confirm == 0 {
		return
	}
	const pedraCount = 8
	limit := activeCarryLimit(e)
	slots := make([]int, 0, pedraCount)
	for j := 0; j < pedraCount; j++ {
		found := -1
		for i := 0; i < limit; i++ {
			it := e.Carry[i]
			if it.Index == int16(1744+j) {
				found = i
				break
			}
		}
		if found < 0 {
			d.notify(w, s, NoticeReqNotMet)
			return
		}
		slots = append(slots, found)
	}
	if e.Coin < compSephiCoinCost {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	slot := firstEmptyCarry(&e.Carry)
	if slot < 0 {
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	e.Coin -= compSephiCoinCost
	d.sendEtc(w, s, e)
	for _, i := range slots {
		e.Carry[i] = world.Item{}
		w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, i, itemToSel(e.Carry[i])))
	}
	// Os templates de Mestre de Skills carregam Class 1..4 (Cap.Cavaleiros=1 ..
	// ForeLearner=4, STRUCT_MOB.Class @20), então o 1759+Class legado produz o
	// Sephirot 1760..1763 da classe base-0 do jogador (_MSG_Quest.cpp:2157). kingArch
	// mapeia de volta com Equip[11]-1760 (:845), então os dois precisam andar juntos.
	e.Carry[slot] = world.Item{Index: archSephirotMin - 1 + int16(npc.Class)}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, slot, itemToSel(e.Carry[slot])))
	d.log.Info("sephirot created", "conn", s.Conn, "name", e.Name, "class", npc.Class)
}

type quest256Step struct {
	flag     uint8
	minLevel int32
	maxLevel int32
	x, y     int16
	area     questArea
}

var quest256Steps = []quest256Step{
	{flag: 1, minLevel: 39, maxLevel: 115, x: 2398, y: 2105, area: questArea{x1: 2379, y1: 2076, x2: 2426, y2: 2133}},
	{flag: 2, minLevel: 115, maxLevel: 190, x: 2234, y: 1714, area: questArea{x1: 2228, y1: 1700, x2: 2257, y2: 1728}},
	{flag: 3, minLevel: 190, maxLevel: 265, x: 464, y: 3902, area: questArea{x1: 459, y1: 3887, x2: 497, y2: 3916}},
	{flag: 4, minLevel: 265, maxLevel: 320, x: 668, y: 3756, area: questArea{x1: 658, y1: 3728, x2: 703, y2: 3762}},
	{flag: 5, minLevel: 320, maxLevel: 350, x: 1322, y: 4041, area: questArea{x1: 1312, y1: 4027, x2: 1348, y2: 4055}},
}

// Os tickets da Quest 256, um por passo da cadeia: o NPC daquele passo só abre sua
// arena para o jogador que carrega o item correspondente (_MSG_Quest.cpp:315/360/404/448/492).
// Eles são consecutivos no ItemList.csv (:5703-5711), o que permite
// quest256StepForTicket mapear um ticket clicado de volta ao seu passo.
const (
	itemVelaDoCoveiro        int16 = 4038 // "Vela_do_Coveiro"
	itemColheitaDoJardineiro int16 = 4039 // "Colheita_do_Jardineiro"
	itemCuraDoBatedor        int16 = 4040 // "Cura_do_Batedor"
	itemManaDoBatedor        int16 = 4041 // "Mana_do_Batedor"
	itemEmblemaDoGuarda      int16 = 4042 // "Emblema_do_Guarda"
)

// quest256NPC trata a entrega de item feita pelos cinco casos legados de NPC da
// Quest 256 (_MSG_Quest.cpp:293/338/382/426/470). Manter o passo e o ticket
// explícitos evita habilitar por acidente rotas de NPC de quest inacabadas.
func (d *Dispatcher) quest256NPC(w *world.World, s *world.Session, e *world.Entity, step quest256Step, ticket int16) {
	// O legado responde cada recusa com SendSay no NPC; este fork usa a
	// mesma caixa de mensagem NoticeReqNotMet que os outros NPCs de quest (perzenExchange,
	// capaverdeTeleport, molarGargula, useQuest256Ticket).
	if e.ClassMaster != classMasterMortal && e.ClassMaster != classMasterArch {
		d.notify(w, s, NoticeReqNotMet) // _NN_Level_Limit2
		return
	}
	if e.Level < step.minLevel || e.Level >= step.maxLevel {
		d.notify(w, s, NoticeReqNotMet) // _NN_Level_limit
		return
	}

	slot := -1
	for i := 0; i < activeCarryLimit(e); i++ {
		if e.Carry[i].Index == ticket {
			slot = i
			break
		}
	}
	if slot < 0 {
		d.notify(w, s, NoticeReqNotMet) // _SN_BRINGITEM
		return
	}

	// BASE_ClearItem limpa o slot legado inteiro (_MSG_Quest.cpp:316); os tickets
	// da Quest 256 não são caminho de consumo de stack (issue #259) mesmo que um item
	// malformado carregue EF_AMOUNT.
	e.Carry[slot] = world.Item{}
	d.sendSlot(w, s, world.ItemPlaceCarry, slot, e.Carry[slot])
	d.teleportQuest256Step(w, s, e, step)
	d.log.Info("quest256 NPC teleport", "conn", s.Conn, "item", ticket, "level", e.Level, "quest_flag", step.flag)
}

var masterGriffTravelDelay = 9500 * time.Millisecond

type masterGriffDestination struct {
	name string
	x, y int16
}

var masterGriffDestinations = []masterGriffDestination{
	{name: "Defensor de Almas", x: 2372, y: 2099},
	{name: "Jardim dos Deuses", x: 2220, y: 1714},
	{name: "Calabouco", x: 2365, y: 2279},
	{name: "Submundo", x: 1826, y: 1771},
}

// mestreGrifo trata o NPC Mestre_Grifo (Merchant 23) entregue no NPCGener.txt
// em Armia. Este atalho de conteúdo do servidor envia o jogador à arena da Quest 256
// que casa com seu nível e precisa definir CMob.QuestFlag antes; senão a guarda
// legada da área retorna o jogador como intruso.
func (d *Dispatcher) mestreGrifo(w *world.World, s *world.Session, e, npc *world.Entity) {
	step, ok := quest256StepForLevel(e.Level)
	if !ok {
		// Fora da faixa nao ha destino: antes o NPC simplesmente nao respondia.
		d.log.Debug("mestre grifo: level outside quest range", "conn", s.Conn, "npc", npc.ID, "level", e.Level)
		d.notify(w, s, NoticeReqNotMet)
		return
	}
	d.teleportQuest256Step(w, s, e, step)
	d.log.Info("mestre grifo teleport", "conn", s.Conn, "npc", npc.ID, "level", e.Level, "quest_flag", step.flag)
}

// masterGriff trata _MSG_MasterGriff (0x0AD9), o pacote que o cliente 7662 envia
// para a UI de warp do Mestre_Grifo. Logs reais do cliente mostram este opcode em vez de
// _MSG_Quest. O cliente toca uma animação de viagem depois de enviá-lo, então o DoTeleport
// final é atrasado um pouco; fazê-lo na hora corta a animação.
func (d *Dispatcher) masterGriff(w *world.World, s *world.Session, _ protocol.Header, payload []byte) {
	e := w.Entity(s.Conn)
	if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
		return
	}
	warpID, ty, ok := protocol.StandardParm2(payload) // MSG_MasterGriff: int WarpID; int Ty
	if !ok {
		return
	}
	dest, ok := masterGriffDestinationForWarpID(warpID)
	if !ok {
		d.log.Debug("master griff: no destination", "conn", s.Conn, "level", e.Level, "warp_id", warpID, "ty", ty)
		d.notify(w, s, NoticeCantUseHere)
		return
	}
	d.log.Info("master griff travel started", "conn", s.Conn, "level", e.Level, "warp_id", warpID, "ty", ty, "dest", dest.name, "x", dest.x, "y", dest.y)
	w.Go(s, func() func(*world.World, *world.Session) {
		time.Sleep(masterGriffTravelDelay)
		return func(w *world.World, s *world.Session) {
			e := w.Entity(s.Conn)
			if e == nil || e.HP <= 0 || s.Mode != world.UserPlay {
				return
			}
			d.doTeleport(w, s, dest.x, dest.y)
			d.log.Info("master griff teleport", "conn", s.Conn, "level", e.Level, "warp_id", warpID, "ty", ty, "dest", dest.name, "x", dest.x, "y", dest.y)
		}
	})
}

func (d *Dispatcher) teleportQuest256Step(w *world.World, s *world.Session, e *world.Entity, step quest256Step) {
	e.QuestFlag = step.flag
	d.doTeleport(w, s, step.x+int16(w.Rand().Intn(5)-3), step.y+int16(w.Rand().Intn(5)-3))
}

func quest256StepForLevel(level int32) (quest256Step, bool) {
	for _, step := range quest256Steps {
		if level >= step.minLevel && level < step.maxLevel {
			return step, true
		}
	}
	return quest256Step{}, false
}

func masterGriffDestinationForWarpID(warpID int32) (masterGriffDestination, bool) {
	switch {
	case warpID == 0:
		return masterGriffDestinations[0], true
	case warpID >= 1 && int(warpID) <= len(masterGriffDestinations):
		return masterGriffDestinations[warpID-1], true
	default:
		return masterGriffDestination{}, false
	}
}

// perzenExchange implementa os NPCs Perzen (_MSG_Quest.cpp PERZEN): se o jogador
// carrega o item que o NPC exige (npc.Carry[0]), consumi-lo e devolver a
// recompensa (npc.Carry[1], uma montaria) no mesmo slot. A troca é totalmente
// dirigida pelos dados do inventário do próprio NPC, então o mesmo código serve toda variante Perzen.
func (d *Dispatcher) perzenExchange(w *world.World, s *world.Session, npc *world.Entity) {
	e := w.Entity(s.Conn)
	input := npc.Carry[0].Index
	reward := npc.Carry[1]
	if e == nil || input == 0 || reward.Index == 0 {
		return
	}
	slot := -1
	for i := 0; i < activeCarryLimit(e); i++ {
		if e.Carry[i].Index == input {
			slot = i
			break
		}
	}
	if slot < 0 {
		// UNVERIFIED: o SendSay original mostra o diálogo "traga o item <nome>"; o
		// formato de pacote de _MSG_NPCQuiz/Say não foi capturado. Falar pelo painel de texto é
		// o caminho confirmado (MSG_MessagePanel, ID 0) e resolve o relato de que a
		// troca da esfera da sorte "não fazia nada".
		d.log.Info("perzen: player lacks input item", "conn", s.Conn, "want", input)
		d.sendClientMessage(w, s, perzenNeedText)
		return
	}
	// Consumir a entrada e conceder a montaria de recompensa com validade de 30 dias
	// (BASE_SetItemDate(30); os itens de recompensa são as montarias "X(30dias)"). A validade
	// é aplicada do lado do servidor no carregamento (dropExpired), independente da
	// codificação de data no item do legado, que é UNVERIFIED.
	e.Carry[slot] = world.Item{
		Index:     reward.Index,
		ExpiresAt: time.Now().Add(mountExpiryDays * 24 * time.Hour).Unix(),
	}
	w.Send(s, protocol.MsgSendItem, protocol.EncodeSendItemBody(protocol.ItemPlaceCarry, slot, itemToSel(e.Carry[slot])))
	// A montaria entra na mochila sem nenhuma confirmação no legado; sem isto a
	// troca parece que não aconteceu (o ícone demora a aparecer na mochila cheia).
	d.sendClientMessage(w, s, perzenDoneText)
	d.log.Info("perzen exchange", "conn", s.Conn, "npc", npc.ID, "input", input, "reward", reward.Index)
}

// capsuleInfo trata _MSG_CapsuleInfo (0x02CD): um repasse puro ao dbServer.
//
// UNVERIFIED: vira um RPC de cash/cápsula do dbServer (Fase 6).
func (d *Dispatcher) capsuleInfo(_ *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	d.log.Debug("CapsuleInfo relay (DB cash RPC pending)", "conn", s.Conn)
}

// putoutSeal trata _MSG_PutoutSeal (0x03CC).
//
// UNVERIFIED: semântica de selo não documentada — stub.
func (d *Dispatcher) putoutSeal(_ *world.World, s *world.Session, _ protocol.Header, _ []byte) {
	d.log.Debug("PutoutSeal not yet implemented (UNVERIFIED)", "conn", s.Conn)
}
