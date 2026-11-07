package handler

import (
	"openwyd-nextgen/tmserver/internal/protocol"
	"openwyd-nextgen/tmserver/internal/refine"
	"openwyd-nextgen/tmserver/internal/world"
)

// Faixa de itens Classe A-E (_MSG_UseItem.cpp:4989): 4016-4020 são os tiers
// simples, 4021-4025 a variante "(P)" — as duas faixas contíguas de 5 itens mapeando
// para o mesmo tier 1..5 (A..E), casadas contra o EF_ITEMLEVEL do alvo.
const classeALo = 4016

// classeTier mapeia o sIndex de um item Classe para seu tier 1..5 (A..E)
// (_MSG_UseItem.cpp:4989). Só faz sentido para itens já classificados como
// volClasses pelo catálogo; quem chama não precisa checar a faixa de idx antes —
// como no legado, que calcula isto aritmeticamente sem checar limites.
func classeTier(idx int16) int {
	return int(idx-classeALo)%5 + 1
}

// useClasseItem é o caminho Classe A-E do _MSG_UseItem (Vol==190,
// _MSG_UseItem.cpp:4958-5012): arrastar um item Classe sobre uma peça de equipamento
// que está no INVENTÁRIO com o grade EF_ITEMLEVEL correspondente sorteia de novo seu sanc
// (limitado a +6) e seus dois atributos de bônus (SetItemBonus2). O item Classe é
// consumido num sorteio bem-sucedido; uma falha de gate ressincroniza o slot arrastado.
//
// O alvo NÃO pode estar equipado (issue #228). O gate legado lê
// `if (!m->DestType || m->DestPos >= 60)` (:4970) — ele rejeita
// DestType==ITEM_PLACE_EQUIP(0) e limita DestPos à faixa do carry. Duas
// coisas confirmam que essa polaridade é proposital, e não o erro de digitação que um
// port anterior tomou: o limite é 60 (maxUnlockedCarry), não 16 (MaxEquip) como nos
// gates genuinamente só-de-equipamento em :3779/:3901/:3979/:4057; e SetItemBonus2
// é seguido só por SendItem, sem recomputar score (:4997-5000), o que
// só é seguro porque o item sorteado de novo não está vestido.
func (d *Dispatcher) useClasseItem(w *world.World, s *world.Session, e *world.Entity, body protocol.MsgUseItemBody, src int) {
	dst := d.itemSlot(w, s, e, int(body.DestType), int(body.DestPos))
	if dst == nil || dst.Empty() {
		return // slot alvo inexistente — o legado só registra e descarta a mensagem
	}

	// O limite companheiro `m->DestPos >= 60` do legado não precisa de port explícito:
	// itemSlot roteia o carry por carrySlotAccessible, que já limita em
	// activeCarryLimit(e) <= maxUnlockedCarry == 60 e devolve nil acima disso —
	// pego pelo retorno dst == nil acima.
	if int(body.DestType) == world.ItemPlaceEquip {
		// O legado usava aqui a string invertida _NN_Only_To_Equips. Dê ao
		// jogador um erro acionável preservando a regra de só inventário.
		d.sendRefineMessage(w, s, classeMessageInventory)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	mobType := d.itemAbility(*dst, efMobType)
	if itemSanc(*dst) >= 10 || (mobType != 0 && mobType != 2) {
		d.sendRefineMessage(w, s, classeMessageCantRefine)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	if d.itemAbility(*dst, efItemLevel) != classeTier(e.Carry[src].Index) {
		d.sendRefineMessage(w, s, classeMessageMismatch)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	// Desvio deliberado do SetItemBonus2 (Server.cpp:2719-2861), que não tem
	// ramo para um nPos fora de helm/chest/legs/glove/boot e por isso cai
	// adiante e consome o item Classe sem efeito — o bug legado conhecido
	// registrado em "Source/Lista de bugs.txt:6". Recuse e ressincronize em vez de
	// destruir o item.
	if !refine.ClasseBonus(dst, d.itemPos[int(dst.Index)], func(n int) int { return w.Rand().Intn(n) }, d.itemAbility) {
		d.sendRefineMessage(w, s, classeMessageCantRefine)
		d.sendSlot(w, s, world.ItemPlaceCarry, src, e.Carry[src])
		return
	}

	d.sendSlot(w, s, int(body.DestType), int(body.DestPos), *dst)
	d.sendRefineMessage(w, s, refineMessageSuccess)
	d.sendRefineMotion(w, s, e.ID, refineMotionSuccess, refineMotionSuccessParm)
	d.log.Info("classe item bonus reroll", "conn", s.Conn, "item", dst.Index, "effects", dst.Effects)
	consumeOneItem(&e.Carry[src])
	// O slot Classe deliberadamente NÃO é reenviado no sucesso, espelhando
	// refineSucceed: o cliente já removeu o item que arrastou.
}
